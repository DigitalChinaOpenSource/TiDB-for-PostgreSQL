package executor

import (
	"context"
	"fmt"
	"github.com/pingcap/tidb/infoschema"
	"github.com/pingcap/tidb/parser/charset"
	"github.com/pingcap/tidb/parser/model"
	"github.com/pingcap/tidb/parser/mysql"
	"github.com/pingcap/tidb/privilege"
	"github.com/pingcap/tidb/sessionctx"
	"github.com/pingcap/tidb/table"
	"github.com/pingcap/tidb/types"
	"github.com/pingcap/tidb/util"
	"github.com/pingcap/tidb/util/collate"
	"sort"
)

type pgMemTableRetriever struct {
	dummyCloser
	table       *model.TableInfo
	columns     []*model.ColumnInfo
	retrieved   bool
	initialized bool
	rows        [][]types.Datum
	dbs         []*model.DBInfo
	dbsIdx      int
	tblIdx      int
	rowIdx      int
}

func (e *pgMemTableRetriever) retrieve(ctx context.Context, sctx sessionctx.Context) ([][]types.Datum, error) {
	if e.retrieved {
		return nil, nil
	}
	//Cache the ret full rows in schemataRetriever
	if !e.initialized {
		is := sctx.GetInfoSchema().(infoschema.InfoSchema)
		dbs := is.AllSchemas()
		sort.Sort(infoschema.SchemasSorter(dbs))
		var err error
		switch e.table.Name.O {
		case infoschema.TablePgInformationsSchemaCatalogName:
			e.setDataForPgInformationSchemaCatalogName()
		case infoschema.TablePgSchemata:
			e.setDataForPgSchemata(sctx, dbs)
		case infoschema.TablePgAdministrableRoleAuthorizations:
			e.setDataForPgAdministrableRoleAuthorizations()
		case infoschema.TablePgEnabledRoles:
			e.setDataForPgEnabledRoles()
		case infoschema.TablePgTables:
			err = e.setDataForPgTables(ctx, sctx, dbs)
		case infoschema.TablePgCollations:
			e.setDataForPgCollations()
		case infoschema.TablePgSequences:
			e.setDataForPgSequences(sctx, dbs)
		case infoschema.TablePgViews:
			e.setDataForPgViews(sctx, dbs)
		case infoschema.TablePgTableConstraints:
			e.setDataForPgTableConstraints(sctx, dbs)
		case infoschema.TablePgCharacterSets:
			e.setDataForPgCharacterSets()
		case infoschema.TablePgKeyColumnUsage:
			e.setDataForPgKeyColumnUsage(sctx, dbs)
		case infoschema.TablePgCollationCharacterSetApplicability:
			e.SetDataForCollationCharacterSetApplicability()
			// todo set data for tables
		}
		if err != nil {
			return nil, err
		}
		e.initialized = true
	}

	//Adjust the amount of each return
	maxCount := 1024
	retCount := maxCount
	if e.rowIdx+maxCount > len(e.rows) {
		retCount = len(e.rows) - e.rowIdx
		e.retrieved = true
	}
	ret := make([][]types.Datum, retCount)
	for i := e.rowIdx; i < e.rowIdx+retCount; i++ {
		ret[i-e.rowIdx] = e.rows[i]
	}
	e.rowIdx += retCount
	return adjustColumns(ret, e.columns, e.table), nil
}

// setDataForPgInformationSchemaCatalogName set data for pgTable information_schema_catalog_name
// todo 这个值应该是动态的，这里先写死
func (e *pgMemTableRetriever) setDataForPgInformationSchemaCatalogName() {
	var rows [][]types.Datum
	rows = append(rows,
		types.MakeDatums(
			"postgres", // catalog_name
		),
	)
	e.rows = rows
}

// setDataForPgAdministrableRoleAuthorizations set data for pgTable administrable_role_authorizations
func (e *pgMemTableRetriever) setDataForPgAdministrableRoleAuthorizations() {
	var rows [][]types.Datum
	rows = append(rows,
		types.MakeDatums(
			"root",  // grantee
			"admin", // role_name
			"YES",   // is_grantable
		),
	)
	e.rows = rows
}

