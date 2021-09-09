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
		ID: pgCatalogDBID,
		Name: util.PgCatalogName,
		Charset: mysql.DefaultCharset,
		Collate: mysql.DefaultCollationName,
		Tables: pgCatalogTables,
	}

	RegisterVirtualTable(pgCatalogDB, createPgCatalogTable)
}
