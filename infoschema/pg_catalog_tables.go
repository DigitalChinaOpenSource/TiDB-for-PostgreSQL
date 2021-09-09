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

package infoschema

import (
	"context"
	"github.com/DigitalChinaOpenSource/DCParser/model"
	"github.com/DigitalChinaOpenSource/DCParser/mysql"
	"github.com/pingcap/tidb/kv"
	"github.com/pingcap/tidb/meta/autoid"
	"github.com/pingcap/tidb/sessionctx"
	"github.com/pingcap/tidb/table"
	"github.com/pingcap/tidb/types"
	"sort"
)

// https://www.postgresql.org/docs/13/catalogs.html
// Define the name of the system catalog in pg_catalog.
// TODO: Complete the definition of the remaining table names in pg_catalog.
const (
	// CatalogPgAggregate is the string constant of pg_catalog table.
	CatalogPgAggregate = "pg_aggregate"
	// CatalogPgAm is the string constant of pg_catalog table.
	CatalogPgAm = "pg_am"
	// CatalogPgAmop is the string constant of pg_catalog table.
	CatalogPgAmop = "pg_amop"
	// CatalogPgAmproc is the string constant of pg_catalog table.
	CatalogPgAmproc = "pg_amproc"
	// CatalogPgAttrdef is the string constant of pg_catalog table.
	CatalogPgAttrdef = "pg_attrdef"
	// CatalogPgAttribute is the string constant of pg_catalog table.
	CatalogPgAttribute = "pg_attribute"
	// CatalogPgAuthMembers is the string constant of pg_catalog table.
	CatalogPgAuthMembers = "pg_auth_members"
	// CatalogPgAuthID is the string constant of pg_catalog table.
	CatalogPgAuthID = "pg_authid"
	// CatalogPgCast is the string constant of pg_catalog table.
	CatalogPgCast = "pg_cast"
	// CatalogPgClass is the string constant of pg_catalog table.
	CatalogPgClass = "pg_class"
	// CatalogPgCollation is the string constant of pg_catalog table.
	CatalogPgCollation = "pg_collation"
	// CatalogPgConstraint is the string constant of pg_catalog table.
	CatalogPgConstraint = "pg_constraint"
	// CatalogPgConversion is the string constant of pg_catalog table.
	CatalogPgConversion = "pg_conversion"
	// CatalogPgDatabase is the string constant of pg_catalog table.
	CatalogPgDatabase = "pg_database"
	// CatalogPgDBRoleSetting is the string constant of pg_catalog table.
	CatalogPgDBRoleSetting = "pg_db_role_setting"
	// CatalogPgDefaultAcl is the string constant of pg_catalog table.
	CatalogPgDefaultAcl = "pg_default_acl"
	// CatalogPgDepend is the string constant of pg_catalog table.
	CatalogPgDepend = "pg_depend"
	// CatalogPgDescription is the string constant of pg_catalog table.
	CatalogPgDescription = "pg_description"
	// CatalogPgEnum is the string constant of pg_catalog table.
	CatalogPgEnum = "pg_enum"
	// CatalogPgEventTrigger is the string constant of pg_catalog table.
	CatalogPgEventTrigger = "pg_event_trigger"
	// CatalogPgExtension is the string constant of pg_catalog table.
	CatalogPgExtension = "pg_extension"
	// CatalogPgForeignDataWrapper is the string constant of pg_catalog table.
	CatalogPgForeignDataWrapper = "pg_foreign_data_wrapper"
	// CatalogPgForeignServer is the string constant of pg_catalog table.
	CatalogPgForeignServer = "pg_foreign_server"
	// CatalogPgForeignTable is the string constant of pg_catalog table.
	CatalogPgForeignTable = "pg_foreign_table"
	// CatalogPgIndex is the string constant of pg_catalog table.
	CatalogPgIndex = "pg_index"
	// CatalogPgInherits is the string constant of pg_catalog table.
	CatalogPgInherits = "pg_inherits"
	// CatalogPgInitPrivs is the string constant of pg_catalog table.
	CatalogPgInitPrivs = "pg_init_privs"
	// CatalogPgLanguage is the string constant of pg_catalog table.
	CatalogPgLanguage = "pg_language"
	// CatalogPgLargeObjectMetadata is the string constant of pg_catalog table.
	CatalogPgLargeObjectMetadata = "pg_largeobject_metadata"
	// CatalogPgNamespace is the string constant of pg_catalog table.
	CatalogPgNamespace = "pg_namespace"
	//CatalogPgOpClass is the string constant of pg_catalog table.
	CatalogPgOpClass = "pg_opclass"
	// CatalogPgOperator is the string constant of pg_catalog table.
	CatalogPgOperator = "pg_operator"
	// CatalogPgOpFamily is the string constant of pg_catalog table.
	CatalogPgOpFamily = "pg_opfamily"
	// CatalogPgPartitionedTable is the string constant of pg_catalog table.
	CatalogPgPartitionedTable = "pg_partitioned_table"
)

