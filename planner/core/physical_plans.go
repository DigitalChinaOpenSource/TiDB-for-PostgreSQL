// Copyright 2016 PingCAP, Inc.
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

// Copyright 2021 Digital China Group Co.,Ltd
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

package core

import (
	driver "github.com/pingcap/tidb/types/parser_driver"
	"unsafe"

	"github.com/DigitalChinaOpenSource/DCParser/ast"
	"github.com/DigitalChinaOpenSource/DCParser/model"
	"github.com/pingcap/tidb/expression"
	"github.com/pingcap/tidb/expression/aggregation"
	"github.com/pingcap/tidb/kv"
	"github.com/pingcap/tidb/planner/property"
	"github.com/pingcap/tidb/planner/util"
	"github.com/pingcap/tidb/sessionctx"
	"github.com/pingcap/tidb/sessionctx/stmtctx"
	"github.com/pingcap/tidb/statistics"
	"github.com/pingcap/tidb/table"
	"github.com/pingcap/tidb/types"
	"github.com/pingcap/tidb/util/ranger"
)

var (
	_ PhysicalPlan = &PhysicalSelection{}
	_ PhysicalPlan = &PhysicalProjection{}
	_ PhysicalPlan = &PhysicalTopN{}
	_ PhysicalPlan = &PhysicalMaxOneRow{}
	_ PhysicalPlan = &PhysicalTableDual{}
	_ PhysicalPlan = &PhysicalUnionAll{}
	_ PhysicalPlan = &PhysicalSort{}
	_ PhysicalPlan = &NominalSort{}
	_ PhysicalPlan = &PhysicalLock{}
	_ PhysicalPlan = &PhysicalLimit{}
	_ PhysicalPlan = &PhysicalIndexScan{}
	_ PhysicalPlan = &PhysicalTableScan{}
	_ PhysicalPlan = &PhysicalTableReader{}
	_ PhysicalPlan = &PhysicalIndexReader{}
	_ PhysicalPlan = &PhysicalIndexLookUpReader{}
	_ PhysicalPlan = &PhysicalIndexMergeReader{}
	_ PhysicalPlan = &PhysicalHashAgg{}
	_ PhysicalPlan = &PhysicalStreamAgg{}
	_ PhysicalPlan = &PhysicalApply{}
	_ PhysicalPlan = &PhysicalIndexJoin{}
	_ PhysicalPlan = &PhysicalBroadCastJoin{}
	_ PhysicalPlan = &PhysicalHashJoin{}
	_ PhysicalPlan = &PhysicalMergeJoin{}
	_ PhysicalPlan = &PhysicalUnionScan{}
	_ PhysicalPlan = &PhysicalWindow{}
	_ PhysicalPlan = &PhysicalShuffle{}
	_ PhysicalPlan = &PhysicalShuffleDataSourceStub{}
	_ PhysicalPlan = &BatchPointGetPlan{}
)

// PhysicalTableReader is the table reader in tidb.
type PhysicalTableReader struct {
	physicalSchemaProducer

	// TablePlans flats the tablePlan to construct executor pb.
	TablePlans []PhysicalPlan
	tablePlan  PhysicalPlan

	// StoreType indicates table read from which type of store.
	StoreType kv.StoreType
}

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

// GetTablePlan exports the tablePlan.
func (p *PhysicalTableReader) GetTablePlan() PhysicalPlan {
	return p.tablePlan
}

// GetTableScan exports the tableScan that contained in tablePlan.
func (p *PhysicalTableReader) GetTableScan() *PhysicalTableScan {
	curPlan := p.tablePlan
	for {
		chCnt := len(curPlan.Children())
		if chCnt == 0 {
			return curPlan.(*PhysicalTableScan)
		} else if chCnt == 1 {
			curPlan = curPlan.Children()[0]
		} else {
			join := curPlan.(*PhysicalBroadCastJoin)
			curPlan = join.children[1-join.globalChildIndex]
		}
	}
}