// setDataForPgSchemata set data for pgTable schemata
// todo 这里的 catalog_name 和 schema_owner 应该是动态获取，暂时写死
func (e *pgMemTableRetriever) setDataForPgSchemata(ctx sessionctx.Context, schemas []*model.DBInfo) {
	checker := privilege.GetPrivilegeManager(ctx)
	rows := make([][]types.Datum, 0, len(schemas))

	for _, schema := range schemas {

		schemaCharset := mysql.DefaultCharset
		collation := mysql.DefaultCollationName

		if len(schema.Charset) > 0 {
			schemaCharset = schema.Charset //overwrite default
		}

		if len(schema.Collate) > 0 {
			collation = schema.Collate //overwrite default
		}

		if checker != nil && !checker.RequestVerification(ctx.GetSessionVars().ActiveRoles, schema.Name.L, "", "", mysql.AllPrivMask) {
			continue
		}

		record := types.MakeDatums(
			"postgres",    // catalog_name
			schema.Name.O, // schema_name
			"root",        // schema_owner
			nil,           // default_character_set_catalog
			nil,           // default_character_set_schema
			schemaCharset, // default_character_set_name
			nil,           // sql_path
			collation,     // DEFAULT_COLLATION_NAME
		)
		rows = append(rows, record)
	}
	e.rows = rows
}

// setDataForPgTables set data for pgTable tables
func (e *pgMemTableRetriever) setDataForPgTables(ctx context.Context, sctx sessionctx.Context, schemas []*model.DBInfo) error {
	tableRowsMap, colLengthMap, err := tableStatsCache.get(ctx, sctx)
	if err != nil {
		return err
	}

	checker := privilege.GetPrivilegeManager(sctx)

	var rows [][]types.Datum
	createTimeTp := mysql.TypeDatetime
	for _, schema := range schemas {
		for _, table := range schema.Tables {
			collation := table.Collate
			if collation == "" {
				collation = mysql.DefaultCollationName
			}
			createTime := types.NewTime(types.FromGoTime(table.GetUpdateTime()), createTimeTp, types.DefaultFsp)

			createOptions := ""

			if table.IsSequence() {
				continue
			}

			if checker != nil && !checker.RequestVerification(sctx.GetSessionVars().ActiveRoles, schema.Name.L, table.Name.L, "", mysql.AllPrivMask) {
				continue
			}

			if !table.IsView() {
				if table.GetPartitionInfo() != nil {
					createOptions = "partitioned"
				}
				var autoIncID interface{}
				hasAutoincID, _ := infoschema.HasAutoIncrementColumn(table)
				if hasAutoincID {
					autoIncID, err = getAutoIncrementID(sctx, schema, table)
					if err != nil {
						return err
					}
				}
				var rowCount, dataLength, indexLength uint64
				if table.GetPartitionInfo() == nil {
					rowCount = tableRowsMap[table.ID]
					dataLength, indexLength = getDataAndIndexLength(table, table.ID, rowCount, colLengthMap)
				} else {
					for _, pi := range table.GetPartitionInfo().Definitions {
						rowCount += tableRowsMap[pi.ID]
						parDataLen, parIndexLen := getDataAndIndexLength(table, pi.ID, tableRowsMap[pi.ID], colLengthMap)
						dataLength += parDataLen
						indexLength += parIndexLen
					}
				}
				avgRowLength := uint64(0)
				if rowCount != 0 {
					avgRowLength = dataLength / rowCount
				}
				var tableType string
				switch schema.Name.L {
				case util.InformationSchemaName.L, util.PerformanceSchemaName.L,
					util.MetricSchemaName.L:
					tableType = "SYSTEM VIEW"
				default:
					tableType = "BASE TABLE"
				}
				shardingInfo := infoschema.GetShardingInfo(schema, table)
				record := types.MakeDatums(
					"postgres",    // TABLE_CATALOG
					schema.Name.O, // TABLE_SCHEMA
					table.Name.O,  // TABLE_NAME
					tableType,     // TABLE_TYPE
					nil,           // self_referencing_column_name
					nil,           // reference_generation
					nil,           // user_defined_type_catalog
					nil,           // user_defined_type_schema
					nil,           // user_defined_type_name
					nil,           // is_insertable_into
					nil,           // is_typed
					nil,           // commit_action
					"InnoDB",      // ENGINE
					uint64(10),    // VERSION
					"Compact",     // ROW_FORMAT
					rowCount,      // TABLE_ROWS
					avgRowLength,  // AVG_ROW_LENGTH
					dataLength,    // DATA_LENGTH
					uint64(0),     // MAX_DATA_LENGTH
					indexLength,   // INDEX_LENGTH
					uint64(0),     // DATA_FREE
					autoIncID,     // AUTO_INCREMENT
					createTime,    // CREATE_TIME
					nil,           // UPDATE_TIME
					nil,           // CHECK_TIME
					collation,     // TABLE_COLLATION
					nil,           // CHECKSUM
					createOptions, // CREATE_OPTIONS
					table.Comment, // TABLE_COMMENT
					table.ID,      // TIDB_TABLE_ID
					shardingInfo,  // TIDB_ROW_ID_SHARDING_INFO
				)
				rows = append(rows, record)
			} else {
				record := types.MakeDatums(
					"postgres",    // TABLE_CATALOG
					schema.Name.O, // TABLE_SCHEMA
					table.Name.O,  // TABLE_NAME
					"VIEW",        // TABLE_TYPE
					nil,           // self_referencing_column_name
					nil,           // reference_generation
					nil,           // user_defined_type_catalog
					nil,           // user_defined_type_schema
					nil,           // user_defined_type_name
					nil,           // is_insertable_into
					nil,           // is_typed
					nil,           // commit_action
					nil,           // ENGINE
					nil,           // VERSION
					nil,           // ROW_FORMAT
					nil,           // TABLE_ROWS
					nil,           // AVG_ROW_LENGTH
					nil,           // DATA_LENGTH
					nil,           // MAX_DATA_LENGTH
					nil,           // INDEX_LENGTH
					nil,           // DATA_FREE
					nil,           // AUTO_INCREMENT
					createTime,    // CREATE_TIME
					nil,           // UPDATE_TIME
					nil,           // CHECK_TIME
					nil,           // TABLE_COLLATION
					nil,           // CHECKSUM
					nil,           // CREATE_OPTIONS
					"VIEW",        // TABLE_COMMENT
					table.ID,      // TIDB_TABLE_ID
					nil,           // TIDB_ROW_ID_SHARDING_INFO
				)
				rows = append(rows, record)
			}
		}
	}
	e.rows = rows
	return nil
}

