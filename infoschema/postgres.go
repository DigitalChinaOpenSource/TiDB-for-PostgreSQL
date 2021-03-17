package infoschema

import (
	"fmt"
	"github.com/pingcap/parser/model"
	"github.com/pingcap/parser/mysql"
)

var (
	//PgCatalogName is the "pg_catalog" database name.
	PgCatalogName = model.NewCIStr("postgres")
)

func init(){
	dbID:=PgCatalogDBID
	pgCatalogTables := make([]*model.TableInfo, 0, len(pgTableNameToColumns))
	for name, cols := range pgTableNameToColumns {
		tableInfo := buildTableMeta(name, cols)
		pgCatalogTables = append(pgCatalogTables, tableInfo)
		var ok bool
		tableInfo.ID, ok = pgTableIDMap[tableInfo.Name.O]
		if !ok {
			panic(fmt.Sprintf("get information_schema table id failed, unknown system table `%v`", tableInfo.Name.O))
		}
		for i, c := range tableInfo.Columns {
			c.ID = int64(i) + 1
		}
	}
	pgCatalogDB := &model.DBInfo{
		ID:      dbID,
		Name:    PgCatalogName,
		Charset: mysql.DefaultCharset,
		Collate: mysql.DefaultCollationName,
		Tables:  pgCatalogTables,
	}
	RegisterVirtualTable(pgCatalogDB, createPgCatalogTable)
}