// GetPhysicalTableReader returns PhysicalTableReader for logical TiKVSingleGather.
func (sg *TiKVSingleGather) GetPhysicalTableReader(schema *expression.Schema, stats *property.StatsInfo, props ...*property.PhysicalProperty) *PhysicalTableReader {
	reader := PhysicalTableReader{}.Init(sg.ctx, sg.blockOffset)
	reader.stats = stats
	reader.SetSchema(schema)
	reader.childrenReqProps = props
	return reader
}

// GetPhysicalIndexReader returns PhysicalIndexReader for logical TiKVSingleGather.
func (sg *TiKVSingleGather) GetPhysicalIndexReader(schema *expression.Schema, stats *property.StatsInfo, props ...*property.PhysicalProperty) *PhysicalIndexReader {
	reader := PhysicalIndexReader{}.Init(sg.ctx, sg.blockOffset)
	reader.stats = stats
	reader.SetSchema(schema)
	reader.childrenReqProps = props
	return reader
}

// SetChildren overrides PhysicalPlan SetChildren interface.
func (p *PhysicalTableReader) SetChildren(children ...PhysicalPlan) {
	p.tablePlan = children[0]
	p.TablePlans = flattenPushDownPlan(p.tablePlan)
}

// PhysicalIndexReader is the index reader in tidb.
type PhysicalIndexReader struct {
	physicalSchemaProducer

	// IndexPlans flats the indexPlan to construct executor pb.
	IndexPlans []PhysicalPlan
	indexPlan  PhysicalPlan

	// OutputColumns represents the columns that index reader should return.
	OutputColumns []*expression.Column
}

// SetParamType set the parameter type in ParamMarkerExpr from physicalIndexReader
// todo 获取PhysicalIndexReader计划中的参数类型
func (p *PhysicalIndexReader) SetParamType(paramExprs *[]ast.ParamMarkerExpr) (err error) {
	return err
}

// SetSchema overrides PhysicalPlan SetSchema interface.
func (p *PhysicalIndexReader) SetSchema(_ *expression.Schema) {
	if p.indexPlan != nil {
		p.IndexPlans = flattenPushDownPlan(p.indexPlan)
		switch p.indexPlan.(type) {
		case *PhysicalHashAgg, *PhysicalStreamAgg:
			p.schema = p.indexPlan.Schema()
		default:
			is := p.IndexPlans[0].(*PhysicalIndexScan)
			p.schema = is.dataSourceSchema
		}
		p.OutputColumns = p.schema.Clone().Columns
	}
}

// SetChildren overrides PhysicalPlan SetChildren interface.
func (p *PhysicalIndexReader) SetChildren(children ...PhysicalPlan) {
	p.indexPlan = children[0]
	p.SetSchema(nil)
}

// PushedDownLimit is the limit operator pushed down into PhysicalIndexLookUpReader.
type PushedDownLimit struct {
	Offset uint64
	Count  uint64
}

// PhysicalIndexLookUpReader is the index look up reader in tidb. It's used in case of double reading.
type PhysicalIndexLookUpReader struct {
	physicalSchemaProducer

	// IndexPlans flats the indexPlan to construct executor pb.
	IndexPlans []PhysicalPlan
	// TablePlans flats the tablePlan to construct executor pb.
	TablePlans []PhysicalPlan
	indexPlan  PhysicalPlan
	tablePlan  PhysicalPlan

	ExtraHandleCol *expression.Column
	// PushedLimit is used to avoid unnecessary table scan tasks of IndexLookUpReader.
	PushedLimit *PushedDownLimit
}

// SetParamType set the parameter type in ParamMarkerExpr from PhysicalIndexLookUpReader
// todo 从PhysicalIndexLookUpReader计划中获取参数类型
func (p *PhysicalIndexLookUpReader) SetParamType(paramExprs *[]ast.ParamMarkerExpr) (err error) {
	return err
}

// PhysicalIndexMergeReader is the reader using multiple indexes in tidb.
type PhysicalIndexMergeReader struct {
	physicalSchemaProducer

	// PartialPlans flats the partialPlans to construct executor pb.
	PartialPlans [][]PhysicalPlan
	// TablePlans flats the tablePlan to construct executor pb.
	TablePlans []PhysicalPlan
	// partialPlans are the partial plans that have not been flatted. The type of each element is permitted PhysicalIndexScan or PhysicalTableScan.
	partialPlans []PhysicalPlan
	// tablePlan is a PhysicalTableScan to get the table tuples. Current, it must be not nil.
	tablePlan PhysicalPlan
}