// setDataForPgEnabledRoles set data for pgTable enabled_roles
func (e *pgMemTableRetriever) setDataForPgEnabledRoles() {
	var rows [][]types.Datum
	rows = append(rows,
		types.MakeDatums(
			"root", // catalog_name
		),
	)
	rows = append(rows,
		types.MakeDatums(
			"admin", // catalog_name
		),
	)
	e.rows = rows
}

// setDataForPgCollations set data for pgTable collations
func (e *pgMemTableRetriever) setDataForPgCollations() {
	var rows [][]types.Datum
	collations := collate.GetSupportedCollations()
	for _, collation := range collations {
		isDefault := ""
		if collation.IsDefault {
			isDefault = "Yes"
		}
		rows = append(rows,
			types.MakeDatums("postgres", "pg_catalog", collation.Name, "NO PAD", collation.CharsetName, collation.ID, isDefault, "Yes", 1),
		)
	}
	e.rows = rows
}

// setDataForPgColumns set data for pgTable Columns
func (e *hugeMemTableRetriever) setDataForPgColumns(ctx sessionctx.Context) error {
	checker := privilege.GetPrivilegeManager(ctx)
	e.rows = e.rows[:0]
	batch := 1024
	for ; e.dbsIdx < len(e.dbs); e.dbsIdx++ {
		schema := e.dbs[e.dbsIdx]
		for e.tblIdx < len(schema.Tables) {
			table := schema.Tables[e.tblIdx]
			e.tblIdx++
			if checker != nil && !checker.RequestVerification(ctx.GetSessionVars().ActiveRoles, schema.Name.L, table.Name.L, "", mysql.AllPrivMask) {
				continue
			}

			e.dataForPgColumnsInTable(schema, table)
			if len(e.rows) >= batch {
				return nil
			}
		}
		e.tblIdx = 0
	}
	return nil
}

