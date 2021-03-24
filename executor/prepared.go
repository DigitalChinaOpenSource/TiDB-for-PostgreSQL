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
	"math"
	"sort"
	"time"

	"github.com/pingcap/errors"
	"github.com/pingcap/log"
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
	"go.uber.org/zap"
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
func NewPrepareExec(ctx sessionctx.Context, is infoschema.InfoSchema, sqlTxt string) *PrepareExec {
	base := newBaseExecutor(ctx, nil, 0)
	base.initCap = chunk.ZeroCapacity
	return &PrepareExec{
		baseExecutor: base,
		is:           is,
		sqlText:      sqlTxt,
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

	err = plannercore.Preprocess(e.ctx, stmt, e.is, plannercore.InPrepare)
	if err != nil {
		return err
	}

	// The parameter markers are appended in visiting order, which may not
	// be the same as the position order in the query string. We need to
	// sort it by position.
	sorter := &paramMarkerSorter{markers: extractor.markers}
	sort.Sort(sorter)
	e.ParamCount = len(sorter.markers)
	for i := 0; i < e.ParamCount; i++ {
		sorter.markers[i].SetOrder(i)
	}
	prepared := &ast.Prepared{
		Stmt:          stmt,
		StmtType:      GetStmtLabel(stmt),
		Params:        sorter.markers,
		SchemaVersion: e.is.SchemaMetaVersion(),
	}

	prepared.UseCache = plannercore.PreparedPlanCacheEnabled() && plannercore.Cacheable(stmt, e.is)

	// We try to build the real statement of preparedStmt.
	for i := range prepared.Params {
		param := prepared.Params[i].(*driver.ParamMarkerExpr)
		param.Datum.SetNull()
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
	}

	if _, ok := stmt.(*ast.SelectStmt); ok {
		e.Fields = colNames2ResultFields(p.Schema(), p.OutputNames(), vars.CurrentDB)
	}
	if e.ID == 0 {
		e.ID = vars.GetNextPreparedStmtID()
	}
	if e.name != "" {
		vars.PreparedStmtNameToID[e.name] = e.ID
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

// SetInsertParamTypeArray 当计划为insert时，将其参数设置到prepared.Param的Type成员中去
//	insertPlan：计划结构体
//	paramExprs：prepared结构体中的Param成员，其中有Type信息，默认都是空的，我们就是要设置这些Type信息
// 目前支持的sql句式：
// insert into table1 values(xxx,xxx), (yyy,yyy)
// insert into table1 where condition1 and condition2 and ...
// insert into table1 where condition2 or condition and
// todo 复杂嵌套条件情况还未支持
// PGSQL Modified
func SetInsertParamType(insertPlan *plannercore.Insert, paramExprs *[]ast.ParamMarkerExpr) {
	if insertPlan.SelectPlan != nil {
		insertPlan.SelectPlan.SetParamType(paramExprs)
	} else {
		//当前计划的tableSchema，也就是表结构。
		paramIndex := 0
		cols := insertPlan.GetTableSchema().Columns
		//lists是参数列表，需要考虑一次性insert多行的情况，在这种情况下，lists数组将有多个元素，只需要将它们依次放入数组中返回即可。
		lists := insertPlan.Lists
		for i := range lists {
			list := lists[i]
			for j := range list {
				cst := list[j].(*expression.Constant)
				//Kind()将返回Datum对象的k成员，我们通过这个标志知道这个参数是传进来值了，还是暂时用？占位。
				if cst.Value.Kind() == 0 {
					if paramMakerExpr, ok := (*paramExprs)[paramIndex].(*driver.ParamMarkerExpr); ok{
						paramMakerExpr.TexprNode.Type = *(cols[j].RetType)
						paramIndex++
					}
				}
			}
		}
	}
}

// SetSelectParamType 当语句为select，获取其参数类型并设置到prepared.Param中去
func SetSelectParamType(projection *plannercore.LogicalProjection, params *[]ast.ParamMarkerExpr) {
	projection.SetParamType(params,nil,0)
}

// SetDeleteParamType 为delete语句设置参数类型
func SetDeleteParamType(delete *plannercore.Delete, params *[]ast.ParamMarkerExpr) {
	if delete.SelectPlan != nil {
		delete.SelectPlan.SetParamType(params)
	}
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
		log.Warn("rebuild plan in EXECUTE statement failed", zap.String("labelName of PREPARE statement", e.name))
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
