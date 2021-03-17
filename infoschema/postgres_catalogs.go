package infoschema

import (
	"github.com/pingcap/parser/model"
	"github.com/pingcap/parser/mysql"
	"github.com/pingcap/tidb/meta/autoid"
	"github.com/pingcap/tidb/table"
)

const (
	// PgCatalogDBID is the pg_catalog schema id, it's exported for test.
	PgCatalogDBID int64 = autoid.SystemSchemaIDFlag | 30000
)

//postgreSQL中,Catalogs分为两部分,一部分是pg_catalog,一部分是information_schema
//其中,pg_catalog分为Tables和Views
//在这个const里面,Table开头的,为pg_catalog中的table;View开头的,就是pg_catalog中的views
//一共是62张Table,64张view
//todo 系统视图的建立
const (
	//pg_aggregate存储关于聚集函数的信息
	TablePgAggregate ="pg_aggregate"

	TablePgAm = "pg_am"

	TablePgAmop="pg_amop"

	TablePgAmproc="pg_amproc"

	TablePgAttrdef="pg_attrdef"

	TablePgAttribute="pg_attribute"

	TablePgAuthMembers="pg_auth_members"

	TablePgAuthId = "pg_authid"

	TablePgCast="pg_cast"

	TablePgClass="pg_calss"

	TablePgCollation="pg_collation"

	TablePgConstraint="pg_constraint"

	TablePgConversion="pg_conversion"

	TablePgDatabase = "pg_database"

	TablePgDbRoleSetting="pg_db_role_setting"

	TablePgDefaultAcl="pg_default_acl"

	TablePgDepend="pg_depend"

	TablePgDescription="pg_description"

	TablePgEnum = "pg_enum"

	TablePgEventTrigger ="pg_event_trigger"

	TablePgExtension ="pg_extension"

	TablePgForeignDataWrapper ="pg_foreign_data_wrapper"

	TablePgForeignServer="pg_foreign_server"

	TablePgForeignTable="pg_foreign_table"

	TablePgIndex="pg_index"

	TablePgInherits="pg_inherits"

	TablePgInitPrivs = "pg_init_privs"

	TablePgLanguage="pg_language"

	TablePgLargeObject="pg_largeobject"

	TablePgLargeObjectMetadata="pg_largeobject_metadata"

	TablePgNamespace = "pg_namespace"

	TablePgOpclass = "pg_opclass"

	TablePgOperator="pg_operator"

	TablePgOpFamily="pg_opfamily"

	TablePgPartitionedTable="pg_partitioned_table"

	TablePgPolicy="pg_policy"

	TablePgProc="pg_proc"

	TablePgPublication="pg_publication"

	TablePgPublicationRel="pg_publication_rel"

	TablePgRange="pg_range"

	TablePgReplicationOrigin="pg_replication_origin"

	TablePgRewrite="pg_rewrite"

	TablePgSeclabel="pg_seclabel"

	TablePgSequence="pg_sequence"

	TablePgShdepend="pg_shdepend"

	TablePgShdescription="pg_shdescription"

	TablePgShseclabel="pg_shseclabel"

	TablePgStatistic="pg_statistic"

	TablePgStatisticExt="pg_statistic_ext"

	TablePgStatisticExtData="pg_statistic_ext_data"

	TablePgSubscription="pg_subscription"

	TablePgSubscriptionRel="pg_subscription_rel"

	TablePgTablespace="pg_tablespace"

	TablePgTransform="pg_transform"

	TablePgTrigger="pg_trigger"

	TablePgTsConfig="pg_ts_config"

	TablePgTsConfigMap="pg_ts_config_map"

	TablePgTsDict="pg_ts_dict"

	TablePgTsParser="pg_ts_parser"

	TablePgTemplate="pg_template"

	TablePgType="pg_type"

	TablePgUserMapping = "pg_user_mapping"

)

//表以及视图的 OID 映射
//这里将这些都写死
var pgTableIDMap = map[string]int64{
	TablePgAggregate: 				2600,
	TablePgAm: 						2601,
	TablePgAmop: 					2602,
	TablePgAmproc: 					2603,
	TablePgAttrdef: 				2604,
	TablePgAttribute: 				1249,
	TablePgAuthMembers: 			1261,
	TablePgAuthId:   				1260,
	TablePgCast: 					2605,
	TablePgClass: 					1259,
	TablePgCollation: 				3456,
	TablePgConstraint: 				2606,
	TablePgConversion: 				2607,
	TablePgDatabase: 				1262,
	TablePgDbRoleSetting: 			2964,
	TablePgDefaultAcl: 				826,
	TablePgDepend: 					2608,
	TablePgDescription: 			2609,
	TablePgEnum:					2501,
	TablePgEventTrigger: 			3466,
	TablePgExtension: 				3079,
	TablePgForeignDataWrapper: 		2328,
	TablePgForeignServer: 			1417,
	TablePgForeignTable: 			3118,
	TablePgIndex: 					2610,
	TablePgInitPrivs: 				3394,
	TablePgInherits: 				2611,
	TablePgLanguage: 				2612,
	TablePgLargeObject: 			2613,
	TablePgLargeObjectMetadata: 	2995,
	TablePgNamespace: 				2615,
	TablePgOpclass: 				2616,
	TablePgOperator: 				2617,
	TablePgOpFamily: 				2753,
	TablePgPolicy: 					3256,
	TablePgProc: 					1255,
	TablePgPublication: 			6104,
	TablePgPublicationRel: 			6106,
	TablePgRange: 					3541,
	TablePgReplicationOrigin: 		6000,
	TablePgRewrite: 				2618,
	TablePgSeclabel:				3596,
	TablePgSequence: 				2224,
	TablePgShdepend: 				1214,
	TablePgShdescription: 			2396,
	TablePgShseclabel: 				3592,
	TablePgStatistic:				2619,
	TablePgStatisticExt:			3381,
	TablePgStatisticExtData:		3429,
	TablePgSubscription: 			6100,
	TablePgSubscriptionRel: 		6102,
	TablePgTablespace: 				1213,
	TablePgTransform: 				3576,
	TablePgTrigger: 				2620,
	TablePgTsConfig: 				3602,
	TablePgTsConfigMap: 			3603,
	TablePgTsDict: 					3600,
	TablePgTsParser: 				3601,
	TablePgTemplate: 				3764,
	TablePgType:					1247,
	TablePgUserMapping: 			1418,
}