// dataForPgColumnsInTable get pgTable pg_columns data from system
func (e *hugeMemTableRetriever) dataForPgColumnsInTable(schema *model.DBInfo, tbl *model.TableInfo) {
	for i, col := range tbl.Columns {
		if col.Hidden {
			continue
		}
		var charMaxLen, charOctLen, numericPrecision, numericScale, datetimePrecision interface{}
		colLen, decimal := col.Flen, col.Decimal
		defaultFlen, defaultDecimal := mysql.GetDefaultFieldLengthAndDecimal(col.Tp)
		if decimal == types.UnspecifiedLength {
			decimal = defaultDecimal
		}
		if colLen == types.UnspecifiedLength {
			colLen = defaultFlen
		}
		if col.Tp == mysql.TypeSet {
			// Example: In MySQL set('a','bc','def','ghij') has length 13, because
			// len('a')+len('bc')+len('def')+len('ghij')+len(ThreeComma)=13
			// Reference link: https://bugs.mysql.com/bug.php?id=22613
			colLen = 0
			for _, ele := range col.Elems {
				colLen += len(ele)
			}
			if len(col.Elems) != 0 {
				colLen += (len(col.Elems) - 1)
			}
			charMaxLen = colLen
			charOctLen = colLen
		} else if col.Tp == mysql.TypeEnum {
			// Example: In MySQL enum('a', 'ab', 'cdef') has length 4, because
			// the longest string in the enum is 'cdef'
			// Reference link: https://bugs.mysql.com/bug.php?id=22613
			colLen = 0
			for _, ele := range col.Elems {
				if len(ele) > colLen {
					colLen = len(ele)
				}
			}
			charMaxLen = colLen
			charOctLen = colLen
		} else if types.IsString(col.Tp) {
			charMaxLen = colLen
			charOctLen = colLen
		} else if types.IsTypeFractionable(col.Tp) {
			datetimePrecision = decimal
		} else if types.IsTypeNumeric(col.Tp) {
			numericPrecision = colLen
			if col.Tp != mysql.TypeFloat && col.Tp != mysql.TypeDouble {
				numericScale = decimal
			} else if decimal != -1 {
				numericScale = decimal
			}
		}
		columnType := col.FieldType.InfoSchemaStr()
		columnDesc := table.NewColDesc(table.ToColumn(col))
		var columnDefault interface{}
		if columnDesc.DefaultValue != nil {
			columnDefault = fmt.Sprintf("%v", columnDesc.DefaultValue)
		}
		var record []types.Datum
		if schema.Name.O == "information_schema" {
			record = types.MakeDatums(
				"postgres",                           // TABLE_CATALOG
				schema.Name.O,                        // TABLE_SCHEMA
				tbl.Name.O,                           // TABLE_NAME
				col.Name.O,                           // COLUMN_NAME
				i+1,                                  // ORIGINAL_POSITION
				columnDefault,                        // COLUMN_DEFAULT
				columnDesc.Null,                      // IS_NULLABLE
				types.TypeToStr(col.Tp, col.Charset), // DATA_TYPE
				charMaxLen,                           // CHARACTER_MAXIMUM_LENGTH
				charOctLen,                           // CHARACTER_OCTET_LENGTH
				numericPrecision,                     // NUMERIC_PRECISION
				nil,                                  // numeric_precision_radix
				numericScale,                         // NUMERIC_SCALE
				datetimePrecision,                    // DATETIME_PRECISION
				nil,                                  // interval_type
				nil,                                  // interval_precision
				nil,                                  // character_set_catalog
				nil,                                  // character_set_schema
				columnDesc.Charset,                   // CHARACTER_SET_NAME
				"postgres",                           // collation_catalog
				"pg_catalog",                         // collation_schema
				columnDesc.Collation,                 // COLLATION_NAME
				"postgres",                           // domain_catalog
				"information_schema",                 // domain_schema
				"sql_identifier",                     // domain_name
				"postgres",                           // udt_catalog
				"pg_catalog",                         // udt_schema
				nil,                                  // udt_name
				nil,                                  // scope_catalog
				nil,                                  // scope_schema
				nil,                                  // scope_name
				nil,                                  // maximum_cardinality
				nil,                                  // dtd_identifier
				nil,                                  // is_self_referencing
				nil,                                  // is_identity
				nil,                                  // identity_generation
				nil,                                  // identity_start
				nil,                                  // identity_increment
				nil,                                  // identity_maximum
				nil,                                  // identity_minimum
				nil,                                  // identity_cycle
				nil,                                  // is_generated
				col.GeneratedExprString,              // GENERATION_EXPRESSION
				nil,                                  // is_updatable
				columnType,                           // COLUMN_TYPE
				columnDesc.Key,                       // COLUMN_KEY
				columnDesc.Extra,                     // EXTRA
				"select,insert,update,references",    // PRIVILEGES
				columnDesc.Comment,                   // COLUMN_COMMENT
			)
		} else {
			record = types.MakeDatums(
				"postgres",                           // TABLE_CATALOG
				schema.Name.O,                        // TABLE_SCHEMA
				tbl.Name.O,                           // TABLE_NAME
				col.Name.O,                           // COLUMN_NAME
				i+1,                                  // ORIGINAL_POSITION
				columnDefault,                        // COLUMN_DEFAULT
				columnDesc.Null,                      // IS_NULLABLE
				types.TypeToStr(col.Tp, col.Charset), // DATA_TYPE
				charMaxLen,                           // CHARACTER_MAXIMUM_LENGTH
				charOctLen,                           // CHARACTER_OCTET_LENGTH
				numericPrecision,                     // NUMERIC_PRECISION
				nil,                                  // numeric_precision_radix
				numericScale,                         // NUMERIC_SCALE
				datetimePrecision,                    // DATETIME_PRECISION
				nil,                                  // interval_type
				nil,                                  // interval_precision
				nil,                                  // character_set_catalog
				nil,                                  // character_set_schema
				columnDesc.Charset,                   // CHARACTER_SET_NAME
				nil,                                  // collation_catalog
				nil,                                  // collation_schema
				columnDesc.Collation,                 // COLLATION_NAME
				nil,                                  // domain_catalog
				nil,                                  // domain_schema
				nil,                                  // domain_name
				"postgres",                           // udt_catalog
				"pg_catalog",                         // udt_schema
				nil,                                  // udt_name
				nil,                                  // scope_catalog
				nil,                                  // scope_schema
				nil,                                  // scope_name
				nil,                                  // maximum_cardinality
				nil,                                  // dtd_identifier
				nil,                                  // is_self_referencing
				nil,                                  // is_identity
				nil,                                  // identity_generation
				nil,                                  // identity_start
				nil,                                  // identity_increment
				nil,                                  // identity_maximum
				nil,                                  // identity_minimum
				nil,                                  // identity_cycle
				nil,                                  // is_generated
				col.GeneratedExprString,              // GENERATION_EXPRESSION
				nil,                                  // is_updatable
				columnType,                           // COLUMN_TYPE
				columnDesc.Key,                       // COLUMN_KEY
				columnDesc.Extra,                     // EXTRA
				"select,insert,update,references",    // PRIVILEGES
				columnDesc.Comment,                   // COLUMN_COMMENT
			)
		}
		e.rows = append(e.rows, record)
	}
}

