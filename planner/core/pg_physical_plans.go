package core

import (
	"github.com/pingcap/tidb/expression"
	"github.com/pingcap/tidb/parser/ast"
	driver "github.com/pingcap/tidb/types/parser_driver"
)

// SetParamType 从tableReader计划中获取参数类型
func (p *PhysicalTableReader) SetParamType(paramExprs *[]ast.ParamMarkerExpr) (err error) {
	if childPlans := p.children; childPlans != nil {
		for _, childPlan := range childPlans {
			if err = childPlan.SetParamType(paramExprs); err != nil {
				return err
			}
		}
	}
	if tablePlans := p.TablePlans; tablePlans != nil {
		for _, tablePlan := range tablePlans {
			if err = tablePlan.SetParamType(paramExprs); err != nil {
				return err
			}
		}
	}
	return err
}

// SetParamType set the parameter type in ParamMarkerExpr from physicalIndexReader
// todo 获取 PhysicalIndexReader 计划中的参数类型
func (p *PhysicalIndexReader) SetParamType(paramExprs *[]ast.ParamMarkerExpr) (err error) {
	return err
}

// SetParamType set the parameter type in ParamMarkerExpr from PhysicalIndexLookUpReader
// todo 从 PhysicalIndexLookUpReader 计划中获取参数类型
func (p *PhysicalIndexLookUpReader) SetParamType(paramExprs *[]ast.ParamMarkerExpr) (err error) {
	return err
}

// SetParamType set the parameter type in ParamMarkerExpr from PhysicalIndexMergeReader
// todo 从 PhysicalIndexMergeReader 计划中获取参数类型
func (p *PhysicalIndexMergeReader) SetParamType(paramExprs *[]ast.ParamMarkerExpr) (err error) {
	return err
}

// SetParamType set the parameter type in ParamMarkerExpr from PhysicalIndexScan
// todo 从 PhysicalIndexScan 计划中获取参数类型
func (p *PhysicalIndexScan) SetParamType(paramExprs *[]ast.ParamMarkerExpr) (err error) {
	return err
}

// SetParamType todo 从 PhysicalMemTable 计划中获取参数类型
func (p *PhysicalMemTable) SetParamType(paramExprs *[]ast.ParamMarkerExpr) (err error) {
	return err
}

// SetParamType 根据PhysicalTableScan计划设置参数类型
func (ts *PhysicalTableScan) SetParamType(paramExprs *[]ast.ParamMarkerExpr) (err error) {
	if childrenPlan := ts.children; childrenPlan != nil {
		for i := range childrenPlan {
			err = childrenPlan[i].SetParamType(paramExprs)
		}
	}
	return err
}

// SetParamType 从PhysicalProjection计划中获取参数类型
func (p *PhysicalProjection) SetParamType(paramExprs *[]ast.ParamMarkerExpr) (err error) {
	if childPlans := p.children; childPlans != nil {
		for _, childPlan := range childPlans {
			err = childPlan.SetParamType(paramExprs)
			if err != nil {
				return nil
			}
		}
	}
	return err
}

// SetParamType set the parameter type in ParamMarkerExpr from PhysicalTopN
// todo 从 PhysicalTopN 中获取参数类型
func (lt *PhysicalTopN) SetParamType(paramExprs *[]ast.ParamMarkerExpr) (err error) {
	return err
}

// SetParamType set the parameter type in ParamMarkerExpr from NominalSort
// todo 从 NominalSort 计划中获取参数类型
func (sort *NominalSort) SetParamType(paramExprs *[]ast.ParamMarkerExpr) (err error) {
	return err
}

// SetParamType 从PhysicalHashJoin计划中获取参数类型
func (p *PhysicalHashJoin) SetParamType(paramExprs *[]ast.ParamMarkerExpr) (err error) {
	if childPlans := p.children; childPlans != nil {
		for _, childPlan := range childPlans {
			err = childPlan.SetParamType(paramExprs)
			if err != nil {
				return err
			}
		}
	}
	return err
}

