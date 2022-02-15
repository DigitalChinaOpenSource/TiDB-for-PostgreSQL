package core

import "github.com/pingcap/tidb/parser/ast"

// SetParamType todo 设置参数类型
func (p *LogicalJoin) SetParamType(paramExprs *[]ast.ParamMarkerExpr) (err error) {
	return err
}

// SetParamType 设置参数类型
func (p *LogicalProjection) SetParamType(paramExprs *[]ast.ParamMarkerExpr) (err error) {
	if childPlan := p.children; childPlan != nil {
		for i := range childPlan {
			if err = childPlan[i].SetParamType(paramExprs); err != nil {

				return err

			}
			return err
		}
	}
	return err
}

// SetParamType todo 设置参数类型
func (la *LogicalAggregation) SetParamType(paramExprs *[]ast.ParamMarkerExpr) (err error) {
	if childs := la.children; childs != nil {
		for _, child := range childs {
			err = child.SetParamType(paramExprs)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// SetParamType 当计划为logic select时，先遍历子计划中的参数类型，再从condition成员中获取参数类型
func (p *LogicalSelection) SetParamType(paramExprs *[]ast.ParamMarkerExpr) (err error) {
	if childPlan := p.children; childPlan != nil {
		for i := range childPlan {
			if err = childPlan[i].SetParamType(paramExprs); err != nil {
				return err
			}
		}
	}
	if p.Conditions != nil {
		DeepFirstTravsalTree(p.Conditions, paramExprs, &p.Schema().Columns)
	}
	return err
}

// SetParamType todo 设置参数类型
func (p *LogicalMaxOneRow) SetParamType(paramExprs *[]ast.ParamMarkerExpr) (err error) {
	return err
}

// SetParamType todo 设置参数类型
func (p *LogicalTableDual) SetParamType(paramExprs *[]ast.ParamMarkerExpr) (err error) {
	return err
}

// SetParamType todo 设置参数类型
func (p *LogicalMemTable) SetParamType(paramExprs *[]ast.ParamMarkerExpr) (err error) {
	return err
}

// SetParamType todo 设置参数类型
func (p *LogicalUnionScan) SetParamType(paramExprs *[]ast.ParamMarkerExpr) (err error) {
	return err
}

// SetParamType set the parameter type of dataSource plan
// dataSource plan is always the last node of o plan tree and it will not contain any parameter.Just return nil
func (ds *DataSource) SetParamType(paramExprs *[]ast.ParamMarkerExpr) (err error) {
	return err
}

// SetParamType todo 设置参数类型
func (p *TiKVSingleGather) SetParamType(paramExprs *[]ast.ParamMarkerExpr) (err error) {
	return err
}

// SetParamType todo 设置参数类型
func (p *LogicalTableScan) SetParamType(paramExprs *[]ast.ParamMarkerExpr) (err error) {
	return err
}

// SetParamType todo 设置参数类型
func (p *LogicalIndexScan) SetParamType(paramExprs *[]ast.ParamMarkerExpr) (err error) {
	return err
}

// SetParamType todo 设置参数类型
func (p *LogicalUnionAll) SetParamType(paramExprs *[]ast.ParamMarkerExpr) (err error) {
	return err
}

// SetParamType todo 设置参数类型
func (ls *LogicalSort) SetParamType(paramExprs *[]ast.ParamMarkerExpr) (err error) {
	if childs := ls.children; childs != nil {
		for _, child := range childs {
			err = child.SetParamType(paramExprs)
			if err != nil {
				return err
			}
		}
	}
	return err
}

// SetParamType todo 设置参数类型
func (lt *LogicalTopN) SetParamType(paramExprs *[]ast.ParamMarkerExpr) (err error) {
	return err
}

// SetParamType set the parameter type of logicalLimit plan
// when send "limit $1 offset $2",planbuilder will convert it to a tableDual plan.see why at /planner/core/logical_plan_builder.go buildLimit
func (ll *LogicalLimit) SetParamType(paramExprs *[]ast.ParamMarkerExpr) (err error) {
	if childs := ll.children; childs != nil {
		for _, child := range childs {
			err = child.SetParamType(paramExprs)
			if err != nil {
				return err
			}
		}
	}
	return err
}

// SetParamType todo 设置参数类型
func (p *LogicalLock) SetParamType(paramExprs *[]ast.ParamMarkerExpr) (err error) {
	return err
}

// SetParamType todo 设置参数类型
func (p *LogicalWindow) SetParamType(paramExprs *[]ast.ParamMarkerExpr) (err error) {
	return err
}

// SetParamType todo 设置参数类型
func (p *LogicalCTE) SetParamType(paramExprs *[]ast.ParamMarkerExpr) (err error) {
	return err
}

// SetParamType todo 设置参数类型
func (p *LogicalCTETable) SetParamType(paramExprs *[]ast.ParamMarkerExpr) (err error) {
	return err
}

// SetParamType set the parameter type of logicalShow plan
// https://www.postgresql.org/docs/current/sql-show.html
// According to the postgresql document, any parameter won't follow the show statement.
// Besides, show statement won't be the sub statement of any other sql statement.
// So, Show statement doesn't need actual implement. Just return nil.
func (p *LogicalShow) SetParamType(paramExprs *[]ast.ParamMarkerExpr) (err error) {
	return err
}

// SetParamType set the parameter type of logicalShowDDLJobs plan
// https://www.postgresql.org/docs/current/sql-show.html
// Actually, postgresql doesn't support the syntax like "show create table tname"
// And showddl statement won't contain parameter. Just return nil
func (p *LogicalShowDDLJobs) SetParamType(paramExprs *[]ast.ParamMarkerExpr) (err error) {
	return err
}

// LogicalReturning represents returning plan.
type LogicalReturning struct {
	baseLogicalPlan

	Offset uint64
	Count  uint64
}

// SetParamType LogicalReturning
func (p *LogicalReturning) SetParamType(paramExprs *[]ast.ParamMarkerExpr) (err error) {
	return err
}
