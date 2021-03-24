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

	//关系访问方法
	TablePgAm = "pg_am"

	//访问方法操作符
	TablePgAmop="pg_amop"

	//访问方法支持函数
	TablePgAmproc="pg_amproc"

	//列默认值
	TablePgAttrdef="pg_attrdef"

	//表列("属性")
	TablePgAttribute="pg_attribute"

	//认证标识符(角色)
	TablePgAuthMembers="pg_auth_members"

	//认证标识符成员关系
	TablePgAuthId = "pg_authid"

	//转换(数据类型转换)
	TablePgCast="pg_cast"

	//表、索引、序列、视图（“关系”）
	TablePgClass="pg_calss"

	//排序规则（locale信息）
	TablePgCollation="pg_collation"

	//检查约束、唯一约束、主键约束、外键约束
	TablePgConstraint="pg_constraint"

	//编码转换信息
	TablePgConversion="pg_conversion"

	//本数据库集簇中的数据库
	TablePgDatabase = "pg_database"

	//每角色和每数据库的设置
	TablePgDbRoleSetting="pg_db_role_setting"

	//对象类型的默认权限
	TablePgDefaultAcl="pg_default_acl"

	//数据库对象间的依赖
	TablePgDepend="pg_depend"

	//数据库对象上的描述或注释
	TablePgDescription="pg_description"

	//枚举标签和值定义
	TablePgEnum = "pg_enum"

	//事件触发器
	TablePgEventTrigger ="pg_event_trigger"

	//已安装拓展
	TablePgExtension ="pg_extension"

	//外部数据包装器的定义
	TablePgForeignDataWrapper ="pg_foreign_data_wrapper"

	//外部服务器的定义
	TablePgForeignServer="pg_foreign_server"

	//外部表信息
	TablePgForeignTable="pg_foreign_table"

	//索引信息
	TablePgIndex="pg_index"

	//表继承层次
	TablePgInherits="pg_inherits"

	//对象初始特权
	TablePgInitPrivs = "pg_init_privs"

	//编写函数的语言
	TablePgLanguage="pg_language"

	//大对象的数据页
	TablePgLargeObject="pg_largeobject"

	//大对象的元数据
	TablePgLargeObjectMetadata="pg_largeobject_metadata"

	//模式
	TablePgNamespace = "pg_namespace"

	//访问方法操作符类
	TablePgOpclass = "pg_opclass"

	//操作符
	TablePgOperator="pg_operator"

	//访问方法操作符族
	TablePgOpFamily="pg_opfamily"

	//表的分区键信息
	TablePgPartitionedTable="pg_partitioned_table"

	//行安全策略
	TablePgPolicy="pg_policy"

	//函数和过程
	TablePgProc="pg_proc"

	//用于逻辑复制的发布
	TablePgPublication="pg_publication"

	//发布映射关系
	TablePgPublicationRel="pg_publication_rel"

	//范围类型的信息
	TablePgRange="pg_range"

	//已注册的复制源
	TablePgReplicationOrigin="pg_replication_origin"

	//查询重写规则
	TablePgRewrite="pg_rewrite"

	//数据库对象上的安全标签
	TablePgSeclabel="pg_seclabel"

	//有关序列的信息
	TablePgSequence="pg_sequence"

	//共享对象上的依赖
	TablePgShdepend="pg_shdepend"

	//共享对象上的注释
	TablePgShdescription="pg_shdescription"

	//共享数据库对象上的安全标签
	TablePgShseclabel="pg_shseclabel"

	//规划器统计
	TablePgStatistic="pg_statistic"

	//扩展的规划器统计信息（定义）
	TablePgStatisticExt="pg_statistic_ext"

	//扩展的规划器统计信息（已构建的统计信息）
	TablePgStatisticExtData="pg_statistic_ext_data"

	//逻辑复制订阅
	TablePgSubscription="pg_subscription"

	//订阅的关系状态
	TablePgSubscriptionRel="pg_subscription_rel"

	//本数据库集簇内的表空间
	TablePgTablespace="pg_tablespace"

	//转换（将数据类型转换为过程语言需要的形式）
	TablePgTransform="pg_transform"

	//触发器
	TablePgTrigger="pg_trigger"

	//文本搜索配置
	TablePgTsConfig="pg_ts_config"

	//文本搜索配置的记号映射
	TablePgTsConfigMap="pg_ts_config_map"

	//文本搜索字典
	TablePgTsDict="pg_ts_dict"

	//文本搜索分析器
	TablePgTsParser="pg_ts_parser"

	//文本搜索模板
	TablePgTemplate="pg_template"

	//数据类型
	TablePgType="pg_type"

	//将用户映射到外部服务器
	TablePgUserMapping = "pg_user_mapping"

)

//表以及视图的 OID 映射,参照pg的做法,使用oid
//这里将这些都写死
//var pgTableOIDMap = map[string]int64{
//	TablePgAggregate: 				2600,
//	TablePgAm: 						2601,
//	TablePgAmop: 					2602,
//	TablePgAmproc: 					2603,
//	TablePgAttrdef: 				2604,
//	TablePgAttribute: 				1249,
//	TablePgAuthMembers: 			1261,
//	TablePgAuthId:   				1260,
//	TablePgCast: 					2605,
//	TablePgClass: 					1259,
//	TablePgCollation: 				3456,
//	TablePgConstraint: 				2606,
//	TablePgConversion: 				2607,
//	TablePgDatabase: 				1262,
//	TablePgDbRoleSetting: 			2964,
//	TablePgDefaultAcl: 				826,
//	TablePgDepend: 					2608,
//	TablePgDescription: 			2609,
//	TablePgEnum:					2501,
//	TablePgEventTrigger: 			3466,
//	TablePgExtension: 				3079,
//	TablePgForeignDataWrapper: 		2328,
//	TablePgForeignServer: 			1417,
//	TablePgForeignTable: 			3118,
//	TablePgIndex: 					2610,
//	TablePgInitPrivs: 				3394,
//	TablePgInherits: 				2611,
//	TablePgLanguage: 				2612,
//	TablePgLargeObject: 			2613,
//	TablePgLargeObjectMetadata: 	2995,
//	TablePgNamespace: 				2615,
//	TablePgOpclass: 				2616,
//	TablePgOperator: 				2617,
//	TablePgOpFamily: 				2753,
//	TablePgPolicy: 					3256,
//	TablePgProc: 					1255,
//	TablePgPublication: 			6104,
//	TablePgPublicationRel: 			6106,
//	TablePgRange: 					3541,
//	TablePgReplicationOrigin: 		6000,
//	TablePgRewrite: 				2618,
//	TablePgSeclabel:				3596,
//	TablePgSequence: 				2224,
//	TablePgShdepend: 				1214,
//	TablePgShdescription: 			2396,
//	TablePgShseclabel: 				3592,
//	TablePgStatistic:				2619,
//	TablePgStatisticExt:			3381,
//	TablePgStatisticExtData:		3429,
//	TablePgSubscription: 			6100,
//	TablePgSubscriptionRel: 		6102,
//	TablePgTablespace: 				1213,
//	TablePgTransform: 				3576,
//	TablePgTrigger: 				2620,
//	TablePgTsConfig: 				3602,
//	TablePgTsConfigMap: 			3603,
//	TablePgTsDict: 					3600,
//	TablePgTsParser: 				3601,
//	TablePgTemplate: 				3764,
//	TablePgType:					1247,
//	TablePgUserMapping: 			1418,
//}