var catalogTableIDMap = map[string]int64{
	CatalogPgAggregate:           autoid.PgCatalogSchemaDBID + 1,
	CatalogPgAm:                  autoid.PgCatalogSchemaDBID + 2,
	CatalogPgAmop:                autoid.PgCatalogSchemaDBID + 3,
	CatalogPgAmproc:              autoid.PgCatalogSchemaDBID + 4,
	CatalogPgAttrdef:             autoid.PgCatalogSchemaDBID + 5,
	CatalogPgAuthMembers:         autoid.PgCatalogSchemaDBID + 6,
	CatalogPgAuthID:              autoid.PgCatalogSchemaDBID + 7,
	CatalogPgCast:                autoid.PgCatalogSchemaDBID + 8,
	CatalogPgClass:               autoid.PgCatalogSchemaDBID + 9,
	CatalogPgCollation:           autoid.PgCatalogSchemaDBID + 10,
	CatalogPgConstraint:          autoid.PgCatalogSchemaDBID + 11,
	CatalogPgConversion:          autoid.PgCatalogSchemaDBID + 12,
	CatalogPgDatabase:            autoid.PgCatalogSchemaDBID + 13,
	CatalogPgDBRoleSetting:       autoid.PgCatalogSchemaDBID + 14,
	CatalogPgDefaultAcl:          autoid.PgCatalogSchemaDBID + 15,
	CatalogPgDepend:              autoid.PgCatalogSchemaDBID + 16,
	CatalogPgDescription:         autoid.PgCatalogSchemaDBID + 17,
	CatalogPgEnum:                autoid.PgCatalogSchemaDBID + 18,
	CatalogPgEventTrigger:        autoid.PgCatalogSchemaDBID + 19,
	CatalogPgExtension:           autoid.PgCatalogSchemaDBID + 20,
	CatalogPgForeignDataWrapper:  autoid.PgCatalogSchemaDBID + 21,
	CatalogPgForeignServer:       autoid.PgCatalogSchemaDBID + 22,
	CatalogPgForeignTable:        autoid.PgCatalogSchemaDBID + 23,
	CatalogPgIndex:               autoid.PgCatalogSchemaDBID + 24,
	CatalogPgInherits:            autoid.PgCatalogSchemaDBID + 25,
	CatalogPgInitPrivs:           autoid.PgCatalogSchemaDBID + 26,
	CatalogPgLanguage:            autoid.PgCatalogSchemaDBID + 27,
	CatalogPgLargeObjectMetadata: autoid.PgCatalogSchemaDBID + 28,
	CatalogPgNamespace:           autoid.PgCatalogSchemaDBID + 29,
	CatalogPgOpClass:             autoid.PgCatalogSchemaDBID + 30,
	CatalogPgOperator:            autoid.PgCatalogSchemaDBID + 31,
	CatalogPgOpFamily:            autoid.PgCatalogSchemaDBID + 32,
	CatalogPgPartitionedTable:    autoid.PgCatalogSchemaDBID + 33,
}