// SetParamType set the parameter type in ParamMarkerExpr from PhysicalIndexMergeReader
// todo 从PhysicalIndexMergeReader计划中获取参数类型
func (p *PhysicalIndexMergeReader) SetParamType(paramExprs *[]ast.ParamMarkerExpr) (err error) {
	return err
}

// PhysicalIndexScan represents an index scan plan.
type PhysicalIndexScan struct {
	physicalSchemaProducer

	// AccessCondition is used to calculate range.
	AccessCondition []expression.Expression

	Table      *model.TableInfo
	Index      *model.IndexInfo
	IdxCols    []*expression.Column
	IdxColLens []int
	Ranges     []*ranger.Range
	Columns    []*model.ColumnInfo
	DBName     model.CIStr

	TableAsName *model.CIStr

	// dataSourceSchema is the original schema of DataSource. The schema of index scan in KV and index reader in TiDB
	// will be different. The schema of index scan will decode all columns of index but the TiDB only need some of them.
	dataSourceSchema *expression.Schema

	// Hist is the histogram when the query was issued.
	// It is used for query feedback.
	Hist *statistics.Histogram

	rangeInfo string

	// The index scan may be on a partition.
	physicalTableID int64

	GenExprs map[model.TableColumnID]expression.Expression

	isPartition bool
	Desc        bool
	KeepOrder   bool
	// DoubleRead means if the index executor will read kv two times.
	// If the query requires the columns that don't belong to index, DoubleRead will be true.
	DoubleRead bool
}

// SetParamType set the parameter type in ParamMarkerExpr from PhysicalIndexScan
// todo 从PhysicalIndexScan计划中获取参数类型
func (p *PhysicalIndexScan) SetParamType(paramExprs *[]ast.ParamMarkerExpr) (err error) {
	return err
}

// PhysicalMemTable reads memory table.
type PhysicalMemTable struct {
	physicalSchemaProducer

	DBName         model.CIStr
	Table          *model.TableInfo
	Columns        []*model.ColumnInfo
	Extractor      MemTablePredicateExtractor
	QueryTimeRange QueryTimeRange
}

// SetParamType todo 从PhysicalMemTable计划中获取参数类型
func (p *PhysicalMemTable) SetParamType(paramExprs *[]ast.ParamMarkerExpr) (err error) {
	return err
}