// pgTableIDMap table的Name与表ID的映射
var pgTableIDMap = map[string]int64{
	TablePgAggregate: 			PgCatalogDBID + 1,
	TablePgAm: 					PgCatalogDBID + 2,
	TablePgAmop:				PgCatalogDBID + 3,
	TablePgAmproc: 				PgCatalogDBID + 4,
	TablePgAttrdef: 			PgCatalogDBID + 5,
	TablePgAttribute:			PgCatalogDBID + 6,
	TablePgAuthMembers:			PgCatalogDBID + 7,
	TablePgAuthId:				PgCatalogDBID + 8,
	TablePgCast:				PgCatalogDBID + 9,
	TablePgClass:				PgCatalogDBID + 10,
	TablePgCollation:			PgCatalogDBID + 11,
	TablePgConstraint:			PgCatalogDBID + 12,
	TablePgConversion: 			PgCatalogDBID + 13,
	TablePgDatabase:			PgCatalogDBID + 14,
	TablePgDbRoleSetting:		PgCatalogDBID + 15,
	TablePgDefaultAcl: 			PgCatalogDBID + 16,
	TablePgDepend:				PgCatalogDBID + 17,
	TablePgDescription:			PgCatalogDBID + 18,
	TablePgEnum:				PgCatalogDBID + 19,
	TablePgEventTrigger: 		PgCatalogDBID + 20,
	TablePgExtension:			PgCatalogDBID + 21,
	TablePgForeignDataWrapper:	PgCatalogDBID + 22,
	TablePgForeignServer:		PgCatalogDBID + 23,
	TablePgForeignTable:		PgCatalogDBID + 24,
	TablePgIndex:				PgCatalogDBID + 25,
	TablePgInherits:			PgCatalogDBID + 26,
	TablePgInitPrivs:			PgCatalogDBID + 27,
	TablePgLanguage:			PgCatalogDBID + 28,
	TablePgLargeObject:			PgCatalogDBID + 29,
	TablePgLargeObjectMetadata:	PgCatalogDBID + 30,
	TablePgNamespace:			PgCatalogDBID + 31,
	TablePgOpclass:				PgCatalogDBID + 32,
	TablePgOperator:			PgCatalogDBID + 33,
	TablePgOpFamily:            PgCatalogDBID + 34,
	TablePgPartitionedTable:	PgCatalogDBID + 35,
	TablePgPolicy:				PgCatalogDBID + 36,
	TablePgProc:				PgCatalogDBID + 37,
	TablePgPublication:			PgCatalogDBID + 38,
	TablePgPublicationRel:		PgCatalogDBID + 39,
	TablePgRange:				PgCatalogDBID + 40,
	TablePgReplicationOrigin:	PgCatalogDBID + 41,
	TablePgRewrite:				PgCatalogDBID + 42,
	TablePgSeclabel:			PgCatalogDBID + 43,
	TablePgSequence:			PgCatalogDBID + 44,
	TablePgShdepend:			PgCatalogDBID + 45,
	TablePgShdescription:		PgCatalogDBID + 46,
	TablePgShseclabel:			PgCatalogDBID + 47,
	TablePgStatistic:			PgCatalogDBID + 48,
	TablePgStatisticExt:		PgCatalogDBID + 49,
	TablePgStatisticExtData:	PgCatalogDBID + 50,
	TablePgSubscription:		PgCatalogDBID + 51,
	TablePgSubscriptionRel:		PgCatalogDBID + 52,
	TablePgTablespace:			PgCatalogDBID + 53,
	TablePgTransform:			PgCatalogDBID + 54,
	TablePgTrigger:				PgCatalogDBID + 55,
	TablePgTsConfig:			PgCatalogDBID + 56,
	TablePgTsConfigMap:			PgCatalogDBID + 57,
	TablePgTsDict:				PgCatalogDBID + 58,
	TablePgTsParser:			PgCatalogDBID + 59,
	TablePgTemplate:			PgCatalogDBID + 60,
	TablePgType:				PgCatalogDBID + 61,
	TablePgUserMapping:			PgCatalogDBID + 62,
}

