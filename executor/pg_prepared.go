package executor

import (
	"context"
	"github.com/pingcap/errors"
	"github.com/pingcap/tidb/expression"
	"github.com/pingcap/tidb/parser"
	"github.com/pingcap/tidb/parser/ast"
	plannercore "github.com/pingcap/tidb/planner/core"
	"github.com/pingcap/tidb/sessionctx"
	"github.com/pingcap/tidb/sessionctx/variable"
	driver "github.com/pingcap/tidb/types/parser_driver"
	"github.com/pingcap/tidb/util"
	"github.com/pingcap/tidb/util/chunk"
	"github.com/pingcap/tidb/util/hint"
	"github.com/pingcap/tidb/util/sqlexec"
	"github.com/pingcap/tidb/util/topsql"
	"math"
	"sort"
	"strings"
)

// NewPrepareExec creates a new PrepareExec.
func NewPrepareExec(ctx sessionctx.Context, sqlTxt string, name string) *PrepareExec {
	base := newBaseExecutor(ctx, nil, 0)
	base.initCap = chunk.ZeroCapacity
	return &PrepareExec{
		baseExecutor: base,
		sqlText:      sqlTxt,
		name:         name,
		needReset:    true,
	}
}

// Next implements the Executor Next interface.
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
		// FIXME: ok... yet another parse API, may need some api interface clean.
		stmts, _, err = sqlParser.ParseSQL(ctx, e.sqlText,
			parser.CharsetConnection(charset),
			parser.CollationConnection(collation))
	} else {
		p := parser.New()
		p.SetParserConfig(vars.BuildParserConfig())
		var warns []error
		stmts, warns, err = p.ParseSQL(e.sqlText,
			parser.CharsetConnection(charset),
			parser.CollationConnection(collation))
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

	if e.needReset {
		err = ResetContextOfStmt(e.ctx, stmt)
		if err != nil {
			return err
		}
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

	ret := &plannercore.PreprocessorReturn{}
	err = plannercore.Preprocess(e.ctx, stmt, plannercore.InPrepare, plannercore.WithPreprocessorReturn(ret))
	if err != nil {
		return err
	}

	//// The parameter markers are appended in visiting order, which may not
	//// be the same as the position order in the query string. We need to
	//// sort it by position.
	//sorter := &paramMarkerSorter{markers: extractor.markers}
	//sort.Sort(sorter)
	//e.ParamCount = len(sorter.markers)
	//for i := 0; i < e.ParamCount; i++ {
	//	sorter.markers[i].SetOrder(i)
	//}

	//Pgsql extend query has its own order
	//sort according to its own order
	err = ParamMakerSorter(extractor.markers)
	if err != nil {
		return err
	}

	// set paramCount to PrepareExec.It was used in step 'handleDescription'
	e.ParamCount = len(extractor.markers)

	prepared := &ast.Prepared{
		Stmt:          stmt,
		StmtType:      GetStmtLabel(stmt),
		Params:        extractor.markers,
		SchemaVersion: ret.InfoSchema.SchemaMetaVersion(),
	}
	normalizedSQL, digest := parser.NormalizeDigest(prepared.Stmt.Text())
	if variable.TopSQLEnabled() {
		ctx = topsql.AttachSQLInfo(ctx, normalizedSQL, digest, "", nil, vars.InRestrictedSQL)
	}

	if !plannercore.PreparedPlanCacheEnabled() {
		prepared.UseCache = false
	} else {
		prepared.UseCache = plannercore.CacheableWithCtx(e.ctx, stmt, ret.InfoSchema)
	}

	// We try to build the real statement of preparedStmt.
	for i := range prepared.Params {
		param := prepared.Params[i].(*driver.ParamMarkerExpr)
		param.Datum.SetNull()
		param.InExecute = false
	}
	var p plannercore.Plan
	e.ctx.GetSessionVars().PlanID = 0
	e.ctx.GetSessionVars().PlanColumnID = 0
	destBuilder, _ := plannercore.NewPlanBuilder().Init(e.ctx, ret.InfoSchema, &hint.BlockHintProcessor{})
	p, err = destBuilder.Build(ctx, stmt)
	if err != nil {
		return err
	}

	// according to plan type. Get param type from the plan tree.
	switch p.(type) {
	case *plannercore.Insert:
		err = SetInsertParamType(p.(*plannercore.Insert), &prepared.Params)
	case *plannercore.LogicalProjection:
		err = SetSelectParamType(p.(*plannercore.LogicalProjection), &prepared.Params)
	case *plannercore.Delete:
		err = SetDeleteParamType(p.(*plannercore.Delete), &prepared.Params)
	case *plannercore.Update:
		err = SetUpdateParamType(p.(*plannercore.Update), &prepared.Params)
	case *plannercore.LogicalSort:
		err = SetSortType(p.(*plannercore.LogicalSort), &prepared.Params)
	case *plannercore.LogicalLimit:
		err = SetLimitType(p.(*plannercore.LogicalLimit), &prepared.Params)
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

	preparedObj := &plannercore.CachedPrepareStmt{
		PreparedAst:         prepared,
		VisitInfos:          destBuilder.GetVisitInfo(),
		NormalizedSQL:       normalizedSQL,
		SQLDigest:           digest,
		ForUpdateRead:       destBuilder.GetIsForUpdateRead(),
		SnapshotTSEvaluator: ret.SnapshotTSEvaluator,
	}
	return vars.AddPreparedStmt(e.ID, preparedObj)
}

// ParamMakerSorter sort by order.
func ParamMakerSorter(markers []ast.ParamMarkerExpr) error {
	// nothing to sort
	if len(markers) == 0 {
		return nil
	}
	// first we sort the given markers by their original order
	sort.Slice(markers, func(i, j int) bool {
		return markers[i].(*driver.ParamMarkerExpr).Order < markers[j].(*driver.ParamMarkerExpr).Order
	})

	// if the smallest order is 0, then we are in mySQL compatible mode
	mySQLCompatibleMode := markers[0].(*driver.ParamMarkerExpr).Order == 0

	// then we check for any error that might exist
	// this checks that there's no mix use of mySQL and postgres' param notation
	// if the first element has order 0, then the last element's order should be 0
	if mySQLCompatibleMode && markers[len(markers)-1].(*driver.ParamMarkerExpr).Order != 0 {
		return errors.Errorf("Mix Use of $ notation and ? notation, or use $0")
	}

	// while checking for repeated use, we change the slice to be 1 indexed (Compatibility with mysql's ?):
	if mySQLCompatibleMode {
		sort.Slice(markers, func(i, j int) bool {
			return markers[i].(*driver.ParamMarkerExpr).Offset < markers[j].(*driver.ParamMarkerExpr).Offset
		})

		for markerIndex, marker := range markers {
			marker.SetOrder(markerIndex) // note that this is 0 indexed
		}
	} else {
		for markerIndex, marker := range markers {
			// TODO: Add support for reusing same paramMarker and remove this check
			if marker.(*driver.ParamMarkerExpr).Order != markerIndex+1 {
				return errors.Errorf("Repeated use of same parameter expression not currently supported")
			}
			marker.SetOrder(marker.(*driver.ParamMarkerExpr).Order - 1) // make it 0 indexed
		}
	}

	return nil
}

//SetInsertParamType when the plan is insert, set the type of parameter expression
func SetInsertParamType(insertPlan *plannercore.Insert, paramExprs *[]ast.ParamMarkerExpr) error {
	// if insertPlan already have a select plan,we could use that to set parameter expressions' type
	if insertPlan.SelectPlan != nil {
		err := insertPlan.SelectPlan.SetParamType(paramExprs)
		return err
	}

	// do nothing if no param to set
	if *paramExprs == nil {
		return nil
	}

	// columns of the table according to the schema, note the order of the columns is set during table definition
	// aka, a table defined as 'test(a, b)' have schema columns [a, b]
	// It holds the correct type information we want
	schemaColumns := insertPlan.GetTableSchema().Columns

	// queryColumn is the column that we are inserting into, note the order here is the same as the sql statement
	// aka, for sql:  "...insert into table(b, a) ... " have query columns [b, a]
	// It holds the order in which we insert
	queryColumns := insertPlan.Columns

	// insertLists is a list of insert values list, it's a list of list for bulk insert
	// aka, sql 'insert into .... values (1, 2), (3, ?) have insert lists [[1, 2], [3, ?]]'
	// It holds the information about which element is a value expression, so we loop through this one
	insertLists := insertPlan.Lists

	for _, insertList := range insertLists {
		for queryOrder := range insertList {
			exprConst := insertList[queryOrder].(*expression.Constant)
			exprOrder := exprConst.Order   // the order of the value expression as they appear on paramExpr
			exprOffset := exprConst.Offset // the offset of the value expression
			// the if the query doesn't specify order, aka 'insert into test values ...', we simply set according to insert order
			if queryColumns == nil {
				if exprOffset != 0 { // a non-zero offset indicates a value expression
					setParam(paramExprs, exprOrder, schemaColumns[queryOrder])
				}
			} else {
				if exprOffset != 0 { // a non-zero offset indicates a value expression
					constShortName := queryColumns[queryOrder].Name.O
					setParamByColName(schemaColumns, constShortName, paramExprs, exprOrder)
				}
			}
		}
	}
	return nil
}

// a helper function that set the specific parameter expression's type to the type of given column's name
func setParamByColName(schemaColumns []*expression.Column, targetName string, target *[]ast.ParamMarkerExpr, targetOrder int) {
	for _, schemaColumn := range schemaColumns { //loop through schema column to find matching name
		schemaNameSplit := strings.Split(schemaColumn.OrigName, ".")
		schemaShortName := schemaNameSplit[len(schemaNameSplit)-1]
		if schemaShortName == targetName {
			setParam(target, targetOrder, schemaColumn)
		}
	}
}

// a helper function that set the specific parameter expression to the type of given column
func setParam(paramExprs *[]ast.ParamMarkerExpr, targetOrder int, givenColumn *expression.Column) {
	if targetParamExpression, ok := (*paramExprs)[targetOrder].(*driver.ParamMarkerExpr); ok {
		targetParamExpression.TexprNode.Type = *givenColumn.RetType
	}
}

// SetSelectParamType 从select计划中获取参数类型
func SetSelectParamType(projection *plannercore.LogicalProjection, params *[]ast.ParamMarkerExpr) error {
	return projection.SetParamType(params)
}

// SetDeleteParamType 从delete计划中获取参数类型
func SetDeleteParamType(delete *plannercore.Delete, params *[]ast.ParamMarkerExpr) error {
	if delete.SelectPlan != nil {
		return delete.SelectPlan.SetParamType(params)
	}
	return nil
}

// SetUpdateParamType 从update计划获取参数类型
func SetUpdateParamType(update *plannercore.Update, params *[]ast.ParamMarkerExpr) error {
	if list := update.OrderedList; list != nil {
		for _, l := range list {
			SetUpdateParamTypes(l, params, &update.SelectPlan.Schema().Columns)
		}
	}
	return update.SelectPlan.SetParamType(params)
}

// SetUpdateParamTypes 这里是处理 update table set name = ?, age = ?这样的位置的参数的
func SetUpdateParamTypes(assignmnet *expression.Assignment, paramExprs *[]ast.ParamMarkerExpr, cols *[]*expression.Column) {
	if constant, ok := assignmnet.Expr.(*expression.Constant); ok {
	cycle:
		for _, col := range *cols {
			for _, expr := range *paramExprs {
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
func SetSortType(sort *plannercore.LogicalSort, i *[]ast.ParamMarkerExpr) error {
	return sort.SetParamType(i)
}

// SetLimitType set the parameter type of limit plan
func SetLimitType(limit *plannercore.LogicalLimit, i *[]ast.ParamMarkerExpr) error {
	return limit.SetParamType(i)
}