// PhysicalTableScan represents a table scan plan.
type PhysicalTableScan struct {
	physicalSchemaProducer

	// AccessCondition is used to calculate range.
	AccessCondition []expression.Expression
	filterCondition []expression.Expression

	Table   *model.TableInfo
	Columns []*model.ColumnInfo
	DBName  model.CIStr
	Ranges  []*ranger.Range
	pkCol   *expression.Column

	TableAsName *model.CIStr

	// Hist is the histogram when the query was issued.
	// It is used for query feedback.
	Hist *statistics.Histogram

	physicalTableID int64

	rangeDecidedBy []*expression.Column

	// HandleIdx is the index of handle, which is only used for admin check table.
	HandleIdx int

	StoreType kv.StoreType

	IsGlobalRead bool

	// The table scan may be a partition, rather than a real table.
	isPartition bool
	// KeepOrder is true, if sort data by scanning pkcol,
	KeepOrder bool
	Desc      bool

	isChildOfIndexLookUp bool
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

// IsPartition returns true and partition ID if it's actually a partition.
func (ts *PhysicalTableScan) IsPartition() (bool, int64) {
	return ts.isPartition, ts.physicalTableID
}

// ExpandVirtualColumn expands the virtual column's dependent columns to ts's schema and column.
func ExpandVirtualColumn(columns []*model.ColumnInfo, schema *expression.Schema,
	colsInfo []*model.ColumnInfo) []*model.ColumnInfo {
	copyColumn := make([]*model.ColumnInfo, len(columns))
	copy(copyColumn, columns)
	var extraColumn *expression.Column
	var extraColumnModel *model.ColumnInfo
	if schema.Columns[len(schema.Columns)-1].ID == model.ExtraHandleID {
		extraColumn = schema.Columns[len(schema.Columns)-1]
		extraColumnModel = copyColumn[len(copyColumn)-1]
		schema.Columns = schema.Columns[:len(schema.Columns)-1]
		copyColumn = copyColumn[:len(copyColumn)-1]
	}
	schemaColumns := schema.Columns
	for _, col := range schemaColumns {
		if col.VirtualExpr == nil {
			continue
		}

		baseCols := expression.ExtractDependentColumns(col.VirtualExpr)
		for _, baseCol := range baseCols {
			if !schema.Contains(baseCol) {
				schema.Columns = append(schema.Columns, baseCol)
				copyColumn = append(copyColumn, FindColumnInfoByID(colsInfo, baseCol.ID))
			}
		}
	}
	if extraColumn != nil {
		schema.Columns = append(schema.Columns, extraColumn)
		copyColumn = append(copyColumn, extraColumnModel)
	}
	return copyColumn
}

//SetIsChildOfIndexLookUp is to set the bool if is a child of IndexLookUpReader
func (ts *PhysicalTableScan) SetIsChildOfIndexLookUp(isIsChildOfIndexLookUp bool) {
	ts.isChildOfIndexLookUp = isIsChildOfIndexLookUp
}

// PhysicalProjection is the physical operator of projection.
type PhysicalProjection struct {
	physicalSchemaProducer

	Exprs                []expression.Expression
	CalculateNoDelay     bool
	AvoidColumnEvaluator bool
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

// PhysicalTopN is the physical operator of topN.
type PhysicalTopN struct {
	basePhysicalPlan

	ByItems []*util.ByItems
	Offset  uint64
	Count   uint64
}

// SetParamType set the parameter type in ParamMarkerExpr from PhysicalTopN
// todo 从PhysicalTopN中获取参数类型
func (lt *PhysicalTopN) SetParamType(paramExprs *[]ast.ParamMarkerExpr) (err error) {
	return err
}

// PhysicalApply represents apply plan, only used for subquery.
type PhysicalApply struct {
	PhysicalHashJoin

	OuterSchema []*expression.CorrelatedColumn
}

// SetParamType set the parameter type in ParamMarkerExpr from PhysicalApply
// todo 从PhysicalApply计划中获取参数类型
func (la *PhysicalApply) SetParamType(paramExprs *[]ast.ParamMarkerExpr) (err error) {
	return err
}

type basePhysicalJoin struct {
	physicalSchemaProducer

	JoinType JoinType

	LeftConditions  expression.CNFExprs
	RightConditions expression.CNFExprs
	OtherConditions expression.CNFExprs

	InnerChildIdx int
	OuterJoinKeys []*expression.Column
	InnerJoinKeys []*expression.Column
	LeftJoinKeys  []*expression.Column
	RightJoinKeys []*expression.Column
	DefaultValues []types.Datum
}

// PhysicalHashJoin represents hash join implementation of LogicalJoin.
type PhysicalHashJoin struct {
	basePhysicalJoin

	Concurrency     uint
	EqualConditions []*expression.ScalarFunction

	// use the outer table to build a hash table when the outer table is smaller.
	UseOuterToBuild bool
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

// NewPhysicalHashJoin creates a new PhysicalHashJoin from LogicalJoin.
func NewPhysicalHashJoin(p *LogicalJoin, innerIdx int, useOuterToBuild bool, newStats *property.StatsInfo, prop ...*property.PhysicalProperty) *PhysicalHashJoin {
	leftJoinKeys, rightJoinKeys := p.GetJoinKeys()
	baseJoin := basePhysicalJoin{
		LeftConditions:  p.LeftConditions,
		RightConditions: p.RightConditions,
		OtherConditions: p.OtherConditions,
		LeftJoinKeys:    leftJoinKeys,
		RightJoinKeys:   rightJoinKeys,
		JoinType:        p.JoinType,
		DefaultValues:   p.DefaultValues,
		InnerChildIdx:   innerIdx,
	}
	hashJoin := PhysicalHashJoin{
		basePhysicalJoin: baseJoin,
		EqualConditions:  p.EqualConditions,
		Concurrency:      uint(p.ctx.GetSessionVars().HashJoinConcurrency),
		UseOuterToBuild:  useOuterToBuild,
	}.Init(p.ctx, newStats, p.blockOffset, prop...)
	return hashJoin
}

// PhysicalIndexJoin represents the plan of index look up join.
type PhysicalIndexJoin struct {
	basePhysicalJoin

	outerSchema *expression.Schema
	innerTask   task

	// Ranges stores the IndexRanges when the inner plan is index scan.
	Ranges []*ranger.Range
	// KeyOff2IdxOff maps the offsets in join key to the offsets in the index.
	KeyOff2IdxOff []int
	// IdxColLens stores the length of each index column.
	IdxColLens []int
	// CompareFilters stores the filters for last column if those filters need to be evaluated during execution.
	// e.g. select * from t, t1 where t.a = t1.a and t.b > t1.b and t.b < t1.b+10
	//      If there's index(t.a, t.b). All the filters can be used to construct index range but t.b > t1.b and t.b < t1.b=10
	//      need to be evaluated after we fetch the data of t1.
	// This struct stores them and evaluate them to ranges.
	CompareFilters *ColWithCmpFuncManager
	// OuterHashKeys indicates the outer keys used to build hash table during
	// execution. OuterJoinKeys is the prefix of OuterHashKeys.
	OuterHashKeys []*expression.Column
	// InnerHashKeys indicates the inner keys used to build hash table during
	// execution. InnerJoinKeys is the prefix of InnerHashKeys.
	InnerHashKeys []*expression.Column
}

// SetParamType set the parameter type in ParamMarkerExpr from PhysicalIndexJoin
// todo 从PhysicalIndexJoin计划中获取参数类型
func (p *PhysicalIndexJoin) SetParamType(paramExprs *[]ast.ParamMarkerExpr) (err error) {
	return err
}

// PhysicalIndexMergeJoin represents the plan of index look up merge join.
type PhysicalIndexMergeJoin struct {
	PhysicalIndexJoin

	// KeyOff2KeyOffOrderByIdx maps the offsets in join keys to the offsets in join keys order by index.
	KeyOff2KeyOffOrderByIdx []int
	// CompareFuncs store the compare functions for outer join keys and inner join key.
	CompareFuncs []expression.CompareFunc
	// OuterCompareFuncs store the compare functions for outer join keys and outer join
	// keys, it's for outer rows sort's convenience.
	OuterCompareFuncs []expression.CompareFunc
	// NeedOuterSort means whether outer rows should be sorted to build range.
	NeedOuterSort bool
	// Desc means whether inner child keep desc order.
	Desc bool
}

// PhysicalIndexHashJoin represents the plan of index look up hash join.
type PhysicalIndexHashJoin struct {
	PhysicalIndexJoin
	// KeepOuterOrder indicates whether keeping the output result order as the
	// outer side.
	KeepOuterOrder bool
}

// PhysicalMergeJoin represents merge join implementation of LogicalJoin.
type PhysicalMergeJoin struct {
	basePhysicalJoin

	CompareFuncs []expression.CompareFunc
	// Desc means whether inner child keep desc order.
	Desc bool
}

// SetParamType set the parameter type in ParamMarkerExpr from PhysicalMergeJoin
// todo 从PhysicalMergeJoin中获取参数类型
func (join *PhysicalMergeJoin) SetParamType(paramExprs *[]ast.ParamMarkerExpr) (err error) {
	return err
}

// PhysicalBroadCastJoin only works for TiFlash Engine, which broadcast the small table to every replica of probe side of tables.
type PhysicalBroadCastJoin struct {
	basePhysicalJoin
	globalChildIndex int
}

// SetParamType set the parameter type in ParamMarkerExpr from PhysicalBroadCastJoin
// todo 从PhysicalBroadCastJoin计划中获取参数类型
func (lock *PhysicalBroadCastJoin) SetParamType(paramExprs *[]ast.ParamMarkerExpr) (err error) {
	return err
}

// PhysicalLock is the physical operator of lock, which is used for `select ... for update` clause.
type PhysicalLock struct {
	basePhysicalPlan

	Lock ast.SelectLockType

	TblID2Handle     map[int64][]*expression.Column
	PartitionedTable []table.PartitionedTable
}

// SetParamType set the parameter type in ParamMarkerExpr from PhysicalLock
// todo 从PhysicalLock计划中获取参数类型
func (lock *PhysicalLock) SetParamType(paramExprs *[]ast.ParamMarkerExpr) (err error) {
	return err
}

// PhysicalLimit is the physical operator of Limit.
type PhysicalLimit struct {
	basePhysicalPlan

	Offset uint64
	Count  uint64
}

// SetParamType set the parameter type in ParamMarkerExpr from PhysicalLimit
// todo 从PhysicalLimit计划中获取参数类型
func (p *PhysicalLimit) SetParamType(paramExprs *[]ast.ParamMarkerExpr) (err error) {
	return err
}

// PhysicalUnionAll is the physical operator of UnionAll.
type PhysicalUnionAll struct {
	physicalSchemaProducer
}

// SetParamType set the parameter type in ParamMarkerExpr from PhysicalUnionAll
// todo 从PhysicalUnionAll计划中获取参数类型
func (unionAll *PhysicalUnionAll) SetParamType(paramExprs *[]ast.ParamMarkerExpr) (err error) {
	return err
}

type basePhysicalAgg struct {
	physicalSchemaProducer

	AggFuncs     []*aggregation.AggFuncDesc
	GroupByItems []expression.Expression
}

// SetParamType set the parameter type in ParamMarkerExpr from basePhysicalAgg
// todo 从basePhysicalAgg计划中获取参数类型
func (p *basePhysicalAgg) SetParamType(paramExprs *[]ast.ParamMarkerExpr) (err error) {
	return err
}

func (p *basePhysicalAgg) numDistinctFunc() (num int) {
	for _, fun := range p.AggFuncs {
		if fun.HasDistinct {
			num++
		}
	}
	return
}

func (p *basePhysicalAgg) getAggFuncCostFactor() (factor float64) {
	factor = 0.0
	for _, agg := range p.AggFuncs {
		if fac, ok := aggFuncFactor[agg.Name]; ok {
			factor += fac
		} else {
			factor += aggFuncFactor["default"]
		}
	}
	if factor == 0 {
		factor = 1.0
	}
	return
}

// PhysicalHashAgg is hash operator of aggregate.
type PhysicalHashAgg struct {
	basePhysicalAgg
}

// SetParamType set the parameter type in ParamMarkerExpr from PhysicalHashAgg
// todo 从PhysicalHashAgg中获取参数类型
func (agg *PhysicalHashAgg) SetParamType(paramExprs *[]ast.ParamMarkerExpr) (err error) {
	return err
}

// NewPhysicalHashAgg creates a new PhysicalHashAgg from a LogicalAggregation.
func NewPhysicalHashAgg(la *LogicalAggregation, newStats *property.StatsInfo, prop *property.PhysicalProperty) *PhysicalHashAgg {
	agg := basePhysicalAgg{
		GroupByItems: la.GroupByItems,
		AggFuncs:     la.AggFuncs,
	}.initForHash(la.ctx, newStats, la.blockOffset, prop)
	return agg
}

// PhysicalStreamAgg is stream operator of aggregate.
type PhysicalStreamAgg struct {
	basePhysicalAgg
}

// SetParamType set the parameter type in ParamMarkerExpr from PhysicalStreamAgg
// todo 从PhysicalStreamAgg孔家中获取参数类型
func (agg *PhysicalStreamAgg) SetParamType(paramExprs *[]ast.ParamMarkerExpr) (err error) {
	return err
}

// PhysicalSort is the physical operator of sort, which implements a memory sort.
type PhysicalSort struct {
	basePhysicalPlan

	ByItems []*util.ByItems
}

// SetParamType set the parameter type in ParamMarkerExpr from PhysicalSort
// todo 从PhysicalSort计划中获取参数类型
func (ls *PhysicalSort) SetParamType(paramExprs *[]ast.ParamMarkerExpr) (err error) {
	return err
}

// NominalSort asks sort properties for its child. It is a fake operator that will not
// appear in final physical operator tree. It will be eliminated or converted to Projection.
type NominalSort struct {
	basePhysicalPlan

	// These two fields are used to switch ScalarFunctions to Constants. For these
	// NominalSorts, we need to converted to Projections check if the ScalarFunctions
	// are out of bounds. (issue #11653)
	ByItems    []*util.ByItems
	OnlyColumn bool
}

// SetParamType set the parameter type in ParamMarkerExpr from NominalSort
// todo 从NominalSort计划中获取参数类型
func (sort *NominalSort) SetParamType(paramExprs *[]ast.ParamMarkerExpr) (err error) {
	return err
}

// PhysicalUnionScan represents a union scan operator.
type PhysicalUnionScan struct {
	basePhysicalPlan

	Conditions []expression.Expression

	HandleCol *expression.Column
}

// SetParamType set the parameter type in ParamMarkerExpr from PhysicalUnionScan
// todo 从PhysicalUnionScan获取参数类型
func (p *PhysicalUnionScan) SetParamType(paramExprs *[]ast.ParamMarkerExpr) (err error) {
	return err
}

// IsPartition returns true and partition ID if it works on a partition.
func (p *PhysicalIndexScan) IsPartition() (bool, int64) {
	return p.isPartition, p.physicalTableID
}

// IsPointGetByUniqueKey checks whether is a point get by unique key.
func (p *PhysicalIndexScan) IsPointGetByUniqueKey(sc *stmtctx.StatementContext) bool {
	return len(p.Ranges) == 1 &&
		p.Index.Unique &&
		len(p.Ranges[0].LowVal) == len(p.Index.Columns) &&
		p.Ranges[0].IsPoint(sc)
}

// PhysicalSelection represents a filter.
type PhysicalSelection struct {
	basePhysicalPlan

	Conditions []expression.Expression
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

// PhysicalMaxOneRow is the physical operator of maxOneRow.
type PhysicalMaxOneRow struct {
	basePhysicalPlan
}

// SetParamType set the parameter type in ParamMarkerExpr from PhysicalMaxOneRow
// todo 从PhysicalMaxOneRow计划中获取参数类型
func (p *PhysicalMaxOneRow) SetParamType(paramExprs *[]ast.ParamMarkerExpr) (err error) {
	return err
}

// PhysicalTableDual is the physical operator of dual.
type PhysicalTableDual struct {
	physicalSchemaProducer

	RowCount int

	// names is used for OutputNames() method. Dual may be inited when building point get plan.
	// So it needs to hold names for itself.
	names []*types.FieldName
}

// SetParamType set the parameter type in ParamMarkerExpr from PhysicalTableDual
// todo 从PhysicalTableDual计划中获取参数类型
func (p *PhysicalTableDual) SetParamType(paramExprs *[]ast.ParamMarkerExpr) (err error) {
	return err
}

// OutputNames returns the outputting names of each column.
func (p *PhysicalTableDual) OutputNames() types.NameSlice {
	return p.names
}

// SetOutputNames sets the outputting name by the given slice.
func (p *PhysicalTableDual) SetOutputNames(names types.NameSlice) {
	p.names = names
}

// PhysicalWindow is the physical operator of window function.
type PhysicalWindow struct {
	physicalSchemaProducer

	WindowFuncDescs []*aggregation.WindowFuncDesc
	PartitionBy     []property.Item
	OrderBy         []property.Item
	Frame           *WindowFrame
}

// SetParamType set the parameter type in ParamMarkerExpr from PhysicalWindow
// todo 从PhysicalWindow获取参数类型
func (p *PhysicalWindow) SetParamType(paramExprs *[]ast.ParamMarkerExpr) (err error) {
	return err
}

// PhysicalShuffle represents a shuffle plan.
// `Tail` and `DataSource` are the last plan within and the first plan following the "shuffle", respectively,
//  to build the child executors chain.
// Take `Window` operator for example:
//  Shuffle -> Window -> Sort -> DataSource, will be separated into:
//    ==> Shuffle: for main thread
//    ==> Window -> Sort(:Tail) -> shuffleWorker: for workers
//    ==> DataSource: for `fetchDataAndSplit` thread
type PhysicalShuffle struct {
	basePhysicalPlan

	Concurrency int
	Tail        PhysicalPlan
	DataSource  PhysicalPlan

	SplitterType PartitionSplitterType
	HashByItems  []expression.Expression
}

// SetParamType set the parameter type in ParamMarkerExpr from PhysicalShuffle
// todo 从PhysicalShuffle计划中获取参数类型
func (shuffle *PhysicalShuffle) SetParamType(paramExprs *[]ast.ParamMarkerExpr) (err error) {
	return err
}

// PartitionSplitterType is the type of `Shuffle` executor splitter, which splits data source into partitions.
type PartitionSplitterType int

const (
	// PartitionHashSplitterType is the splitter splits by hash.
	PartitionHashSplitterType = iota
)

// PhysicalShuffleDataSourceStub represents a data source stub of `PhysicalShuffle`,
// and actually, is executed by `executor.shuffleWorker`.
type PhysicalShuffleDataSourceStub struct {
	physicalSchemaProducer

	// Worker points to `executor.shuffleWorker`.
	Worker unsafe.Pointer
}

// SetParamType set the parameter type in ParamMarkerExpr from PhysicalShuffleDataSourceStub
// todo 从PhysicalShuffleDataSourceStub计划中获取参数类型
func (p *PhysicalShuffleDataSourceStub) SetParamType(paramExprs *[]ast.ParamMarkerExpr) (err error) {
	return err
}

// CollectPlanStatsVersion uses to collect the statistics version of the plan.
func CollectPlanStatsVersion(plan PhysicalPlan, statsInfos map[string]uint64) map[string]uint64 {
	for _, child := range plan.Children() {
		statsInfos = CollectPlanStatsVersion(child, statsInfos)
	}
	switch copPlan := plan.(type) {
	case *PhysicalTableReader:
		statsInfos = CollectPlanStatsVersion(copPlan.tablePlan, statsInfos)
	case *PhysicalIndexReader:
		statsInfos = CollectPlanStatsVersion(copPlan.indexPlan, statsInfos)
	case *PhysicalIndexLookUpReader:
		// For index loop up, only the indexPlan is necessary,
		// because they use the same stats and we do not set the stats info for tablePlan.
		statsInfos = CollectPlanStatsVersion(copPlan.indexPlan, statsInfos)
	case *PhysicalIndexScan:
		statsInfos[copPlan.Table.Name.O] = copPlan.stats.StatsVersion
	case *PhysicalTableScan:
		statsInfos[copPlan.Table.Name.O] = copPlan.stats.StatsVersion
	}

	return statsInfos
}

// PhysicalShow represents a show plan.
type PhysicalShow struct {
	physicalSchemaProducer

	ShowContents
}

// SetParamType set the parameter type of physicalShow plan
// https://www.postgresql.org/docs/current/sql-show.html
// According to the postgresql document, any parameter won't follow the show statement.
// Besides, show statement won't be the sub statement of any other sql statement.
// So, Show statement doesn't need actual implement. Just return nil.
func (show *PhysicalShow) SetParamType(paramExprs *[]ast.ParamMarkerExpr) (err error) {
	return err
}

// PhysicalShowDDLJobs is for showing DDL job list.
type PhysicalShowDDLJobs struct {
	physicalSchemaProducer

	JobNumber int64
}

// SetParamType set the parameter type of physicalShowDDLJobs plan
// https://www.postgresql.org/docs/current/sql-show.html
// Actually, postgresql doesn't support the syntax like "show create table tname"
// And showddl statement won't contain parameter. Just return nil
func (ddl *PhysicalShowDDLJobs) SetParamType(paramExprs *[]ast.ParamMarkerExpr) (err error) {
	return err
}

// BuildMergeJoinPlan builds a PhysicalMergeJoin from the given fields. Currently, it is only used for test purpose.
func BuildMergeJoinPlan(ctx sessionctx.Context, joinType JoinType, leftKeys, rightKeys []*expression.Column) *PhysicalMergeJoin {
	baseJoin := basePhysicalJoin{
		JoinType:      joinType,
		DefaultValues: []types.Datum{types.NewDatum(1), types.NewDatum(1)},
		LeftJoinKeys:  leftKeys,
		RightJoinKeys: rightKeys,
	}
	return PhysicalMergeJoin{basePhysicalJoin: baseJoin}.Init(ctx, nil, 0)
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
