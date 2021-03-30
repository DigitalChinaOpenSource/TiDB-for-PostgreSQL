package session

// PG SYSTEM TABLE SQL
const(
	CreateTablePgAggregate = `CREATE TABLE IF NOT EXISTS postgres.pg_aggregate
(
    aggfnoid 			VARCHAR(64) NOT NULL,
    aggkind 			CHAR(1) NOT NULL,
    aggnumdirectargs 	SMALLINT NOT NULL,
    aggtransfn 			VARCHAR(64) NOT NULL,
    aggfinalfn 			VARCHAR(64) NOT NULL,
    aggcombinefn 		VARCHAR(64) NOT NULL,
    aggserialfn 		VARCHAR(64) NOT NULL,
    aggdeserialfn 		VARCHAR(64) NOT NULL,
    aggmtransfn 		VARCHAR(64) NOT NULL,
    aggminvtransfn 		VARCHAR(64) NOT NULL,
    aggmfinalfn 		VARCHAR(64) NOT NULL,
    aggfinalextra 		TINYINT NOT NULL,
    aggmfinalextra 		TINYINT NOT NULL,
    aggfinalmodify 		CHAR(1) NOT NULL,
    aggmfinalmodify 	CHAR(1) NOT NULL,
    aggsortop 			INTEGER(32)  NOT NULL,
    aggtranstype 		INTEGER(32)  NOT NULL,
    aggtransspace 		INTEGER(32) NOT NULL,
    aggmtranstype 		INTEGER(32)  NOT NULL,
    aggmtransspace 		INTEGER(32) NOT NULL,
    agginitval 			longtext,
    aggminitval 		longtext
);`

	CreateTablePgAm =`CREATE TABLE IF NOT EXISTS postgres.pg_am
(
    oid INTEGER(32) NOT NULL,
    amname VARCHAR(32) NOT NULL,
    amhandler VARCHAR(64) NOT NULL,
    amtype CHAR(1) NOT NULL
);`

	CreateTablePgAmop = `CREATE TABLE IF NOT EXISTS postgres.pg_amop
(
	oid INTEGER(32) NOT NULL,
    amopfamily INTEGER(32) NOT NULL,
    amoplefttype INTEGER(32) NOT NULL,
    amoprighttype INTEGER(32) NOT NULL,
    amopstrategy SMALLINT NOT NULL,
    amoppurpose CHAR(1) NOT NULL,
    amopopr INTEGER(32) NOT NULL,
    amopmethod INTEGER(32) NOT NULL,
    amopsortfamily INTEGER(32) NOT NULL
);`

	CreateTablePgAmproc = `CREATE TABLE IF NOT EXISTS postgres.pg_amproc
(
    oid INTEGER(32) NOT NULL,
    amprocfamily INTEGER(32) NOT NULL,
    amproclefttype INTEGER(32) NOT NULL,
    amprocrighttype INTEGER(32) NOT NULL,
    amprocnum SMALLINT NOT NULL,
    amproc VARCHAR(64) NOT NULL
);`

	CreateTablePgAttrdef = `CREATE TABLE IF NOT EXISTS postgres.pg_attrdef
(
    oid INTEGER(32) NOT NULL,
    adrelid INTEGER(32) NOT NULL,
    adnum SMALLINT NOT NULL,
    adbin VARCHAR(64) NOT NULL
);`

	CreateTablePgAttribute = `CREATE TABLE IF NOT EXISTS postgres.pg_attribute
(
    attrelid INTEGER(32) NOT NULL,
    attname VARCHAR(32) NOT NULL,
    atttypid INTEGER(32) NOT NULL,
    attstattarget INTEGER(32) NOT NULL,
    attlen SMALLINT NOT NULL,
    attnum SMALLINT NOT NULL,
    attndims INTEGER(32) NOT NULL,
    attcacheoff INTEGER(32) NOT NULL,
    atttypmod INTEGER(32) NOT NULL,
    attbyval TINYINT(1) NOT NULL,
    attstorage CHAR(1) NOT NULL,
    attalign CHAR(1) NOT NULL,
    attnotnull TINYINT(1) NOT NULL,
    atthasdef TINYINT(1) NOT NULL,
    atthasmissing TINYINT(1) NOT NULL,
    attidentity CHAR(1) NOT NULL,
    attgenerated CHAR(1) NOT NULL,
    attisdropped TINYINT(1) NOT NULL,
    attislocal TINYINT(1) NOT NULL,
    attinhcount INTEGER(32) NOT NULL,
    attcollation INTEGER(32) NOT NULL,
    attacl VARCHAR(64),
    attoptions VARCHAR(64),
    attfdwoptions VARCHAR(64),
    attmissingval BLOB
);`

	CreateTablePgAuthMembers = `CREATE TABLE IF NOT EXISTS postgres.pg_auth_members
(
    roleid INTEGER(32) NOT NULL,
    member INTEGER(32) NOT NULL,
    grantor INTEGER(32) NOT NULL,
    admin_option TINYINT(1) NOT NULL
);`

	CreateTablePgAuthId = `CREATE TABLE IF NOT EXISTS postgres.pg_authid
(
    oid INTEGER(32) NOT NULL,
    rolname VARCHAR(32) NOT NULL,
    rolsuper TINYINT(1) NOT NULL,
    rolinherit TINYINT(1) NOT NULL,
    rolcreaterole TINYINT(1) NOT NULL,
    rolcreatedb TINYINT(1) NOT NULL,
    rolcanlogin TINYINT(1) NOT NULL,
    rolreplication TINYINT(1) NOT NULL,
    rolbypassrls TINYINT(1) NOT NULL,
    rolconnlimit INTEGER(32) NOT NULL,
    rolpassword TEXT,
    rolvaliduntil timestamp
);`

	CreateTablePgCast = `CREATE TABLE IF NOT EXISTS postgres.pg_cast
(
    oid INTEGER(32) NOT NULL,
    castsource INTEGER(32) NOT NULL,
    casttarget INTEGER(32) NOT NULL,
    castfunc INTEGER(32) NOT NULL,
    castcontext CHAR(1) NOT NULL,
    castmethod CHAR(1) NOT NULL
);`

	CreateTablePgClass = `CREATE TABLE IF NOT EXISTS postgres.pg_class
(
    oid INTEGER(32) NOT NULL,
    relname VARCHAR(32) NOT NULL,
    relnamespace INTEGER(32) NOT NULL,
    reltype INTEGER(32) NOT NULL,
    reloftype INTEGER(32) NOT NULL,
    relowner INTEGER(32) NOT NULL,
    relam INTEGER(32) NOT NULL,
    relfilenode INTEGER(32) NOT NULL,
    reltablespace INTEGER(32) NOT NULL,
    relpages INTEGER(32) NOT NULL,
    reltuples FLOAT NOT NULL,
    relallvisible INTEGER(32) NOT NULL,
    reltoastrelid INTEGER(32) NOT NULL,
    relhasindex TINYINT(1) NOT NULL,
    relisshared TINYINT(1) NOT NULL,
    relpersistence CHAR(1) NOT NULL,
    relkind CHAR(1) NOT NULL,
    relnatts SMALLINT NOT NULL,
    relchecks SMALLINT NOT NULL,
    relhasrules TINYINT(1) NOT NULL,
    relhastriggers TINYINT(1) NOT NULL,
    relhassubclass TINYINT(1) NOT NULL,
    relrowsecurity TINYINT(1) NOT NULL,
    relforcerowsecurity TINYINT(1) NOT NULL,
    relispopulated TINYINT(1) NOT NULL,
    relreplident CHAR(1) NOT NULL,
    relispartition TINYINT(1) NOT NULL,
    relrewrite INTEGER(32) NOT NULL,
    relfrozenxid INTEGER(32) NOT NULL,
    relminmxid INTEGER(32) NOT NULL,
    relacl VARCHAR(255),
    reloptions TEXT,
    relpartbound TEXT
);`

	CreateTablePgCollation = `CREATE TABLE IF NOT EXISTS postgres.pg_collation
(
    oid INTEGER(32) NOT NULL,
    collname VARCHAR(32) NOT NULL,
    collnamespace INTEGER(32) NOT NULL,
    collowner INTEGER(32) NOT NULL,
    collprovider CHAR(1) NOT NULL,
    collisdeterministic TINYINT(1) NOT NULL,
    collencoding INTEGER(32) NOT NULL,
    collcollate VARCHAR(32) NOT NULL,
    collctype VARCHAR(32) NOT NULL,
    collversion TEXT
);`

	CreateTablePgConstraint = `CREATE TABLE IF NOT EXISTS postgres.pg_constraint
(
    oid INTEGER(32) NOT NULL,
    conname VARCHAR(32) NOT NULL,
    connamespace INTEGER(32) NOT NULL,
    contype CHAR(1) NOT NULL,
    condeferrable TINYINT(1) NOT NULL,
    condeferred TINYINT(1) NOT NULL,
    convalidated TINYINT(1) NOT NULL,
    conrelid INTEGER(32) NOT NULL,
    contypid INTEGER(32) NOT NULL,
    conindid INTEGER(32) NOT NULL,
    conparentid INTEGER(32) NOT NULL,
    confrelid INTEGER(32) NOT NULL,
    confupdtype CHAR(1) NOT NULL,
    confdeltype CHAR(1) NOT NULL,
    confmatchtype CHAR(1) NOT NULL,
    conislocal TINYINT(1) NOT NULL,
    coninhcount INTEGER(32) NOT NULL,
    connoinherit TINYINT(1) NOT NULL,
    conkey SMALLINT,
    confkey SMALLINT,
    conpfeqop INTEGER(32),
    conppeqop INTEGER(32),
    conffeqop INTEGER(32),
    conexclop INTEGER(32),
    conbin TEXT
);`

	CreateTablePgConversion = `CREATE TABLE IF NOT EXISTS postgres.pg_conversion
(
    oid INTEGER(32) NOT NULL,
    conname VARCHAR(32) NOT NULL,
    connamespace INTEGER(32) NOT NULL,
    conowner INTEGER(32) NOT NULL,
    conforencoding INTEGER(32) NOT NULL,
    contoencoding INTEGER(32) NOT NULL,
    conproc TEXT NOT NULL,
    condefault TINYINT(1) NOT NULL
);`

	CreateTablePgDatabase = `CREATE TABLE IF NOT EXISTS postgres.pg_database
(
    oid INTEGER(32) NOT NULL,
    datname VARCHAR(32) NOT NULL,
    datdba INTEGER(32) NOT NULL,
    encoding INTEGER(32) NOT NULL,
    datcollate VARCHAR(32) NOT NULL,
    datctype VARCHAR(32) NOT NULL,
    datistemplate TINYINT(1) NOT NULL,
    datallowconn TINYINT(1) NOT NULL,
    datconnlimit INTEGER(32) NOT NULL,
    datlastsysoid INTEGER(32) NOT NULL,
    datfrozenxid INTEGER(32) NOT NULL,
    datminmxid INTEGER(32) NOT NULL,
    dattablespace INTEGER(32) NOT NULL,
    datacl VARCHAR(255)
);`

	 CreateTablePgDbRoleSetting = `CREATE TABLE IF NOT EXISTS postgres.pg_db_role_setting
(
    setdatabase INTEGER(32) NOT NULL,
    setrole INTEGER(32) NOT NULL,
    setconfig TEXT
);`

	 CreateTablePgDefaultAcl = `CREATE TABLE IF NOT EXISTS postgres.pg_default_acl
(
    oid INTEGER(32) NOT NULL,
    defaclrole INTEGER(32) NOT NULL,
    defaclnamespace INTEGER(32) NOT NULL,
    defaclobjtype CHAR(1) NOT NULL,
    defaclacl VARCHAR(255) NOT NULL
);`

	 CreateTablePgDepend = `CREATE TABLE IF NOT EXISTS postgres.pg_depend
(
    classid INTEGER(32) NOT NULL,
    objid INTEGER(32) NOT NULL,
    objsubid INTEGER(32) NOT NULL,
    refclassid INTEGER(32) NOT NULL,
    refobjid INTEGER(32) NOT NULL,
    refobjsubid INTEGER(32) NOT NULL,
    deptype CHAR(1) NOT NULL
);`

	  CreateTablePgDescription = `CREATE TABLE IF NOT EXISTS postgres.pg_description
(
    objoid INTEGER(32) NOT NULL,
    classoid INTEGER(32) NOT NULL,
    objsubid INTEGER(32) NOT NULL,
    description TEXT NOT NULL
);`

	  CreateTablePgEnum = `CREATE TABLE IF NOT EXISTS postgres.pg_enum
(
    oid INTEGER(32) NOT NULL,
    enumtypid INTEGER(32) NOT NULL,
    enumsortorder FLOAT NOT NULL,
    enumlabel VARCHAR(32) NOT NULL
);`

	  CreateTablePgEventTrigger = `CREATE TABLE IF NOT EXISTS postgres.pg_event_trigger
(
    oid INTEGER(32) NOT NULL,
    evtname VARCHAR(32)  NOT NULL,
    evtevent VARCHAR(32)  NOT NULL,
    evtowner INTEGER(32) NOT NULL,
    evtfoid INTEGER(32) NOT NULL,
    evtenabled CHAR(1) NOT NULL,
    evttags TEXT
);`

	  CreateTablePgExtension = `CREATE TABLE IF NOT EXISTS postgres.pg_extension
(
    oid INTEGER(32) NOT NULL,
    extname VARCHAR(32)  NOT NULL,
    extowner INTEGER(32) NOT NULL,
    extnamespace INTEGER(32) NOT NULL,
    extrelocatable TINYINT(1) NOT NULL,
    extversion TEXT NOT NULL,
    extconfig INTEGER(32),
    extcondition TEXT
);`

	  CreateTablePgForeignDataWrapper = `CREATE TABLE IF NOT EXISTS postgres.pg_foreign_data_wrapper
(
    oid INTEGER(32) NOT NULL,
    fdwname VARCHAR(32) NOT NULL,
    fdwowner INTEGER(32) NOT NULL,
    fdwhandler INTEGER(32) NOT NULL,
    fdwvalidator INTEGER(32) NOT NULL,
    fdwacl VARCHAR(255),
    fdwoptions TEXT
);`

	  CreateTablePgForeignServer = `CREATE TABLE IF NOT EXISTS postgres.pg_foreign_server
(
    oid INTEGER(32) NOT NULL,
    srvname VARCHAR(32) NOT NULL,
    srvowner INTEGER(32) NOT NULL,
    srvfdw INTEGER(32) NOT NULL,
    srvtype TEXT,
    srvversion TEXT,
    srvacl VARCHAR(255),
    srvoptions TEXT
);`

	  CreateTablePgForeignTable = `CREATE TABLE IF NOT EXISTS postgres.pg_foreign_table
(
    ftrelid INTEGER(32) NOT NULL,
    ftserver INTEGER(32) NOT NULL,
    ftoptions TEXT
);`

	  CreateTablePgIndex = `CREATE TABLE IF NOT EXISTS postgres.pg_index
(
    indexrelid INTEGER(32) NOT NULL,
    indrelid INTEGER(32) NOT NULL,
    indnatts SMALLINT NOT NULL,
    indnkeyatts SMALLINT NOT NULL,
    indisunique TINYINT(1) NOT NULL,
    indisprimary TINYINT(1) NOT NULL,
    indisexclusion TINYINT(1) NOT NULL,
    indimmediate TINYINT(1) NOT NULL,
    indisclustered TINYINT(1) NOT NULL,
    indisvalid TINYINT(1) NOT NULL,
    indcheckxmin TINYINT(1) NOT NULL,
    indisready TINYINT(1) NOT NULL,
    indislive TINYINT(1) NOT NULL,
    indisreplident TINYINT(1) NOT NULL,
    indkey VARCHAR(64) NOT NULL,
    indcollation VARCHAR(64) NOT NULL,
    indclass VARCHAR(64) NOT NULL,
    indoption VARCHAR(64) NOT NULL,
    indexprs TEXT,
    indpred TEXT
);`

	  CreateTablePgInherits = `CREATE TABLE IF NOT EXISTS postgres.pg_inherits
(
    inhrelid INTEGER(32) NOT NULL,
    inhparent INTEGER(32) NOT NULL,
    inhseqno INTEGER(32) NOT NULL
);`

	  CreateTablePgInitPrivs = `CREATE TABLE IF NOT EXISTS postgres.pg_init_privs
(
    objoid INTEGER(32) NOT NULL,
    classoid INTEGER(32) NOT NULL,
    objsubid INTEGER(32) NOT NULL,
    privtype CHAR(1) NOT NULL,
    initprivs VARCHAR(255) NOT NULL
);`

	  CreateTablePgLanguage = `CREATE TABLE IF NOT EXISTS postgres.pg_language
(
    oid INTEGER(32) NOT NULL,
    lanname VARCHAR(32),
    lanowner INTEGER(32) NOT NULL,
    lanispl TINYINT(1) NOT NULL,
    lanpltrusted TINYINT(1) NOT NULL,
    lanplcallfoid INTEGER(32) NOT NULL,
    laninline INTEGER(32) NOT NULL,
    lanvalidator INTEGER(32) NOT NULL,
    lanacl VARCHAR(255)
);`

	  CreateTablePgLargeObject = `CREATE TABLE IF NOT EXISTS postgres.pg_largeobject
(
    loid INTEGER(32) NOT NULL,
    pageno INTEGER(32) NOT NULL,
    data BINARY NOT NULL
);`

	  CreateTablePgLargeObjectMetadata = `CREATE TABLE IF NOT EXISTS postgres.pg_largeobject_metadata
(
    oid INTEGER(32) NOT NULL,
    lomowner INTEGER(32) NOT NULL,
    lomacl VARCHAR(255)
);`

	  CreateTablePgNamespace = `CREATE TABLE IF NOT EXISTS postgres.pg_namespace
(
    oid INTEGER(32) NOT NULL,
    nspname VARCHAR(32) NOT NULL,
    nspowner INTEGER(32) NOT NULL,
    nspacl VARCHAR(255)
);`

	  CreateTablePgOpclass = `CREATE TABLE IF NOT EXISTS postgres.pg_opclass
(
    oid INTEGER(32) NOT NULL,
    opcmethod INTEGER(32) NOT NULL,
    opcname VARCHAR(32) NOT NULL,
    opcnamespace INTEGER(32) NOT NULL,
    opcowner INTEGER(32) NOT NULL,
    opcfamily INTEGER(32) NOT NULL,
    opcintype INTEGER(32) NOT NULL,
    opcdefault TINYINT(1) NOT NULL,
    opckeytype INTEGER(32) NOT NULL
);`

	  CreateTablePgOperator = `CREATE TABLE IF NOT EXISTS postgres.pg_operator
(
    oid INTEGER(32) NOT NULL,
    oprname VARCHAR(32) NOT NULL,
    oprnamespace INTEGER(32) NOT NULL,
    oprowner INTEGER(32) NOT NULL,
    oprkind CHAR(1) NOT NULL,
    oprcanmerge TINYINT(1) NOT NULL,
    oprcanhash TINYINT(1) NOT NULL,
    oprleft INTEGER(32) NOT NULL,
    oprright INTEGER(32) NOT NULL,
    oprresult INTEGER(32) NOT NULL,
    oprcom INTEGER(32) NOT NULL,
    oprnegate INTEGER(32) NOT NULL,
    oprcode TEXT NOT NULL,
    oprrest TEXT NOT NULL,
    oprjoin TEXT NOT NULL
);`

	  CreateTablePgOpFamily = `CREATE TABLE IF NOT EXISTS postgres.pg_opfamily
(
    oid INTEGER(32) NOT NULL,
    opfmethod INTEGER(32) NOT NULL,
    opfname VARCHAR(32) NOT NULL,
    opfnamespace INTEGER(32) NOT NULL,
    opfowner INTEGER(32) NOT NULL
);`

	  CreateTablePgPartitionedTable = `CREATE TABLE IF NOT EXISTS postgres.pg_partitioned_table
(
    partrelid INTEGER(32) NOT NULL,
    partstrat CHAR(1) NOT NULL,
    partnatts SMALLINT NOT NULL,
    partdefid INTEGER(32) NOT NULL,
    partattrs VARCHAR(64) NOT NULL,
    partclass VARCHAR(64) NOT NULL,
    partcollation VARCHAR(64) NOT NULL,
    partexprs TEXT
);`

	  CreateTablePgPolicy = `CREATE TABLE IF NOT EXISTS postgres.pg_policy
(
    oid INTEGER(32) NOT NULL,
    polname VARCHAR(32) NOT NULL,
    polrelid INTEGER(32) NOT NULL,
    polcmd CHAR(1) NOT NULL,
    polpermissive TINYINT(1) NOT NULL,
    polroles INTEGER(32) NOT NULL,
    polqual TEXT,
    polwithcheck TEXT
);`

	CreateTablePgProc = `CREATE TABLE IF NOT EXISTS postgres.pg_proc
(
    oid INTEGER(32) NOT NULL,
    proname VARCHAR(32) NOT NULL,
    pronamespace INTEGER(32) NOT NULL,
    proowner INTEGER(32) NOT NULL,
    prolang INTEGER(32) NOT NULL,
    procost FLOAT NOT NULL,
    prorows FLOAT NOT NULL,
    provariadic INTEGER(32) NOT NULL,
    prosupport TEXT NOT NULL,
    prokind CHAR(1) NOT NULL,
    prosecdef TINYINT(1) NOT NULL,
    proleakproof TINYINT(1) NOT NULL,
    proisstrict TINYINT(1) NOT NULL,
    proretset TINYINT(1) NOT NULL,
    provolatile CHAR(1) NOT NULL,
    proparallel CHAR(1) NOT NULL,
    pronargs SMALLINT NOT NULL,
    pronargdefaults SMALLINT NOT NULL,
    prorettype INTEGER(32) NOT NULL,
    proargtypes VARCHAR(64) NOT NULL,
    proallargtypes INTEGER(32),
    proargmodes CHAR(1),
    proargnames TEXT,
    proargdefaults TEXT,
    protrftypes INTEGER(32),
    prosrc TEXT NOT NULL,
    probin TEXT,
    proconfig TEXT,
    proacl VARCHAR(255)
);`

	CreateTablePgPublication = `CREATE TABLE IF NOT EXISTS postgres.pg_publication
(
    oid INTEGER(32) NOT NULL,
    pubname VARCHAR(32) NOT NULL,
    pubowner INTEGER(32) NOT NULL,
    puballtables TINYINT(1) NOT NULL,
    pubinsert TINYINT(1) NOT NULL,
    pubupdate TINYINT(1) NOT NULL,
    pubdelete TINYINT(1) NOT NULL,
    pubtruncate TINYINT(1) NOT NULL,
    pubviaroot TINYINT(1) NOT NULL
);`

	CreateTablePgPublicationRel = `CREATE TABLE IF NOT EXISTS postgres.pg_publication_rel
(
    oid INTEGER(32) NOT NULL,
    prpubid INTEGER(32) NOT NULL,
    prrelid INTEGER(32) NOT NULL
);`

	CreateTablePgRange = `CREATE TABLE IF NOT EXISTS postgres.pg_range
(
    rngtypid INTEGER(32) NOT NULL,
    rngsubtype INTEGER(32) NOT NULL,
    rngcollation INTEGER(32) NOT NULL,
    rngsubopc INTEGER(32) NOT NULL,
    rngcanonical TEXT NOT NULL,
    rngsubdiff TEXT NOT NULL
);`

	CreateTablePgReplicationOrigin = `CREATE TABLE IF NOT EXISTS postgres.pg_replication_origin
(
    roident INTEGER(32) NOT NULL,
    roname TEXT NOT NULL
);`

	CreateTablePgRewrite = `CREATE TABLE IF NOT EXISTS postgres.pg_rewrite
(
    oid INTEGER(32) NOT NULL,
    rulename VARCHAR(32) NOT NULL,
    ev_class INTEGER(32) NOT NULL,
    ev_type CHAR(1) NOT NULL,
    ev_enabled CHAR(1) NOT NULL,
    is_instead TINYINT(1) NOT NULL,
    ev_qual TEXT NOT NULL,
    ev_action TEXT NOT NULL
);`

	CreateTablePgSeclabel = `CREATE TABLE IF NOT EXISTS postgres.pg_seclabel
(
    objoid INTEGER(32) NOT NULL,
    classoid INTEGER(32) NOT NULL,
    objsubid INTEGER(32) NOT NULL,
    provider TEXT NOT NULL,
    label TEXT NOT NULL
);`

	CreateTablePgSequence = `CREATE TABLE IF NOT EXISTS postgres.pg_sequence
(
    seqrelid INTEGER(32) NOT NULL,
    seqtypid INTEGER(32) NOT NULL,
    seqstart BIGINT NOT NULL,
    seqincrement BIGINT NOT NULL,
    seqmax BIGINT NOT NULL,
    seqmin BIGINT NOT NULL,
    seqcache BIGINT NOT NULL,
    seqcycle TINYINT(1) NOT NULL
);`

	CreateTablePgShdepend = `CREATE TABLE IF NOT EXISTS postgres.pg_shdepend
(
    dbid INTEGER(32) NOT NULL,
    classid INTEGER(32) NOT NULL,
    objid INTEGER(32) NOT NULL,
    objsubid INTEGER(32) NOT NULL,
    refclassid INTEGER(32) NOT NULL,
    refobjid INTEGER(32) NOT NULL,
    deptype CHAR(1) NOT NULL
);`

	CreateTablePgShdescription = `CREATE TABLE IF NOT EXISTS postgres.pg_shdescription
(
    objoid INTEGER(32) NOT NULL,
    classoid INTEGER(32) NOT NULL,
    description TEXT NOT NULL
);`

	CreateTablePgShseclabel = `CREATE TABLE IF NOT EXISTS postgres.pg_shseclabel
(
    objoid INTEGER(32) NOT NULL,
    classoid INTEGER(32) NOT NULL,
    provider TEXT NOT NULL,
    label TEXT NOT NULL
);`

	CreateTablePgStatistic = `CREATE TABLE IF NOT EXISTS postgres.pg_statistic
(
    starelid INTEGER(32) NOT NULL,
    staattnum SMALLINT NOT NULL,
    stainherit TINYINT(1) NOT NULL,
    stanullfrac FLOAT NOT NULL,
    stawidth INTEGER(32) NOT NULL,
    stadistinct FLOAT NOT NULL,
    stakind1 SMALLINT NOT NULL,
    stakind2 SMALLINT NOT NULL,
    stakind3 SMALLINT NOT NULL,
    stakind4 SMALLINT NOT NULL,
    stakind5 SMALLINT NOT NULL,
    staop1 INTEGER(32) NOT NULL,
    staop2 INTEGER(32) NOT NULL,
    staop3 INTEGER(32) NOT NULL,
    staop4 INTEGER(32) NOT NULL,
    staop5 INTEGER(32) NOT NULL,
    stacoll1 INTEGER(32) NOT NULL,
    stacoll2 INTEGER(32) NOT NULL,
    stacoll3 INTEGER(32) NOT NULL,
    stacoll4 INTEGER(32) NOT NULL,
    stacoll5 INTEGER(32) NOT NULL,
    stanumbers1 FLOAT,
    stanumbers2 FLOAT,
    stanumbers3 FLOAT,
    stanumbers4 FLOAT,
    stanumbers5 FLOAT,
    stavalues1 VARCHAR(64),
    stavalues2 VARCHAR(64),
    stavalues3 VARCHAR(64),
    stavalues4 VARCHAR(64),
    stavalues5 VARCHAR(64)
);`

	CreateTablePgStatisticExt = `CREATE TABLE IF NOT EXISTS postgres.pg_statistic_ext
(
    oid INTEGER(32) NOT NULL,
    stxrelid INTEGER(32) NOT NULL,
    stxname VARCHAR(32) NOT NULL,
    stxnamespace INTEGER(32) NOT NULL,
    stxowner INTEGER(32) NOT NULL,
    stxstattarget INTEGER(32) NOT NULL,
    stxkeys VARCHAR(64) NOT NULL,
    stxkind CHAR(1) NOT NULL
);`

	CreateTablePgStatisticExtData = `CREATE TABLE IF NOT EXISTS postgres.pg_statistic_ext_data
(
    stxoid INTEGER(32) NOT NULL,
    stxdndistinct VARCHAR(64),
    stxddependencies VARCHAR(64),
    stxdmcv VARCHAR(64)
);`

	CreateTablePgSubscription = `CREATE TABLE IF NOT EXISTS postgres.pg_subscription
(
    oid INTEGER(32) NOT NULL,
    subdbid INTEGER(32) NOT NULL,
    subname VARCHAR(32) NOT NULL,
    subowner INTEGER(32) NOT NULL,
    subenabled TINYINT(1) NOT NULL,
    subconninfo TEXT NOT NULL,
    subslotname VARCHAR(32),
    subsynccommit TEXT NOT NULL,
    subpublications TEXT NOT NULL
);`

	CreateTablePgSubscriptionRel = `CREATE TABLE IF NOT EXISTS postgres.pg_subscription_rel
(
    srsubid INTEGER(32) NOT NULL,
    srrelid INTEGER(32) NOT NULL,
    srsubstate CHAR(1) NOT NULL,
    srsublsn VARCHAR(64)
);`

	CreateTablePgTablespace = `CREATE TABLE IF NOT EXISTS postgres.pg_tablespace
(
    oid INTEGER(32) NOT NULL,
    spcname VARCHAR(32)  NOT NULL,
    spcowner INTEGER(32) NOT NULL,
    spcacl VARCHAR(255),
    spcoptions TEXT
);`

	CreateTablePgTransform = `CREATE TABLE IF NOT EXISTS postgres.pg_transform
(
    oid INTEGER(32) NOT NULL,
    trftype INTEGER(32) NOT NULL,
    trflang INTEGER(32) NOT NULL,
    trffromsql TEXT NOT NULL,
    trftosql TEXT NOT NULL
);`

	CreateTablePgTrigger = `CREATE TABLE IF NOT EXISTS postgres.pg_trigger
(
    oid INTEGER(32) NOT NULL,
    tgrelid INTEGER(32) NOT NULL,
    tgparentid INTEGER(32) NOT NULL,
    tgname VARCHAR(32) NOT NULL,
    tgfoid INTEGER(32) NOT NULL,
    tgtype SMALLINT NOT NULL,
    tgenabled CHAR(1) NOT NULL,
    tgisinternal TINYINT(1) NOT NULL,
    tgconstrrelid INTEGER(32) NOT NULL,
    tgconstrindid INTEGER(32) NOT NULL,
    tgconstraint INTEGER(32) NOT NULL,
    tgdeferrable TINYINT(1) NOT NULL,
    tginitdeferred TINYINT(1) NOT NULL,
    tgnargs SMALLINT NOT NULL,
    tgattr VARCHAR(64) NOT NULL,
    tgargs BINARY NOT NULL,
    tgqual TEXT,
    tgoldtable VARCHAR(32),
    tgnewtable VARCHAR(32)
);`

	CreateTablePgTsConfig = `CREATE TABLE IF NOT EXISTS postgres.pg_ts_config
(
    oid INTEGER(32) NOT NULL,
    cfgname VARCHAR(32),
    cfgnamespace INTEGER(32) NOT NULL,
    cfgowner INTEGER(32) NOT NULL,
    cfgparser INTEGER(32) NOT NULL
);`

	CreateTablePgTsConfigMap = `CREATE TABLE IF NOT EXISTS postgres.pg_ts_config_map
(
    mapcfg INTEGER(32) NOT NULL,
    maptokentype INTEGER(32) NOT NULL,
    mapseqno INTEGER(32) NOT NULL,
    mapdict INTEGER(32) NOT NULL
);`

	CreateTablePgTsDict = `CREATE TABLE IF NOT EXISTS postgres.pg_ts_dict
(
    oid INTEGER(32) NOT NULL,
    dictname VARCHAR(32),
    dictnamespace INTEGER(32) NOT NULL,
    dictowner INTEGER(32) NOT NULL,
    dicttemplate INTEGER(32) NOT NULL,
    dictinitoption TEXT
);`

	CreateTablePgTsParser = `CREATE TABLE IF NOT EXISTS postgres.pg_ts_parser
(
    oid INTEGER(32) NOT NULL,
    prsname VARCHAR(32) NOT NULL,
    prsnamespace INTEGER(32) NOT NULL,
    prsstart TEXT NOT NULL,
    prstoken TEXT NOT NULL,
    prsend TEXT NOT NULL,
    prsheadline TEXT NOT NULL,
    prslextype TEXT NOT NULL
);`

	CreateTablePgTsTemplate = `CREATE TABLE IF NOT EXISTS postgres.pg_ts_template
(
    oid INTEGER(32) NOT NULL,
    tmplname VARCHAR(32) NOT NULL,
    tmplnamespace INTEGER(32) NOT NULL,
    tmplinit TEXT NOT NULL,
    tmpllexize TEXT NOT NULL
);`

	CreateTablePgType = `CREATE TABLE IF NOT EXISTS postgres.pg_type
(
    oid INTEGER(32) NOT NULL,
    typname VARCHAR(32) NOT NULL,
    typnamespace INTEGER(32) NOT NULL,
    typowner INTEGER(32) NOT NULL,
    typlen SMALLINT NOT NULL,
    typbyval TINYINT(1) NOT NULL,
    typtype CHAR(1) NOT NULL,
    typcategory CHAR(1) NOT NULL,
    typispreferred TINYINT(1) NOT NULL,
    typisdefined TINYINT(1) NOT NULL,
    typdelim CHAR(1) NOT NULL,
    typrelid INTEGER(32) NOT NULL,
    typelem INTEGER(32) NOT NULL,
    typarray INTEGER(32) NOT NULL,
    typinput TEXT NOT NULL,
    typoutput TEXT NOT NULL,
    typreceive TEXT NOT NULL,
    typsend TEXT NOT NULL,
    typmodin TEXT NOT NULL,
    typmodout TEXT NOT NULL,
    typanalyze TEXT NOT NULL,
    typalign CHAR(1) NOT NULL,
    typstorage CHAR(1) NOT NULL,
    typnotnull TINYINT(1) NOT NULL,
    typbasetype INTEGER(32) NOT NULL,
    typtypmod INTEGER(32) NOT NULL,
    typndims INTEGER(32) NOT NULL,
    typcollation INTEGER(32) NOT NULL,
    typdefaultbin TEXT,
    typdefault TEXT,
    typacl VARCHAR(255)
);`
	CreateTablePgUserMapping = `CREATE TABLE IF NOT EXISTS postgres.pg_user_mapping
(
    oid INTEGER(32) NOT NULL,
    umuser INTEGER(32) NOT NULL,
    umserver INTEGER(32) NOT NULL,
    umoptions TEXT
);`
)