// pgAggregateCols 是 TablePgAggregate的表结构信息
var pgAggregateCols = []columnInfo{
	{name:"aggfnoid", tp: mysql.TypeVarchar, size: 64, flag: mysql.NotNullFlag},
	{name:"aggkind", tp:mysql.TypeVarchar, size: 1, flag: mysql.NotNullFlag},
	{name:"aggnumdirectargs", tp: mysql.TypeShort, size: 2, flag: mysql.NotNullFlag},
	{name:"aggtransfn", tp: mysql.TypeVarchar, size: 64, flag: mysql.NotNullFlag},
	{name:"aggfinalfn", tp: mysql.TypeVarchar, size: 64, flag: mysql.NotNullFlag},
	{name:"aggcombinefn", tp: mysql.TypeVarchar, size: 64, flag: mysql.NotNullFlag},
	{name:"aggserialfn", tp: mysql.TypeVarchar, size: 64, flag: mysql.NotNullFlag},
	{name:"aggdeserialfn", tp: mysql.TypeVarchar, size: 64, flag: mysql.NotNullFlag},
	{name:"aggmtransfn", tp: mysql.TypeVarchar, size: 64, flag: mysql.NotNullFlag},
	{name:"aggminvtransfn", tp: mysql.TypeVarchar, size: 64, flag: mysql.NotNullFlag},
	{name:"aggmfinalfn", tp: mysql.TypeVarchar, size: 64, flag: mysql.NotNullFlag},
	{name:"aggfinalextra", tp: mysql.TypeTiny, size: 1, flag: mysql.NotNullFlag},
	{name:"aggmfinalextra", tp: mysql.TypeTiny, size: 1, flag: mysql.NotNullFlag},
	{name:"aggfinalmodify", tp: mysql.TypeVarchar, size: 1, flag: mysql.NotNullFlag},
	{name:"aggmfinalmodify", tp: mysql.TypeVarchar, size: 1, flag: mysql.NotNullFlag},
	{name:"aggsortop", tp: mysql.TypeLong, size: 32, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
	{name:"aggtranstype", tp: mysql.TypeLong, size: 32, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
	{name:"aggtransspace", tp: mysql.TypeLong, size: 64, flag: mysql.NotNullFlag},
	{name:"aggmtranstype", tp: mysql.TypeLong, size: 32, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
	{name:"aggmtransspace", tp: mysql.TypeLong, size: 64, flag: mysql.NotNullFlag},
	{name:"agginitval", tp: mysql.TypeString, size: 64, flag: mysql.NotNullFlag},
	{name:"aggminitval", tp: mysql.TypeString, size: 64, flag: mysql.NotNullFlag},
}

// pgAmCols 是 TablePgAm 的表结构信息
var pgAmCols =[]columnInfo{
	{name:"oid", tp: mysql.TypeLong, size: 32, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
	{name:"amname", tp: mysql.TypeString, size: 64, flag: mysql.NotNullFlag},
	{name:"amhandler", tp: mysql.TypeVarchar, size: 64, flag: mysql.NotNullFlag},
	{name:"amtype", tp: mysql.TypeVarchar, size: 1, flag: mysql.NotNullFlag},
}

// pgAmopCols 是TablePgAmop 的表结构信息
var pgAmopCols =[]columnInfo{
	{name:"oid", tp: mysql.TypeLong, size: 32, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
	{name:"amopfamily", tp: mysql.TypeLong, size: 32, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
	{name:"amoplefttype", tp: mysql.TypeLong, size: 32, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
	{name:"amoprighttype", tp: mysql.TypeLong, size: 32, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
	{name:"amopstrategy", tp: mysql.TypeShort, size: 2, flag: mysql.NotNullFlag},
	{name:"amoppurpose", tp: mysql.TypeVarchar, size: 1, flag: mysql.NotNullFlag},
	{name:"amopopr", tp: mysql.TypeLong, size: 32, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
	{name:"amopmethod", tp: mysql.TypeLong, size: 32, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
	{name:"amopsortfamily", tp: mysql.TypeLong, size: 32, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
}

// pgAttrdef 是TableAttrdef的表结构信息
var pgAttrdef =[]columnInfo{
	{name:"oid", tp: mysql.TypeLong, size: 32, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
	{name:"adrelid", tp: mysql.TypeLong, size: 32, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
	{name:"adnum", tp: mysql.TypeShort, size: 2, flag: mysql.NotNullFlag},
	{name:"adbin", tp: mysql.TypeString, size: 32, flag: mysql.NotNullFlag},
}

// pgAttribute 是TableAttribute的表结构信息
var pgAttribute = []columnInfo{
	{name:"oid", tp: mysql.TypeLong, size: 32, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
	{name:"attname", tp: mysql.TypeString, size: 16, flag: mysql.NotNullFlag},
	{name:"atttypid", tp: mysql.TypeLong, size: 32, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
	{name:"attstattarget", tp: mysql.TypeLong, size: 32, flag: mysql.NotNullFlag},
	{name:"attlen", tp: mysql.TypeShort, size: 2, flag: mysql.NotNullFlag},
	{name:"attnum", tp: mysql.TypeShort, size: 2, flag: mysql.NotNullFlag},
	{name:"attndims", tp: mysql.TypeLong, size: 32, flag: mysql.NotNullFlag},
	{name:"attcacheoff", tp: mysql.TypeLong, size: 32, flag: mysql.NotNullFlag},
	{name:"atttypmod", tp: mysql.TypeLong, size: 32, flag: mysql.NotNullFlag},
	{name:"attbyval", tp: mysql.TypeTiny, size: 1, flag: mysql.NotNullFlag},
	{name:"attstorage", tp: mysql.TypeVarchar, size: 1, flag: mysql.NotNullFlag},
	{name:"attalign", tp: mysql.TypeVarchar, size: 1, flag: mysql.NotNullFlag},
	{name:"attnotnull", tp: mysql.TypeTiny, size: 1, flag: mysql.NotNullFlag},
	{name:"atthasdef", tp: mysql.TypeTiny, size: 1, flag: mysql.NotNullFlag},
	{name:"atthasmissing", tp: mysql.TypeTiny, size: 1, flag: mysql.NotNullFlag},
	{name:"attidentity", tp: mysql.TypeVarchar, size: 1, flag: mysql.NotNullFlag},
	{name:"attgenerated", tp: mysql.TypeVarchar, size: 1, flag: mysql.NotNullFlag},
	{name:"attisdropped", tp: mysql.TypeTiny, size: 1, flag: mysql.NotNullFlag},
	{name:"attislocal", tp: mysql.TypeTiny, size: 1, flag: mysql.NotNullFlag},
	{name:"attinhcount", tp: mysql.TypeLong, size: 32, flag: mysql.NotNullFlag},
	{name:"attcollation", tp: mysql.TypeLong, size: 32, flag: mysql.NotNullFlag & mysql.UnsignedFlag},
	{name:"attacl", tp: mysql.TypeString, size: 2},
	{name:"attoptions", tp: mysql.TypeString, size: 2},
	{name:"attfdwoptions", tp: mysql.TypeString, size: 2},
	{name:"attmissingval", tp: mysql.TypeBlob, size: 2},
}

// PgAuthMembersCols 是 TableAuthMembers的表结构信息
var  PgAuthMembersCols = []columnInfo{
	{name:"roleid", tp: mysql.TypeLong, size: 32, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
	{name:"member", tp: mysql.TypeLong, size: 32, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
	{name:"grantor", tp: mysql.TypeLong, size: 32, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
	{name:"admin_option", tp: mysql.TypeTiny, size: 1, flag: mysql.NotNullFlag},
}

// pgAuthIDCols 是TablePgAuthId的表结构信息
var pgAuthIDCols = []columnInfo{
	{name:"oid",tp: mysql.TypeLong, size: 32, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
	{name:"rolname",tp: mysql.TypeVarchar, size: 32, flag: mysql.NotNullFlag},
	{name:"rolsuper",tp: mysql.TypeTiny, size: 1, flag: mysql.NotNullFlag},
	{name:"rolinherit",tp: mysql.TypeTiny, size: 1, flag: mysql.NotNullFlag},
	{name:"rolcreaterole",tp: mysql.TypeTiny, size: 1, flag: mysql.NotNullFlag},
	{name:"rolcreatedb",tp: mysql.TypeTiny, size: 1, flag: mysql.NotNullFlag},
	{name:"rolcanlogin",tp: mysql.TypeTiny, size: 1, flag: mysql.NotNullFlag},
	{name:"rolreplication",tp: mysql.TypeTiny, size: 1, flag: mysql.NotNullFlag},
	{name:"rolbypassrls",tp: mysql.TypeTiny, size: 1, flag: mysql.NotNullFlag},
	{name:"rolconnlimit",tp: mysql.TypeLong, size: 32,flag: mysql.NotNullFlag},
	{name:"rollpassword",tp: mysql.TypeVarchar,size: 64},
	{name:"rolvaliduntil",tp: mysql.TypeTimestamp},
}

// pgCastCols 是 TablePgCast的表结构信息
var pgCastCols = []columnInfo{
	{name:"oid", tp: mysql.TypeLong, size: 32, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
	{name:"castsource", tp: mysql.TypeLong, size: 32, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
	{name:"casttarget", tp: mysql.TypeLong, size: 32, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
	{name:"castfunc", tp: mysql.TypeLong, size: 32, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
	{name:"castcontext", tp: mysql.TypeVarchar, size: 1, flag: mysql.NotNullFlag},
	{name:"castmethod", tp: mysql.TypeVarchar, size: 1, flag: mysql.NotNullFlag},
}

// pgClassCols 是 TablePgClass的表结构信息
var pgClassCols = []columnInfo{
	{name:"oid", tp: mysql.TypeLong, size: 32, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
	{name:"relname",tp: mysql.TypeVarchar, size: 32, flag: mysql.NotNullFlag},
	{name:"relnamespace", tp: mysql.TypeLong, size: 32, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
	{name:"reltype", tp: mysql.TypeLong, size: 32, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
	{name:"reloftype", tp: mysql.TypeLong, size: 32, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
	{name:"relowner", tp: mysql.TypeLong, size: 32, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
	{name:"relam", tp: mysql.TypeLong, size: 32, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
	{name:"relfilenode", tp: mysql.TypeLong, size: 32, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
	{name:"reltablespace", tp: mysql.TypeLong, size: 32, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
	{name:"relpages", tp: mysql.TypeLong, size: 32, flag: mysql.NotNullFlag},
	{name:"reltuples", tp: mysql.TypeFloat, size: 4, flag: mysql.NotNullFlag},
	{name:"relallvisible", tp: mysql.TypeLong, size: 32, flag: mysql.NotNullFlag},
	{name:"reltoastrelid",tp: mysql.TypeLong, size: 32, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
	{name:"relhasindex",tp: mysql.TypeTiny, size: 1, flag: mysql.NotNullFlag},
	{name:"relisshared",tp: mysql.TypeTiny, size: 1, flag: mysql.NotNullFlag},
	{name:"relpersistence",tp: mysql.TypeVarchar, size: 1, flag: mysql.NotNullFlag},
	{name:"relkind",tp: mysql.TypeVarchar, size: 1, flag: mysql.NotNullFlag},
	{name:"relnatts",tp: mysql.TypeShort, size: 2, flag: mysql.NotNullFlag},
	{name:"relchecks",tp: mysql.TypeShort, size: 2, flag: mysql.NotNullFlag},
	{name:"relhasrules",tp: mysql.TypeTiny, size: 1, flag: mysql.NotNullFlag},
	{name:"relhastriggers",tp: mysql.TypeTiny, size: 1, flag: mysql.NotNullFlag},
	{name:"relhassubclass",tp: mysql.TypeTiny, size: 1, flag: mysql.NotNullFlag},
	{name:"relrowsecurity",tp: mysql.TypeTiny, size: 1, flag: mysql.NotNullFlag},
	{name:"relforcerowsecurity",tp: mysql.TypeTiny, size: 1, flag: mysql.NotNullFlag},
	{name:"relispopulated",tp: mysql.TypeTiny, size: 1, flag: mysql.NotNullFlag},
	{name:"relreplident",tp: mysql.TypeVarchar, size: 1, flag: mysql.NotNullFlag},
	{name:"relispartition",tp: mysql.TypeTiny, size: 1, flag: mysql.NotNullFlag},
	{name:"relrewrite",tp: mysql.TypeLong, size: 32, flag: mysql.NotNullFlag & mysql.UnsignedFlag},
	{name:"relfrozenxid",tp: mysql.TypeLong, size: 32, flag: mysql.NotNullFlag & mysql.UnsignedFlag},
	{name:"relminmxid",tp: mysql.TypeLong, size: 32, flag: mysql.NotNullFlag & mysql.UnsignedFlag},
	{name:"relacl",tp: mysql.TypeVarchar, size: 64},
	{name:"reloptions",tp: mysql.TypeVarchar, size: 64},
	{name:"relpartbound",tp: mysql.TypeVarchar, size: 64},
}

// pgCollationCols 是TablePgCollation的表结构信息
var pgCollationCols = []columnInfo{
	{name:"oid", tp: mysql.TypeLong, size: 32, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
	{name:"collname", tp: mysql.TypeVarchar, size: 64, flag: mysql.NotNullFlag},
	{name:"collnamespace", tp: mysql.TypeLong, size: 32, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
	{name:"collowner", tp: mysql.TypeLong, size: 32, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
	{name:"collprovider", tp: mysql.TypeVarchar, size: 1, flag: mysql.NotNullFlag},
	{name:"collisdeterministic", tp: mysql.TypeTiny, size: 1, flag: mysql.NotNullFlag},
	{name:"collencoding", tp: mysql.TypeLong, size: 32, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
	{name:"collcollate", tp: mysql.TypeVarchar, size: 64, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
	{name:"collctype", tp: mysql.TypeVarchar, size: 64, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
	{name:"collversion", tp: mysql.TypeVarchar, size: 64},
}

// pgConstraintCols 是 TablePgConstraint的表结构信息
var pgConstraintCols = []columnInfo{
	{name:"oid", tp: mysql.TypeLong, size: 32, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
	{name:"conname", tp: mysql.TypeVarchar, size: 32, flag: mysql.NotNullFlag},
	{name:"connamespace", tp: mysql.TypeLong, size: 32, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
	{name:"contype", tp: mysql.TypeVarchar, size: 1, flag: mysql.NotNullFlag},
	{name:"condeferrable", tp: mysql.TypeTiny, size: 1, flag: mysql.NotNullFlag},
	{name:"condeferred", tp: mysql.TypeTiny, size: 1, flag: mysql.NotNullFlag},
	{name:"convalidated", tp: mysql.TypeTiny, size: 1, flag: mysql.NotNullFlag},
	{name:"conrelid", tp: mysql.TypeLong, size: 32, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
	{name:"contypid", tp: mysql.TypeLong, size: 32, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
	{name:"conindid", tp: mysql.TypeLong, size: 32, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
	{name:"conparentid", tp: mysql.TypeLong, size: 32, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
	{name:"confrelid", tp: mysql.TypeLong, size: 32, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
	{name:"confupdtype", tp: mysql.TypeVarchar, size: 1, flag: mysql.NotNullFlag},
	{name:"confdeltype", tp: mysql.TypeVarchar, size: 1, flag: mysql.NotNullFlag},
	{name:"confmatchtype", tp: mysql.TypeVarchar, size: 1, flag: mysql.NotNullFlag},
	{name:"conislocal", tp: mysql.TypeTiny, size: 1, flag: mysql.NotNullFlag},
	{name:"coninhcount", tp: mysql.TypeLong, size: 32, flag: mysql.NotNullFlag},
	{name:"connoinherit", tp: mysql.TypeTiny, size: 1, flag: mysql.NotNullFlag},
	{name:"conkey", tp: mysql.TypeShort},
	{name:"confkey", tp: mysql.TypeShort},
	{name:"conpfeqop", tp: mysql.TypeLong},
	{name:"conppeqop", tp: mysql.TypeLong},
	{name:"conffeqop", tp: mysql.TypeLong},
	{name:"conexclop", tp: mysql.TypeLong},
	{name:"conbin", tp: mysql.TypeVarchar, size: 255},
}

// pgConversionCols 是TablePgConversionCols的表结构信息
var pgConversionCols = []columnInfo{
	{name:"oid", tp: mysql.TypeLong, size: 32, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
	{name:"conname", tp: mysql.TypeVarchar, size: 32, flag: mysql.NotNullFlag},
	{name:"connamespace", tp: mysql.TypeLong, size: 32, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
	{name:"conowner", tp: mysql.TypeLong, size: 32, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
	{name:"conforencoding", tp: mysql.TypeLong, size: 32, flag: mysql.NotNullFlag},
	{name:"contoencoding", tp: mysql.TypeLong, size: 32, flag: mysql.NotNullFlag},
	{name:"conproc", tp: mysql.TypeVarchar, size: 32, flag: mysql.NotNullFlag},
	{name:"condefault", tp: mysql.TypeTiny, size: 1, flag: mysql.NotNullFlag},
}

// pgDatabaseCols 是 TablePgDatabase的表结构信息
var pgDatabaseCols = []columnInfo{
	{name:"oid", tp: mysql.TypeLong, size: 32, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
	{name:"datname", tp: mysql.TypeVarchar, size: 32, flag: mysql.NotNullFlag},
	{name:"datdba", tp: mysql.TypeLong, size: 32, flag: mysql.NotNullFlag & mysql.UnsignedFlag},
	{name:"encoding",tp: mysql.TypeLong, size: 32, flag: mysql.NotNullFlag},
	{name:"datcollate", tp: mysql.TypeVarchar, size: 32, flag: mysql.NotNullFlag},
	{name:"datctype", tp: mysql.TypeVarchar, size: 32, flag: mysql.NotNullFlag},
	{name:"datistemplate", tp: mysql.TypeTiny,size: 1},
	{name:"datallowconn", tp: mysql.TypeTiny,size: 1},
	{name:"datconnlimit", tp: mysql.TypeVarchar, size: 32, flag: mysql.NotNullFlag},
	{name:"datlastsysoid", tp: mysql.TypeLong, size: 32, flag: mysql.NotNullFlag & mysql.UnsignedFlag},
	{name:"datfrozenxid", tp: mysql.TypeLong, size: 32, flag: mysql.NotNullFlag & mysql.UnsignedFlag},
	{name:"datminmxid", tp: mysql.TypeLong, size: 32, flag: mysql.NotNullFlag & mysql.UnsignedFlag},
	{name:"dattablespace", tp: mysql.TypeLong, size: 32, flag: mysql.NotNullFlag & mysql.UnsignedFlag},
	{name:"datacl", tp: mysql.TypeVarchar,size: 32},
}

// pgDbSettingCols 是 TablePgDbRoleSetting的表结构信息
var pgDbSettingCols = []columnInfo{
	{name:"setdatabase", tp: mysql.TypeLong, size: 32, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
	{name:"setrole", tp: mysql.TypeLong, size: 32, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
	{name:"setconfig", tp: mysql.TypeVarchar, size: 32},
}

// pgDefaultAclCols 是 TablePgDefaultAcl 的表结构信息
var pgDefaultAclCols = []columnInfo{
	{name:"oid", tp: mysql.TypeLong, size: 32, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
	{name:"defaclrole", tp: mysql.TypeLong, size: 32, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
	{name:"defaclnamespace", tp: mysql.TypeLong, size: 32, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
	{name:"defaclobjtype", tp: mysql.TypeVarchar, size: 32, flag: mysql.NotNullFlag},
	{name:"defaclacl", tp: mysql.TypeVarchar, size: 32, flag: mysql.NotNullFlag},
}

// pgDepnedCols 是 TablePgDepend 的表结构信息
var pgDepnedCols = []columnInfo{
	{name:"classid", tp: mysql.TypeLong, size: 32, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
	{name:"objid", tp: mysql.TypeLong, size: 32, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
	{name:"objsubid", tp: mysql.TypeLong, size: 32, flag: mysql.NotNullFlag},
	{name:"refclassid", tp: mysql.TypeLong, size: 32, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
	{name:"refobjid", tp: mysql.TypeLong, size: 32, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
	{name:"refobjsubid", tp: mysql.TypeLong, size: 32, flag: mysql.NotNullFlag},
	{name:"deptype", tp: mysql.TypeVarchar, size: 1, flag: mysql.NotNullFlag},
}

// pgDescriptionCols 是 TablePgDescription 的表结构信息
var pgDescriptionCols = []columnInfo{
	{name:"objoid", tp: mysql.TypeLong, size: 32, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
	{name:"classoid", tp: mysql.TypeLong, size: 32, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
	{name:"objsubid", tp: mysql.TypeLong, size: 32, flag: mysql.NotNullFlag},
	{name:"description", tp: mysql.TypeString, size: 64, flag: mysql.NotNullFlag},
}

// pgEnumCols 是 TablePgEnum 的表结构信息
var pgEnumCols = []columnInfo{
	{name:"oid", tp: mysql.TypeLong, size: 32, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
	{name:"enumtypid", tp: mysql.TypeLong, size: 32, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
	{name:"enumsortorder", tp: mysql.TypeFloat, size: 32, flag: mysql.NotNullFlag},
	{name:"enumlabel", tp: mysql.TypeVarchar, size: 32, flag: mysql.NotNullFlag},
}

// pgEventTriggerCols 是 TablePgEventTrigger 的表结构信息
var pgEventTriggerCols = []columnInfo{
	{name:"oid", tp: mysql.TypeLong, size: 32, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
	{name:"evtname", tp: mysql.TypeVarchar, size: 32, flag: mysql.NotNullFlag},
	{name:"evtevent", tp: mysql.TypeVarchar, size: 32, flag: mysql.NotNullFlag},
	{name:"evtowner", tp: mysql.TypeLong, size: 32, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
	{name:"evtfoid", tp: mysql.TypeLong, size: 32, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
	{name:"evtenabled", tp: mysql.TypeVarchar, size: 1, flag: mysql.NotNullFlag},
	{name:"evttags", tp: mysql.TypeVarchar},
}

// pgExtensionCols 是 TablePgExtension 的表结构信息
var pgExtensionCols = []columnInfo{
	{name:"oid", tp: mysql.TypeLong, size: 32, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
	{name:"extname", tp: mysql.TypeVarchar, size: 32, flag: mysql.NotNullFlag},
	{name:"extowner", tp: mysql.TypeLong, size: 32, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
	{name:"extnamespace", tp: mysql.TypeLong, size: 32, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
	{name:"extrelocatable", tp: mysql.TypeTiny, size: 1, flag: mysql.NotNullFlag},
	{name:"extversion", tp: mysql.TypeVarchar, size: 64, flag: mysql.NotNullFlag},
	{name:"extconfig", tp: mysql.TypeLong, size: 32, flag: mysql.UnsignedFlag},
	{name:"extcondition", tp: mysql.TypeVarchar, size: 32},
}

// pgForeignDataCols 是 TablePgForeignDataWrapper 的表结构信息
var pgForeignDataCols = []columnInfo{
	{name:"oid", tp: mysql.TypeLong, size: 32, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
	{name:"fdwname", tp: mysql.TypeVarchar, size: 32, flag: mysql.NotNullFlag},
	{name:"fdwowner", tp: mysql.TypeLong, size: 32, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
	{name:"fdwhandler", tp: mysql.TypeLong, size: 32, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
	{name:"fdwvalidator", tp: mysql.TypeLong, size: 32, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
	{name:"fdwacl", tp: mysql.TypeVarchar, size: 64},
	{name:"fdwoptions", tp: mysql.TypeVarchar, size: 64},
}

// pgForeignServerCols 是 TablePgForeignServer 的表结构信息
var pgForeignServerCols = []columnInfo{
	{name:"oid", tp: mysql.TypeLong, size: 32, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
	{name:"srvname", tp: mysql.TypeVarchar, size: 32, flag: mysql.NotNullFlag},
	{name:"srvowner", tp: mysql.TypeLong, size: 32, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
	{name:"srvfdw", tp: mysql.TypeLong, size: 32, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
	{name:"srvtype", tp: mysql.TypeVarchar, size: 64},
	{name:"srvversion", tp: mysql.TypeVarchar, size: 64},
	{name:"srvacl", tp: mysql.TypeVarchar, size: 64},
	{name:"srvoptions", tp: mysql.TypeVarchar, size: 64},
}

// pgForeignTableCols 是 TablePgForeignServer 的表结构信息
var pgForeignTableCols = []columnInfo{
	{name:"ftrelid", tp: mysql.TypeLong, size: 32, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
	{name:"ftserver", tp: mysql.TypeLong, size: 32, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
	{name:"ftoptions", tp: mysql.TypeVarchar, size: 64},
}

// pgIndexCols 是 TablePgForeignServer 的表结构信息
var pgIndexCols = []columnInfo{
	{name:"indexrelid", tp: mysql.TypeLong, size: 32, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
	{name:"indrelid", tp: mysql.TypeLong, size: 32, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
	{name:"indnatts",tp: mysql.TypeShort, size: 2, flag: mysql.NotNullFlag},
	{name:"indnkeyatts",tp: mysql.TypeShort, size: 2, flag: mysql.NotNullFlag},
	{name:"indisunique",tp: mysql.TypeTiny, size: 1, flag: mysql.NotNullFlag},
	{name:"indisprimary",tp: mysql.TypeTiny, size: 1, flag: mysql.NotNullFlag},
	{name:"indisexclusion",tp: mysql.TypeTiny, size: 1, flag: mysql.NotNullFlag},
	{name:"indimmediate",tp: mysql.TypeTiny, size: 1, flag: mysql.NotNullFlag},
	{name:"indisclustered",tp: mysql.TypeTiny, size: 1, flag: mysql.NotNullFlag},
	{name:"indisvalid",tp: mysql.TypeTiny, size: 1, flag: mysql.NotNullFlag},
	{name:"indcheckxmin",tp: mysql.TypeTiny, size: 1, flag: mysql.NotNullFlag},
	{name:"indisready",tp: mysql.TypeTiny, size: 1, flag: mysql.NotNullFlag},
	{name:"indislive",tp: mysql.TypeTiny, size: 1, flag: mysql.NotNullFlag},
	{name:"indisreplident",tp: mysql.TypeTiny, size: 1, flag: mysql.NotNullFlag},
	{name:"indkey",tp: mysql.TypeLong, size: 32},
	{name:"indcollation",tp: mysql.TypeLong, size: 32},
	{name:"indclass",tp: mysql.TypeLong, size: 32},
	{name:"indoption",tp: mysql.TypeLong, size: 32},
	{name:"indexprs",tp: mysql.TypeVarchar, size: 32},
	{name:"indpred",tp: mysql.TypeVarchar, size: 32},
}

// pgInheritsCols 是 TablePgInherits 的表结构信息
var pgInheritsCols = []columnInfo{
	{name:"inhrelid", tp: mysql.TypeLong, size: 32, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
	{name:"inhparent", tp: mysql.TypeLong, size: 32, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
	{name:"inhseqno", tp: mysql.TypeLong, size: 32, flag: mysql.NotNullFlag},
}

// pgInitPrivsCols 是 TablePgInitPrivs 的表结构信息
var pgInitPrivsCols = []columnInfo{
	{name:"objoid", tp: mysql.TypeLong, size: 32, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
	{name:"classoid", tp: mysql.TypeLong, size: 32, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
	{name:"objsubid", tp: mysql.TypeLong, size: 32, flag: mysql.NotNullFlag},
	{name:"privtype", tp: mysql.TypeVarchar, size: 1, flag: mysql.NotNullFlag},
	{name:"initprivs", tp: mysql.TypeVarchar, size: 32, flag: mysql.NotNullFlag},
}

// pgLanguageCols 是 TablePgLanguage 的表结构信息
var pgLanguageCols = []columnInfo{
	{name:"objoid", tp: mysql.TypeLong, size: 32, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
	{name:"lanname", tp: mysql.TypeVarchar, size: 32, flag: mysql.NotNullFlag},
	{name:"lanowner", tp: mysql.TypeLong, size: 32, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
	{name:"lanispl", tp: mysql.TypeTiny, size: 1, flag: mysql.NotNullFlag},
	{name:"lanpltrusted", tp: mysql.TypeTiny, size: 1, flag: mysql.NotNullFlag},
	{name:"lanplcallfoid", tp: mysql.TypeLong, size: 32, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
	{name:"laninline", tp: mysql.TypeLong, size: 32, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
	{name:"lanvalidator", tp: mysql.TypeLong, size: 32, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
	{name:"lanacl", tp: mysql.TypeVarchar, size: 64},
}

// pgLargeObjectCols 是 TablePgLargeObject 的表结构信息
var pgLargeObjectCols = []columnInfo{
	{name:"loid", tp: mysql.TypeLong, size: 32, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
	{name:"pageno", tp: mysql.TypeLong, size: 32, flag: mysql.NotNullFlag},
	{name:"data", tp: mysql.TypeBit, size: 64, flag: mysql.NotNullFlag},
}

// pgLargeObjMetadataCols 是 TablePgLargeObjectMetadata 的表结构信息
var pgLargeObjMetadataCols = []columnInfo{
	{name:"oid", tp: mysql.TypeLong, size: 32, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
	{name:"lomowner", tp: mysql.TypeLong, size: 32, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
	{name:"lomacl", tp: mysql.TypeVarchar, size: 64},
}

// pgNamespaceCols 是 TablePgNamespace 的表结构信息
var pgNamespaceCols = []columnInfo{
	{name:"oid", tp: mysql.TypeLong, size: 32, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
	{name:"nspname", tp: mysql.TypeVarchar, size: 32, flag: mysql.NotNullFlag},
	{name:"nspowner", tp: mysql.TypeLong, size: 32, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
	{name:"nspacl", tp: mysql.TypeVarchar, size: 64, flag: mysql.NotNullFlag},
}

// pgOpclassCols 是 TablePgOpclass 的表结构信息
var pgOpclassCols = []columnInfo{
	{name:"oid", tp: mysql.TypeLong, size: 32, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
	{name:"opcmethod", tp: mysql.TypeLong, size: 32, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
	{name:"opcname", tp: mysql.TypeVarchar, size: 32, flag: mysql.NotNullFlag},
	{name:"opcnamespace", tp: mysql.TypeLong, size: 32, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
	{name:"opcowner", tp: mysql.TypeLong, size: 32, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
	{name:"opcfamily", tp: mysql.TypeLong, size: 32, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
	{name:"opcintype", tp: mysql.TypeLong, size: 32, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
	{name:"opcdefault", tp: mysql.TypeTiny, size: 1, flag: mysql.NotNullFlag},
	{name:"opckeytype", tp: mysql.TypeLong, size: 32, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
}

// pgOperatorCols 是 TablePgOperator 的表结构信息
var pgOperatorCols = []columnInfo{
	{name:"oid", tp: mysql.TypeLong, size: 32, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
	{name:"oprname", tp: mysql.TypeVarchar, size: 32, flag: mysql.NotNullFlag},
	{name:"oprnamespace", tp: mysql.TypeLong, size: 32, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
	{name:"oprowner", tp: mysql.TypeLong, size: 32, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
	{name:"oprkind", tp: mysql.TypeVarchar, size: 1, flag: mysql.NotNullFlag},
	{name:"oprcanmerge", tp: mysql.TypeTiny, size: 1, flag: mysql.NotNullFlag},
	{name:"oprcanhash", tp: mysql.TypeTiny, size: 1, flag: mysql.NotNullFlag},
	{name:"oprleft", tp: mysql.TypeLong, size: 32, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
	{name:"oprright", tp: mysql.TypeLong, size: 32, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
	{name:"oprresult", tp: mysql.TypeLong, size: 32, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
	{name:"oprcom", tp: mysql.TypeLong, size: 32, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
	{name:"oprnegate", tp: mysql.TypeLong, size: 32, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
	{name:"oprcode", tp: mysql.TypeVarchar, size: 32, flag: mysql.NotNullFlag},
	{name:"oprrest", tp: mysql.TypeVarchar, size: 32, flag: mysql.NotNullFlag},
	{name:"oprjoin", tp: mysql.TypeVarchar, size: 32, flag: mysql.NotNullFlag},
}

// pgOpfamilyCols 是 TablePgOpFamily 的表结构信息
var pgOpfamilyCols = []columnInfo{
	{name:"oid", tp: mysql.TypeLong, size: 32, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
	{name:"opfmethod", tp: mysql.TypeLong, size: 32, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
	{name:"opfname", tp: mysql.TypeVarchar, size: 32, flag: mysql.NotNullFlag},
	{name:"opfnamespace", tp: mysql.TypeLong, size: 32, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
	{name:"opfowner", tp: mysql.TypeLong, size: 32, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
}

// pgPartitionedTableCols 是 TablePgPartitionedTable 的表结构信息
var pgPartitionedTableCols = []columnInfo{
	{name:"partrelid", tp: mysql.TypeLong, size: 32, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
	{name:"partstrat", tp: mysql.TypeVarchar, size: 1, flag: mysql.NotNullFlag},
	{name:"partnatts", tp: mysql.TypeShort, size: 2, flag: mysql.NotNullFlag},
	{name:"partdefid", tp: mysql.TypeLong, size: 32, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
	{name:"partattrs", tp: mysql.TypeVarchar, size: 32, flag: mysql.NotNullFlag},
	{name:"partclass", tp: mysql.TypeVarchar, size: 32, flag: mysql.NotNullFlag},
	{name:"partcollation", tp: mysql.TypeVarchar, size: 32, flag: mysql.NotNullFlag},
	{name:"partexprs", tp: mysql.TypeVarchar, size: 32},
}

//pgPolicyCols 是 TablePgPolicy 的表结构信息
var pgPolicyCols = []columnInfo{
	{name:"oid", tp: mysql.TypeLong, size: 32, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
	{name:"polname", tp: mysql.TypeVarchar, size: 32, flag: mysql.NotNullFlag},
	{name:"polrelid", tp: mysql.TypeLong, size: 32, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
	{name:"polcmd", tp: mysql.TypeVarchar, size: 1, flag: mysql.NotNullFlag},
	{name:"polpermissive", tp: mysql.TypeTiny, size: 1, flag: mysql.NotNullFlag},
	{name:"polroles", tp: mysql.TypeVarchar, size: 32, flag: mysql.NotNullFlag},
	{name:"polqual", tp: mysql.TypeVarchar, size: 64},
	{name:"polwithcheck", tp: mysql.TypeVarchar, size: 64},
}

// pgProcCols 是 TablePgProc 的表结构信息
var pgProcCols = []columnInfo{
	{name:"oid", tp: mysql.TypeLong, size: 32, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
	{name:"proname", tp: mysql.TypeVarchar, size: 32, flag: mysql.NotNullFlag},
	{name:"pronamespace", tp: mysql.TypeLong, size: 32, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
	{name:"proowner", tp: mysql.TypeLong, size: 32, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
	{name:"prolang", tp: mysql.TypeLong, size: 32, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
	{name:"procost", tp: mysql.TypeFloat, size: 4, flag: mysql.NotNullFlag},
	{name:"prorows", tp: mysql.TypeFloat, size: 4, flag: mysql.NotNullFlag},
	{name:"provariadic", tp: mysql.TypeLong, size: 32, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
	{name:"prosupport", tp: mysql.TypeVarchar, size: 32, flag: mysql.NotNullFlag},
	{name:"prokind", tp: mysql.TypeVarchar, size: 1, flag: mysql.NotNullFlag},
	{name:"prosecdef", tp: mysql.TypeTiny, size: 1, flag: mysql.NotNullFlag},
	{name:"proleakproof", tp: mysql.TypeTiny, size: 1, flag: mysql.NotNullFlag},
	{name:"proisstrict", tp: mysql.TypeTiny, size: 1, flag: mysql.NotNullFlag},
	{name:"proretset", tp: mysql.TypeTiny, size: 1, flag: mysql.NotNullFlag},
	{name:"provolatile", tp: mysql.TypeVarchar, size: 1, flag: mysql.NotNullFlag},
	{name:"proparallel", tp: mysql.TypeVarchar, size: 1, flag: mysql.NotNullFlag},
	{name:"pronargs", tp: mysql.TypeShort, size: 2, flag: mysql.NotNullFlag},
	{name:"pronargdefaults", tp: mysql.TypeShort, size: 2, flag: mysql.NotNullFlag},
	{name:"prorettype", tp: mysql.TypeLong, size: 32, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
	{name:"proargtypes", tp: mysql.TypeLong, size: 32, flag: mysql.NotNullFlag},
	{name:"proallargtypes", tp: mysql.TypeVarchar, size: 32},
	{name:"proargmodes", tp: mysql.TypeVarchar, size: 64},
	{name:"proargnames", tp: mysql.TypeVarchar, size: 64},
	{name:"proargdefaults", tp: mysql.TypeVarchar, size: 64},
	{name:"protrftypes", tp: mysql.TypeVarchar, size: 64},
	{name:"prosrc", tp: mysql.TypeString, size: 64, flag: mysql.NotNullFlag},
	{name:"probin", tp: mysql.TypeString, size: 64},
	{name:"proconfig", tp: mysql.TypeVarchar, size: 64},
	{name:"proacl", tp: mysql.TypeVarchar, size: 64},
}

// pgPublicationCols 是 TablePgPublication 的表结构信息
var pgPublicationCols = []columnInfo{
	{name:"oid", tp: mysql.TypeLong, size: 32, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
	{name:"pubname", tp: mysql.TypeVarchar, size: 32, flag: mysql.NotNullFlag},
	{name:"pubowner", tp: mysql.TypeLong, size: 32, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
	{name:"puballtables", tp: mysql.TypeTiny, size: 1, flag: mysql.NotNullFlag},
	{name:"pubinsert", tp: mysql.TypeTiny, size: 1, flag: mysql.NotNullFlag},
	{name:"pubupdate", tp: mysql.TypeTiny, size: 1, flag: mysql.NotNullFlag},
	{name:"pubdelete", tp: mysql.TypeTiny, size: 1, flag: mysql.NotNullFlag},
	{name:"pubtruncate", tp: mysql.TypeTiny, size: 1, flag: mysql.NotNullFlag},
	{name:"pubviaroot", tp: mysql.TypeTiny, size: 1, flag: mysql.NotNullFlag},
}

// pgPublicationRelCols 是 TablePgPublicationRel 的表结构信息
var pgPublicationRelCols = []columnInfo{
	{name:"oid", tp: mysql.TypeLong, size: 32, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
	{name:"prpubid", tp: mysql.TypeLong, size: 32, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
	{name:"prrelid", tp: mysql.TypeLong, size: 32, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
}

// pgRangeCols 是 TablePgRange 的表结构信息
var pgRangeCols = []columnInfo{
	{name:"rngtypid", tp: mysql.TypeLong, size: 32, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
	{name:"rngsubtype", tp: mysql.TypeLong, size: 32, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
	{name:"rngcollation", tp: mysql.TypeLong, size: 32, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
	{name:"rngsubopc", tp: mysql.TypeLong, size: 32, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
	{name:"rngcanonical", tp: mysql.TypeVarchar, size: 32, flag: mysql.NotNullFlag},
	{name:"rngsubdiff", tp: mysql.TypeVarchar, size: 32, flag: mysql.NotNullFlag},
}

// pgReplicationOriginCols 是 TablePgReplicationOrigin 的表结构信息
var pgReplicationOriginCols = []columnInfo{
	{name:"roident", tp: mysql.TypeLong, size: 32, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
	{name:"roname", tp: mysql.TypeVarchar, size: 64, flag: mysql.NotNullFlag},
}

// pgRewriteCols 是 TablePgRewrite 的表结构信息
var pgRewriteCols = []columnInfo{
	{name:"oid", tp: mysql.TypeLong, size: 32, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
	{name:"rulename", tp: mysql.TypeVarchar, size: 32, flag: mysql.NotNullFlag},
	{name:"ev_class", tp: mysql.TypeLong, size: 32, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
	{name:"ev_type", tp: mysql.TypeVarchar, size: 1, flag: mysql.NotNullFlag},
	{name:"ev_enabled", tp: mysql.TypeVarchar, size: 1, flag: mysql.NotNullFlag},
	{name:"is_instead", tp: mysql.TypeTiny, size: 1, flag: mysql.NotNullFlag},
	{name:"ev_qual", tp: mysql.TypeVarchar, size: 64, flag: mysql.NotNullFlag},
	{name:"ev_action", tp: mysql.TypeVarchar, size: 64, flag: mysql.NotNullFlag},
}

// pgSeclabelCols 是 TablePgSeclabel 的表结构信息
var pgSeclabelCols = []columnInfo{
	{name:"objoid", tp: mysql.TypeLong, size: 32, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
	{name:"classoid", tp: mysql.TypeLong, size: 32, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
	{name:"objsubid", tp: mysql.TypeLong, size: 32, flag: mysql.NotNullFlag},
	{name:"provider", tp: mysql.TypeVarchar, size: 32, flag: mysql.NotNullFlag},
	{name:"label", tp: mysql.TypeVarchar, size: 32, flag: mysql.NotNullFlag},
}

// pgSequenceCols 是 TablePgSequence 的表结构信息
var pgSequenceCols = []columnInfo{
	{name:"seqrelid", tp: mysql.TypeLong, size: 32, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
	{name:"seqtypid", tp: mysql.TypeLong, size: 32, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
	{name:"seqstart", tp: mysql.TypeLonglong, size: 64, flag: mysql.NotNullFlag},
	{name:"seqincrement", tp: mysql.TypeLonglong, size: 64, flag: mysql.NotNullFlag},
	{name:"seqmax", tp: mysql.TypeLonglong, size: 64, flag: mysql.NotNullFlag},
	{name:"seqmin", tp: mysql.TypeLonglong, size: 64, flag: mysql.NotNullFlag},
	{name:"seqcache", tp: mysql.TypeLonglong, size: 64, flag: mysql.NotNullFlag},
	{name:"seqcycle", tp: mysql.TypeTiny, size: 1, flag: mysql.NotNullFlag},
}

// pgShdepenCols 是 TablePgShdepend 的表结构信息
var pgShdepenCols =	[]columnInfo{
	{name:"dbid", tp: mysql.TypeLong, size: 32, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
	{name:"classid", tp: mysql.TypeLong, size: 32, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
	{name:"objid", tp: mysql.TypeLong, size: 32, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
	{name:"objsubid", tp: mysql.TypeLong, size: 32, flag: mysql.NotNullFlag},
	{name:"refclassid", tp: mysql.TypeLong, size: 32, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
	{name:"refobjid", tp: mysql.TypeLong, size: 32, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
	{name:"deptype", tp: mysql.TypeVarchar, size: 1, flag: mysql.NotNullFlag},
}

// pgShdescriptionCols 是 TablePgShdescription 的表结构信息
var pgShdescriptionCols = []columnInfo{
	{name:"objoid", tp: mysql.TypeLong, size: 32, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
	{name:"classoid", tp: mysql.TypeLong, size: 32, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
	{name:"description", tp: mysql.TypeVarchar, size: 64, flag: mysql.NotNullFlag},
}

// pgShseclabelCols 是 TablePgShseclabel 的表结构信息
var pgShseclabelCols = []columnInfo{
	{name:"objoid", tp: mysql.TypeLong, size: 32, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
	{name:"classoid", tp: mysql.TypeLong, size: 32, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
	{name:"provider", tp: mysql.TypeVarchar, size: 64, flag: mysql.NotNullFlag},
	{name:"label", tp: mysql.TypeVarchar, size: 64, flag: mysql.NotNullFlag},
}

// pgStatisticCols 是 TablePgStatistic 的表结构信息
var pgStatisticCols = []columnInfo{
	{name:"starelid", tp: mysql.TypeLong, size: 32, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
	{name:"staattnum", tp: mysql.TypeShort, size: 2, flag: mysql.NotNullFlag},
	{name:"stainherit", tp: mysql.TypeTiny, size: 1, flag: mysql.NotNullFlag},
	{name:"stanullfrac", tp: mysql.TypeFloat, size: 32, flag: mysql.NotNullFlag},
	{name:"stawidth", tp: mysql.TypeLong, size: 32, flag: mysql.NotNullFlag},
	{name:"stadistinct", tp: mysql.TypeFloat, size: 32, flag: mysql.NotNullFlag},
	{name:"stakind1", tp: mysql.TypeShort, size: 2, flag: mysql.NotNullFlag},
	{name:"stakind2", tp: mysql.TypeShort, size: 2, flag: mysql.NotNullFlag},
	{name:"stakind3", tp: mysql.TypeShort, size: 2, flag: mysql.NotNullFlag},
	{name:"stakind4", tp: mysql.TypeShort, size: 2, flag: mysql.NotNullFlag},
	{name:"stakind5", tp: mysql.TypeShort, size: 2, flag: mysql.NotNullFlag},
	{name:"staop1", tp: mysql.TypeLong, size: 32, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
	{name:"staop2", tp: mysql.TypeLong, size: 32, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
	{name:"staop3", tp: mysql.TypeLong, size: 32, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
	{name:"staop4", tp: mysql.TypeLong, size: 32, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
	{name:"staop5", tp: mysql.TypeLong, size: 32, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
	{name:"stacoll1", tp: mysql.TypeLong, size: 32, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
	{name:"stacoll2", tp: mysql.TypeLong, size: 32, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
	{name:"stacoll3", tp: mysql.TypeLong, size: 32, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
	{name:"stacoll4", tp: mysql.TypeLong, size: 32, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
	{name:"stacoll5", tp: mysql.TypeLong, size: 32, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
	{name:"stanumbers1", tp: mysql.TypeVarchar, size: 10},
	{name:"stanumbers2", tp: mysql.TypeVarchar, size: 10},
	{name:"stanumbers3", tp: mysql.TypeVarchar, size: 10},
	{name:"stanumbers4", tp: mysql.TypeVarchar, size: 10},
	{name:"stanumbers5", tp: mysql.TypeVarchar, size: 10},
	{name:"stavalues1", tp: mysql.TypeBit},
	{name:"stavalues2", tp: mysql.TypeBit},
	{name:"stavalues3", tp: mysql.TypeBit},
	{name:"stavalues4", tp: mysql.TypeBit},
	{name:"stavalues5", tp: mysql.TypeBit},
}

// pgStatisticExtCols 是 TablePgStatisticExt 的表结构信息
var pgStatisticExtCols = []columnInfo{
	{name:"oid", tp: mysql.TypeLong, size: 32, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
	{name:"stxrelid", tp: mysql.TypeLong, size: 32, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
	{name:"stxname", tp: mysql.TypeVarchar, size: 16, flag: mysql.NotNullFlag},
	{name:"stxnamespace", tp: mysql.TypeLong, size: 32, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
	{name:"stxowner", tp: mysql.TypeLong, size: 32, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
	{name:"stxstattarget", tp: mysql.TypeLong, size: 32, flag: mysql.NotNullFlag},
	{name:"stxkeys", tp: mysql.TypeVarchar, size: 32, flag: mysql.NotNullFlag},
	{name:"stxkind", tp: mysql.TypeVarchar, size: 32, flag: mysql.NotNullFlag},
}

// pgStatisticExtDataCols 是 TablePgStatisticExtData 的表结构信息
var pgStatisticExtDataCols = []columnInfo{
	{name:"stxoid", tp: mysql.TypeLong, size: 32, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
	{name:"stxdndistinct", tp: mysql.TypeVarchar, size: 32},
	{name:"stxddependencies", tp: mysql.TypeVarchar, size: 32},
	{name:"stxdmcv", tp: mysql.TypeVarchar, size: 32},
}

// pgSubscriptionCols 是 TablePgSubscription 的表结构信息
var pgSubscriptionCols = []columnInfo{
	{name:"oid", tp: mysql.TypeLong, size: 32, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
	{name:"subdbid", tp: mysql.TypeLong, size: 32, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
	{name:"subname", tp: mysql.TypeVarchar, size: 16, flag: mysql.NotNullFlag},
	{name:"subowner", tp: mysql.TypeLong, size: 32, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
	{name:"subenabled", tp: mysql.TypeTiny, size: 1, flag: mysql.NotNullFlag},
	{name:"subconninfo", tp: mysql.TypeVarchar, size: 64, flag: mysql.NotNullFlag},
	{name:"subslotname", tp: mysql.TypeVarchar, size: 16},
	{name:"subsynccommit", tp: mysql.TypeVarchar, size: 64, flag: mysql.NotNullFlag},
	{name:"subpublications", tp: mysql.TypeVarchar, size: 64, flag: mysql.NotNullFlag},
}

// pgSubscriptionRelCols 是 TablePgSubscriptionRel 的表结构信息
var pgSubscriptionRelCols = []columnInfo{
	{name:"srsubid", tp: mysql.TypeLong, size: 32, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
	{name:"srrelid", tp: mysql.TypeLong, size: 32, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
	{name:"srsubstate", tp: mysql.TypeVarchar, size: 1, flag: mysql.NotNullFlag},
	{name:"srsublsn", tp: mysql.TypeVarchar, size: 32},
}

// pgTableSpaceCols 是 TablePgSubscriptionRel 的表结构信息
var pgTableSpaceCols = []columnInfo{
	{name:"oid", tp: mysql.TypeLong, size: 32, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
	{name:"spcname", tp: mysql.TypeVarchar, size: 16, flag: mysql.NotNullFlag},
	{name:"spcowner", tp: mysql.TypeLong, size: 32, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
	{name:"spcacl", tp: mysql.TypeVarchar, size: 64},
	{name:"spcoptions", tp: mysql.TypeVarchar, size: 64},
}

// pgTransformCols 是 TablePgTransform 的表结构信息
var pgTransformCols = []columnInfo{
	{name:"oid", tp: mysql.TypeLong, size: 32, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
	{name:"trftype", tp: mysql.TypeLong, size: 32, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
	{name:"trflang", tp: mysql.TypeLong, size: 32, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
	{name:"trffromsql", tp: mysql.TypeVarchar, size: 32, flag: mysql.NotNullFlag},
	{name:"trftosql", tp: mysql.TypeVarchar, size: 32, flag: mysql.NotNullFlag},
}

// pgTriggerCols 是 TablePgTrigger 的表结构信息
var pgTriggerCols = []columnInfo{
	{name:"oid", tp: mysql.TypeLong, size: 32, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
	{name:"tgrelid", tp: mysql.TypeLong, size: 32, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
	{name:"tgparentid", tp: mysql.TypeLong, size: 32, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
	{name:"tgname", tp: mysql.TypeLong, size: 32, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
	{name:"tgfoid", tp: mysql.TypeLong, size: 32, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
	{name:"tgtype", tp: mysql.TypeLong, size: 32, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
	{name:"tgenabled", tp: mysql.TypeLong, size: 32, flag: mysql.UnsignedFlag & mysql.NotNullFlag},
}

var pgTableNameToColumns = map[string][]columnInfo{
	TablePgAggregate: 				pgAggregateCols,
	TablePgAm: 						pgAmCols,
	TablePgAmop: 					pgAmopCols,
	TablePgAmproc: 					pgAmopCols,
	TablePgAttrdef: 				pgAttrdef,
	TablePgAttribute: 				pgAttribute,
	TablePgAuthMembers: 			PgAuthMembersCols,
	TablePgAuthId:   				pgAuthIDCols,
	TablePgCast: 					pgCastCols,
	TablePgClass: 					pgClassCols,
	TablePgCollation: 				pgCollationCols,
	TablePgConstraint: 				pgConstraintCols,
	TablePgConversion: 				pgConversionCols,
	TablePgDatabase: 				pgDatabaseCols,
	TablePgDbRoleSetting: 			pgDbSettingCols,
	TablePgDefaultAcl: 				pgDefaultAclCols,
	TablePgDepend: 					pgDepnedCols,
	TablePgDescription: 			pgDescriptionCols,
	TablePgEnum: 					pgEnumCols,
	TablePgEventTrigger: 			pgEventTriggerCols,
	TablePgExtension: 				pgExtensionCols,
	TablePgForeignDataWrapper: 		pgForeignDataCols,
	TablePgForeignServer: 			pgForeignServerCols,
	TablePgForeignTable: 			pgForeignTableCols,
	TablePgIndex: 					pgIndexCols,
	TablePgInherits: 				pgInheritsCols,
	TablePgInitPrivs: 				pgInitPrivsCols,
	TablePgLanguage: 				pgLanguageCols,
	TablePgLargeObject: 			pgLargeObjectCols,
	TablePgLargeObjectMetadata: 	pgLargeObjMetadataCols,
	TablePgNamespace: 				pgNamespaceCols,
	TablePgOpclass: 				pgOpclassCols,
	TablePgOperator: 				pgOperatorCols,
	TablePgOpFamily: 				pgOpfamilyCols,
	TablePgPartitionedTable: 		pgPartitionedTableCols,
	TablePgPolicy: 					pgPolicyCols,
	TablePgProc: 					pgProcCols,
	TablePgPublication:				pgPublicationCols,
	TablePgPublicationRel: 			pgPublicationRelCols,
	TablePgRange: 					pgRangeCols,
	TablePgReplicationOrigin: 		pgReplicationOriginCols,
	TablePgRewrite: 				pgRewriteCols,
	TablePgSeclabel: 				pgSeclabelCols,
	TablePgSequence: 				pgSequenceCols,
	TablePgShdepend: 				pgShdepenCols,
	TablePgShdescription: 			pgShdescriptionCols,
	TablePgShseclabel: 				pgShseclabelCols,
	TablePgStatistic: 				pgStatisticCols,
	TablePgStatisticExt: 			pgStatisticExtCols,
	TablePgStatisticExtData: 		pgStatisticExtDataCols,
	TablePgSubscription: 			pgSubscriptionCols,
	TablePgSubscriptionRel: 		pgSubscriptionRelCols,
	TablePgTablespace: 				pgTableSpaceCols,
	TablePgTransform: 				pgTransformCols,
	TablePgTrigger: 				pgTriggerCols,
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





