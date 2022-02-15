package core

import "github.com/pingcap/tidb/expression"

// GetTableSchema 获取insert计划的表结构
func (s *Insert) GetTableSchema() *expression.Schema {
	return s.tableSchema
}
