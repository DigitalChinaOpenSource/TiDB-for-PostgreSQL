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
	"fmt"
	"github.com/pingcap/tidb/ddl/placement"
	"github.com/pingcap/tidb/meta/autoid"
	"github.com/pingcap/tidb/parser/model"
	"github.com/pingcap/tidb/parser/mysql"
	"github.com/pingcap/tidb/table"
	"github.com/pingcap/tidb/util"
	"sync"
)

func init() {
	// Initialize the pg_catalog shema database and register the driver to `drivers`
	pgCatalogDBID := autoid.PgCatalogSchemaDBID
	pgCatalogTables := make([]*model.TableInfo, 0, len(catalogTableNameToColumns))
	for name, cols := range catalogTableNameToColumns {
		tableInfo := buildTableMeta(name, cols)
		pgCatalogTables = append(pgCatalogTables, tableInfo)
		var ok bool
		tableInfo.ID, ok = catalogTableIDMap[tableInfo.Name.O]
		if !ok {
			panic(fmt.Sprintf("get information_schema table id failed, unknown system table `%v`", tableInfo.Name.O))
		}
		for i, c := range tableInfo.Columns {
			c.ID = int64(i) + 1
		}
	}

	pgCatalogDB := &model.DBInfo{
		ID:      pgCatalogDBID,
		Name:    util.PgCatalogName,
		Charset: mysql.DefaultCharset,
		Collate: mysql.DefaultCollationName,
		Tables:  pgCatalogTables,
	}

	RegisterVirtualTable(pgCatalogDB, createPgCatalogTable)
}

// PgCatalog is the interface used to retrieve the schema information.
// It works as a in memory cache and doesn't handle any schema change.
// PgCatalog is read-only, and the returned value is a copy.
// TODO: add more methods to retrieve tables and columns.
type PgCatalog interface {
	SchemaByName(schema model.CIStr) (*model.DBInfo, bool)
	SchemaExists(schema model.CIStr) bool
	TableByName(schema, table model.CIStr) (table.Table, error)
	TableExists(schema, table model.CIStr) bool
	SchemaByID(id int64) (*model.DBInfo, bool)
	SchemaByTable(tableInfo *model.TableInfo) (*model.DBInfo, bool)
	PolicyByName(name model.CIStr) (*model.PolicyInfo, bool)
	TableByID(id int64) (table.Table, bool)
	AllocByID(id int64) (autoid.Allocators, bool)
	AllSchemaNames() []string
	AllSchemas() []*model.DBInfo
	Clone() (result []*model.DBInfo)
	SchemaTables(schema model.CIStr) []table.Table
	SchemaMetaVersion() int64
	// TableIsView indicates whether the schema.table is a view.
	TableIsView(schema, table model.CIStr) bool
	// TableIsSequence indicates whether the schema.table is a sequence.
	TableIsSequence(schema, table model.CIStr) bool
	FindTableByPartitionID(partitionID int64) (table.Table, *model.DBInfo, *model.PartitionDefinition)
	// BundleByName is used to get a rule bundle.
	BundleByName(name string) (*placement.Bundle, bool)
	// SetBundle is used internally to update rule bundles or mock tests.
	SetBundle(*placement.Bundle)
	// RuleBundles will return a copy of all rule bundles.
	RuleBundles() []*placement.Bundle
	// AllPlacementPolicies returns all placement policies
	AllPlacementPolicies() []*model.PolicyInfo
}

type pgCatalog struct {
	// ruleBundleMap stores all placement rules
	ruleBundleMutex sync.RWMutex
	ruleBundleMap   map[string]*placement.Bundle

	// policyMap stores all placement policies.
	policyMutex sync.RWMutex
	policyMap   map[string]*model.PolicyInfo

	schemaMap map[string]*schemaTables

	// sortedTablesBuckets is a slice of sortedTables, a table's bucket index is (tableID % bucketCount).
	sortedTablesBuckets []sortedTables

	// schemaMetaVersion is the version of schema, and we should check version when change schema.
	schemaMetaVersion int64
}

var _ PgCatalog = (*pgCatalog)(nil)

func (is *pgCatalog) SchemaByName(schema model.CIStr) (val *model.DBInfo, ok bool) {
	tableNames, ok := is.schemaMap[schema.L]
	if !ok {
		return
	}
	return tableNames.dbInfo, true
}

func (is *pgCatalog) SchemaMetaVersion() int64 {
	return is.schemaMetaVersion
}

func (is *pgCatalog) SchemaExists(schema model.CIStr) bool {
	_, ok := is.schemaMap[schema.L]
	return ok
}

func (is *pgCatalog) TableByName(schema, table model.CIStr) (t table.Table, err error) {
	if tbNames, ok := is.schemaMap[schema.L]; ok {
		if t, ok = tbNames.tables[table.L]; ok {
			return
		}
	}
	return nil, ErrTableNotExists.GenWithStackByArgs(schema, table)
}

func (is *pgCatalog) TableIsView(schema, table model.CIStr) bool {
	if tbNames, ok := is.schemaMap[schema.L]; ok {
		if t, ok := tbNames.tables[table.L]; ok {
			return t.Meta().IsView()
		}
	}
	return false
}

func (is *pgCatalog) TableIsSequence(schema, table model.CIStr) bool {
	if tbNames, ok := is.schemaMap[schema.L]; ok {
		if t, ok := tbNames.tables[table.L]; ok {
			return t.Meta().IsSequence()
		}
	}
	return false
}

