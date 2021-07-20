// Copyright 2015 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// See the License for the specific language governing permissions and
// limitations under the License.

// Copyright 2021 Digital China Group Co.,Ltd
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// See the License for the specific language governing permissions and
// limitations under the License.

package executor

import (
	"context"
	"github.com/pingcap/errors"
	"github.com/DigitalChinaOpenSource/DCParser"
	"github.com/DigitalChinaOpenSource/DCParser/ast"
	"github.com/DigitalChinaOpenSource/DCParser/mysql"
	"github.com/pingcap/tidb/expression"
	"github.com/pingcap/tidb/infoschema"
	"github.com/pingcap/tidb/planner"
	plannercore "github.com/pingcap/tidb/planner/core"
	"github.com/pingcap/tidb/sessionctx"
	"github.com/pingcap/tidb/types"
	driver "github.com/pingcap/tidb/types/parser_driver"
	"github.com/pingcap/tidb/util"
	"github.com/pingcap/tidb/util/chunk"
	"github.com/pingcap/tidb/util/hint"
	"github.com/pingcap/tidb/util/sqlexec"
	"math"
	"strings"
	"time"
)

var (
	_ Executor = &DeallocateExec{}
	_ Executor = &ExecuteExec{}
	_ Executor = &PrepareExec{}
)

type paramMarkerSorter struct {
	markers []ast.ParamMarkerExpr
}

func (p *paramMarkerSorter) Len() int {
	return len(p.markers)
}

func (p *paramMarkerSorter) Less(i, j int) bool {
	return p.markers[i].(*driver.ParamMarkerExpr).Offset < p.markers[j].(*driver.ParamMarkerExpr).Offset
}

func (p *paramMarkerSorter) Swap(i, j int) {
	p.markers[i], p.markers[j] = p.markers[j], p.markers[i]
}

type paramMarkerExtractor struct {
	markers []ast.ParamMarkerExpr
}

func (e *paramMarkerExtractor) Enter(in ast.Node) (ast.Node, bool) {
	return in, false
}

func (e *paramMarkerExtractor) Leave(in ast.Node) (ast.Node, bool) {
	if x, ok := in.(*driver.ParamMarkerExpr); ok {
		e.markers = append(e.markers, x)
	}
	return in, true
}

// PrepareExec represents a PREPARE executor.
type PrepareExec struct {
	baseExecutor

	is      infoschema.InfoSchema
	name    string
	sqlText string

	ID         uint32
	ParamCount int
	Fields     []*ast.ResultField
}

// NewPrepareExec creates a new PrepareExec.
func NewPrepareExec(ctx sessionctx.Context, is infoschema.InfoSchema, sqlTxt string, name string) *PrepareExec {
	base := newBaseExecutor(ctx, nil, 0)
	base.initCap = chunk.ZeroCapacity
	return &PrepareExec{
		baseExecutor: base,
		is:           is,
		sqlText:      sqlTxt,
		name:         name,
	}
}