// PG SYSTEM VIEW SQL
const (
	CreateViewPgRoles = `CREATE DEFINER = root VIEW postgres.pg_roles AS
    SELECT
        rolname,
        rolsuper,
        rolinherit,
        rolcreaterole,
        rolcreatedb,
        rolcanlogin,
        rolreplication,
        rolconnlimit,
        '********' as rolpassword,
        rolvaliduntil,
        rolbypassrls,
        setconfig as rolconfig,
        pg_authid.oid
    FROM postgres.pg_authid LEFT JOIN postgres.pg_db_role_setting s
    ON (postgres.pg_authid.oid = setrole AND setdatabase = 0);`

	CreateViewPgShadow = `CREATE DEFINER = root VIEW pg_shadow AS
    SELECT
        rolname AS usename,
        pg_authid.oid AS usesysid,
        rolcreatedb AS usecreatedb,
        rolsuper AS usesuper,
        rolreplication AS userepl,
        rolbypassrls AS usebypassrls,
        rolpassword AS passwd,
        rolvaliduntil AS valuntil,
        setconfig AS useconfig
    FROM pg_authid LEFT JOIN pg_db_role_setting s
    ON (pg_authid.oid = setrole AND setdatabase = 0)
    WHERE rolcanlogin;`

	CreateViewPgGroup = `CREATE DEFINER = root VIEW pg_group AS
    SELECT
        rolname AS groname,
        oid AS grosysid,
        (SELECT member FROM pg_auth_members WHERE roleid = oid) AS grolist
    FROM pg_authid
    WHERE NOT rolcanlogin;`

	CreateViewPgUser = `CREATE DEFINER = root VIEW pg_user AS
    SELECT
        usename,
        usesysid,
        usecreatedb,
        usesuper,
        userepl,
        usebypassrls,
        '********' as passwd,
        valuntil,
        useconfig
    FROM pg_shadow;`
)

