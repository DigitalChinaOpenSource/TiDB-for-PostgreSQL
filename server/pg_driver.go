package server

import (
	"context"
	"github.com/pingcap/tidb/types"
	"github.com/pingcap/tidb/util/chunk"
)

// PreparedStatement is the interface to use a prepared statement.
type PreparedStatement interface {
	// ID returns statement ID
	ID() int

	// Execute executes the statement.
	Execute(context.Context, []types.Datum) (ResultSet, error)

	// AppendParam appends parameter to the statement.
	AppendParam(paramID int, data []byte) error

	// NumParams returns number of parameters.
	NumParams() int

	// BoundParams returns bound parameters.
	BoundParams() [][]byte

	// SetParamsType sets type for parameters.
	SetParamsType([]byte)

	// GetParamsType returns the type for parameters.
	GetParamsType() []byte

	// StoreResultSet stores ResultSet for subsequent stmt fetching
	StoreResultSet(rs ResultSet)

	// GetResultSet gets ResultSet associated this statement
	GetResultSet() ResultSet

	// Reset removes all bound parameters.
	Reset()

	// Close closes the statement.
	Close() error

	// SetColumnInfo 设置statement中返回行数据信息
	// PgSQL Modified
	SetColumnInfo(columns []*ColumnInfo)

	// GetColumnInfo 获取statement中返回行数据信息
	// PgSQL Modified
	GetColumnInfo() []*ColumnInfo

	// SetArgs 在bind阶段绑定参数具体值
	// PgSQL Modified
	SetArgs(args []types.Datum)

	// GetArgs 获取Args的具体值
	// PgSQL Modified
	GetArgs() []types.Datum

	// GetResultFormat 获取结果返回的格式 0 为 Text, 1 为 Binary
	// PgSQL Modified
	GetResultFormat() []int16

	// SetResultFormat 设置结果返回的格式 0 为 Text, 1 为 Binary
	// PgSQL Modified
	SetResultFormat(rf []int16)

	// GetOIDs returns the postgres OIDs
	// PgSQL Modified
	GetOIDs() []uint32

	// SetOIDs set the postgres OIDs
	// PgSQL Modified
	SetOIDs(pgOIDs []uint32)
}

// ResultSet is the result set of an query.
type ResultSet interface {
	Columns() []*ColumnInfo
	NewChunk() *chunk.Chunk
	Next(context.Context, *chunk.Chunk) error
	StoreFetchedRows(rows []chunk.Row)
	GetFetchedRows() []chunk.Row
	Close() error
	IsPrepareStmt() bool
}