func (is *pgCatalog) TableExists(schema, table model.CIStr) bool {
	if tbNames, ok := is.schemaMap[schema.L]; ok {
		if _, ok = tbNames.tables[table.L]; ok {
			return true
		}
	}
	return false
}

func (is *pgCatalog) PolicyByID(id int64) (val *model.PolicyInfo, ok bool) {
	// TODO: use another hash map to avoid traveling on the policy map
	for _, v := range is.policyMap {
		if v.ID == id {
			return v, true
		}
	}
	return nil, false
}

func (is *pgCatalog) SchemaByID(id int64) (val *model.DBInfo, ok bool) {
	for _, v := range is.schemaMap {
		if v.dbInfo.ID == id {
			return v.dbInfo, true
		}
	}
	return nil, false
}

func (is *pgCatalog) SchemaByTable(tableInfo *model.TableInfo) (val *model.DBInfo, ok bool) {
	if tableInfo == nil {
		return nil, false
	}
	for _, v := range is.schemaMap {
		if tbl, ok := v.tables[tableInfo.Name.L]; ok {
			if tbl.Meta().ID == tableInfo.ID {
				return v.dbInfo, true
			}
		}
	}
	return nil, false
}

func (is *pgCatalog) TableByID(id int64) (val table.Table, ok bool) {
	slice := is.sortedTablesBuckets[tableBucketIdx(id)]
	idx := slice.searchTable(id)
	if idx == -1 {
		return nil, false
	}
	return slice[idx], true
}

func (is *pgCatalog) AllocByID(id int64) (autoid.Allocators, bool) {
	tbl, ok := is.TableByID(id)
	if !ok {
		return nil, false
	}
	return tbl.Allocators(nil), true
}

func (is *pgCatalog) AllSchemaNames() (names []string) {
	for _, v := range is.schemaMap {
		names = append(names, v.dbInfo.Name.O)
	}
	return
}

func (is *pgCatalog) AllSchemas() (schemas []*model.DBInfo) {
	for _, v := range is.schemaMap {
		schemas = append(schemas, v.dbInfo)
	}
	return
}

func (is *pgCatalog) SchemaTables(schema model.CIStr) (tables []table.Table) {
	schemaTables, ok := is.schemaMap[schema.L]
	if !ok {
		return
	}
	for _, tbl := range schemaTables.tables {
		tables = append(tables, tbl)
	}
	return
}

// FindTableByPartitionID finds the partition-table info by the partitionID.
// FindTableByPartitionID will traverse all the tables to find the partitionID partition in which partition-table.
func (is *pgCatalog) FindTableByPartitionID(partitionID int64) (table.Table, *model.DBInfo, *model.PartitionDefinition) {
	for _, v := range is.schemaMap {
		for _, tbl := range v.tables {
			pi := tbl.Meta().GetPartitionInfo()
			if pi == nil {
				continue
			}
			for _, p := range pi.Definitions {
				if p.ID == partitionID {
					return tbl, v.dbInfo, &p
				}
			}
		}
	}
	return nil, nil, nil
}

func (is *pgCatalog) Clone() (result []*model.DBInfo) {
	for _, v := range is.schemaMap {
		result = append(result, v.dbInfo.Clone())
	}
	return
}

// PolicyByName is used to find the policy.
func (is *pgCatalog) PolicyByName(name model.CIStr) (*model.PolicyInfo, bool) {
	is.policyMutex.RLock()
	defer is.policyMutex.RUnlock()
	t, r := is.policyMap[name.L]
	return t, r
}

// AllPlacementPolicies returns all placement policies
func (is *pgCatalog) AllPlacementPolicies() []*model.PolicyInfo {
	is.policyMutex.RLock()
	defer is.policyMutex.RUnlock()
	policies := make([]*model.PolicyInfo, 0, len(is.policyMap))
	for _, policy := range is.policyMap {
		policies = append(policies, policy)
	}
	return policies
}

func (is *pgCatalog) BundleByName(name string) (*placement.Bundle, bool) {
	is.ruleBundleMutex.RLock()
	defer is.ruleBundleMutex.RUnlock()
	t, r := is.ruleBundleMap[name]
	return t, r
}

func (is *pgCatalog) RuleBundles() []*placement.Bundle {
	is.ruleBundleMutex.RLock()
	defer is.ruleBundleMutex.RUnlock()
	bundles := make([]*placement.Bundle, 0, len(is.ruleBundleMap))
	for _, bundle := range is.ruleBundleMap {
		bundles = append(bundles, bundle)
	}
	return bundles
}

func (is *pgCatalog) setPolicy(policy *model.PolicyInfo) {
	is.policyMutex.Lock()
	defer is.policyMutex.Unlock()
	is.policyMap[policy.Name.L] = policy
}

func (is *pgCatalog) deletePolicy(name string) {
	is.policyMutex.Lock()
	defer is.policyMutex.Unlock()
	delete(is.policyMap, name)
}

func (is *pgCatalog) SetBundle(bundle *placement.Bundle) {
	is.ruleBundleMutex.Lock()
	defer is.ruleBundleMutex.Unlock()
	is.ruleBundleMap[bundle.ID] = bundle
}

func (is *pgCatalog) deleteBundle(id string) {
	is.ruleBundleMutex.Lock()
	defer is.ruleBundleMutex.Unlock()
	delete(is.ruleBundleMap, id)
}