// setDataForPgSequences set data for pgTable sequences
func (e *pgMemTableRetriever) setDataForPgSequences(ctx sessionctx.Context, schemas []*model.DBInfo) {
	checker := privilege.GetPrivilegeManager(ctx)
	var rows [][]types.Datum
	for _, schema := range schemas {
		for _, table := range schema.Tables {
			if !table.IsSequence() {
				continue
			}
			if checker != nil && !checker.RequestVerification(ctx.GetSessionVars().ActiveRoles, schema.Name.L, table.Name.L, "", mysql.AllPrivMask) {
				continue
			}
			record := types.MakeDatums(
				"postgres",                // sequence_catalog
				schema.Name.O,             // sequence_schema
				table.Name.O,              // sequence_name
				nil,                       // data_type
				nil,                       // numeric_precision
				nil,                       // numeric_precision_radix
				nil,                       // numeric_scale
				nil,                       // start_value
				nil,                       // minimum_value
				nil,                       // maximum_value
				table.Sequence.Increment,  // INCREMENT
				nil,                       // cycle_option
				"postgres",                // TABLE_CATALOG
				table.Sequence.Cache,      // Cache
				table.Sequence.CacheValue, // CACHE_VALUE
				table.Sequence.Cycle,      // CYCLE
				table.Sequence.MaxValue,   // MAX_VALUE
				table.Sequence.MinValue,   // MIN_VALUE
				table.Sequence.Start,      // START
				table.Sequence.Comment,    // COMMENT
			)
			rows = append(rows, record)
		}
	}
	e.rows = rows
}