// TODO: Complete all table structure definitions in pg_catalog.

// catalogPgAggregateCols is table pg_aggregate columns.
// https://www.postgresql.org/docs/13/catalog-pg-aggregate.html
var catalogPgAggregateCols = []columnInfo{
	{name: "aggfnoid", tp: mysql.TypeVarchar, size: 64, flag: mysql.NotNullFlag},
	{name: "aggkind", tp: mysql.TypeVarchar, size: 1, flag: mysql.NotNullFlag},
	{name: "aggnumdirectargs", tp: mysql.TypeTiny, size: 1, flag: mysql.NotNullFlag},
	{name: "aggtransfn", tp: mysql.TypeVarchar, size: 64, flag: mysql.NotNullFlag},
	{name: "aggcombinefn", tp: mysql.TypeVarchar, size: 64, flag: mysql.NotNullFlag},
	{name: "aggserialfn", tp: mysql.TypeVarchar, size: 64, flag: mysql.NotNullFlag},
	{name: "aggdeserialfn", tp: mysql.TypeVarchar, size: 64, flag: mysql.NotNullFlag},
	{name: "aggmtransfn", tp: mysql.TypeVarchar, size: 64, flag: mysql.NotNullFlag},
	{name: "aggminvtransfn", tp: mysql.TypeVarchar, size: 64, flag: mysql.NotNullFlag},
	{name: "aggmfinalfn", tp: mysql.TypeVarchar, size: 64, flag: mysql.NotNullFlag},
	{name: "aggfinalextra", tp: mysql.TypeTiny, size: 1, flag: mysql.NotNullFlag},
	{name: "aggmfinalextra", tp: mysql.TypeTiny, size: 1, flag: mysql.NotNullFlag},
	{name: "aggfinalmodify", tp: mysql.TypeVarchar, size: 1, flag: mysql.NotNullFlag},
	{name: "aggmfinalmodify", tp: mysql.TypeVarchar, size: 1, flag: mysql.NotNullFlag},
	{name: "aggsortop", tp: mysql.TypeLonglong, size: 32, flag: mysql.NotNullFlag | mysql.UnsignedFlag},
	{name: "aggtranstype", tp: mysql.TypeLonglong, size: 32, flag: mysql.NotNullFlag | mysql.UnsignedFlag},
	{name: "aggtransspace", tp: mysql.TypeLonglong, size: 32, flag: mysql.NotNullFlag},
	{name: "aggmtranstype", tp: mysql.TypeLonglong, size: 32, flag: mysql.NotNullFlag | mysql.UnsignedFlag},
	{name: "aggmtransspace", tp: mysql.TypeLonglong, size: 32, flag: mysql.NotNullFlag},
	{name: "agginitval", tp: mysql.TypeJSON, size: 128, flag: mysql.NotNullFlag},
	{name: "aggminitval", tp: mysql.TypeJSON, size: 128, flag: mysql.NotNullFlag},
}

