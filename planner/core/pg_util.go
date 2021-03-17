package core

import "github.com/pingcap/tidb/expression"

//insert的架构
func (s *Insert) TableSchema() *expression.Schema {
	return s.tableSchema
}