// SetParamType set the parameter type in ParamMarkerExpr from PhysicalApply
// todo 从 PhysicalApply 计划中获取参数类型
func (la *PhysicalApply) SetParamType(paramExprs *[]ast.ParamMarkerExpr) (err error) {
	return err
}

// SetParamType set the parameter type in ParamMarkerExpr from PhysicalIndexJoin
// todo 从 PhysicalIndexJoin 计划中获取参数类型
func (p *PhysicalIndexJoin) SetParamType(paramExprs *[]ast.ParamMarkerExpr) (err error) {
	return err
}

// SetParamType set the parameter type in ParamMarkerExpr from PhysicalMergeJoin
// todo 从 PhysicalMergeJoin 中获取参数类型
func (join *PhysicalMergeJoin) SetParamType(paramExprs *[]ast.ParamMarkerExpr) (err error) {
	return err
}

// SetParamType set the parameter type in ParamMarkerExpr from PhysicalLock
// todo 从 PhysicalLock 计划中获取参数类型
func (lock *PhysicalLock) SetParamType(paramExprs *[]ast.ParamMarkerExpr) (err error) {
	return err
}

// SetParamType set the parameter type in ParamMarkerExpr from PhysicalLimit
// todo 从 PhysicalLimit 计划中获取参数类型
func (p *PhysicalLimit) SetParamType(paramExprs *[]ast.ParamMarkerExpr) (err error) {
	return err
}

// SetParamType set the parameter type in ParamMarkerExpr from basePhysicalAgg
// todo 从 basePhysicalAgg 计划中获取参数类型
func (p *basePhysicalAgg) SetParamType(paramExprs *[]ast.ParamMarkerExpr) (err error) {
	return err
}

// SetParamType set the parameter type in ParamMarkerExpr from PhysicalHashAgg
// todo 从 PhysicalHashAgg 中获取参数类型
func (agg *PhysicalHashAgg) SetParamType(paramExprs *[]ast.ParamMarkerExpr) (err error) {
	return err
}

// SetParamType set the parameter type in ParamMarkerExpr from PhysicalStreamAgg
// todo 从 PhysicalStreamAgg 孔家中获取参数类型
func (agg *PhysicalStreamAgg) SetParamType(paramExprs *[]ast.ParamMarkerExpr) (err error) {
	return err
}

// SetParamType set the parameter type in ParamMarkerExpr from PhysicalSort
// todo 从 PhysicalSort 计划中获取参数类型
func (ls *PhysicalSort) SetParamType(paramExprs *[]ast.ParamMarkerExpr) (err error) {
	return err
}

// SetParamType set the parameter type in ParamMarkerExpr from PhysicalUnionScan
// todo 从 PhysicalUnionScan 获取参数类型
func (p *PhysicalUnionScan) SetParamType(paramExprs *[]ast.ParamMarkerExpr) (err error) {
	return err
}

// SetParamType 从PhysicalSelection计划中获取参数类型
func (p *PhysicalSelection) SetParamType(paramExprs *[]ast.ParamMarkerExpr) (err error) {
	if childPlans := p.children; childPlans != nil {
		for _, childPlan := range childPlans {
			if err = childPlan.SetParamType(paramExprs); err != nil {
				return err
			}
		}
	}

	if p.Conditions != nil {
		DeepFirstTravsalTree(p.Conditions, paramExprs, &p.Schema().Columns)
	}
	return err
}

// SetParamType set the parameter type in ParamMarkerExpr from PhysicalMaxOneRow
// todo 从 PhysicalMaxOneRow 计划中获取参数类型
func (p *PhysicalMaxOneRow) SetParamType(paramExprs *[]ast.ParamMarkerExpr) (err error) {
	return err
}

// SetParamType set the parameter type in ParamMarkerExpr from PhysicalTableDual
// todo 从 PhysicalTableDual 计划中获取参数类型
func (p *PhysicalTableDual) SetParamType(paramExprs *[]ast.ParamMarkerExpr) (err error) {
	return err
}

// SetParamType set the parameter type in ParamMarkerExpr from PhysicalWindow
// todo 从 PhysicalWindow 获取参数类型
func (p *PhysicalWindow) SetParamType(paramExprs *[]ast.ParamMarkerExpr) (err error) {
	return err
}

