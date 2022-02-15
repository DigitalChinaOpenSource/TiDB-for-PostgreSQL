package core

import "github.com/pingcap/tidb/parser/ast"

// SetParamType set the parameter type in ParamMarkerExpr from PointGetPlan
// todo PointGetPlan计划中获取参数类型
func (p *PointGetPlan) SetParamType(paramExprs *[]ast.ParamMarkerExpr) error {
	return nil
}

// SetParamType set the parameter type in ParamMarkerExpr from BatchPointGetPlan
// todo 从BatchPointGetPlan计划中获取参数类型
func (p *BatchPointGetPlan) SetParamType(paramExprs *[]ast.ParamMarkerExpr) error {
	return nil
}
