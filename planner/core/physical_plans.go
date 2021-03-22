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

package core

import (
	driver "github.com/pingcap/tidb/types/parser_driver"
	"unsafe"

	"github.com/pingcap/parser/ast"
	"github.com/pingcap/parser/model"
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

// PGSQL Modified
// 在tableReader计划中获取参数类型，目前考虑的sql比较简单，也就是在tablereader的tabplan有个selection子计划，调用子计划的setParamType方法即可
// 注意处理 paramExprs 的里面各参数类型的index
func (tableReader *PhysicalTableReader) SetParamType(paramExprs *[]ast.ParamMarkerExpr, cols *[]*expression.Column, paramMarkerIndex int) (currIndex int, err error) {
	if tablePlan := tableReader.tablePlan; tablePlan != nil {
		paramMarkerIndex,err = tablePlan.SetParamType(paramExprs, &tableReader.physicalSchemaProducer.schema.Columns, paramMarkerIndex)
	}
	return paramMarkerIndex, err
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

//PGSQL Modified
func (p *PhysicalIndexReader) SetParamType(paramExprs *[]ast.ParamMarkerExpr, cols *[]*expression.Column, paramMarkerIndex int) (currIndex int, err error) {
	return currIndex, err
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

// PGSQL Modified
func (p *PhysicalIndexLookUpReader) SetParamType(paramExprs *[]ast.ParamMarkerExpr, cols *[]*expression.Column, paramMarkerIndex int) (currIndex int, err error) {
	return currIndex, err
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

//PGSQL Modified
func (p *PhysicalIndexMergeReader) SetParamType(paramExprs *[]ast.ParamMarkerExpr, cols *[]*expression.Column, paramMarkerIndex int) (currIndex int, err error) {
	return currIndex, err
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

// PGSQL Modified
func (p *PhysicalIndexScan) SetParamType(paramExprs *[]ast.ParamMarkerExpr, cols *[]*expression.Column, paramMarkerIndex int) (currIndex int, err error) {
	return currIndex, err
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

// PGSQL Modified
// SetParamType 设置参数类型
func (men *PhysicalMemTable) SetParamType(paramExprs *[]ast.ParamMarkerExpr, cols *[]*expression.Column, paramMarkerIndex int) (currParamMarkerIndex int, err error) {
	return currParamMarkerIndex, err
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

// PGSQL Modified
func (ts *PhysicalTableScan) SetParamType(paramExprs *[]ast.ParamMarkerExpr, cols *[]*expression.Column, paramMarkerIndex int) (currIndex int, err error) {
	if childrenPlan := ts.children; childrenPlan != nil {
		for i := range childrenPlan {
			paramMarkerIndex,err = childrenPlan[i].SetParamType(paramExprs,cols,paramMarkerIndex)
		}
	}
	return paramMarkerIndex, err
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

// PGSQL Modified
// SetParamType 设置参数类型，需要考虑具体计划内部参数体哦阿健的分布
func (p *PhysicalProjection) SetParamType(paramExprs *[]ast.ParamMarkerExpr, cols *[]*expression.Column, paramMarkerIndex int) (currIndex int, err error) {
	return currIndex, err
}

// PhysicalTopN is the physical operator of topN.
type PhysicalTopN struct {
	basePhysicalPlan

	ByItems []*util.ByItems
	Offset  uint64
	Count   uint64
}

// PGSQL Modified
func (lt *PhysicalTopN) SetParamType(paramExprs *[]ast.ParamMarkerExpr, cols *[]*expression.Column, paramMarkerIndex int) (currIndex int, err error) {
	return currIndex, err
}

// PhysicalApply represents apply plan, only used for subquery.
type PhysicalApply struct {
	PhysicalHashJoin

	OuterSchema []*expression.CorrelatedColumn
}

// PGSQL Modified
func (la *PhysicalApply) SetParamType(paramExprs *[]ast.ParamMarkerExpr, cols *[]*expression.Column, paramMarkerIndex int) (currIndex int, err error) {
	return currIndex, err
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

// PGSQL Modified
func (p *PhysicalHashJoin) SetParamType(paramExprs *[]ast.ParamMarkerExpr, cols *[]*expression.Column, paramMarkerIndex int) (currIndex int, err error) {
	return currIndex, err
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

// PGSQL Modified
func (indexJoin *PhysicalIndexJoin) SetParamType(paramExprs *[]ast.ParamMarkerExpr, cols *[]*expression.Column, paramMarkerIndex int) (currIndex int, err error) {
	return currIndex, err
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

//PGSQL Modified
func (join *PhysicalMergeJoin) SetParamType(paramExprs *[]ast.ParamMarkerExpr, cols *[]*expression.Column, paramMarkerIndex int) (currIndex int, err error) {
	return currIndex, err
}

// PhysicalBroadCastJoin only works for TiFlash Engine, which broadcast the small table to every replica of probe side of tables.
type PhysicalBroadCastJoin struct {
	basePhysicalJoin
	globalChildIndex int
}

// PGSQL Modified
func (lock *PhysicalBroadCastJoin) SetParamType(paramExprs *[]ast.ParamMarkerExpr, cols *[]*expression.Column, paramMarkerIndex int) (currIndex int, err error) {
	return currIndex, err
}

// PhysicalLock is the physical operator of lock, which is used for `select ... for update` clause.
type PhysicalLock struct {
	basePhysicalPlan

	Lock ast.SelectLockType

	TblID2Handle     map[int64][]*expression.Column
	PartitionedTable []table.PartitionedTable
}

// PGSQL Modified
func (lock *PhysicalLock) SetParamType(paramExprs *[]ast.ParamMarkerExpr, cols *[]*expression.Column, paramMarkerIndex int) (currIndex int, err error) {
	return currIndex, err
}

// PhysicalLimit is the physical operator of Limit.
type PhysicalLimit struct {
	basePhysicalPlan

	Offset uint64
	Count  uint64
}

// PGSQL Modified
func (p *PhysicalLimit) SetParamType(paramExprs *[]ast.ParamMarkerExpr, cols *[]*expression.Column, paramMarkerIndex int) (currIndex int, err error) {
	return currIndex, err
}

// PhysicalUnionAll is the physical operator of UnionAll.
type PhysicalUnionAll struct {
	physicalSchemaProducer
}

// PGSQL Modified
func (unionAll *PhysicalUnionAll) SetParamType(paramExprs *[]ast.ParamMarkerExpr, cols *[]*expression.Column, paramMarkerIndex int) (currIndex int, err error) {
	return currIndex, err
}

type basePhysicalAgg struct {
	physicalSchemaProducer

	AggFuncs     []*aggregation.AggFuncDesc
	GroupByItems []expression.Expression
}

// PGSQL Modified
func (agg *basePhysicalAgg) SetParamType(paramExprs *[]ast.ParamMarkerExpr, cols *[]*expression.Column, paramMarkerIndex int) (currParamMarkerIndex int, err error) {
	return currParamMarkerIndex, err
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

// PGSQL Modified
func (agg *PhysicalHashAgg) SetParamType(paramExprs *[]ast.ParamMarkerExpr, cols *[]*expression.Column, paramMarkerIndex int) (currIndex int, err error) {
	return currIndex, err
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

// SetParamType
// PGSQL Modified
func (agg *PhysicalStreamAgg) SetParamType(paramExprs *[]ast.ParamMarkerExpr, cols *[]*expression.Column, paramMarkerIndex int) (currIndex int, err error) {
	return currIndex, err
}

// PhysicalSort is the physical operator of sort, which implements a memory sort.
type PhysicalSort struct {
	basePhysicalPlan

	ByItems []*util.ByItems
}

// PGSQL Modified
func (ls *PhysicalSort) SetParamType(paramExprs *[]ast.ParamMarkerExpr, cols *[]*expression.Column, paramMarkerIndex int) (currIndex int, err error) {
	return currIndex, err
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

//PGSQL Modified
func (sort *NominalSort) SetParamType(paramExprs *[]ast.ParamMarkerExpr, cols *[]*expression.Column, paramMarkerIndex int) (currIndex int, err error) {
	return currIndex, err
}

// PhysicalUnionScan represents a union scan operator.
type PhysicalUnionScan struct {
	basePhysicalPlan

	Conditions []expression.Expression

	HandleCol *expression.Column
}

// PGSQL Modified
func (p *PhysicalUnionScan) SetParamType(paramExprs *[]ast.ParamMarkerExpr, cols *[]*expression.Column, paramMarkerIndex int) (currIndex int, err error) {
	return currIndex, err
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

// PGSQL Modified
// SetParamType 设置paramExprs参数类型，需要考虑具体计划内部参数的分布
func (p *PhysicalSelection) SetParamType(paramExprs *[]ast.ParamMarkerExpr, cols *[]*expression.Column, paramMarkerIndex int) (currIndex int, err error) {
	if childrenPlan := p.children; childrenPlan != nil {
		for i := range childrenPlan {
			plan := childrenPlan[i]
			if paramMarkerIndex ,err = plan.SetParamType(paramExprs,cols,paramMarkerIndex); err != nil {
				return paramMarkerIndex, err
			}
		}
	}

	if p.Conditions != nil {
		paramMarkerIndex = DeepFirstTravsalTree(p.Conditions,paramExprs,cols,paramMarkerIndex)
	}
	return paramMarkerIndex, err
}

//深度优先遍历条件节点，获取到参数类型数组
// todo 完善多层select嵌套的情况。
func DeepFirstTravsalTree(exprs []expression.Expression, paramExprs *[]ast.ParamMarkerExpr, cols *[]*expression.Column,paramIndex int) int {
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
			paramIndex = DoDeepFirstTraverSal(scalar.Function.Args(), paramExprs, cols, paramIndex)
		}
	}
	return paramIndex
}

// 深度优先遍历树
func DoDeepFirstTraverSal(args []expression.Expression, paramExprs *[]ast.ParamMarkerExpr, cols *[]*expression.Column, paramIndex int) int {
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

// SetParamTypes 设置参数类型，根据给出的column信息和当前节点信息对比，找到其参数类型
func SetParamTypes(args []expression.Expression, paramExprs *[]ast.ParamMarkerExpr, cols *[]*expression.Column, paramIndex int) bool {
	if constant, ok := args[1].(*expression.Constant); ok && constant.Value.Kind() == 0 {
		if column, ok := args[0].(*expression.Column); ok {
			for i := range *cols {
				if paramMarker, ok := (*paramExprs)[paramIndex].(*driver.ParamMarkerExpr); (*cols)[i].OrigName == column.OrigName && ok {
					paramMarker.TexprNode.Type = *(*cols)[i].RetType
					break
				}
			}
		}
		return true
	} else {
		return false
	}
}

// PhysicalMaxOneRow is the physical operator of maxOneRow.
type PhysicalMaxOneRow struct {
	basePhysicalPlan
}

// PGSQL Modified
func (maxOneRow *PhysicalMaxOneRow) SetParamType(paramExprs *[]ast.ParamMarkerExpr, cols *[]*expression.Column, paramMarkerIndex int) (currIndex int, err error) {
	return currIndex, err
}

// PhysicalTableDual is the physical operator of dual.
type PhysicalTableDual struct {
	physicalSchemaProducer

	RowCount int

	// names is used for OutputNames() method. Dual may be inited when building point get plan.
	// So it needs to hold names for itself.
	names []*types.FieldName
}

//PGSQL Modified
func (p *PhysicalTableDual) SetParamType(paramExprs *[]ast.ParamMarkerExpr, cols *[]*expression.Column, paramMarkerIndex int) (currIndex int, err error) {
	return currIndex, err
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

// PGSQL Modified
func (p *PhysicalWindow) SetParamType(paramExprs *[]ast.ParamMarkerExpr, cols *[]*expression.Column, paramMarkerIndex int) (currIndex int, err error) {
	return currIndex, err
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

// PGSQL Modified
func (shuffle *PhysicalShuffle) SetParamType(paramExprs *[]ast.ParamMarkerExpr, cols *[]*expression.Column, paramMarkerIndex int) (currIndex int, err error) {
	return currIndex, err
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

// PGSQL Modified
func (p *PhysicalShuffleDataSourceStub) SetParamType(paramExprs *[]ast.ParamMarkerExpr, cols *[]*expression.Column, paramMarkerIndex int) (currIndex int, err error) {
	return currIndex, err
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

// PGSQL Modified
func (show *PhysicalShow) SetParamType(paramExprs *[]ast.ParamMarkerExpr, cols *[]*expression.Column, paramMarkerIndex int) (currParamMarkerIndex int, err error) {
	return currParamMarkerIndex,err
}

// PhysicalShowDDLJobs is for showing DDL job list.
type PhysicalShowDDLJobs struct {
	physicalSchemaProducer

	JobNumber int64
}

//PGSQL Modified
func (ddl *PhysicalShowDDLJobs) SetParamType(paramExprs *[]ast.ParamMarkerExpr, cols *[]*expression.Column, paramMarkerIndex int) (currParamMarkerIndex int, err error) {
	return currParamMarkerIndex, err
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