// catalogPgClassCols is table pg_class columns.
// https://www.postgresql.org/docs/13/catalog-pg-class.html
var catalogPgClassCols = []columnInfo{
	{name: "oid", tp: mysql.TypeLonglong, size: 32, flag: mysql.NotNullFlag | mysql.UnsignedFlag},
	{name: "relname", tp: mysql.TypeVarchar, size: 64, flag: mysql.NotNullFlag},
	{name: "relnamespace", tp: mysql.TypeLonglong, size: 32, flag: mysql.NotNullFlag | mysql.UnsignedFlag},
	{name: "reltype", tp: mysql.TypeLonglong, size: 32, flag: mysql.NotNullFlag | mysql.UnsignedFlag},
	{name: "reloftype", tp: mysql.TypeLonglong, size: 32, flag: mysql.NotNullFlag | mysql.UnsignedFlag},
	{name: "relowner", tp: mysql.TypeLonglong, size: 32, flag: mysql.NotNullFlag | mysql.UnsignedFlag},
	{name: "relam", tp: mysql.TypeLonglong, size: 32, flag: mysql.NotNullFlag | mysql.UnsignedFlag},
	{name: "relfilenode", tp: mysql.TypeLonglong, size: 32, flag: mysql.NotNullFlag | mysql.UnsignedFlag},
	{name: "reltablespace", tp: mysql.TypeLonglong, size: 32, flag: mysql.NotNullFlag | mysql.UnsignedFlag},
	{name: "relpages", tp: mysql.TypeLonglong, size: 32, flag: mysql.NotNullFlag},
	{name: "reltuples", tp: mysql.TypeFloat, size: 1, flag: mysql.NotNullFlag},
	{name: "relallvisible", tp: mysql.TypeLonglong, size: 32, flag: mysql.NotNullFlag},
	{name: "reltoastrelid", tp: mysql.TypeLonglong, size: 32, flag: mysql.NotNullFlag | mysql.UnsignedFlag},
	{name: "relhasindex", tp: mysql.TypeTiny, size: 1, flag: mysql.NotNullFlag},
	{name: "relisshared", tp: mysql.TypeTiny, size: 1, flag: mysql.NotNullFlag},
	{name: "relpersistence", tp: mysql.TypeVarchar, size: 1, flag: mysql.NotNullFlag},
	{name: "relkind", tp: mysql.TypeVarchar, size: 1, flag: mysql.NotNullFlag},
	{name: "relnatts", tp: mysql.TypeLong, size: 4, flag: mysql.NotNullFlag},
	{name: "relchecks", tp: mysql.TypeLong, size: 4, flag: mysql.NotNullFlag},
	{name: "relhasrules", tp: mysql.TypeTiny, size: 1, flag: mysql.NotNullFlag},
	{name: "relhastriggers", tp: mysql.TypeTiny, size: 1, flag: mysql.NotNullFlag},
	{name: "relhassubclass", tp: mysql.TypeTiny, size: 1, flag: mysql.NotNullFlag},
	{name: "relrowsecurity", tp: mysql.TypeTiny, size: 1, flag: mysql.NotNullFlag},
	{name: "relforcerowsecurity", tp: mysql.TypeTiny, size: 1, flag: mysql.NotNullFlag},
	{name: "relispopulated", tp: mysql.TypeTiny, size: 1, flag: mysql.NotNullFlag},
	{name: "relreplident", tp: mysql.TypeVarchar, size: 1, flag: mysql.NotNullFlag},
	{name: "relispartition", tp: mysql.TypeTiny, size: 1, flag: mysql.NotNullFlag},
	{name: "relrewrite", tp: mysql.TypeLonglong, size: 32, flag: mysql.NotNullFlag | mysql.UnsignedFlag},
	{name: "relfrozenxid", tp: mysql.TypeLonglong, size: 32, flag: mysql.NotNullFlag | mysql.UnsignedFlag},
	{name: "relminmxid", tp: mysql.TypeLonglong, size: 32, flag: mysql.NotNullFlag | mysql.UnsignedFlag},
	{name: "relacl", tp: mysql.TypeVarchar, size: 255},
	{name: "reloptions", tp: mysql.TypeJSON, size: 128},
	{name: "relpartbound", tp: mysql.TypeJSON, size: 128},
}

// catalogPgInheritsCols is table pg_inherits columns.
// https://www.postgresql.org/docs/13/catalog-pg-inherits.html
var catalogPgInheritsCols = []columnInfo{
	{name: "inhrelid", tp: mysql.TypeLonglong, size: 32, flag: mysql.NotNullFlag | mysql.UnsignedFlag},
	{name: "inhparent", tp: mysql.TypeLonglong, size: 32, flag: mysql.NotNullFlag | mysql.UnsignedFlag},
	{name: "inhseqno", tp: mysql.TypeLonglong, size: 32, flag: mysql.NotNullFlag},
}