//pgAuthIDCols 是TablePgAuthId的表结构信息
var pgAuthIDCols = []columnInfo{
	{name:"oid",tp: mysql.TypeInt24,size: 64},
	{name:"rolname",tp: mysql.TypeVarchar,size: 64},
	{name:"rolsuper",tp: mysql.TypeTiny,size: 1},
	{name:"rolinherit",tp: mysql.TypeTiny,size: 1},
	{name:"rolcreaterole",tp: mysql.TypeTiny,size: 1},
	{name:"rolcreatedb",tp: mysql.TypeTiny,size: 1},
	{name:"rolcanlogin",tp: mysql.TypeTiny,size: 1},
	{name:"rolreplication",tp: mysql.TypeTiny,size: 1},
	{name:"rolbypassrls",tp: mysql.TypeTiny,size: 1},
	{name:"rolconnlimit",tp: mysql.TypeInt24,size: 64},
	{name:"rollpassword",tp: mysql.TypeVarchar,size: 64},
	{name:"rolvaliduntil",tp: mysql.TypeDatetime},
}

var pgDatabaseCols = []columnInfo{
	{name:"oid",tp: mysql.TypeInt24,size: 64},
	{name:"datname",tp: mysql.TypeVarchar,size: 64},
	{name:"datdba",tp: mysql.TypeInt24,size: 64},
	{name:"encoding",tp: mysql.TypeInt24,size: 64},
	{name:"datcollate",tp: mysql.TypeVarchar,size: 64},
	{name:"datctype",tp: mysql.TypeVarchar,size: 64},
	{name:"datistemplate",tp: mysql.TypeTiny,size: 1},
	{name:"datallowconn",tp: mysql.TypeTiny,size: 1},
	{name:"datconnlimit",tp: mysql.TypeInt24,size: 64},
	{name:"datlastsysoid",tp: mysql.TypeInt24,size: 64},
	{name:"datfrozenxid",tp: mysql.TypeInt24,size: 64},
	{name:"datminmxid",tp: mysql.TypeInt24,size: 64},
	{name:"dattablespace",tp: mysql.TypeInt24,size: 64},
	{name:"datacl",tp: mysql.TypeVarchar,size: 64},
}

var pgSettingsCols = []columnInfo{
	{name:"name",tp: mysql.TypeVarchar,size: 64},
	{name:"setting",tp: mysql.TypeVarchar,size: 64},
	{name:"unit",tp: mysql.TypeVarchar,size: 64},
	{name:"category",tp: mysql.TypeVarchar,size: 64},
	{name:"short_desc",tp: mysql.TypeVarchar,size: 64},
	{name:"extra_desc",tp: mysql.TypeVarchar,size: 64},
	{name:"context",tp: mysql.TypeVarchar,size: 64},
	{name:"vartype",tp: mysql.TypeVarchar,size: 64},
	{name:"source",tp: mysql.TypeVarchar,size: 64},
	{name:"min_val",tp: mysql.TypeVarchar,size: 64},
	{name:"max_val",tp: mysql.TypeVarchar,size: 64},
	{name:"enumvals",tp: mysql.TypeEnum,size: 64},
	{name:"bool_val",tp: mysql.TypeVarchar,size: 64},
	{name:"reset_val",tp: mysql.TypeVarchar,size: 64},
	{name:"sourcefile",tp: mysql.TypeVarchar,size: 64},
	{name:"sourceline",tp: mysql.TypeInt24,size: 64},
	{name:"pending_restart",tp: mysql.TypeTiny,size: 64},
}


type pgCatalogTable struct {
	infoschemaTable
}

func createPgCatalogTable(_ autoid.Allocators, meta *model.TableInfo) (table.Table, error){
	columns := make([]*table.Column, len(meta.Columns))
	for i, col := range meta.Columns {
		columns[i] = table.ToColumn(col)
	}
	return &pgCatalogTable{
		infoschemaTable:infoschemaTable{
			meta: meta,
			cols: columns,
			tp:   table.VirtualTable,
		},
	}, nil
}

var pgTableNameToColumns = map[string][]columnInfo{
	TablePgAuthId:   pgAuthIDCols,
	TablePgDatabase: pgDatabaseCols,
}