// setDataForPgViews set data for pgTable pg_views
func (e *pgMemTableRetriever) setDataForPgViews(ctx sessionctx.Context, schemas []*model.DBInfo) {
	checker := privilege.GetPrivilegeManager(ctx)
	var rows [][]types.Datum
	for _, schema := range schemas {
		for _, table := range schema.Tables {
			if !table.IsView() {
				continue
			}
			collation := table.Collate
			charset := table.Charset
			if collation == "" {
				collation = mysql.DefaultCollationName
			}
			if charset == "" {
				charset = mysql.DefaultCharset
			}
			if checker != nil && !checker.RequestVerification(ctx.GetSessionVars().ActiveRoles, schema.Name.L, table.Name.L, "", mysql.AllPrivMask) {
				continue
			}
			record := types.MakeDatums(
				"postgres",                      // TABLE_CATALOG
				schema.Name.O,                   // TABLE_SCHEMA
				table.Name.O,                    // TABLE_NAME
				table.View.SelectStmt,           // VIEW_DEFINITION
				table.View.CheckOption.String(), // CHECK_OPTION
				"NO",                            // IS_UPDATABLE
				nil,                             // is_insertable_into
				nil,                             // is_trigger_updatable
				nil,                             // is_trigger_deletable
				nil,                             // is_trigger_insertable_into
				table.View.Definer.String(),     // DEFINER
				table.View.Security.String(),    // SECURITY_TYPE
				charset,                         // CHARACTER_SET_CLIENT
				collation,                       // COLLATION_CONNECTION
			)
			rows = append(rows, record)
		}
	}
	e.rows = rows
}

// setDataForPgTableConstraints set data for pgTable table_constraints
func (e *pgMemTableRetriever) setDataForPgTableConstraints(ctx sessionctx.Context, schemas []*model.DBInfo) {
	checker := privilege.GetPrivilegeManager(ctx)
	var rows [][]types.Datum
	for _, schema := range schemas {
		for _, tbl := range schema.Tables {
			if checker != nil && !checker.RequestVerification(ctx.GetSessionVars().ActiveRoles, schema.Name.L, tbl.Name.L, "", mysql.AllPrivMask) {
				continue
			}

			if tbl.PKIsHandle {
				record := types.MakeDatums(
					"postgres",                // CONSTRAINT_CATALOG
					schema.Name.O,             // CONSTRAINT_SCHEMA
					mysql.PrimaryKeyName,      // CONSTRAINT_NAME
					"postgres",                // table_catalog
					schema.Name.O,             // TABLE_SCHEMA
					tbl.Name.O,                // TABLE_NAME
					infoschema.PrimaryKeyType, // CONSTRAINT_TYPE
					nil,                       // is_deferrable
					nil,                       // initially_deferred
					nil,                       // enforced
				)
				rows = append(rows, record)
			}

			for _, idx := range tbl.Indices {
				var cname, ctype string
				if idx.Primary {
					cname = mysql.PrimaryKeyName
					ctype = infoschema.PrimaryKeyType
				} else if idx.Unique {
					cname = idx.Name.O
					ctype = infoschema.UniqueKeyType
				} else {
					// The index has no constriant.
					continue
				}
				record := types.MakeDatums(
					"postgres",    // CONSTRAINT_CATALOG
					schema.Name.O, // CONSTRAINT_SCHEMA
					cname,         // CONSTRAINT_NAME
					"postgres",    // table_catalog
					schema.Name.O, // TABLE_SCHEMA
					tbl.Name.O,    // TABLE_NAME
					ctype,         // CONSTRAINT_TYPE
					nil,           // is_deferrable
					nil,           // initially_deferred
					nil,           // enforced
				)
				rows = append(rows, record)
			}
		}
	}
	e.rows = rows
}

// setDataForPgCharacterSets set data for pgTable character_sets
func (e *pgMemTableRetriever) setDataForPgCharacterSets() {
	var rows [][]types.Datum
	charsets := charset.GetSupportedCharsets()
	for _, charset := range charsets {
		rows = append(rows,
			types.MakeDatums(nil, nil, charset.Name, "UCS", charset.Name, "postgres", nil, charset.DefaultCollation, charset.Desc, charset.Maxlen),
		)
	}
	e.rows = rows
}

// SetDataForCollationCharacterSetApplicability set data for pgTable collation_character_set_applicability
func (e *pgMemTableRetriever) SetDataForCollationCharacterSetApplicability() {
	var rows [][]types.Datum
	collations := collate.GetSupportedCollations()
	for _, collation := range collations {
		rows = append(rows,
			types.MakeDatums("postgres", "pg_catalog", collation.Name, nil, nil, collation.CharsetName),
		)
	}
	e.rows = rows
}