// catalogPgNamespaceCols is table pg_namespace columns.
// https://www.postgresql.org/docs/13/catalog-pg-namespace.html
var catalogPgNamespaceCols = []columnInfo{
	{name: "oid", tp: mysql.TypeLonglong, size: 32, flag: mysql.NotNullFlag | mysql.UnsignedFlag},
	{name: "nspname", tp: mysql.TypeVarchar, size: 32, flag: mysql.NotNullFlag},
	{name: "nspowner", tp: mysql.TypeLonglong, size: 32, flag: mysql.NotNullFlag | mysql.UnsignedFlag},
	{name: "nspacl", tp: mysql.TypeVarchar, size: 255},
}

// catalogPgPartitionedTableCols is table pg_partitioned_table columns
// https://www.postgresql.org/docs/13/catalog-pg-partitioned-table.html
var catalogPgPartitionedTableCols = []columnInfo{
	{name: "partrelid", tp: mysql.TypeLonglong, size: 32, flag: mysql.NotNullFlag | mysql.UnsignedFlag},
	{name: "partstrat", tp: mysql.TypeVarchar, size: 1, flag: mysql.NotNullFlag},
	{name: "partnatts", tp: mysql.TypeLong, flag: mysql.NotNullFlag},
	{name: "partdefid", tp: mysql.TypeLonglong, size: 32, flag: mysql.NotNullFlag},
	{name: "partattrs", tp: mysql.TypeVarchar, size: 64, flag: mysql.NotNullFlag},
	{name: "partclass", tp: mysql.TypeVarchar, size: 64, flag: mysql.NotNullFlag},
	{name: "partcollation", tp: mysql.TypeVarchar, size: 64, flag: mysql.NotNullFlag},
	{name: "partexprs", tp: mysql.TypeJSON},
}

var catalogTableNameToColumns = map[string][]columnInfo{
	CatalogPgAggregate:        catalogPgAggregateCols,
	CatalogPgClass:            catalogPgClassCols,
	CatalogPgInherits:         catalogPgInheritsCols,
	CatalogPgNamespace:        catalogPgNamespaceCols,
	CatalogPgPartitionedTable: catalogPgPartitionedTableCols,
}

func createPgCatalogTable(_ autoid.Allocators, meta *model.TableInfo) (table.Table, error) {
	columns := make([]*table.Column, len(meta.Columns))
	for i, col := range meta.Columns {
		columns[i] = table.ToColumn(col)
	}
	tp := table.VirtualTable
	return &pgCatalogTable{meta: meta, cols: columns, tp: tp}, nil
}

type pgCatalogTable struct {
	meta *model.TableInfo
	cols []*table.Column
	tp   table.Type
}

func (pt *pgCatalogTable) getRows(ctx sessionctx.Context, cols []*table.Column) (fullRows [][]types.Datum, err error) {
	is := GetInfoSchema(ctx)
	dbs := is.AllSchemas()
	sort.Sort(SchemasSorter(dbs))
	switch pt.meta.Name.O {
	// TODO: Fill the rest tables
	case tableTiKVRegionStatus:
		fullRows, err = dataForTiKVRegionStatus(ctx)
	}
	if err != nil {
		return nil, err
	}
	if len(cols) == len(pt.cols) {
		return
	}
	rows := make([][]types.Datum, len(fullRows))
	for i, fullRow := range fullRows {
		row := make([]types.Datum, len(cols))
		for j, col := range cols {
			row[j] = fullRow[col.Offset]
		}
		rows[i] = row
	}
	return rows, nil
}

// IterRecords implements table.Table IterRecords interface.
func (pt *pgCatalogTable) IterRecords(ctx sessionctx.Context, startKey kv.Key, cols []*table.Column,
	fn table.RecordIterFunc) error {
	if len(startKey) != 0 {
		return table.ErrUnsupportedOp
	}
	rows, err := pt.getRows(ctx, cols)
	if err != nil {
		return err
	}
	for i, row := range rows {
		more, err := fn(int64(i), row, cols)
		if err != nil {
			return err
		}
		if !more {
			break
		}
	}
	return nil
}

