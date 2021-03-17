package executor

import (
	"github.com/pingcap/parser/ast"
	"github.com/pingcap/parser/model"
	"github.com/pingcap/tidb/expression"
	plannercore "github.com/pingcap/tidb/planner/core"
	driver "github.com/pingcap/tidb/types/parser_driver"
	"strings"
)

func SetInsertParamTypeArray(insertPlan *plannercore.Insert, paramExprs *[]ast.ParamMarkerExpr) (paramType []byte) {
	//if insertPlan, ok := insertPlan.(*plannercore.Insert); ok {
	if insertPlan.SelectPlan != nil {
		if selectPlan, ok := insertPlan.SelectPlan.(*plannercore.PhysicalTableReader); ok {
			//获取子查询中各字段类型的map，给下文获取参数类型提供参照。
			if tableScan, ok := selectPlan.TablePlans[0].(*plannercore.PhysicalTableScan); ok {
				cols := tableScan.Columns
				//从condition中获取参数类型
				if selection, ok := selectPlan.TablePlans[1].(*plannercore.PhysicalSelection); ok {
					conditions := selection.Conditions
					//调用递归方法，获取参数类型，填充到paramType数组
					DeepFirstTravsalTree(conditions, paramExprs, &cols)
				}
			}
		}
	} else {
		//当前计划的tableSchema，也就是表结构。
		ts := insertPlan.TableSchema()
		cols := ts.Columns
		//将要插入字段的各个类型翻入到数组中，以待下文通过
		colsType := make([]byte,0)
		for index := range cols {
			colsType = append(colsType,cols[index].RetType.Tp)
		}
		//lists是参数列表，需要考虑一次性insert多行的情况，在这种情况下，lists数组将有多个元素，只需要将它们依次放入数组中返回即可。
		lists := insertPlan.Lists

		//遍历lists，将参数类型都
		for i := range lists {
			list := lists[i]
			for j := range list {
				cst := list[j].(*expression.Constant)
				//Kind()将返回Datum对象的k成员，我们通过这个标志知道这个参数是传进来值了，还是暂时用？占位。
				if cst.Value.Kind() == 0 {
					//等于0就意味着？占位，没有传实际值。
					paramType = append(paramType,colsType[j])
				}
			}
		}
	}
	//}
	return paramType
}

//深度优先遍历条件节点，获取到参数类型数组
// todo 完善多层select嵌套的情况。
func DeepFirstTravsalTree(exprs []expression.Expression, paramExprs *[]ast.ParamMarkerExpr, cols *[]*model.ColumnInfo){
	paramIndex := 0
	if len(exprs) > 1 {
		// sql where condition like this : x = aaa and y > bbb and z < ccc
		for i := range exprs {
			if scalar, ok := exprs[i].(*expression.ScalarFunction); ok {
				if setOk := SetParamTypes(scalar.Function.Args(), paramExprs, cols, paramIndex); setOk{
					paramIndex++
				}
			}
		}
	} else if len(exprs) == 1 {
		// sql where condition like this : x = aaa or y < bbb and z = zzz
		// the struct is a tree , not even array, we should use deep-first traversal to resolve
		if scalar, ok := exprs[0].(*expression.ScalarFunction); ok {
			DoDeepFirstTraverSal(scalar.Function.Args(), paramExprs, cols, paramIndex)
		}
	}
}

func DoDeepFirstTraverSal(args []expression.Expression, paramExprs *[]ast.ParamMarkerExpr, cols *[]*model.ColumnInfo, paramIndex int) int {
	//left不是终结点，还可以往下遍历
	if left, ok := args[0].(*expression.ScalarFunction); ok {
		paramIndex = DoDeepFirstTraverSal(left.Function.Args(), paramExprs, cols, paramIndex)
	}
	// 右子树不是终结点，还可以往下遍历
	if right, ok := args[1].(*expression.ScalarFunction); ok {
		paramIndex = DoDeepFirstTraverSal(right.Function.Args(), paramExprs, cols, paramIndex)
	}
	//深度优先遍历，先往下一直遍历，知道叶子节点在做具体的处理逻辑，也就是判断节点参数类型。
	//程序到达这里，args左右子树就是column和constant了。
	if setOk := SetParamTypes(args, paramExprs, cols, paramIndex); setOk {
		paramIndex++
	}
	return paramIndex
}

func SetParamTypes(args []expression.Expression, paramExprs *[]ast.ParamMarkerExpr, cols *[]*model.ColumnInfo, paramIndex int) bool {
	if constant, ok := args[1].(*expression.Constant); ok && constant.Value.Kind() == 0 {
		if column, ok := args[0].(*expression.Column); ok {
			nameSplit := strings.Split(column.OrigName,".")
			shortName := nameSplit[len(nameSplit) - 1]
			for i := range *cols {
				if paramMarker, ok := (*paramExprs)[paramIndex].(*driver.ParamMarkerExpr); (*cols)[i].Name.O == shortName && ok {
					paramMarker.TexprNode.Type = (*cols)[i].FieldType
					break
				}
			}
		}
		return true
	} else {
		return false
	}
}
