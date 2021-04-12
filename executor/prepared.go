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

package executor

import (
	"context"
	"github.com/pingcap/errors"
	"github.com/pingcap/parser"
	"github.com/pingcap/parser/ast"
	"github.com/pingcap/parser/mysql"
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

	//根据parammaker自带的的order成员排序
	ParamMakerSortor(extractor.markers)

	err = plannercore.Preprocess(e.ctx, stmt, e.is, plannercore.InPrepare)
	if err != nil {
		return err
	}

	// The parameter markers are appended in visiting order, which may not
	// be the same as the position order in the query string. We need to
	// sort it by position.
	//sorter := &paramMarkerSorter{markers: extractor.markers}
	//sort.Sort(sorter)
	//e.ParamCount = len(sorter.markers)
	e.ParamCount = len(extractor.markers)
	//for i := 0; i < e.ParamCount; i++ {
	//	sorter.markers[i].SetOrder(i)
	//}
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
		//param.Datum.SetNull()
		//每个datum设置类型为interface，保证计划中的条件都会存在不会被优化掉。
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
		// 当没有 Stmt 没有Name时，则表示该预处理语句为临时语句，我们会分配 0 作为其 Name
		// 后面在获取临时预处理语句 ID 的时候，通过 0 获取
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

func ParamMakerSortor(markers []ast.ParamMarkerExpr) {
	for i := 0; i< len(markers); i++ {
		for j := i + 1; j < len(markers); j++ {
			if markers[j].(*driver.ParamMarkerExpr).Order < markers[i].(*driver.ParamMarkerExpr).Order {
				markers[i], markers[j] = markers[j], markers[i]
			}
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

// SetSortType
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
