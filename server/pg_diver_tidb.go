package server

import (
	"github.com/pingcap/tidb/parser/auth"
	"github.com/pingcap/tidb/parser/mysql"
	"github.com/pingcap/tidb/types"
)

type TiDBStatement struct {
	id           uint32
	numParams    int
	boundParams  [][]byte
	paramsType   []byte
	ctx          *TiDBContext
	rs           ResultSet
	sql          string
	columnInfo   []*ColumnInfo
	args         []types.Datum
	resultFormat []int16
	paramOIDs    []uint32
}

// SetColumnInfo 设置 TiDBStatement 中返回行数据信息
// 如果有返回数据的语句，则存储返回数据的结构，如果没有则为空
func (ts *TiDBStatement) SetColumnInfo(columns []*ColumnInfo) {
	ts.columnInfo = columns
}

// GetColumnInfo 获取 TiDBStatement 中返回行数据信息
func (ts *TiDBStatement) GetColumnInfo() []*ColumnInfo {
	return ts.columnInfo
}


// GetArgs 获取绑定后的参数值
func (ts *TiDBStatement) GetArgs() []types.Datum {
	return ts.args
}

// SetArgs 进行参数绑定值
func (ts *TiDBStatement) SetArgs(args []types.Datum) {
	ts.args = args
}

// GetResultFormat 获取结果返回的格式 0 为 Text, 1 为 Binary
func (ts *TiDBStatement) GetResultFormat() []int16 {
	return ts.resultFormat
}

// SetResultFormat 设置结果返回的格式 0 为 Text, 1 为 Binary
func (ts *TiDBStatement) SetResultFormat(rf []int16) {
	ts.resultFormat = rf
}

// GetOIDs return OIDs for the current statement
func (ts *TiDBStatement) GetOIDs() []uint32 {
	return ts.paramOIDs
}

// SetOIDs set OIDs for the current statement
func (ts *TiDBStatement) SetOIDs(pgOIDs []uint32) {
	ts.paramOIDs = pgOIDs
}

// IsPrepareStmt 判断是否为预处理查询
func (trs *tidbResultSet) IsPrepareStmt() bool {
	if trs.preparedStmt != nil {
		return true
	}
	return false
}


// Prepare implements QueryCtx Prepare method.
func (tc *TiDBContext) Prepare(sql string, name string) (statement PreparedStatement, columns, params []*ColumnInfo, err error) {
	stmtID, paramCount, fields, err := tc.Session.PrepareStmt(sql, name)
	if err != nil {
		return
	}

	columns = make([]*ColumnInfo, len(fields))
	for i := range fields {
		columns[i] = convertColumnInfo(fields[i])
	}
	params = make([]*ColumnInfo, paramCount)
	for i := range params {
		params[i] = &ColumnInfo{
			Type: mysql.TypeBlob,
		}
	}
	stmt := &TiDBStatement{
		sql:         sql,
		id:          stmtID,
		numParams:   paramCount,
		boundParams: make([][]byte, paramCount),
		ctx:         tc,
		columnInfo:  columns,
	}

	statement = stmt
	tc.stmts[int(stmtID)] = stmt
	return
}

// NeedPassword implements QueryCtx NeedPassword method
func (tc *TiDBContext) NeedPassword(user *auth.UserIdentity) bool {
	return tc.Session.NeedPassword(user)
}