// Next implements the Executor Next interface.
// 通过过程中生成的计划p，设置参数类型
// PGSQL Modified
func (e *PrepareExec) Next(ctx context.Context, req *chunk.Chunk) error {
	vars := e.ctx.GetSessionVars()
	if e.ID != 0 {
		// Must be the case when we retry a prepare.
		// Make sure it is idempotent.
		_, ok := vars.PreparedStmts[e.ID]
		if ok {
			return nil
		}
	}
	charset, collation := vars.GetCharsetInfo()
	var (
		stmts []ast.StmtNode
		err   error
	)
	if sqlParser, ok := e.ctx.(sqlexec.SQLParser); ok {
		stmts, err = sqlParser.ParseSQL(e.sqlText, charset, collation)
	} else {
		p := parser.New()
		p.EnableWindowFunc(vars.EnableWindowFunction)
		var warns []error
		stmts, warns, err = p.Parse(e.sqlText, charset, collation)
		for _, warn := range warns {
			e.ctx.GetSessionVars().StmtCtx.AppendWarning(util.SyntaxWarn(warn))
		}
	}
	if err != nil {
		return util.SyntaxError(err)
	}
	if len(stmts) != 1 {
		return ErrPrepareMulti
	}
	stmt := stmts[0]

	err = ResetContextOfStmt(e.ctx, stmt)
	if err != nil {
		return err
	}

	var extractor paramMarkerExtractor
	stmt.Accept(&extractor)

	// DDL Statements can not accept parameters
	if _, ok := stmt.(ast.DDLNode); ok && len(extractor.markers) > 0 {
		return ErrPrepareDDL
	}

	switch stmt.(type) {
	case *ast.LoadDataStmt, *ast.PrepareStmt, *ast.ExecuteStmt, *ast.DeallocateStmt:
		return ErrUnsupportedPs
	}

	// Prepare parameters should NOT over 2 bytes(MaxUint16)
	// https://dev.mysql.com/doc/internals/en/com-stmt-prepare-response.html#packet-COM_STMT_PREPARE_OK.
	if len(extractor.markers) > math.MaxUint16 {
		return ErrPsManyParam
	}

	//Pgsql extend query has its own order
	//sort according to its own order
	ParamMakerSortor(extractor.markers)

	err = plannercore.Preprocess(e.ctx, stmt, e.is, plannercore.InPrepare)

	if err != nil {
		return err
	}

	// set paramCount to PrepareExec.It was used in step 'handleDescription'
	e.ParamCount = len(extractor.markers)

	// set Params use extractor's member 'markers'
	prepared := &ast.Prepared{
		Stmt:          stmt,
		StmtType:      GetStmtLabel(stmt),
		Params:        extractor.markers,
		SchemaVersion: e.is.SchemaMetaVersion(),
	}
	prepared.UseCache = plannercore.PreparedPlanCacheEnabled() && plannercore.Cacheable(stmt, e.is)

	// We try to build the real statement of preparedStmt.
	for i := range prepared.Params {
		param := prepared.Params[i].(*driver.ParamMarkerExpr)
		//set every paramType to interface. It keeps every param in the plan tree whatever the param is Primary key index
		param.Datum.SetInterfaceType()
		param.InExecute = false
	}
	var p plannercore.Plan
	e.ctx.GetSessionVars().PlanID = 0
	e.ctx.GetSessionVars().PlanColumnID = 0
	destBuilder, _ := plannercore.NewPlanBuilder(e.ctx, e.is, &hint.BlockHintProcessor{})
	p, err = destBuilder.Build(ctx, stmt)
	if err != nil {
		return err
	}
	// according to plan type. Get param type from the plan tree.
	switch p.(type) {
	case *plannercore.Insert:
		SetInsertParamType(p.(*plannercore.Insert), &prepared.Params)
	case *plannercore.LogicalProjection:
		SetSelectParamType(p.(*plannercore.LogicalProjection), &prepared.Params)
	case *plannercore.Delete:
		SetDeleteParamType(p.(*plannercore.Delete), &prepared.Params)
	case *plannercore.Update:
		SetUpdateParamType(p.(*plannercore.Update), &prepared.Params)
	case *plannercore.LogicalSort:
		SetSortType(p.(*plannercore.LogicalSort), &prepared.Params)
	}
	if _, ok := stmt.(*ast.SelectStmt); ok {
		e.Fields = colNames2ResultFields(p.Schema(), p.OutputNames(), vars.CurrentDB)
	}
	if e.ID == 0 {
		e.ID = vars.GetNextPreparedStmtID()
	}
	if e.name != "" {
		vars.PreparedStmtNameToID[e.name] = e.ID
	} else {
		// When Stmt does not have a Name, it means that the prepared statement is a temporary statement, and we will assign 0 as its Name
		// When obtaining the temporary prepared statement ID later, pass 0 to obtain
		vars.PreparedStmtNameToID["0"] = e.ID
	}

	normalized, digest := parser.NormalizeDigest(prepared.Stmt.Text())
	preparedObj := &plannercore.CachedPrepareStmt{
		PreparedAst:   prepared,
		VisitInfos:    destBuilder.GetVisitInfo(),
		NormalizedSQL: normalized,
		SQLDigest:     digest,
	}
	return vars.AddPreparedStmt(e.ID, preparedObj)
}