// setDataForPgKeyColumnUsage set data for pgTable KEY_COLUMN_USAGE
func (e *pgMemTableRetriever) setDataForPgKeyColumnUsage(ctx sessionctx.Context, schemas []*model.DBInfo) {
	checker := privilege.GetPrivilegeManager(ctx)
	rows := make([][]types.Datum, 0, len(schemas)) // The capacity is not accurate, but it is not a big problem.
	for _, schema := range schemas {
		for _, table := range schema.Tables {
			if checker != nil && !checker.RequestVerification(ctx.GetSessionVars().ActiveRoles, schema.Name.L, table.Name.L, "", mysql.AllPrivMask) {
				continue
			}
			rs := pgKeyColumnUsageInTable(schema, table)
			rows = append(rows, rs...)
		}
	}
	e.rows = rows
}

// pgKeyColumnUsageInTable get table KEY_COLUMN_USAGE data from system
func pgKeyColumnUsageInTable(schema *model.DBInfo, table *model.TableInfo) [][]types.Datum {
	var rows [][]types.Datum
	if table.PKIsHandle {
		for _, col := range table.Columns {
			if mysql.HasPriKeyFlag(col.Flag) {
				record := types.MakeDatums(
					"postgres",                   // CONSTRAINT_CATALOG
					schema.Name.O,                // CONSTRAINT_SCHEMA
					infoschema.PrimaryConstraint, // CONSTRAINT_NAME
					"postgres",                   // TABLE_CATALOG
					schema.Name.O,                // TABLE_SCHEMA
					table.Name.O,                 // TABLE_NAME
					col.Name.O,                   // COLUMN_NAME
					1,                            // ORDINAL_POSITION
					1,                            // POSITION_IN_UNIQUE_CONSTRAINT
					nil,                          // REFERENCED_TABLE_SCHEMA
					nil,                          // REFERENCED_TABLE_NAME
					nil,                          // REFERENCED_COLUMN_NAME
				)
				rows = append(rows, record)
				break
			}
		}
	}
	nameToCol := make(map[string]*model.ColumnInfo, len(table.Columns))
	for _, c := range table.Columns {
		nameToCol[c.Name.L] = c
	}
	for _, index := range table.Indices {
		var idxName string
		if index.Primary {
			idxName = infoschema.PrimaryConstraint
		} else if index.Unique {
			idxName = index.Name.O
		} else {
			// Only handle unique/primary key
			continue
		}
		for i, key := range index.Columns {
			col := nameToCol[key.Name.L]
			record := types.MakeDatums(
				"postgres",    // CONSTRAINT_CATALOG
				schema.Name.O, // CONSTRAINT_SCHEMA
				idxName,       // CONSTRAINT_NAME
				"postgres",    // TABLE_CATALOG
				schema.Name.O, // TABLE_SCHEMA
				table.Name.O,  // TABLE_NAME
				col.Name.O,    // COLUMN_NAME
				i+1,           // ORDINAL_POSITION,
				nil,           // POSITION_IN_UNIQUE_CONSTRAINT
				nil,           // REFERENCED_TABLE_SCHEMA
				nil,           // REFERENCED_TABLE_NAME
				nil,           // REFERENCED_COLUMN_NAME
			)
			rows = append(rows, record)
		}
	}
	for _, fk := range table.ForeignKeys {
		fkRefCol := ""
		if len(fk.RefCols) > 0 {
			fkRefCol = fk.RefCols[0].O
		}
		for i, key := range fk.Cols {
			col := nameToCol[key.L]
			record := types.MakeDatums(
				"postgres",    // CONSTRAINT_CATALOG
				schema.Name.O, // CONSTRAINT_SCHEMA
				fk.Name.O,     // CONSTRAINT_NAME
				"postgres",    // TABLE_CATALOG
				schema.Name.O, // TABLE_SCHEMA
				table.Name.O,  // TABLE_NAME
				col.Name.O,    // COLUMN_NAME
				i+1,           // ORDINAL_POSITION,
				1,             // POSITION_IN_UNIQUE_CONSTRAINT
				schema.Name.O, // REFERENCED_TABLE_SCHEMA
				fk.RefTable.O, // REFERENCED_TABLE_NAME
				fkRefCol,      // REFERENCED_COLUMN_NAME
			)
			rows = append(rows, record)
		}
	}
	return rows
}
