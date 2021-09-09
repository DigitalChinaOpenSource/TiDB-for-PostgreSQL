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
	"github.com/DigitalChinaOpenSource/DCParser/model"
	"github.com/DigitalChinaOpenSource/DCParser/mysql"
	"github.com/pingcap/tidb/meta/autoid"
	"github.com/pingcap/tidb/util"
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