// ParamMakerSortor sort by order.
// in the query, most situations are in order.so bubble sort and insert sort are Preferred
// we choose insert sort here.
// todo: According to different parameters situations, choose the most suitable sorting method
func ParamMakerSortor(markers []ast.ParamMarkerExpr) {
	if len(markers) <=  1 {
		return
	}

	var val ast.ParamMarkerExpr
	var index int
	for i := 1; i < len(markers); i++ {
		val, index = markers[i], i -1
		for {
			if val.(*driver.ParamMarkerExpr).Order < markers[index].(*driver.ParamMarkerExpr).Order ||
				(val.(*driver.ParamMarkerExpr).Order == markers[index].(*driver.ParamMarkerExpr).Order &&
					val.(*driver.ParamMarkerExpr).Offset < markers[index].(*driver.ParamMarkerExpr).Offset){
				markers[index + 1] = markers[index]
			} else {
				break
			}
			index--
			if index < 0 {
				break
			}
		}
		markers[index + 1] = val
	}

	//todo Eliminate compatibility with "?"

	// If more than two ParamMarkerExpr.Order are zero, it means that the placeholder is "?".
	// So we need reassign order.
	if markers[1].(*driver.ParamMarkerExpr).Order == 0 {
		for i := 0; i < len(markers); i++ {
			markers[i].SetOrder(i)
		}
	}
}


// SetInsertParamTypeArray 当计划为insert时，将其参数设置到prepared.Param的Type成员中去
//	insertPlan：计划结构体
//	paramExprs：prepared结构体中的Param成员，其中有Type信息，默认都是空的，我们就是要设置这些Type信息
func SetInsertParamType(insertPlan *plannercore.Insert, paramExprs *[]ast.ParamMarkerExpr) {
	if insertPlan.SelectPlan != nil {
		insertPlan.SelectPlan.SetParamType(paramExprs)
	} else {
		paramIndex := 0
		//当前计划的tableSchema，也就是表结构。
		cols := insertPlan.GetTableSchema().Columns
		// 这里有参数传进来的顺序。
		orderedColumns := insertPlan.Columns

		//lists是参数列表，需要考虑一次性insert多行的情况，在这种情况下，lists数组将有多个元素，只需要将它们依次放入数组中返回即可。
		for _, list := range insertPlan.Lists {
			for j := range list {
				if orderedColumns != nil {
					if cst := list[j].(*expression.Constant); cst.Offset != 0 {
						if paramMakerExpr, ok := (*paramExprs)[paramIndex].(*driver.ParamMarkerExpr); ok {
							for _, col := range cols {
								nameSplit := strings.Split(col.OrigName,".")
								shortName := nameSplit[len(nameSplit) - 1]
								if shortName == orderedColumns[cst.Order - 1].Name.O {
									paramMakerExpr.TexprNode.Type = *col.RetType
									paramIndex++
								}
							}
						}
					}
				} else {
					if paramMakerExpr, ok := (*paramExprs)[paramIndex].(*driver.ParamMarkerExpr); ok {
						paramMakerExpr.TexprNode.Type = *cols[j].RetType
					}
				}

			}
		}
	}
}

// SetSelectParamType 从select计划中获取参数类型
func SetSelectParamType(projection *plannercore.LogicalProjection, params *[]ast.ParamMarkerExpr) {
	projection.SetParamType(params)
}

// SetDeleteParamType 从delete计划中获取参数类型
func SetDeleteParamType(delete *plannercore.Delete, params *[]ast.ParamMarkerExpr) {
	if delete.SelectPlan != nil {
		delete.SelectPlan.SetParamType(params)
	}
}

// SetUpdateParamType 从update计划获取参数类型
func SetUpdateParamType(update *plannercore.Update, params *[]ast.ParamMarkerExpr) {
	if list := update.OrderedList; list != nil {
		for _,l := range list {
			SetUpdateParamTypes(l,params,&update.SelectPlan.Schema().Columns)
		}
	}
	update.SelectPlan.SetParamType(params)
}

// SetUpdateParamTypes 这里是处理 update table set name = ?, age = ?这样的位置的参数的
func SetUpdateParamTypes(assignmnet *expression.Assignment, paramExprs *[]ast.ParamMarkerExpr, cols *[]*expression.Column) {
	if constant, ok := assignmnet.Expr.(*expression.Constant); ok {
	cycle:
		for _, col := range *cols {
			for _,expr := range *paramExprs {
				if paramMarker, ok := expr.(*driver.ParamMarkerExpr); ok && col.OrigName == assignmnet.Col.OrigName &&
					paramMarker.Offset == constant.Offset {
					paramMarker.TexprNode.Type = *col.RetType
					break cycle
				}
			}
		}
	}
}

// SetSortType 从根节点计划是logicalSort的计划中获取参数类型
func SetSortType(sort *plannercore.LogicalSort, i *[]ast.ParamMarkerExpr) {
	sort.SetParamType(i)
}