// SetParamType set the parameter type in ParamMarkerExpr from PhysicalShuffle
// todo 从 PhysicalShuffle 计划中获取参数类型
func (shuffle *PhysicalShuffle) SetParamType(paramExprs *[]ast.ParamMarkerExpr) (err error) {
	return err
}

// SetParamType set the parameter type in ParamMarkerExpr from PhysicalShuffleReceiverStub
// todo 从 PhysicalShuffleReceiverStub 计划中获取参数类型
func (p *PhysicalShuffleReceiverStub) SetParamType(paramExprs *[]ast.ParamMarkerExpr) (err error) {
	return err
}

// SetParamType set the parameter type of physicalShow plan
// https://www.postgresql.org/docs/current/sql-show.html
// According to the postgresql document, any parameter won't follow the show statement.
// Besides, show statement won't be the sub statement of any other sql statement.
// So, Show statement doesn't need actual implement. Just return nil.
func (show *PhysicalShow) SetParamType(paramExprs *[]ast.ParamMarkerExpr) (err error) {
	return err
}

// SetParamType set the parameter type of physicalShowDDLJobs plan
// https://www.postgresql.org/docs/current/sql-show.html
// Actually, postgresql doesn't support the syntax like "show create table tname"
// And showddl statement won't contain parameter. Just return nil
func (ddl *PhysicalShowDDLJobs) SetParamType(paramExprs *[]ast.ParamMarkerExpr) (err error) {
	return err
}

// SetParamType set the parameter type in ParamMarkerExpr from PhysicalUnionAll
// todo 从 PhysicalUnionAll 计划中获取参数类型
func (unionAll *PhysicalUnionAll) SetParamType(paramExprs *[]ast.ParamMarkerExpr) (err error) {
	return err
}

// SetParamType set the parameter type in ParamMarkerExpr from PhysicalTableSample
// todo 从 PhysicalTableSample 计划中获取参数类型
func (sample *PhysicalTableSample) SetParamType(paramExprs *[]ast.ParamMarkerExpr) (err error) {
	return err
}

// SetParamType set the parameter type in ParamMarkerExpr from PhysicalExchangeReceiver
// todo 从 PhysicalExchangeReceiver 计划中获取参数类型
func (er *PhysicalExchangeReceiver) SetParamType(paramExprs *[]ast.ParamMarkerExpr) (err error) {
	return err
}

// SetParamType set the parameter type in ParamMarkerExpr from PhysicalExchangeSender
// todo 从 PhysicalExchangeSender 计划中获取参数类型
func (es *PhysicalExchangeSender) SetParamType(paramExprs *[]ast.ParamMarkerExpr) (err error) {
	return err
}

// SetParamType set the parameter type in ParamMarkerExpr from PhysicalCTE
// todo 从 PhysicalCTE 计划中获取参数类型
func (er *PhysicalCTE) SetParamType(paramExprs *[]ast.ParamMarkerExpr) (err error) {
	return err
}

// SetParamType set the parameter type in ParamMarkerExpr from PhysicalCTETable
// todo 从 PhysicalCTETable 计划中获取参数类型
func (er *PhysicalCTETable) SetParamType(paramExprs *[]ast.ParamMarkerExpr) (err error) {
	return err
}

// SetParamType set the parameter type in ParamMarkerExpr from PhysicalSimpleWrapper
// todo 从 PhysicalSimpleWrapper 计划中获取参数类型
func (er *PhysicalSimpleWrapper) SetParamType(paramExprs *[]ast.ParamMarkerExpr) (err error) {
	return err
}