//DATA FOR PG SYSTEM TABLE
const (
	DataForTablePgAggregate = `INSERT INTO
postgres.pg_authid VALUES
(3373, 'pg_monitor', 0, 1, 0, 0, 0, 0, 0, -1, NULL, NULL),
(3374, 'pg_read_all_settings', 0, 1, 0, 0, 0, 0, 0, -1, NULL, NULL),
(3375, 'pg_read_all_stats', 0, 1, 0, 0, 0, 0, 0, -1, NULL, NULL),
(3377, 'pg_stat_scan_tables', 0, 1, 0, 0, 0, 0, 0, -1, NULL, NULL),
(4569, 'pg_read_server_files', 0, 1, 0, 0, 0, 0, 0, -1, NULL, NULL),
(4570, 'pg_write_server_files', 0, 1, 0, 0, 0, 0, 0, -1, NULL, NULL),
(4571, 'pg_execute_server_program', 0, 1, 0, 0, 0, 0, 0, -1, NULL, NULL),
(4200, 'pg_signal_backend', 0, 1, 0, 0, 0, 0, 0, -1, NULL, NULL),
(10, 'postgres', 1, 1, 1, 1, 1, 1, 1, -1, NULL, NULL)
;`

	DataForTablePgAuthMembers = `INSERT INTO
postgres.pg_auth_members
VALUES
(3374,3373,10,0),
(3375,3373,10,0),
(3377,3373,10,0);`

	DataForTablePgDatabase = `INSERT INTO
postgres.pg_database
VALUES
(1,'template1',10,6,'Chinese(Simplified)_China.936','Chinese(Simplified)_China.936',1,1,-1,13441,479,1,1663,'{=c/postgres,postgres=CTc/postgres}'),
(13441,'template0',10,6,'Chinese(Simplified)_China.936','Chinese(Simplified)_China.936',1,0,-1,13441,479,1,1663,'{=c/postgres,postgres=CTc/postgres}'),
(13442,'postgres',10,6,'Chinese(Simplified)_China.936','Chinese(Simplified)_China.936',0,1,-1,13441,479,1,1663,NULL);`

	DataForTablePgClass = ``

	DataForTablePgType = `INSERT INTO
postgres.pg_type VALUES
(25,'text',11,10,-1,0,'b','S',1,1,',',0,0,1009,'textin','textout','textrecv','textsend','-','-','-','i','x',0,0,-1,0,100,NULL,NULL,NULL),
(1043,'varchar',11,10,-1,0,'b','S',0,1,',',0,0,1015,'varcharin','varcharout','varcharrecv','varcharsend','varchartypmodin','varchartypmodout','-','i','x',0,0,-1,0,100,NULL,NULL,NULL);
`
	DataForTablePgTablespace =`INSERT INTO
postgres.pg_tablespace VALUES
(1663,'pg_default',10,NULL,NULL),
(1664,'pg_global',10,NULL,NULL);`

	DataForTablePgNamespace = `INSERT INTO
postgres.pg_namespace VALUES
(99,'ph_toast',10,NULL),
(11,'pg_catalog',10,'{postgres=UC/postgres,=U/postgres}'),
(2200,'public',10,'{postgres=UC/postgres,=U/postgres}'),
(13158,'information_schema',10,'{postgres=UC/postgres,=U/postgres}');`

)

//todo list 暂且考虑连接到PgAdmin需要的系统数据表、系统数据视图，因而，中间会牵扯到一些系统变量以及系统函数
//
//系统视图
//pg_settings
//
//系统函数
//pg_show_all_settings、set_config、pg_encoding_to_char、has_database_privilege、current_database、pg_is_in_recovery
//pg_is_wal_replay_paused、has_table_privilege、has_schema_privilege、has_database_privilege