// RowWithCols implements table.Table RowWithCols interface.
func (pt *pgCatalogTable) RowWithCols(ctx sessionctx.Context, h int64, cols []*table.Column) ([]types.Datum, error) {
	return nil, table.ErrUnsupportedOp
}

// Row implements table.Table Row interface.
func (pt *pgCatalogTable) Row(ctx sessionctx.Context, h int64) ([]types.Datum, error) {
	return nil, table.ErrUnsupportedOp
}

// Cols implements table.Table Cols interface.
func (pt *pgCatalogTable) Cols() []*table.Column {
	return pt.cols
}

// VisibleCols implements table.Table VisibleCols interface.
func (pt *pgCatalogTable) VisibleCols() []*table.Column {
	return pt.cols
}

// HiddenCols implements table.Table HiddenCols interface.
func (pt *pgCatalogTable) HiddenCols() []*table.Column {
	return nil
}

// WritableCols implements table.Table WritableCols interface.
func (pt *pgCatalogTable) WritableCols() []*table.Column {
	return pt.cols
}

// DeletableCols implements table DeletableCols interface.
func (pt *pgCatalogTable) DeletableCols() []*table.Column {
	return pt.cols
}

// Indices implements table.Table Indices interface.
func (pt *pgCatalogTable) Indices() []table.Index {
	return nil
}

// WritableIndices implements table.Table WritableIndices interface.
func (pt *pgCatalogTable) WritableIndices() []table.Index {
	return nil
}

// DeletableIndices implements table.Table DeletableIndices interface.
func (pt *pgCatalogTable) DeletableIndices() []table.Index {
	return nil
}

// RecordPrefix implements table.Table RecordPrefix interface.
func (pt *pgCatalogTable) RecordPrefix() kv.Key {
	return nil
}

// IndexPrefix implements table.Table IndexPrefix interface.
func (pt *pgCatalogTable) IndexPrefix() kv.Key {
	return nil
}

// FirstKey implements table.Table FirstKey interface.
func (pt *pgCatalogTable) FirstKey() kv.Key {
	return nil
}

// RecordKey implements table.Table RecordKey interface.
func (pt *pgCatalogTable) RecordKey(h int64) kv.Key {
	return nil
}

// AddRecord implements table.Table AddRecord interface.
func (pt *pgCatalogTable) AddRecord(ctx sessionctx.Context, r []types.Datum, opts ...table.AddRecordOption) (recordID int64, err error) {
	return 0, table.ErrUnsupportedOp
}

// RemoveRecord implements table.Table RemoveRecord interface.
func (pt *pgCatalogTable) RemoveRecord(ctx sessionctx.Context, h int64, r []types.Datum) error {
	return table.ErrUnsupportedOp
}

// UpdateRecord implements table.Table UpdateRecord interface.
func (pt *pgCatalogTable) UpdateRecord(ctx context.Context, sctx sessionctx.Context, h int64, oldData, newData []types.Datum, touched []bool) error {
	return table.ErrUnsupportedOp
}

// Allocators implements table.Table Allocators interface.
func (pt *pgCatalogTable) Allocators(_ sessionctx.Context) autoid.Allocators {
	return nil
}

// RebaseAutoID implements table.Table RebaseAutoID interface.
func (pt *pgCatalogTable) RebaseAutoID(ctx sessionctx.Context, newBase int64, isSetStep bool, tp autoid.AllocatorType) error {
	return table.ErrUnsupportedOp
}

// Meta implements table.Table Meta interface.
func (pt *pgCatalogTable) Meta() *model.TableInfo {
	return pt.meta
}

// GetPhysicalID implements table.Table GetPhysicalID interface.
func (pt *pgCatalogTable) GetPhysicalID() int64 {
	return pt.meta.ID
}

// Seek implements table.Table Seek interface.
func (pt *pgCatalogTable) Seek(ctx sessionctx.Context, h int64) (int64, bool, error) {
	return 0, false, table.ErrUnsupportedOp
}

// Type implements table.Table Type interface.
func (pt *pgCatalogTable) Type() table.Type {
	return pt.tp
}