// DeepFirstTravsalTree 当insert计划嵌套select子查询时，将select子查询的参数类型设置到prepared.Param中去。
// 其分为两种情况，一种是 condition1 and condition2 and condition 3，其参数放在select子计划中的Condition成员中，是一个数组
// 另一种情况是 condition1 or condition2 and condition3 。其参数构造成了一个树，需要深度优先遍历
// exprs：select子计划的参数，可能是个数组，也可能是个树
// paramExprs：prepared.Param，我们需要往里面设置参数类型。
// cols：select计划查询的字段信息
func DeepFirstTravsalTree(exprs []expression.Expression, paramExprs *[]ast.ParamMarkerExpr, cols *[]*expression.Column) {
	for i := range exprs {
		if scalar, ok := exprs[i].(*expression.ScalarFunction); ok {
			DoDeepFirstTraverSal(scalar.Function.GetArgs(), paramExprs, cols)
		}
	}
}

// DoDeepFirstTraverSal 深度优先遍历树形结构的参数，设置到paramExprs中去
// 这个递归方法返回值的情况是当遇到 cast 这样的一个参数的函数时，将其参数返回但递归的上一层处理。
// 这是一种特殊情况，我们在前面构造计划 p 之前，设置prepared.Param参数类型为interface以保证无论子查询条件中是否包含主键字段，都能够在计划中找到条件字段。
// 这样做的代价就是生成的 condition 在条件左侧的字段类型如果与默认类型不同，那么条件左右两侧的column和constant都要再包上一层cast方法。
// 当遇到这种情况，我们就在递归过程中直接将参数（这里可能是column或者是constant）返回上层。在上层的逻辑中再调用SetParamTypes设置参数进去。
func DoDeepFirstTraverSal(args []expression.Expression, paramExprs *[]ast.ParamMarkerExpr, cols *[]*expression.Column) []expression.Expression {
	//left不是终结点，还可以往下遍历
	var lRet, rRet []expression.Expression
	if len(args) == 2 {
		if left, ok := args[0].(*expression.ScalarFunction); ok {
			lRet = DoDeepFirstTraverSal(left.Function.GetArgs(), paramExprs, cols)
		}
		// 右子树不是终结点，还可以往下遍历
		if right, ok := args[1].(*expression.ScalarFunction); ok {
			rRet = DoDeepFirstTraverSal(right.Function.GetArgs(), paramExprs, cols)
		}
	}
	if lRet != nil && rRet != nil {
		newArgs := append(lRet, rRet...)
		SetParamTypes(newArgs, paramExprs, cols)
	}
	//深度优先遍历，先往下一直遍历，知道叶子节点在做具体的处理逻辑，也就是判断节点参数类型。
	return SetParamTypes(args, paramExprs, cols)
}

// SetParamTypes 设置参数类型.
// 只考虑了参数一个方法参数数量为1,或者是2的情况，比如 cast一个参数。eq 参数数量是2.
func SetParamTypes(args []expression.Expression, paramExprs *[]ast.ParamMarkerExpr, cols *[]*expression.Column) []expression.Expression {
	//如果参数类型已经完全设置完毕，则退出
	if CheckParamFullySeted(paramExprs) {
		return nil
	}
	if len(args) == 1 {
		// 当前考虑的是cast方法，只有一个参数的情况。直接返回上层处理。
		return args
	} else if len(args) == 2 {
		if constant, ok := args[1].(*expression.Constant); ok {
			if column, ok := args[0].(*expression.Column); ok {
			cycle:
				for _, col := range *cols {
					for _, expr := range *paramExprs {
						if paramMarker, ok := expr.(*driver.ParamMarkerExpr); ok && col.OrigName == column.OrigName &&
							paramMarker.Offset == constant.Offset {
							paramMarker.TexprNode.Type = *col.RetType
							break cycle
						}
					}
				}
			}
		}
		return nil
	} else {
		// todo 完善多参数的处理逻辑
		return nil
	}
}

// CheckParamFullySeted 检查params是否完全设置完毕
func CheckParamFullySeted(paramExprs *[]ast.ParamMarkerExpr) bool {
	for _, p := range *paramExprs {
		if p.(*driver.ParamMarkerExpr).Type.Tp == 0 {
			return false
		}
	}
	return true
}

// PhysicalReturning presents a returning plan
type PhysicalReturning struct {
	basePhysicalPlan
}

// SetParamType of PhysicalReturning
func (p *PhysicalReturning) SetParamType(paramExprs *[]ast.ParamMarkerExpr) (err error) {
	return nil
}