// ExecuteExec represents an EXECUTE executor.
// It cannot be executed by itself, all it needs to do is to build
// another Executor from a prepared statement.
type ExecuteExec struct {
	baseExecutor

	is            infoschema.InfoSchema
	name          string
	usingVars     []expression.Expression
	stmtExec      Executor
	stmt          ast.StmtNode
	plan          plannercore.Plan
	id            uint32
	lowerPriority bool
	outputNames   []*types.FieldName
}

// Next implements the Executor Next interface.
func (e *ExecuteExec) Next(ctx context.Context, req *chunk.Chunk) error {
	return nil
}

// Build builds a prepared statement into an executor.
// After Build, e.StmtExec will be used to do the real execution.
func (e *ExecuteExec) Build(b *executorBuilder) error {
	if snapshotTS := e.ctx.GetSessionVars().SnapshotTS; snapshotTS != 0 {
		if err := e.ctx.InitTxnWithStartTS(snapshotTS); err != nil {
			return err
		}
	} else {
		ok, err := plannercore.IsPointGetWithPKOrUniqueKeyByAutoCommit(e.ctx, e.plan)
		if err != nil {
			return err
		}
		if ok {
			err = e.ctx.InitTxnWithStartTS(math.MaxUint64)
			if err != nil {
				return err
			}
		}
	}
	stmtExec := b.build(e.plan)
	if b.err != nil {
		//log.Warn("rebuild plan in EXECUTE statement failed", zap.String("labelName of PREPARE statement", e.name))
		return errors.Trace(b.err)
	}
	e.stmtExec = stmtExec
	if e.ctx.GetSessionVars().StmtCtx.Priority == mysql.NoPriority {
		e.lowerPriority = needLowerPriority(e.plan)
	}
	return nil
}

// DeallocateExec represent a DEALLOCATE executor.
type DeallocateExec struct {
	baseExecutor

	Name string
}

// Next implements the Executor Next interface.
func (e *DeallocateExec) Next(ctx context.Context, req *chunk.Chunk) error {
	vars := e.ctx.GetSessionVars()
	id, ok := vars.PreparedStmtNameToID[e.Name]
	if !ok {
		return errors.Trace(plannercore.ErrStmtNotFound)
	}
	preparedPointer := vars.PreparedStmts[id]
	preparedObj, ok := preparedPointer.(*plannercore.CachedPrepareStmt)
	if !ok {
		return errors.Errorf("invalid CachedPrepareStmt type")
	}
	prepared := preparedObj.PreparedAst
	delete(vars.PreparedStmtNameToID, e.Name)
	if plannercore.PreparedPlanCacheEnabled() {
		e.ctx.PreparedPlanCache().Delete(plannercore.NewPSTMTPlanCacheKey(
			vars, id, prepared.SchemaVersion,
		))
	}
	vars.RemovePreparedStmt(id)
	return nil
}

// CompileExecutePreparedStmt compiles a session Execute command to a stmt.Statement.
func CompileExecutePreparedStmt(ctx context.Context, sctx sessionctx.Context,
	ID uint32, args []types.Datum) (sqlexec.Statement, error) {
	startTime := time.Now()
	defer func() {
		sctx.GetSessionVars().DurationCompile = time.Since(startTime)
	}()
	execStmt := &ast.ExecuteStmt{ExecID: ID}
	if err := ResetContextOfStmt(sctx, execStmt); err != nil {
		return nil, err
	}
	execStmt.BinaryArgs = args
	is := infoschema.GetInfoSchema(sctx)
	execPlan, names, err := planner.Optimize(ctx, sctx, execStmt, is)
	if err != nil {
		return nil, err
	}

	stmt := &ExecStmt{
		GoCtx:       ctx,
		InfoSchema:  is,
		Plan:        execPlan,
		StmtNode:    execStmt,
		Ctx:         sctx,
		OutputNames: names,
	}
	if preparedPointer, ok := sctx.GetSessionVars().PreparedStmts[ID]; ok {
		preparedObj, ok := preparedPointer.(*plannercore.CachedPrepareStmt)
		if !ok {
			return nil, errors.Errorf("invalid CachedPrepareStmt type")
		}
		stmtCtx := sctx.GetSessionVars().StmtCtx
		stmt.Text = preparedObj.PreparedAst.Stmt.Text()
		stmtCtx.OriginalSQL = stmt.Text
		stmtCtx.InitSQLDigest(preparedObj.NormalizedSQL, preparedObj.SQLDigest)
	}
	return stmt, nil
}
