package infoschema

import (
	"fmt"
	"github.com/pingcap/tidb/meta/autoid"
	"github.com/pingcap/tidb/parser/model"
	"github.com/pingcap/tidb/parser/mysql"
	"github.com/pingcap/tidb/util"
)

const (
	// TablePgInformationsSchemaCatalogName is a table that always contains one row and one column containing the name of the current database (current catalog, in SQL terminology).
	// https://www.postgresql.org/docs/13/infoschema-information-schema-catalog-name.html
	TablePgInformationsSchemaCatalogName = "information_schema_catalog_name"
	// TablePgAdministrableRoleAuthorizations identifies all roles that the current user has the admin option for.
	//https://www.postgresql.org/docs/13/infoschema-administrable-role-authorizations.html
	TablePgAdministrableRoleAuthorizations = "administrable_role_authorizations"
	// TablePgApplicableRole identifies all roles whose privileges the current user can use.
	//https://www.postgresql.org/docs/13/infoschema-applicable-roles.html
	TablePgApplicableRole = "applicable_roles"
	// TablePgAttributes contains information about the attributes of composite data types defined in the database.
	// https://www.postgresql.org/docs/13/infoschema-attributes.html
	TablePgAttributes = "attributes"
	// TablePgCharacterSets identifies the character sets available in the current database.
	// https://www.postgresql.org/docs/13/infoschema-character-sets.html
	TablePgCharacterSets = "character_sets"
	// TablePgCheckConstraintRoutineUsage identifies routines (functions and procedures) that are used by a check constraint.
	// https://www.postgresql.org/docs/13/infoschema-check-constraint-routine-usage.html
	TablePgCheckConstraintRoutineUsage = "check_constraint_routine_usage"
	// TablePgCheckConstraints contains all check constraints, either defined on a table or on a domain, that are owned by a currently enabled role.
	// https://www.postgresql.org/docs/13/infoschema-check-constraints.html
	TablePgCheckConstraints = "check_constraints"
	// TablePgCollations contains the collations available in the current database.
	// https://www.postgresql.org/docs/13/infoschema-collations.html
	TablePgCollations = "collations"
	// TablePgCollationCharacterSetApplicability  identifies which character set the available collations are applicable to.
	// https://www.postgresql.org/docs/13/infoschema-collation-character-set-applicab.html
	TablePgCollationCharacterSetApplicability = "collation_character_set_applicability"
	// TablePgColumnColumnUsage identifies all generated columns that depend on another base column in the same table.
	// https://www.postgresql.org/docs/13/infoschema-column-column-usage.html
	TablePgColumnColumnUsage = "column_column_usage"
	// TablePgColumnDomainUsage identifies all columns (of a table or a view) that make use of some domain defined in the current database and owned by a currently enabled role.
	// https://www.postgresql.org/docs/13/infoschema-column-domain-usage.html
	TablePgColumnDomainUsage = "column_domain_usage"
	// TablePgColumnOptions contains all the options defined for foreign table columns in the current database.
	// https://www.postgresql.org/docs/13/infoschema-column-options.html
	TablePgColumnOptions = "column_options"
	// TablePgColumnPrivileges identifies all privileges granted on columns to a currently enabled role or by a currently enabled role.
	// https://www.postgresql.org/docs/13/infoschema-column-privileges.html
	TablePgColumnPrivileges = "column_privileges"
	// TablePgColumnUdtUsage identifies all columns that use data types owned by a currently enabled role.
	// https://www.postgresql.org/docs/13/infoschema-column-udt-usage.html
	TablePgColumnUdtUsage = "column_udt_usage"
	// TablePgColumns https://www.postgresql.org/docs/13/infoschema-columns.html
	// https://www.postgresql.org/docs/13/infoschema-columns.html
	TablePgColumns = "columns"
	// TablePgConstraintColumnUsage  identifies all columns in the current database that are used by some constraint.
	// https://www.postgresql.org/docs/13/infoschema-constraint-column-usage.html
	TablePgConstraintColumnUsage = "constraint_column_usage"
	// TablePgConstraintTableUsage identifies all tables in the current database that are used by some constraint and are owned by a currently enabled role.
	// https://www.postgresql.org/docs/13/infoschema-constraint-table-usage.html
	TablePgConstraintTableUsage = "constraint_table_usage"
	// TablePgDataTypePrivileges identifies all data type descriptors that the current user has access to, by way of being the owner of the described object or having some privilege for it.
	// https://www.postgresql.org/docs/13/infoschema-data-type-privileges.html
	TablePgDataTypePrivileges = "data_type_privileges"
	// TablePgDomainConstraints contains all constraints belonging to domains defined in the current database.
	// https://www.postgresql.org/docs/13/infoschema-domain-constraints.html
	TablePgDomainConstraints = "domain_constraints"
	// TablePgDomainUdtUsage identifies all domains that are based on data types owned by a currently enabled role.
	// https://www.postgresql.org/docs/13/infoschema-domain-udt-usage.html
	TablePgDomainUdtUsage = "domain_udt_usage"
	// TablePgDomains contains all domains defined in the current database.
	// https://www.postgresql.org/docs/13/infoschema-domains.html
	TablePgDomains = "domains"
	// TablePgElementTypes contains the data type descriptors of the elements of arrays.
	// https://www.postgresql.org/docs/13/infoschema-element-types.html
	TablePgElementTypes = "element_types"
	// TablePgEnabledRoles identifies the currently “enabled roles”.
	// https://www.postgresql.org/docs/13/infoschema-enabled-roles.html
	TablePgEnabledRoles = "enabled_roles"
	// TablePgForeignDataWrapperOptions contains all the options defined for foreign-data wrappers in the current database.
	// https://www.postgresql.org/docs/13/infoschema-foreign-data-wrapper-options.html
	TablePgForeignDataWrapperOptions = "foreign_data_wrapper_options"
	// TablePgForeignDataWrappers contains all foreign-data wrappers defined in the current database.
	// https://www.postgresql.org/docs/13/infoschema-foreign-data-wrappers.html
	TablePgForeignDataWrappers = "foreign_data_wrappers"
	// TablePgForeignServerOptions contains all the options defined for foreign servers in the current database.
	// https://www.postgresql.org/docs/13/infoschema-foreign-server-options.html
	TablePgForeignServerOptions = "foreign_server_options"
	// TablePgForeignServers contains all foreign servers defined in the current database.
	// https://www.postgresql.org/docs/13/infoschema-foreign-servers.html
	TablePgForeignServers = "foreign_servers"
	// TablePgForeignTableOptions contains all the options defined for foreign tables in the current database.
	// https://www.postgresql.org/docs/13/infoschema-foreign-table-options.html
	TablePgForeignTableOptions = "foreign_table_options"
	// TablePgForeignTales contains all foreign tables defined in the current database.
	// https://www.postgresql.org/docs/13/infoschema-foreign-tables.html
	TablePgForeignTales = "foreign_tables"
	// TablePgKeyColumnUsage  identifies all columns in the current database that are restricted by some unique, primary key, or foreign key constraint.
	// https://www.postgresql.org/docs/13/infoschema-key-column-usage.html
	TablePgKeyColumnUsage = "key_column_usage"
	// TablePgParameters  contains information about the parameters (arguments) of all functions in the current database.
	// https://www.postgresql.org/docs/13/infoschema-parameters.html
	TablePgParameters = "parameters"
	// TablePgReferentialConstraints contains all referential (foreign key) constraints in the current database.
	// https://www.postgresql.org/docs/13/infoschema-referential-constraints.html
	TablePgReferentialConstraints = "referential_constraints"
	// TablePgRoleColumnGrants identifies all privileges granted on columns where the grantor or grantee is a currently enabled role.
	// https://www.postgresql.org/docs/13/infoschema-role-column-grants.html
	TablePgRoleColumnGrants = "role_column_grants"
	// TablePgRoleRoutineGrants identifies all privileges granted on functions where the grantor or grantee is a currently enabled role.
	// https://www.postgresql.org/docs/13/infoschema-role-routine-grants.html
	TablePgRoleRoutineGrants = "role_routine_grants"
	// TablePgRoleTableGrants identifies all privileges granted on tables or views where the grantor or grantee is a currently enabled role.
	// https://www.postgresql.org/docs/13/infoschema-role-table-grants.html
	TablePgRoleTableGrants = "role_table_grants"
	// TablePgRoleUdtGrants is intended to identify USAGE privileges granted on user-defined types where the grantor or grantee is a currently enabled role.
	// https://www.postgresql.org/docs/13/infoschema-role-udt-grants.html
	TablePgRoleUdtGrants = "role_udt_grants"
	// TablePgRoleUsageGrants identifies USAGE privileges granted on various kinds of objects where the grantor or grantee is a currently enabled role.
	// https://www.postgresql.org/docs/13/infoschema-role-usage-grants.html
	TablePgRoleUsageGrants = "role_usage_grants"
	// TablePgRoutinePrivileges identifies all privileges granted on functions to a currently enabled role or by a currently enabled role.
	// https://www.postgresql.org/docs/13/infoschema-routine-privileges.html
	TablePgRoutinePrivileges = "routine_privileges"
	// TablePgRoutines contains all functions and procedures in the current database.
	// https://www.postgresql.org/docs/13/infoschema-routines.html
	TablePgRoutines = "routines"
	// TablePgSchemata contains all schemas in the current database that the current user has access to (by way of being the owner or having some privilege).
	// https://www.postgresql.org/docs/13/infoschema-schemata.html
	TablePgSchemata = "schemata"
	// TablePgSequences contains all sequences defined in the current database.
	// https://www.postgresql.org/docs/13/infoschema-sequences.html
	TablePgSequences = "sequences"
	// TablePgSQLFeatures contains information about which formal features defined in the SQL standard are supported by PostgreSQL.
	// https://www.postgresql.org/docs/13/infoschema-sql-features.html
	TablePgSQLFeatures = "sql_features"
	// TablePgSQLImplementationInfo contains information about various aspects that are left implementation-defined by the SQL standard.
	// https://www.postgresql.org/docs/13/infoschema-sql-implementation-info.html
	TablePgSQLImplementationInfo = "sql_implementation_info"
	// TablePgSQLParts contains information about which of the several parts of the SQL standard are supported by PostgreSQL.
	// https://www.postgresql.org/docs/13/infoschema-sql-parts.html
	TablePgSQLParts = "sql_parts"
	// TablePgSQLSizing contains information about various size limits and maximum values in PostgreSQL.
	// https://www.postgresql.org/docs/13/infoschema-sql-sizing.html
	TablePgSQLSizing = "sql_sizing"
	// TablePgTableConstraints contains all constraints belonging to tables that the current user owns or has some privilege other than SELECT on.
	// https://www.postgresql.org/docs/13/infoschema-table-constraints.html
	TablePgTableConstraints = "table_constraints"
	// TablePgTablePrivileges identifies all privileges granted on tables or views to a currently enabled role or by a currently enabled role.
	// https://www.postgresql.org/docs/13/infoschema-table-privileges.html
	TablePgTablePrivileges = "table_privileges"
	// TablePgTables contains all tables and views defined in the current database.
	// https://www.postgresql.org/docs/13/infoschema-tables.html
	TablePgTables = "TABLES"
	// TablePgTransforms contains information about the transforms defined in the current database.
	// https://www.postgresql.org/docs/13/infoschema-transforms.html
	TablePgTransforms = "transforms"
	// TablePgTriggeredUpdateColumns For triggers in the current database that specify a column list (like UPDATE OF column1, column2), the view TableTriggeredUpdateColumns identifies these columns.
	// https://www.postgresql.org/docs/13/infoschema-triggered-update-columns.html
	TablePgTriggeredUpdateColumns = "triggered_update_columns"
	// TablePgTriggers contains all triggers defined in the current database on tables and views that the current user owns or has some privilege other than SELECT on.
	// https://www.postgresql.org/docs/13/infoschema-triggers.html
	TablePgTriggers = "triggers"
	// TablePgUdtPrivileges identifies USAGE privileges granted on user-defined types to a currently enabled role or by a currently enabled role.
	// https://www.postgresql.org/docs/13/infoschema-udt-privileges.html
	TablePgUdtPrivileges = "udt_privileges"
	// TablePgUsagePrivileges  identifies USAGE privileges granted on various kinds of objects to a currently enabled role or by a currently enabled role.
	// https://www.postgresql.org/docs/13/infoschema-usage-privileges.html
	TablePgUsagePrivileges = "usage_privileges"
	// TablePgUserDefinedTypes currently contains all composite types defined in the current database.
	// https://www.postgresql.org/docs/13/infoschema-user-defined-types.html
	TablePgUserDefinedTypes = "user_defined_types"
	// TablePgUserMappingOptions contains all the options defined for user mappings in the current database.
	// https://www.postgresql.org/docs/13/infoschema-user-mapping-options.html
	TablePgUserMappingOptions = "user_mapping_options"
	// TablePgUserMappings contains all user mappings defined in the current database.
	// https://www.postgresql.org/docs/13/infoschema-user-mappings.html
	TablePgUserMappings = "user_mapping"
	// TablePgViewColumnUsage  identifies all columns that are used in the query expression of a view (the SELECT statement that defines the view).
	// https://www.postgresql.org/docs/13/infoschema-view-column-usage.html
	TablePgViewColumnUsage = "view_column_usage"
	// TablePgViewRoutineUsage  identifies all routines (functions and procedures) that are used in the query expression of a view (the SELECT statement that defines the view).
	// https://www.postgresql.org/docs/13/infoschema-view-routine-usage.html
	TablePgViewRoutineUsage = "view_routine_usage"
	// TablePgViewTableUsage identifies all tables that are used in the query expression of a view
	// https://www.postgresql.org/docs/13/infoschema-view-table-usage.html
	TablePgViewTableUsage = "view_table_usage"
	// TablePgViews contains all views defined in the current database.
	// https://www.postgresql.org/docs/13/infoschema-views.html
	TablePgViews = "views"
)

var pgTableIDMap = map[string]int64{
	TablePgInformationsSchemaCatalogName:      autoid.InformationSchemaDBID + 80,
	TablePgAdministrableRoleAuthorizations:    autoid.InformationSchemaDBID + 81,
	TablePgApplicableRole:                     autoid.InformationSchemaDBID + 82,
	TablePgAttributes:                         autoid.InformationSchemaDBID + 83,
	TablePgCharacterSets:                      autoid.InformationSchemaDBID + 84,
	TablePgCheckConstraintRoutineUsage:        autoid.InformationSchemaDBID + 85,
	TablePgCheckConstraints:                   autoid.InformationSchemaDBID + 86,
	TablePgCollationCharacterSetApplicability: autoid.InformationSchemaDBID + 87,
	TablePgCollations:                         autoid.InformationSchemaDBID + 88,
	TablePgColumnColumnUsage:                  autoid.InformationSchemaDBID + 89,
	TablePgColumnDomainUsage:                  autoid.InformationSchemaDBID + 90,
	TablePgColumnOptions:                      autoid.InformationSchemaDBID + 91,
	TablePgColumnPrivileges:                   autoid.InformationSchemaDBID + 92,
	TablePgColumnUdtUsage:                     autoid.InformationSchemaDBID + 93,
	TablePgColumns:                            autoid.InformationSchemaDBID + 94,
	TablePgConstraintColumnUsage:              autoid.InformationSchemaDBID + 95,
	TablePgConstraintTableUsage:               autoid.InformationSchemaDBID + 96,
	TablePgDataTypePrivileges:                 autoid.InformationSchemaDBID + 97,
	TablePgDomainConstraints:                  autoid.InformationSchemaDBID + 98,
	TablePgDomainUdtUsage:                     autoid.InformationSchemaDBID + 99,
	TablePgDomains:                            autoid.InformationSchemaDBID + 100,
	TablePgElementTypes:                       autoid.InformationSchemaDBID + 101,
	TablePgEnabledRoles:                       autoid.InformationSchemaDBID + 102,
	TablePgForeignDataWrapperOptions:          autoid.InformationSchemaDBID + 103,
	TablePgForeignDataWrappers:                autoid.InformationSchemaDBID + 104,
	TablePgForeignServerOptions:               autoid.InformationSchemaDBID + 105,
	TablePgForeignServers:                     autoid.InformationSchemaDBID + 106,
	TablePgForeignTableOptions:                autoid.InformationSchemaDBID + 107,
	TablePgForeignTales:                       autoid.InformationSchemaDBID + 108,
	TablePgKeyColumnUsage:                     autoid.InformationSchemaDBID + 109,
	TablePgParameters:                         autoid.InformationSchemaDBID + 110,
	TablePgReferentialConstraints:             autoid.InformationSchemaDBID + 111,
	TablePgRoleColumnGrants:                   autoid.InformationSchemaDBID + 112,
	TablePgRoleRoutineGrants:                  autoid.InformationSchemaDBID + 113,
	TablePgRoleTableGrants:                    autoid.InformationSchemaDBID + 114,
	TablePgRoleUdtGrants:                      autoid.InformationSchemaDBID + 115,
	TablePgRoleUsageGrants:                    autoid.InformationSchemaDBID + 116,
	TablePgRoutinePrivileges:                  autoid.InformationSchemaDBID + 117,
	TablePgRoutines:                           autoid.InformationSchemaDBID + 118,
	TablePgSchemata:                           autoid.InformationSchemaDBID + 119,
	TablePgSequences:                          autoid.InformationSchemaDBID + 120,
	TablePgSQLFeatures:                        autoid.InformationSchemaDBID + 121,
	TablePgSQLImplementationInfo:              autoid.InformationSchemaDBID + 122,
	TablePgSQLParts:                           autoid.InformationSchemaDBID + 123,
	TablePgSQLSizing:                          autoid.InformationSchemaDBID + 124,
	TablePgTableConstraints:                   autoid.InformationSchemaDBID + 125,
	TablePgTablePrivileges:                    autoid.InformationSchemaDBID + 126,
	TablePgTables:                             autoid.InformationSchemaDBID + 127,
	TablePgTransforms:                         autoid.InformationSchemaDBID + 128,
	TablePgTriggeredUpdateColumns:             autoid.InformationSchemaDBID + 129,
	TablePgTriggers:                           autoid.InformationSchemaDBID + 130,
	TablePgUdtPrivileges:                      autoid.InformationSchemaDBID + 131,
	TablePgUsagePrivileges:                    autoid.InformationSchemaDBID + 132,
	TablePgUserDefinedTypes:                   autoid.InformationSchemaDBID + 133,
	TablePgUserMappingOptions:                 autoid.InformationSchemaDBID + 134,
	TablePgUserMappings:                       autoid.InformationSchemaDBID + 135,
	TablePgViewColumnUsage:                    autoid.InformationSchemaDBID + 136,
	TablePgViewRoutineUsage:                   autoid.InformationSchemaDBID + 137,
	TablePgViewTableUsage:                     autoid.InformationSchemaDBID + 138,
	TablePgViews:                              autoid.InformationSchemaDBID + 139,
}

// pgTableInformationSchemaCatalogName is table information_schema_catalog_name columns
// https://www.postgresql.org/docs/13/infoschema-information-schema-catalog-name.html
var pgTableInformationSchemaCatalogNameCols = []columnInfo{
	{name: "catalog_name", tp: mysql.TypeVarchar, size: 32},
}

// pgTableAdministrableRoleAuthorizationsCols is table administrable_role_authorizations columns
// https://www.postgresql.org/docs/13/infoschema-administrable-role-authorizations.html
var pgTableAdministrableRoleAuthorizationsCols = []columnInfo{
	{name: "grantee", tp: mysql.TypeVarchar, size: 64},
	{name: "role_name", tp: mysql.TypeVarchar, size: 64},
	{name: "is_grantable", tp: mysql.TypeVarchar, size: 32},
}

// pgTableApplicableRolesCols is table application_roles columns
// https://www.postgresql.org/docs/13/infoschema-applicable-roles.html
var pgTableApplicableRolesCols = []columnInfo{
	{name: "grantee", tp: mysql.TypeVarchar, size: 64},
	{name: "role_name", tp: mysql.TypeVarchar, size: 64},
	{name: "is_grantable", tp: mysql.TypeVarchar, size: 32},
}

// pgTableAttributesCols is table attribute columns
// https://www.postgresql.org/docs/13/infoschema-attributes.html
var pgTableAttributesCols = []columnInfo{
	{name: "udt_catalog", tp: mysql.TypeVarchar, size: 64},
	{name: "udt_schema", tp: mysql.TypeVarchar, size: 64},
	{name: "udt_name", tp: mysql.TypeVarchar, size: 64},
	{name: "attribute_name", tp: mysql.TypeVarchar, size: 64},
	{name: "ordinal_position", tp: mysql.TypeVarchar, size: 64},
	{name: "attribute_default", tp: mysql.TypeVarchar, size: 64},
	{name: "is_nullable", tp: mysql.TypeVarchar, size: 64},
	{name: "data_type", tp: mysql.TypeVarchar, size: 64},
	{name: "character_maximum_length", tp: mysql.TypeVarchar, size: 64},
	{name: "character_octet_length", tp: mysql.TypeVarchar, size: 64},
	{name: "character_set_catalog", tp: mysql.TypeVarchar, size: 64},
	{name: "character_set_schema", tp: mysql.TypeVarchar, size: 64},
	{name: "character_set_name", tp: mysql.TypeVarchar, size: 64},
	{name: "collation_catalog", tp: mysql.TypeVarchar, size: 64},
	{name: "collation_schema", tp: mysql.TypeVarchar, size: 64},
	{name: "collation_name", tp: mysql.TypeVarchar, size: 64},
	{name: "numeric_precision", tp: mysql.TypeVarchar, size: 64},
	{name: "numeric_precision_radix", tp: mysql.TypeVarchar, size: 64},
	{name: "numeric_scale", tp: mysql.TypeVarchar, size: 64},
	{name: "datetime_precision", tp: mysql.TypeVarchar, size: 64},
	{name: "interval_type", tp: mysql.TypeVarchar, size: 64},
	{name: "interval_precision", tp: mysql.TypeVarchar, size: 64},
	{name: "attribute_udt_catalog", tp: mysql.TypeVarchar, size: 64},
	{name: "attribute_udt_schema", tp: mysql.TypeVarchar, size: 64},
	{name: "attribute_udt_name", tp: mysql.TypeVarchar, size: 64},
	{name: "scope_catalog", tp: mysql.TypeVarchar, size: 64},
	{name: "scope_schema", tp: mysql.TypeVarchar, size: 64},
	{name: "scope_name", tp: mysql.TypeVarchar, size: 64},
	{name: "maximum_cardinality", tp: mysql.TypeVarchar, size: 64},
	{name: "dtd_identifier", tp: mysql.TypeVarchar, size: 64},
	{name: "is_derived_reference_attribute", tp: mysql.TypeVarchar, size: 64},
}

// pgTableCharacterSetsCols is table character_sets columns
// https://www.postgresql.org/docs/13/infoschema-character-sets.html
var pgTableCharacterSetsCols = []columnInfo{
	{name: "character_set_catalog", tp: mysql.TypeVarchar, size: 64},
	{name: "character_set_schema", tp: mysql.TypeVarchar, size: 64},
	{name: "character_set_name", tp: mysql.TypeVarchar, size: 32},
	{name: "character_repertoire", tp: mysql.TypeVarchar, size: 64},
	{name: "form_of_use", tp: mysql.TypeVarchar, size: 64},
	{name: "default_collate_catalog", tp: mysql.TypeVarchar, size: 64},
	{name: "default_collate_schema", tp: mysql.TypeVarchar, size: 64},
	{name: "default_collate_name", tp: mysql.TypeVarchar, size: 32},
	{name: "DESCRIPTION", tp: mysql.TypeVarchar, size: 60, comment: "TiDB table cols"},
	{name: "MAXLEN", tp: mysql.TypeLonglong, size: 3, comment: "TiDB table cols"},
}

// pgTableCheckConstraintRoutineUsageCols is table check_constraint_routine_usage columns
// https://www.postgresql.org/docs/13/infoschema-check-constraint-routine-usage.html
var pgTableCheckConstraintRoutineUsageCols = []columnInfo{
	{name: "constraint_catalog", tp: mysql.TypeVarchar, size: 64},
	{name: "constraint_schema", tp: mysql.TypeVarchar, size: 64},
	{name: "constraint_name", tp: mysql.TypeVarchar, size: 64},
	{name: "specific_catalog", tp: mysql.TypeVarchar, size: 64},
	{name: "specific_schema", tp: mysql.TypeVarchar, size: 64},
	{name: "specific_name", tp: mysql.TypeVarchar, size: 64},
}

// pgTableCheckConstraintsCols is table check_constraints columns
// https://www.postgresql.org/docs/13/infoschema-check-constraints.html
var pgTableCheckConstraintsCols = []columnInfo{
	{name: "constraint_catalog", tp: mysql.TypeVarchar, size: 64},
	{name: "constraint_schema", tp: mysql.TypeVarchar, size: 128},
	{name: "constraint_name", tp: mysql.TypeVarchar, size: 64},
	{name: "check_clause", tp: mysql.TypeVarchar, size: 64},
}

// pgTableCollationsCols is table collations columns
// https://www.postgresql.org/docs/13/infoschema-collations.html
var pgTableCollationsCols = []columnInfo{
	{name: "collation_catalog", tp: mysql.TypeVarchar, size: 64},
	{name: "collation_schema", tp: mysql.TypeVarchar, size: 64},
	{name: "collation_name", tp: mysql.TypeVarchar, size: 64},
	{name: "pad_attribute", tp: mysql.TypeVarchar, size: 64},
	{name: "CHARACTER_SET_NAME", tp: mysql.TypeVarchar, size: 32, comment: "TiDB table cols"},
	{name: "ID", tp: mysql.TypeLonglong, size: 11, comment: "TiDB table cols"},
	{name: "IS_DEFAULT", tp: mysql.TypeVarchar, size: 3, comment: "TiDB table cols"},
	{name: "IS_COMPILED", tp: mysql.TypeVarchar, size: 3, comment: "TiDB table cols"},
	{name: "SORTLEN", tp: mysql.TypeLonglong, size: 3, comment: "TiDB table cols"},
}

// pgTableCollationCharacterSetApplicabilityCols is table collation_character_set_applicability columns
// https://www.postgresql.org/docs/13/infoschema-collation-character-set-applicab.html
var pgTableCollationCharacterSetApplicabilityCols = []columnInfo{
	{name: "collation_catalog", tp: mysql.TypeVarchar, size: 64},
	{name: "collation_schema", tp: mysql.TypeVarchar, size: 64},
	{name: "collation_name", tp: mysql.TypeVarchar, size: 32, flag: mysql.NotNullFlag},
	{name: "character_set_catalog", tp: mysql.TypeVarchar, size: 64},
	{name: "character_set_schema", tp: mysql.TypeVarchar, size: 64},
	{name: "character_set_name", tp: mysql.TypeVarchar, size: 32, flag: mysql.NotNullFlag},
}

// pgTableColumnColumnUsageCols is table column_column_usage columns
// https://www.postgresql.org/docs/13/infoschema-column-column-usage.html
var pgTableColumnColumnUsageCols = []columnInfo{
	{name: "table_catalog", tp: mysql.TypeVarchar, size: 64},
	{name: "table_schema", tp: mysql.TypeVarchar, size: 64},
	{name: "table_name", tp: mysql.TypeVarchar, size: 64},
	{name: "column_name", tp: mysql.TypeVarchar, size: 64},
	{name: "dependent_column", tp: mysql.TypeVarchar, size: 64},
}

// pgTableColumnDomainUsageCols is table column_domain_usage columns
// https://www.postgresql.org/docs/13/infoschema-column-domain-usage.html
var pgTableColumnDomainUsageCols = []columnInfo{
	{name: "domain_catalog", tp: mysql.TypeVarchar, size: 64},
	{name: "domain_schema", tp: mysql.TypeVarchar, size: 64},
	{name: "domain_name", tp: mysql.TypeVarchar, size: 64},
	{name: "table_catalog", tp: mysql.TypeVarchar, size: 64},
	{name: "table_schema", tp: mysql.TypeVarchar, size: 64},
	{name: "table_name", tp: mysql.TypeVarchar, size: 64},
	{name: "column_name", tp: mysql.TypeVarchar, size: 64},
}

// pgTableColumnOptionsCols is table column_options columns
// https://www.postgresql.org/docs/13/infoschema-column-options.html
var pgTableColumnOptionsCols = []columnInfo{
	{name: "table_catalog", tp: mysql.TypeVarchar, size: 64},
	{name: "table_schema", tp: mysql.TypeVarchar, size: 64},
	{name: "table_name", tp: mysql.TypeVarchar, size: 64},
	{name: "column_name", tp: mysql.TypeVarchar, size: 64},
	{name: "option_name", tp: mysql.TypeVarchar, size: 64},
	{name: "option_value", tp: mysql.TypeVarchar, size: 64},
}

// pgTableColumnPrivilegesCols is table column_privileges columns
// https://www.postgresql.org/docs/13/infoschema-column-privileges.html
var pgTableColumnPrivilegesCols = []columnInfo{
	{name: "grantor", tp: mysql.TypeVarchar, size: 64},
	{name: "grantee", tp: mysql.TypeVarchar, size: 64},
	{name: "table_catalog", tp: mysql.TypeVarchar, size: 64},
	{name: "table_schema", tp: mysql.TypeVarchar, size: 64},
	{name: "table_name", tp: mysql.TypeVarchar, size: 64},
	{name: "column_name", tp: mysql.TypeVarchar, size: 64},
	{name: "privilege_type", tp: mysql.TypeVarchar, size: 64},
	{name: "is_grantable", tp: mysql.TypeVarchar, size: 64},
}

// pgTableColumnUdtUsageCols is table column_udt_usage columns
// https://www.postgresql.org/docs/13/infoschema-column-udt-usage.html
var pgTableColumnUdtUsageCols = []columnInfo{
	{name: "udt_catalog", tp: mysql.TypeVarchar, size: 64},
	{name: "udt_schema", tp: mysql.TypeVarchar, size: 64},
	{name: "udt_name", tp: mysql.TypeVarchar, size: 64},
	{name: "table_catalog", tp: mysql.TypeVarchar, size: 64},
	{name: "table_schema", tp: mysql.TypeVarchar, size: 64},
	{name: "table_name", tp: mysql.TypeVarchar, size: 64},
	{name: "column_name", tp: mysql.TypeVarchar, size: 64},
}

// pgTableColumnsCols is table columns columns
// https://www.postgresql.org/docs/13/infoschema-columns.html
var pgTableColumnsCols = []columnInfo{
	{name: "table_catalog", tp: mysql.TypeVarchar, size: 512},
	{name: "table_schema", tp: mysql.TypeVarchar, size: 64},
	{name: "table_name", tp: mysql.TypeVarchar, size: 64},
	{name: "column_name", tp: mysql.TypeVarchar, size: 64},
	{name: "ordinal_position", tp: mysql.TypeLonglong, size: 64},
	{name: "column_default", tp: mysql.TypeBlob, size: 196606},
	{name: "is_nullable", tp: mysql.TypeVarchar, size: 3},
	{name: "data_type", tp: mysql.TypeVarchar, size: 64},
	{name: "character_maximum_length", tp: mysql.TypeLonglong, size: 21},
	{name: "character_octet_length", tp: mysql.TypeLonglong, size: 21},
	{name: "numeric_precision", tp: mysql.TypeLonglong, size: 21},
	{name: "numeric_precision_radix", tp: mysql.TypeLonglong, size: 21},
	{name: "numeric_scale", tp: mysql.TypeLonglong, size: 21},
	{name: "datetime_precision", tp: mysql.TypeLonglong, size: 21},
	{name: "interval_type", tp: mysql.TypeVarchar, size: 64},
	{name: "interval_precision", tp: mysql.TypeInt24, size: 64},
	{name: "character_set_catalog", tp: mysql.TypeVarchar, size: 64},
	{name: "character_set_schema", tp: mysql.TypeVarchar, size: 64},
	{name: "character_set_name", tp: mysql.TypeVarchar, size: 64},
	{name: "collation_catalog", tp: mysql.TypeVarchar, size: 64},
	{name: "collation_schema", tp: mysql.TypeVarchar, size: 64},
	{name: "collation_name", tp: mysql.TypeVarchar, size: 32},
	{name: "domain_catalog", tp: mysql.TypeVarchar, size: 64},
	{name: "domain_schema", tp: mysql.TypeVarchar, size: 64},
	{name: "domain_name", tp: mysql.TypeVarchar, size: 64},
	{name: "udt_catalog", tp: mysql.TypeVarchar, size: 64},
	{name: "udt_schema", tp: mysql.TypeVarchar, size: 64},
	{name: "udt_name", tp: mysql.TypeVarchar, size: 64},
	{name: "scope_catalog", tp: mysql.TypeVarchar, size: 64},
	{name: "scope_schema", tp: mysql.TypeVarchar, size: 64},
	{name: "scope_name", tp: mysql.TypeVarchar, size: 64},
	{name: "maximum_cardinality", tp: mysql.TypeInt24, size: 64},
	{name: "dtd_identifier", tp: mysql.TypeVarchar, size: 64},
	{name: "is_self_referencing", tp: mysql.TypeVarchar, size: 64},
	{name: "is_identity", tp: mysql.TypeVarchar, size: 64},
	{name: "identity_generation", tp: mysql.TypeVarchar, size: 64},
	{name: "identity_start", tp: mysql.TypeVarchar, size: 64},
	{name: "identity_increment", tp: mysql.TypeVarchar, size: 64},
	{name: "identity_maximum", tp: mysql.TypeVarchar, size: 64},
	{name: "identity_minimum", tp: mysql.TypeVarchar, size: 64},
	{name: "identity_cycle", tp: mysql.TypeVarchar, size: 64},
	{name: "is_generated", tp: mysql.TypeVarchar, size: 64},
	{name: "generation_expression", tp: mysql.TypeBlob, size: 589779, flag: mysql.NotNullFlag},
	{name: "is_updatable", tp: mysql.TypeVarchar, size: 64},
	{name: "COLUMN_TYPE", tp: mysql.TypeBlob, size: 196606, comment: "TiDB Table Cols"},
	{name: "COLUMN_KEY", tp: mysql.TypeVarchar, size: 3, comment: "TiDB Table Cols"},
	{name: "EXTRA", tp: mysql.TypeVarchar, size: 30, comment: "TiDB Table Cols"},
	{name: "PRIVILEGES", tp: mysql.TypeVarchar, size: 80, comment: "TiDB Table Cols"},
	{name: "COLUMN_COMMENT", tp: mysql.TypeVarchar, size: 1024, comment: "TiDB Table Cols"},
}

// pgTableConstraintColumnUsageCols is table constraint_column_usage columns
// https://www.postgresql.org/docs/13/infoschema-constraint-column-usage.html
var pgTableConstraintColumnUsageCols = []columnInfo{
	{name: "table_catalog", tp: mysql.TypeVarchar, size: 64},
	{name: "table_schema", tp: mysql.TypeVarchar, size: 64},
	{name: "table_name", tp: mysql.TypeVarchar, size: 64},
	{name: "column_name", tp: mysql.TypeVarchar, size: 64},
	{name: "constraint_catalog", tp: mysql.TypeVarchar, size: 64},
	{name: "constraint_schema", tp: mysql.TypeVarchar, size: 64},
	{name: "constraint_name", tp: mysql.TypeVarchar, size: 64},
}

// pgTableConstraintTableUsageCols is table constraint_tale_usage columns
// https://www.postgresql.org/docs/13/infoschema-constraint-table-usage.html
var pgTableConstraintTableUsageCols = []columnInfo{
	{name: "table_catalog", tp: mysql.TypeVarchar, size: 64},
	{name: "table_schema", tp: mysql.TypeVarchar, size: 64},
	{name: "table_name", tp: mysql.TypeVarchar, size: 64},
	{name: "constraint_catalog", tp: mysql.TypeVarchar, size: 64},
	{name: "constraint_schema", tp: mysql.TypeVarchar, size: 64},
	{name: "constraint_name", tp: mysql.TypeVarchar, size: 64},
}

// pgTableDataTypePrivilegesCols is table data_type_privileges columns
// https://www.postgresql.org/docs/13/infoschema-data-type-privileges.html
var pgTableDataTypePrivilegesCols = []columnInfo{
	{name: "object_catalog", tp: mysql.TypeVarchar, size: 64},
	{name: "object_schema", tp: mysql.TypeVarchar, size: 64},
	{name: "object_name", tp: mysql.TypeVarchar, size: 64},
	{name: "object_type", tp: mysql.TypeVarchar, size: 64},
	{name: "dtd_identifier", tp: mysql.TypeVarchar, size: 64},
}

// pgTableDomainConstraintsCols is table domain_constraints columns
// https://www.postgresql.org/docs/13/infoschema-domain-constraints.html
var pgTableDomainConstraintsCols = []columnInfo{
	{name: "constraint_catalog", tp: mysql.TypeVarchar, size: 64},
	{name: "constraint_schema", tp: mysql.TypeVarchar, size: 64},
	{name: "constraint_name", tp: mysql.TypeVarchar, size: 64},
	{name: "domain_catalog", tp: mysql.TypeVarchar, size: 64},
	{name: "domain_schema", tp: mysql.TypeVarchar, size: 64},
	{name: "domain_name", tp: mysql.TypeVarchar, size: 64},
	{name: "is_deferrable", tp: mysql.TypeVarchar, size: 64},
	{name: "initially_deferred", tp: mysql.TypeVarchar, size: 64},
}

// pgTableDomainUdtUsageCols is table domain_udt_usage columns
// https://www.postgresql.org/docs/13/infoschema-domain-udt-usage.html
var pgTableDomainUdtUsageCols = []columnInfo{
	{name: "udt_catalog", tp: mysql.TypeVarchar, size: 64},
	{name: "udt_schema", tp: mysql.TypeVarchar, size: 64},
	{name: "udt_name", tp: mysql.TypeVarchar, size: 64},
	{name: "domain_catalog", tp: mysql.TypeVarchar, size: 64},
	{name: "domain_schema", tp: mysql.TypeVarchar, size: 64},
	{name: "domain_name", tp: mysql.TypeVarchar, size: 64},
}

// pgTableDomainsCols is table domain columns
// https://www.postgresql.org/docs/13/infoschema-domains.html
var pgTableDomainsCols = []columnInfo{
	{name: "domain_catalog", tp: mysql.TypeVarchar, size: 64},
	{name: "domain_schema", tp: mysql.TypeVarchar, size: 64},
	{name: "domain_name", tp: mysql.TypeVarchar, size: 64},
	{name: "data_type", tp: mysql.TypeVarchar, size: 64},
	{name: "character_maximum_length", tp: mysql.TypeInt24, size: 64},
	{name: "character_octet_length", tp: mysql.TypeInt24, size: 64},
	{name: "character_set_catalog", tp: mysql.TypeVarchar, size: 64},
	{name: "character_set_schema", tp: mysql.TypeVarchar, size: 64},
	{name: "character_set_name", tp: mysql.TypeVarchar, size: 64},
	{name: "collation_catalog", tp: mysql.TypeVarchar, size: 64},
	{name: "collation_schema", tp: mysql.TypeVarchar, size: 64},
	{name: "collation_name", tp: mysql.TypeVarchar, size: 64},
	{name: "numeric_precision", tp: mysql.TypeVarchar, size: 64},
	{name: "numeric_precision_radix", tp: mysql.TypeVarchar, size: 64},
	{name: "numeric_scale", tp: mysql.TypeVarchar, size: 64},
	{name: "datetime_precision", tp: mysql.TypeVarchar, size: 64},
	{name: "interval_type", tp: mysql.TypeVarchar, size: 64},
	{name: "interval_precision", tp: mysql.TypeVarchar, size: 64},
	{name: "domain_default", tp: mysql.TypeVarchar, size: 64},
	{name: "udt_catalog", tp: mysql.TypeVarchar, size: 64},
	{name: "udt_schema", tp: mysql.TypeVarchar, size: 64},
	{name: "udt_name", tp: mysql.TypeVarchar, size: 64},
	{name: "scope_catalog", tp: mysql.TypeVarchar, size: 64},
	{name: "scope_schema", tp: mysql.TypeVarchar, size: 64},
	{name: "scope_name", tp: mysql.TypeVarchar, size: 64},
	{name: "maximum_cardinality", tp: mysql.TypeVarchar, size: 64},
	{name: "dtd_identifier", tp: mysql.TypeVarchar, size: 64},
}

// pgTableElementTypesCols is table element_types columns
// https://www.postgresql.org/docs/13/infoschema-element-types.html
var pgTableElementTypesCols = []columnInfo{
	{name: "object_catalog", tp: mysql.TypeVarchar, size: 64},
	{name: "object_schema", tp: mysql.TypeVarchar, size: 64},
	{name: "object_name", tp: mysql.TypeVarchar, size: 64},
	{name: "object_type", tp: mysql.TypeVarchar, size: 64},
	{name: "collection_type_identifier", tp: mysql.TypeVarchar, size: 64},
	{name: "data_type", tp: mysql.TypeVarchar, size: 64},
	{name: "character_maximum_length", tp: mysql.TypeInt24, size: 64},
	{name: "character_octet_length", tp: mysql.TypeInt24, size: 64},
	{name: "character_set_catalog", tp: mysql.TypeVarchar, size: 64},
	{name: "character_set_schema", tp: mysql.TypeVarchar, size: 64},
	{name: "character_set_name", tp: mysql.TypeVarchar, size: 64},
	{name: "collation_catalog", tp: mysql.TypeVarchar, size: 64},
	{name: "collation_schema", tp: mysql.TypeVarchar, size: 64},
	{name: "collation_name", tp: mysql.TypeVarchar, size: 64},
	{name: "numeric_precision", tp: mysql.TypeVarchar, size: 64},
	{name: "numeric_precision_radix", tp: mysql.TypeVarchar, size: 64},
	{name: "numeric_scale", tp: mysql.TypeVarchar, size: 64},
	{name: "datetime_precision", tp: mysql.TypeVarchar, size: 64},
	{name: "interval_type", tp: mysql.TypeVarchar, size: 64},
	{name: "interval_precision", tp: mysql.TypeVarchar, size: 64},
	{name: "domain_default", tp: mysql.TypeVarchar, size: 64},
	{name: "udt_catalog", tp: mysql.TypeVarchar, size: 64},
	{name: "udt_schema", tp: mysql.TypeVarchar, size: 64},
	{name: "udt_name", tp: mysql.TypeVarchar, size: 64},
	{name: "scope_catalog", tp: mysql.TypeVarchar, size: 64},
	{name: "scope_schema", tp: mysql.TypeVarchar, size: 64},
	{name: "scope_name", tp: mysql.TypeVarchar, size: 64},
	{name: "maximum_cardinality", tp: mysql.TypeVarchar, size: 64},
	{name: "dtd_identifier", tp: mysql.TypeVarchar, size: 64},
}

// pgTableEnabledRolesCols is table enabled_roles columns
// https://www.postgresql.org/docs/13/infoschema-enabled-roles.html
var pgTableEnabledRolesCols = []columnInfo{
	{name: "role_name", tp: mysql.TypeVarchar, size: 64},
}

// pgTableForeignDataWrapperOptionsCols is table foreign_data_wrapper_options columns
// https://www.postgresql.org/docs/13/infoschema-foreign-data-wrapper-options.html
var pgTableForeignDataWrapperOptionsCols = []columnInfo{
	{name: "foreign_data_wrapper_catalog", tp: mysql.TypeVarchar, size: 64},
	{name: "foreign_data_wrapper_name", tp: mysql.TypeVarchar, size: 64},
	{name: "option_name", tp: mysql.TypeVarchar, size: 64},
	{name: "option_value", tp: mysql.TypeVarchar, size: 64},
}

// pgTableForeignDataWrappersCols is table foreign_data_wrapper columns
// https://www.postgresql.org/docs/13/infoschema-foreign-data-wrappers.html
var pgTableForeignDataWrappersCols = []columnInfo{
	{name: "foreign_data_wrapper_catalog", tp: mysql.TypeVarchar, size: 64},
	{name: "foreign_data_wrapper_name", tp: mysql.TypeVarchar, size: 64},
	{name: "authorization_identifier", tp: mysql.TypeVarchar, size: 64},
	{name: "library_name", tp: mysql.TypeVarchar, size: 64},
	{name: "foreign_data_wrapper_language", tp: mysql.TypeVarchar, size: 64},
}

// pgTableForeignServerOptionsCols is table foreign_server_options columns
// https://www.postgresql.org/docs/13/infoschema-foreign-server-options.html
var pgTableForeignServerOptionsCols = []columnInfo{
	{name: "foreign_server_catalog", tp: mysql.TypeVarchar, size: 64},
	{name: "foreign_server_name", tp: mysql.TypeVarchar, size: 64},
	{name: "option_name", tp: mysql.TypeVarchar, size: 64},
	{name: "option_value", tp: mysql.TypeVarchar, size: 64},
}

// pgTableForeignServers is table foreign_servers columns
// https://www.postgresql.org/docs/13/infoschema-foreign-servers.html
var pgTableForeignServers = []columnInfo{
	{name: "foreign_server_catalog", tp: mysql.TypeVarchar, size: 64},
	{name: "foreign_server_name", tp: mysql.TypeVarchar, size: 64},
	{name: "foreign_data_wrapper_catalog", tp: mysql.TypeVarchar, size: 64},
	{name: "foreign_data_wrapper_name", tp: mysql.TypeVarchar, size: 64},
	{name: "foreign_server_type", tp: mysql.TypeVarchar, size: 64},
	{name: "foreign_server_version", tp: mysql.TypeVarchar, size: 64},
	{name: "authorization_identifier", tp: mysql.TypeVarchar, size: 64},
}

// pgTableForeignTableOptionsCols is table foreign_table_options columns
// https://www.postgresql.org/docs/13/infoschema-foreign-table-options.html
var pgTableForeignTableOptionsCols = []columnInfo{
	{name: "foreign_table_catalog", tp: mysql.TypeVarchar, size: 64},
	{name: "foreign_table_schema", tp: mysql.TypeVarchar, size: 64},
	{name: "foreign_table_name", tp: mysql.TypeVarchar, size: 64},
	{name: "option_name", tp: mysql.TypeVarchar, size: 64},
	{name: "option_value", tp: mysql.TypeVarchar, size: 64},
}

// pgTableForeignTablesCols is table foreign_tables columns
// https://www.postgresql.org/docs/13/infoschema-foreign-tables.html
var pgTableForeignTablesCols = []columnInfo{
	{name: "foreign_table_catalog", tp: mysql.TypeVarchar, size: 64},
	{name: "foreign_table_schema", tp: mysql.TypeVarchar, size: 64},
	{name: "foreign_table_name", tp: mysql.TypeVarchar, size: 64},
	{name: "foreign_server_catalog", tp: mysql.TypeVarchar, size: 64},
	{name: "foreign_server_name", tp: mysql.TypeVarchar, size: 64},
}

// pgTableKeyColumnUsageCols is table key_column_usage columns
// https://www.postgresql.org/docs/13/infoschema-key-column-usage.html
var pgTableKeyColumnUsageCols = []columnInfo{
	{name: "constraint_catalog", tp: mysql.TypeVarchar, size: 512, flag: mysql.NotNullFlag},
	{name: "constraint_schema", tp: mysql.TypeVarchar, size: 64, flag: mysql.NotNullFlag},
	{name: "constraint_name", tp: mysql.TypeVarchar, size: 64, flag: mysql.NotNullFlag},
	{name: "table_catalog", tp: mysql.TypeVarchar, size: 512, flag: mysql.NotNullFlag},
	{name: "table_schema", tp: mysql.TypeVarchar, size: 64, flag: mysql.NotNullFlag},
	{name: "table_name", tp: mysql.TypeVarchar, size: 64, flag: mysql.NotNullFlag},
	{name: "column_name", tp: mysql.TypeVarchar, size: 64, flag: mysql.NotNullFlag},
	{name: "ordinal_position", tp: mysql.TypeLonglong, size: 10, flag: mysql.NotNullFlag},
	{name: "position_in_unique_constraint", tp: mysql.TypeLonglong, size: 10},
	{name: "REFERENCED_TABLE_SCHEMA", tp: mysql.TypeVarchar, size: 64},
	{name: "REFERENCED_TABLE_NAME", tp: mysql.TypeVarchar, size: 64},
	{name: "REFERENCED_COLUMN_NAME", tp: mysql.TypeVarchar, size: 64},
}

// pgTableParametersCols is table parameters columns
// https://www.postgresql.org/docs/13/infoschema-parameters.html
var pgTableParametersCols = []columnInfo{
	{name: "specific_catalog", tp: mysql.TypeVarchar, size: 64},
	{name: "specific_schema", tp: mysql.TypeVarchar, size: 64},
	{name: "specific_name", tp: mysql.TypeVarchar, size: 64},
	{name: "ordinal_position", tp: mysql.TypeVarchar, size: 64},
	{name: "parameter_mode", tp: mysql.TypeVarchar, size: 64},
	{name: "is_result", tp: mysql.TypeVarchar, size: 64},
	{name: "as_local", tp: mysql.TypeVarchar, size: 64},
	{name: "parameter_name", tp: mysql.TypeVarchar, size: 64},
	{name: "data_type", tp: mysql.TypeVarchar, size: 64},
	{name: "character_maximum_length", tp: mysql.TypeVarchar, size: 64},
	{name: "character_octet_length", tp: mysql.TypeVarchar, size: 64},
	{name: "character_set_catalog", tp: mysql.TypeVarchar, size: 64},
	{name: "character_set_schema", tp: mysql.TypeVarchar, size: 64},
	{name: "character_set_name", tp: mysql.TypeVarchar, size: 64},
	{name: "collation_catalog", tp: mysql.TypeVarchar, size: 64},
	{name: "collation_schema", tp: mysql.TypeVarchar, size: 64},
	{name: "collation_name", tp: mysql.TypeVarchar, size: 64},
	{name: "numeric_precision", tp: mysql.TypeVarchar, size: 64},
	{name: "numeric_precision_radix", tp: mysql.TypeVarchar, size: 64},
	{name: "numeric_scale", tp: mysql.TypeVarchar, size: 64},
	{name: "datetime_precision", tp: mysql.TypeVarchar, size: 64},
	{name: "interval_type", tp: mysql.TypeVarchar, size: 64},
	{name: "interval_precision", tp: mysql.TypeVarchar, size: 64},
	{name: "udt_catalog", tp: mysql.TypeVarchar, size: 64},
	{name: "udt_schema", tp: mysql.TypeVarchar, size: 64},
	{name: "udt_name", tp: mysql.TypeVarchar, size: 64},
	{name: "scope_catalog", tp: mysql.TypeVarchar, size: 64},
	{name: "scope_schema", tp: mysql.TypeVarchar, size: 64},
	{name: "scope_name", tp: mysql.TypeVarchar, size: 64},
	{name: "maximum_cardinality", tp: mysql.TypeVarchar, size: 64},
	{name: "dtd_identifier", tp: mysql.TypeVarchar, size: 64},
	{name: "parameter_default", tp: mysql.TypeVarchar, size: 64},
}

// pgTableReferentialConstraintsCols is table referential_constraints columns
// https://www.postgresql.org/docs/13/infoschema-referential-constraints.html
var pgTableReferentialConstraintsCols = []columnInfo{
	{name: "constraint_catalog", tp: mysql.TypeVarchar, size: 64},
	{name: "constraint_schema", tp: mysql.TypeVarchar, size: 64},
	{name: "constraint_name", tp: mysql.TypeVarchar, size: 64},
	{name: "unique_constraint_catalog", tp: mysql.TypeVarchar, size: 64},
	{name: "unique_constraint_schema", tp: mysql.TypeVarchar, size: 64},
	{name: "unique_constraint_name", tp: mysql.TypeVarchar, size: 64},
	{name: "match_option", tp: mysql.TypeVarchar, size: 64},
	{name: "update_rule", tp: mysql.TypeVarchar, size: 64},
	{name: "delete_rule", tp: mysql.TypeVarchar, size: 64},
}

// pgTableRoleColumnGrantsCols is table role_column_grants columns
// https://www.postgresql.org/docs/13/infoschema-role-column-grants.html
var pgTableRoleColumnGrantsCols = []columnInfo{
	{name: "grantor", tp: mysql.TypeVarchar, size: 64},
	{name: "grantee", tp: mysql.TypeVarchar, size: 64},
	{name: "table_catalog", tp: mysql.TypeVarchar, size: 64},
	{name: "table_schema", tp: mysql.TypeVarchar, size: 64},
	{name: "table_name", tp: mysql.TypeVarchar, size: 64},
	{name: "column_name", tp: mysql.TypeVarchar, size: 64},
	{name: "privilege_type", tp: mysql.TypeVarchar, size: 64},
	{name: "is_grantable", tp: mysql.TypeVarchar, size: 64},
}

// pgTableRoleRoutineGrantsCols is table role_routine_grants columns
// https://www.postgresql.org/docs/13/infoschema-role-routine-grants.html
var pgTableRoleRoutineGrantsCols = []columnInfo{
	{name: "grantor", tp: mysql.TypeVarchar, size: 64},
	{name: "grantee", tp: mysql.TypeVarchar, size: 64},
	{name: "specific_catalog", tp: mysql.TypeVarchar, size: 64},
	{name: "specific_schema", tp: mysql.TypeVarchar, size: 64},
	{name: "specific_name", tp: mysql.TypeVarchar, size: 64},
	{name: "routine_catalog", tp: mysql.TypeVarchar, size: 64},
	{name: "routine_schema", tp: mysql.TypeVarchar, size: 64},
	{name: "routine_name", tp: mysql.TypeVarchar, size: 64},
	{name: "privilege_type", tp: mysql.TypeVarchar, size: 64},
	{name: "is_grantable", tp: mysql.TypeVarchar, size: 64},
}

// pgTableRoleTableGrantsCols is table role_table_grants columns
// https://www.postgresql.org/docs/13/infoschema-role-table-grants.html
var pgTableRoleTableGrantsCols = []columnInfo{
	{name: "grantor", tp: mysql.TypeVarchar, size: 64},
	{name: "grantee", tp: mysql.TypeVarchar, size: 64},
	{name: "table_catalog", tp: mysql.TypeVarchar, size: 64},
	{name: "table_schema", tp: mysql.TypeVarchar, size: 64},
	{name: "table_name", tp: mysql.TypeVarchar, size: 64},
	{name: "privilege_type", tp: mysql.TypeVarchar, size: 64},
	{name: "is_grantable", tp: mysql.TypeVarchar, size: 64},
	{name: "with_hierarchy", tp: mysql.TypeVarchar, size: 64},
}

// pgTableRoleUdtGrantsCols is table role_udt_grants columns
// https://www.postgresql.org/docs/13/infoschema-role-udt-grants.html
var pgTableRoleUdtGrantsCols = []columnInfo{
	{name: "grantor", tp: mysql.TypeVarchar, size: 64},
	{name: "grantee", tp: mysql.TypeVarchar, size: 64},
	{name: "udt_catalog", tp: mysql.TypeVarchar, size: 64},
	{name: "udt_schema", tp: mysql.TypeVarchar, size: 64},
	{name: "udt_name", tp: mysql.TypeVarchar, size: 64},
	{name: "privilege_type", tp: mysql.TypeVarchar, size: 64},
	{name: "is_grantable", tp: mysql.TypeVarchar, size: 64},
}

// pgTableRoleUsageGrantsCols is table role_usage_grants columns
// https://www.postgresql.org/docs/13/infoschema-role-usage-grants.html
var pgTableRoleUsageGrantsCols = []columnInfo{
	{name: "grantor", tp: mysql.TypeVarchar, size: 64},
	{name: "grantee", tp: mysql.TypeVarchar, size: 64},
	{name: "object_catalog", tp: mysql.TypeVarchar, size: 64},
	{name: "object_schema", tp: mysql.TypeVarchar, size: 64},
	{name: "object_name", tp: mysql.TypeVarchar, size: 64},
	{name: "object_type", tp: mysql.TypeVarchar, size: 64},
	{name: "privilege_type", tp: mysql.TypeVarchar, size: 64},
	{name: "is_grantable", tp: mysql.TypeVarchar, size: 64},
}

// pgTableRoutinePrivilegesCols is table routine_privileges columns
// https://www.postgresql.org/docs/13/infoschema-routine-privileges.html
var pgTableRoutinePrivilegesCols = []columnInfo{
	{name: "grantor", tp: mysql.TypeVarchar, size: 64},
	{name: "grantee", tp: mysql.TypeVarchar, size: 64},
	{name: "specific_catalog", tp: mysql.TypeVarchar, size: 64},
	{name: "specific_schema", tp: mysql.TypeVarchar, size: 64},
	{name: "specific_name", tp: mysql.TypeVarchar, size: 64},
	{name: "routine_catalog", tp: mysql.TypeVarchar, size: 64},
	{name: "routine_schema", tp: mysql.TypeVarchar, size: 64},
	{name: "routine_name", tp: mysql.TypeVarchar, size: 64},
	{name: "privilege_type", tp: mysql.TypeVarchar, size: 64},
	{name: "is_grantable", tp: mysql.TypeVarchar, size: 64},
}

// pgTableRoutinesCols is table routines columns
// https://www.postgresql.org/docs/13/infoschema-routines.html
var pgTableRoutinesCols = []columnInfo{
	{name: "specific_catalog", tp: mysql.TypeVarchar, size: 64},
	{name: "specific_schema", tp: mysql.TypeVarchar, size: 64},
	{name: "specific_name", tp: mysql.TypeVarchar, size: 64},
	{name: "routine_catalog", tp: mysql.TypeVarchar, size: 64},
	{name: "routine_schema", tp: mysql.TypeVarchar, size: 64},
	{name: "routine_name", tp: mysql.TypeVarchar, size: 64},
	{name: "routine_type", tp: mysql.TypeVarchar, size: 64},
	{name: "module_catalog", tp: mysql.TypeVarchar, size: 64},
	{name: "module_schema", tp: mysql.TypeVarchar, size: 64},
	{name: "module_name", tp: mysql.TypeVarchar, size: 64},
	{name: "udt_catalog", tp: mysql.TypeVarchar, size: 64},
	{name: "udt_schema", tp: mysql.TypeVarchar, size: 64},
	{name: "udt_name", tp: mysql.TypeVarchar, size: 64},
	{name: "data_type", tp: mysql.TypeVarchar, size: 64},
	{name: "character_maximum_length", tp: mysql.TypeVarchar, size: 64},
	{name: "character_octet_length", tp: mysql.TypeVarchar, size: 64},
	{name: "character_set_catalog", tp: mysql.TypeVarchar, size: 64},
	{name: "character_set_schema", tp: mysql.TypeVarchar, size: 64},
	{name: "character_set_name", tp: mysql.TypeVarchar, size: 64},
	{name: "collation_catalog", tp: mysql.TypeVarchar, size: 64},
	{name: "collation_schema", tp: mysql.TypeVarchar, size: 64},
	{name: "collation_name", tp: mysql.TypeVarchar, size: 64},
	{name: "numeric_precision", tp: mysql.TypeVarchar, size: 64},
	{name: "numeric_precision_radix", tp: mysql.TypeVarchar, size: 64},
	{name: "numeric_scale", tp: mysql.TypeVarchar, size: 64},
	{name: "datetime_precision", tp: mysql.TypeVarchar, size: 64},
	{name: "interval_type", tp: mysql.TypeVarchar, size: 64},
	{name: "interval_precision", tp: mysql.TypeVarchar, size: 64},
	{name: "type_udt_catalog", tp: mysql.TypeVarchar, size: 64},
	{name: "type_udt_schema", tp: mysql.TypeVarchar, size: 64},
	{name: "type_udt_name", tp: mysql.TypeVarchar, size: 64},
	{name: "scope_catalog", tp: mysql.TypeVarchar, size: 64},
	{name: "scope_schema", tp: mysql.TypeVarchar, size: 64},
	{name: "scope_name", tp: mysql.TypeVarchar, size: 64},
	{name: "maximum_cardinality", tp: mysql.TypeVarchar, size: 64},
	{name: "dtd_identifier", tp: mysql.TypeVarchar, size: 64},
	{name: "routine_body", tp: mysql.TypeVarchar, size: 64},
	{name: "routine_definition", tp: mysql.TypeVarchar, size: 64},
	{name: "external_name", tp: mysql.TypeVarchar, size: 64},
	{name: "external_language", tp: mysql.TypeVarchar, size: 64},
	{name: "parameter_style", tp: mysql.TypeVarchar, size: 64},
	{name: "is_deterministic", tp: mysql.TypeVarchar, size: 64},
	{name: "sql_data_access", tp: mysql.TypeVarchar, size: 64},
	{name: "is_null_call", tp: mysql.TypeVarchar, size: 64},
	{name: "sql_path", tp: mysql.TypeVarchar, size: 64},
	{name: "schema_level_routine", tp: mysql.TypeVarchar, size: 64},
	{name: "max_dynamic_result_sets", tp: mysql.TypeVarchar, size: 64},
	{name: "is_user_defined_cast", tp: mysql.TypeVarchar, size: 64},
	{name: "is_implicitly_invocable", tp: mysql.TypeVarchar, size: 64},
	{name: "security_type", tp: mysql.TypeVarchar, size: 64},
	{name: "to_sql_specific_catalog", tp: mysql.TypeVarchar, size: 64},
	{name: "to_sql_specific_schema", tp: mysql.TypeVarchar, size: 64},
	{name: "to_sql_specific_name", tp: mysql.TypeVarchar, size: 64},
	{name: "as_locator", tp: mysql.TypeVarchar, size: 64},
	{name: "created", tp: mysql.TypeVarchar, size: 64},
	{name: "last_altered", tp: mysql.TypeVarchar, size: 64},
	{name: "new_savepoint_level", tp: mysql.TypeVarchar, size: 64},
	{name: "id_udt_dependent", tp: mysql.TypeVarchar, size: 64},
	{name: "result_cast_from_data_type", tp: mysql.TypeVarchar, size: 64},
	{name: "result_cast_as_locator", tp: mysql.TypeVarchar, size: 64},
	{name: "result_cast_char_max_length", tp: mysql.TypeVarchar, size: 64},
	{name: "result_cast_char_octet_length", tp: mysql.TypeVarchar, size: 64},
	{name: "result_cast_char_set_catalog", tp: mysql.TypeVarchar, size: 64},
	{name: "result_cast_char_set_schema", tp: mysql.TypeVarchar, size: 64},
	{name: "result_cast_char_set_name", tp: mysql.TypeVarchar, size: 64},
	{name: "result_cast_collation_catalog", tp: mysql.TypeVarchar, size: 64},
	{name: "result_cast_collation_schema", tp: mysql.TypeVarchar, size: 64},
	{name: "result_cast_collation_name", tp: mysql.TypeVarchar, size: 64},
	{name: "result_cast_numeric_precision", tp: mysql.TypeVarchar, size: 64},
	{name: "result_cast_numeric_precision_radix", tp: mysql.TypeVarchar, size: 64},
	{name: "result_cast_numeric_scale", tp: mysql.TypeVarchar, size: 64},
	{name: "result_cast_datetime_precision", tp: mysql.TypeVarchar, size: 64},
	{name: "result_cast_interval_type", tp: mysql.TypeVarchar, size: 64},
	{name: "result_cast_interval_precision", tp: mysql.TypeVarchar, size: 64},
	{name: "result_cast_type_udt_catalog", tp: mysql.TypeVarchar, size: 64},
	{name: "result_cast_type_udt_schema", tp: mysql.TypeVarchar, size: 64},
	{name: "result_cast_type_udt_name", tp: mysql.TypeVarchar, size: 64},
	{name: "result_cast_scope_catalog", tp: mysql.TypeVarchar, size: 64},
	{name: "result_cast_scope_schema", tp: mysql.TypeVarchar, size: 64},
	{name: "result_cast_scope_name", tp: mysql.TypeVarchar, size: 64},
	{name: "result_cast_maximum_cardinality", tp: mysql.TypeVarchar, size: 64},
	{name: "result_cast_dtd_identifier", tp: mysql.TypeVarchar, size: 64},
}

// pgTableSchemataCols is table schemata columns
// https://www.postgresql.org/docs/13/infoschema-schemata.html
var pgTableSchemataCols = []columnInfo{
	{name: "catalog_name", tp: mysql.TypeVarchar, size: 512},
	{name: "schema_name", tp: mysql.TypeVarchar, size: 64},
	{name: "schema_owner", tp: mysql.TypeVarchar, size: 64},
	{name: "default_character_set_catalog", tp: mysql.TypeVarchar, size: 64},
	{name: "default_character_set_schema", tp: mysql.TypeVarchar, size: 64},
	{name: "default_character_set_name", tp: mysql.TypeVarchar, size: 64},
	{name: "sql_path", tp: mysql.TypeVarchar, size: 512},
	{name: "DEFAULT_COLLATION_NAME", tp: mysql.TypeVarchar, size: 32, comment: "TiDB Schemata col"},
}

// pgTableSequencesCols is table sequences columns
// https://www.postgresql.org/docs/13/infoschema-sequences.html
var pgTableSequencesCols = []columnInfo{
	{name: "sequence_catalog", tp: mysql.TypeVarchar, size: 64},
	{name: "sequence_schema", tp: mysql.TypeVarchar, size: 64, flag: mysql.NotNullFlag},
	{name: "sequence_name", tp: mysql.TypeVarchar, size: 64, flag: mysql.NotNullFlag},
	{name: "data_type", tp: mysql.TypeVarchar, size: 64},
	{name: "numeric_precision", tp: mysql.TypeVarchar, size: 64},
	{name: "numeric_precision_radix", tp: mysql.TypeVarchar, size: 32},
	{name: "numeric_scale", tp: mysql.TypeVarchar, size: 64},
	{name: "start_value", tp: mysql.TypeVarchar, size: 64},
	{name: "minimum_value", tp: mysql.TypeVarchar, size: 64},
	{name: "maximum_value", tp: mysql.TypeVarchar, size: 64},
	{name: "increment", tp: mysql.TypeLonglong, size: 21, flag: mysql.NotNullFlag},
	{name: "cycle_option", tp: mysql.TypeVarchar, size: 64},
	{name: "TABLE_CATALOG", tp: mysql.TypeVarchar, size: 512, flag: mysql.NotNullFlag, comment: "TiDB Table Cols"},
	{name: "CACHE", tp: mysql.TypeTiny, flag: mysql.NotNullFlag, comment: "TiDB Table Cols"},
	{name: "CACHE_VALUE", tp: mysql.TypeLonglong, size: 21, comment: "TiDB Table Cols"},
	{name: "CYCLE", tp: mysql.TypeTiny, flag: mysql.NotNullFlag, comment: "TiDB Table Cols"},
	{name: "MAX_VALUE", tp: mysql.TypeLonglong, size: 21, comment: "TiDB Table Cols"},
	{name: "MIN_VALUE", tp: mysql.TypeLonglong, size: 21, comment: "TiDB Table Cols"},
	{name: "START", tp: mysql.TypeLonglong, size: 21, comment: "TiDB Table Cols"},
	{name: "COMMENT", tp: mysql.TypeVarchar, size: 64, comment: "TiDB Table Cols"},
}

// pgTableSQLFeaturesCols is table sql_features columns
// https://www.postgresql.org/docs/13/infoschema-sql-features.html
var pgTableSQLFeaturesCols = []columnInfo{
	{name: "feature_id", tp: mysql.TypeVarchar, size: 64},
	{name: "feature_name", tp: mysql.TypeVarchar, size: 64},
	{name: "sub_feature_id", tp: mysql.TypeVarchar, size: 64},
	{name: "sub_feature_name", tp: mysql.TypeVarchar, size: 64},
	{name: "is_supported", tp: mysql.TypeVarchar, size: 64},
	{name: "is_verified_by", tp: mysql.TypeVarchar, size: 32},
	{name: "comments", tp: mysql.TypeVarchar, size: 64},
}

// pgTableSQLImplementationInfoCols is table sql_implementation_info columns
// https://www.postgresql.org/docs/13/infoschema-sql-implementation-info.html
var pgTableSQLImplementationInfoCols = []columnInfo{
	{name: "implementation_info_id", tp: mysql.TypeVarchar, size: 64},
	{name: "implementation_info_name", tp: mysql.TypeVarchar, size: 64},
	{name: "integer_value", tp: mysql.TypeVarchar, size: 64},
	{name: "character_value", tp: mysql.TypeVarchar, size: 64},
	{name: "comments", tp: mysql.TypeVarchar, size: 64},
}

// pgTableSQLPartsCols is table sql_parts columns
// https://www.postgresql.org/docs/13/infoschema-sql-parts.html
var pgTableSQLPartsCols = []columnInfo{
	{name: "feature_id", tp: mysql.TypeVarchar, size: 64},
	{name: "feature_name", tp: mysql.TypeVarchar, size: 64},
	{name: "is_supported", tp: mysql.TypeVarchar, size: 64},
	{name: "is_verified_by", tp: mysql.TypeVarchar, size: 64},
	{name: "comments", tp: mysql.TypeVarchar, size: 64},
}

// pgTableSqlSizing is table sql_sizing columns
// https://www.postgresql.org/docs/13/infoschema-sql-sizing.html
var pgTableSQLSizingCols = []columnInfo{
	{name: "sizing_id", tp: mysql.TypeInt24, size: 64},
	{name: "sizing_name", tp: mysql.TypeVarchar, size: 64},
	{name: "supported_value", tp: mysql.TypeInt24, size: 64},
	{name: "comments", tp: mysql.TypeVarchar, size: 64},
}

// pgTableTableConstraintsCols is table table_constraints columns
// https://www.postgresql.org/docs/13/infoschema-table-constraints.html
var pgTableTableConstraintsCols = []columnInfo{
	{name: "constraint_catalog", tp: mysql.TypeVarchar, size: 512},
	{name: "constraint_schema", tp: mysql.TypeVarchar, size: 64},
	{name: "constraint_name", tp: mysql.TypeVarchar, size: 64},
	{name: "table_catalog", tp: mysql.TypeVarchar, size: 64},
	{name: "table_schema", tp: mysql.TypeVarchar, size: 64},
	{name: "table_name", tp: mysql.TypeVarchar, size: 64},
	{name: "constraint_type", tp: mysql.TypeVarchar, size: 64},
	{name: "is_deferrable", tp: mysql.TypeVarchar, size: 64},
	{name: "initially_deferred", tp: mysql.TypeVarchar, size: 64},
	{name: "enforced", tp: mysql.TypeVarchar, size: 64},
}

// pgTableTablePrivilegesCols is table table_privileges columns
// https://www.postgresql.org/docs/13/infoschema-table-privileges.html
var pgTableTablePrivilegesCols = []columnInfo{
	{name: "grantor", tp: mysql.TypeVarchar, size: 64},
	{name: "grantee", tp: mysql.TypeVarchar, size: 64},
	{name: "table_catalog", tp: mysql.TypeVarchar, size: 64},
	{name: "table_schema", tp: mysql.TypeVarchar, size: 64},
	{name: "table_name", tp: mysql.TypeVarchar, size: 64},
	{name: "privilege_type", tp: mysql.TypeVarchar, size: 64},
	{name: "is_grantable", tp: mysql.TypeVarchar, size: 64},
	{name: "with_hierarchy", tp: mysql.TypeVarchar, size: 64},
}

// pgTableTablesCols is table tables columns
// https://www.postgresql.org/docs/13/infoschema-tables.html
var pgTableTablesCols = []columnInfo{
	{name: "table_catalog", tp: mysql.TypeVarchar, size: 64},
	{name: "table_schema", tp: mysql.TypeVarchar, size: 64},
	{name: "table_name", tp: mysql.TypeVarchar, size: 64},
	{name: "table_type", tp: mysql.TypeVarchar, size: 64},
	{name: "self_referencing_column_name", tp: mysql.TypeVarchar, size: 64},
	{name: "reference_generation", tp: mysql.TypeVarchar, size: 64},
	{name: "user_defined_type_catalog", tp: mysql.TypeVarchar, size: 64},
	{name: "user_defined_type_schema", tp: mysql.TypeVarchar, size: 64},
	{name: "user_defined_type_name", tp: mysql.TypeVarchar, size: 64},
	{name: "is_insertable_into", tp: mysql.TypeVarchar, size: 64},
	{name: "is_typed", tp: mysql.TypeVarchar, size: 64},
	{name: "commit_action", tp: mysql.TypeVarchar, size: 64},
	{name: "ENGINE", tp: mysql.TypeVarchar, size: 64, comment: "TiDB Table Cols"},
	{name: "VERSION", tp: mysql.TypeLonglong, size: 21, comment: "TiDB Table Cols"},
	{name: "ROW_FORMAT", tp: mysql.TypeVarchar, size: 10, comment: "TiDB Table Cols"},
	{name: "TABLE_ROWS", tp: mysql.TypeLonglong, size: 21, comment: "TiDB Table Cols"},
	{name: "AVG_ROW_LENGTH", tp: mysql.TypeLonglong, size: 21, comment: "TiDB Table Cols"},
	{name: "DATA_LENGTH", tp: mysql.TypeLonglong, size: 21, comment: "TiDB Table Cols"},
	{name: "MAX_DATA_LENGTH", tp: mysql.TypeLonglong, size: 21, comment: "TiDB Table Cols"},
	{name: "INDEX_LENGTH", tp: mysql.TypeLonglong, size: 21, comment: "TiDB Table Cols"},
	{name: "DATA_FREE", tp: mysql.TypeLonglong, size: 21, comment: "TiDB Table Cols"},
	{name: "AUTO_INCREMENT", tp: mysql.TypeLonglong, size: 21, comment: "TiDB Table Cols"},
	{name: "CREATE_TIME", tp: mysql.TypeDatetime, size: 19, comment: "TiDB Table Cols"},
	{name: "UPDATE_TIME", tp: mysql.TypeDatetime, size: 19, comment: "TiDB Table Cols"},
	{name: "CHECK_TIME", tp: mysql.TypeDatetime, size: 19, comment: "TiDB Table Cols"},
	{name: "TABLE_COLLATION", tp: mysql.TypeVarchar, size: 32, flag: mysql.NotNullFlag, deflt: "utf8_bin", comment: "TiDB Table Cols"},
	{name: "CHECKSUM", tp: mysql.TypeLonglong, size: 21, comment: "TiDB Table Cols"},
	{name: "CREATE_OPTIONS", tp: mysql.TypeVarchar, size: 255, comment: "TiDB Table Cols"},
	{name: "TABLE_COMMENT", tp: mysql.TypeVarchar, size: 2048, comment: "TiDB Table Cols"},
	{name: "TIDB_TABLE_ID", tp: mysql.TypeLonglong, size: 21, comment: "TiDB Table Cols"},
	{name: "TIDB_ROW_ID_SHARDING_INFO", tp: mysql.TypeVarchar, size: 255, comment: "TiDB Table Cols"},
}

// pgTableTransformsCols is table transforms columns
// https://www.postgresql.org/docs/13/infoschema-transforms.html
var pgTableTransformsCols = []columnInfo{
	{name: "udt_catalog", tp: mysql.TypeVarchar, size: 64},
	{name: "udt_schema", tp: mysql.TypeVarchar, size: 64},
	{name: "udt_name", tp: mysql.TypeVarchar, size: 64},
	{name: "specific_catalog", tp: mysql.TypeVarchar, size: 64},
	{name: "specific_schema", tp: mysql.TypeVarchar, size: 64},
	{name: "specific_name", tp: mysql.TypeVarchar, size: 64},
	{name: "group_name", tp: mysql.TypeVarchar, size: 64},
	{name: "transform_type", tp: mysql.TypeVarchar, size: 64},
}

// pgTableTriggeredUpdateColumns is table triggered_update_columns columns
// https://www.postgresql.org/docs/13/infoschema-triggered-update-columns.html
var pgTableTriggeredUpdateColumns = []columnInfo{
	{name: "trigger_catalog", tp: mysql.TypeVarchar, size: 64},
	{name: "trigger_schema", tp: mysql.TypeVarchar, size: 64},
	{name: "trigger_name", tp: mysql.TypeVarchar, size: 64},
	{name: "event_object_catalog", tp: mysql.TypeVarchar, size: 64},
	{name: "event_object_schema", tp: mysql.TypeVarchar, size: 64},
	{name: "event_object_name", tp: mysql.TypeVarchar, size: 64},
	{name: "event_object_column", tp: mysql.TypeVarchar, size: 64},
}

// pgTableTriggersCols is table triggers columns
// https://www.postgresql.org/docs/13/infoschema-triggers.html
var pgTableTriggersCols = []columnInfo{
	{name: "trigger_catalog", tp: mysql.TypeVarchar, size: 64},
	{name: "trigger_schema", tp: mysql.TypeVarchar, size: 64},
	{name: "trigger_name", tp: mysql.TypeVarchar, size: 64},
	{name: "event_manipulation", tp: mysql.TypeVarchar, size: 64},
	{name: "event_object_catalog", tp: mysql.TypeVarchar, size: 64},
	{name: "event_object_schema", tp: mysql.TypeVarchar, size: 64},
	{name: "event_object_name", tp: mysql.TypeVarchar, size: 64},
	{name: "action_order", tp: mysql.TypeVarchar, size: 64},
	{name: "action_condition", tp: mysql.TypeVarchar, size: 64},
	{name: "action_statement", tp: mysql.TypeVarchar, size: 64},
	{name: "action_orientation", tp: mysql.TypeVarchar, size: 64},
	{name: "action_timing", tp: mysql.TypeVarchar, size: 64},
	{name: "action_reference_old_table", tp: mysql.TypeVarchar, size: 64},
	{name: "action_reference_new_table", tp: mysql.TypeVarchar, size: 64},
	{name: "action_reference_old_row", tp: mysql.TypeVarchar, size: 64},
	{name: "action_reference_new_row", tp: mysql.TypeVarchar, size: 64},
	{name: "created", tp: mysql.TypeVarchar, size: 64},
}

// pgTableUdtPrivilegesCols is table udt_privileges columns
// https://www.postgresql.org/docs/13/infoschema-udt-privileges.html
var pgTableUdtPrivilegesCols = []columnInfo{
	{name: "grantor", tp: mysql.TypeVarchar, size: 64},
	{name: "grantee", tp: mysql.TypeVarchar, size: 64},
	{name: "udt_catalog", tp: mysql.TypeVarchar, size: 64},
	{name: "udt_schema", tp: mysql.TypeVarchar, size: 64},
	{name: "udt_name", tp: mysql.TypeVarchar, size: 64},
	{name: "privilege_type", tp: mysql.TypeVarchar, size: 64},
	{name: "is_grantable", tp: mysql.TypeVarchar, size: 64},
}

// pgTableUsagePrivilegesCols is table usage_privileges columns
// https://www.postgresql.org/docs/13/infoschema-usage-privileges.html
var pgTableUsagePrivilegesCols = []columnInfo{
	{name: "grantor", tp: mysql.TypeVarchar, size: 64},
	{name: "grantee", tp: mysql.TypeVarchar, size: 64},
	{name: "object_catalog", tp: mysql.TypeVarchar, size: 64},
	{name: "object_schema", tp: mysql.TypeVarchar, size: 64},
	{name: "object_name", tp: mysql.TypeVarchar, size: 64},
	{name: "object_type", tp: mysql.TypeVarchar, size: 64},
	{name: "privilege_type", tp: mysql.TypeVarchar, size: 64},
	{name: "is_grantable", tp: mysql.TypeVarchar, size: 64},
}

// pgTableUserDefinedTypesCols is table user_defined_types columns
// https://www.postgresql.org/docs/13/infoschema-user-defined-types.html
var pgTableUserDefinedTypesCols = []columnInfo{
	{name: "user_defined_type_catalog", tp: mysql.TypeVarchar, size: 64},
	{name: "user_defined_type_schema", tp: mysql.TypeVarchar, size: 64},
	{name: "user_defined_type_name", tp: mysql.TypeVarchar, size: 64},
	{name: "user_defined_type_category", tp: mysql.TypeVarchar, size: 64},
	{name: "is_instantiable", tp: mysql.TypeVarchar, size: 64},
	{name: "is_final", tp: mysql.TypeVarchar, size: 64},
	{name: "ordering_form", tp: mysql.TypeVarchar, size: 64},
	{name: "ordering_category", tp: mysql.TypeVarchar, size: 64},
	{name: "ordering_routine_catalog", tp: mysql.TypeVarchar, size: 64},
	{name: "ordering_routine_schema", tp: mysql.TypeVarchar, size: 64},
	{name: "ordering_routine_name", tp: mysql.TypeVarchar, size: 64},
	{name: "reference_type", tp: mysql.TypeVarchar, size: 64},
	{name: "data_type", tp: mysql.TypeVarchar, size: 64},
	{name: "character_maximum_length", tp: mysql.TypeVarchar, size: 64},
	{name: "character_octet_length", tp: mysql.TypeVarchar, size: 64},
	{name: "character_set_catalog", tp: mysql.TypeVarchar, size: 64},
	{name: "character_set_schema", tp: mysql.TypeVarchar, size: 64},
	{name: "character_set_name", tp: mysql.TypeVarchar, size: 64},
	{name: "collation_catalog", tp: mysql.TypeVarchar, size: 64},
	{name: "collation_schema", tp: mysql.TypeVarchar, size: 64},
	{name: "collation_name", tp: mysql.TypeVarchar, size: 64},
	{name: "numeric_precision", tp: mysql.TypeVarchar, size: 64},
	{name: "numeric_precision_radix", tp: mysql.TypeVarchar, size: 64},
	{name: "numeric_scale", tp: mysql.TypeVarchar, size: 64},
	{name: "datetime_precision", tp: mysql.TypeVarchar, size: 64},
	{name: "interval_type", tp: mysql.TypeVarchar, size: 64},
	{name: "interval_precision", tp: mysql.TypeVarchar, size: 64},
	{name: "source_dtd_identifier", tp: mysql.TypeVarchar, size: 64},
	{name: "ref_dtd_identifier", tp: mysql.TypeVarchar, size: 64},
}

// pgTableUserMappingOptionsCols is table user_mapping_options columns
// https://www.postgresql.org/docs/13/infoschema-user-mapping-options.html
var pgTableUserMappingOptionsCols = []columnInfo{
	{name: "authorization_identifier", tp: mysql.TypeVarchar, size: 64},
	{name: "foreign_server_catalog", tp: mysql.TypeVarchar, size: 64},
	{name: "foreign_server_name", tp: mysql.TypeVarchar, size: 64},
	{name: "option_name", tp: mysql.TypeVarchar, size: 64},
	{name: "option_value", tp: mysql.TypeVarchar, size: 64},
}

// pgTableUserMappingsCols is table user_mappings columns
// https://www.postgresql.org/docs/13/infoschema-user-mappings.html
var pgTableUserMappingsCols = []columnInfo{
	{name: "authorization_identifier", tp: mysql.TypeVarchar, size: 64},
	{name: "foreign_server_catalog", tp: mysql.TypeVarchar, size: 64},
	{name: "foreign_server_name", tp: mysql.TypeVarchar, size: 64},
}

// pgTableViewColumnUsageCols is table view_column_usage columns
// https://www.postgresql.org/docs/13/infoschema-view-column-usage.html
var pgTableViewColumnUsageCols = []columnInfo{
	{name: "view_catalog", tp: mysql.TypeVarchar, size: 64},
	{name: "view_schema", tp: mysql.TypeVarchar, size: 64},
	{name: "view_name", tp: mysql.TypeVarchar, size: 64},
	{name: "table_catalog", tp: mysql.TypeVarchar, size: 64},
	{name: "table_schema", tp: mysql.TypeVarchar, size: 64},
	{name: "table_name", tp: mysql.TypeVarchar, size: 64},
	{name: "column_name", tp: mysql.TypeVarchar, size: 64},
}

// pgTableViewRoutineUsageCols is table view_routine_usage
// https://www.postgresql.org/docs/13/infoschema-view-routine-usage.html
var pgTableViewRoutineUsageCols = []columnInfo{
	{name: "table_catalog", tp: mysql.TypeVarchar, size: 64},
	{name: "table_schema", tp: mysql.TypeVarchar, size: 64},
	{name: "table_name", tp: mysql.TypeVarchar, size: 64},
	{name: "specific_catalog", tp: mysql.TypeVarchar, size: 64},
	{name: "specific_schema", tp: mysql.TypeVarchar, size: 64},
	{name: "specific_name", tp: mysql.TypeVarchar, size: 64},
}

// pgTableViewTableUsageCols is table view_table_usage columns
// https://www.postgresql.org/docs/13/infoschema-view-table-usage.html
var pgTableViewTableUsageCols = []columnInfo{
	{name: "view_catalog", tp: mysql.TypeVarchar, size: 64},
	{name: "view_schema", tp: mysql.TypeVarchar, size: 64},
	{name: "view_name", tp: mysql.TypeVarchar, size: 64},
	{name: "table_catalog", tp: mysql.TypeVarchar, size: 64},
	{name: "table_schema", tp: mysql.TypeVarchar, size: 64},
	{name: "table_name", tp: mysql.TypeVarchar, size: 64},
}

// pgTableViewsCols is table views columns
// https://www.postgresql.org/docs/13/infoschema-views.html
var pgTableViewsCols = []columnInfo{
	{name: "table_catalog", tp: mysql.TypeVarchar, size: 512, flag: mysql.NotNullFlag},
	{name: "table_schema", tp: mysql.TypeVarchar, size: 64, flag: mysql.NotNullFlag},
	{name: "table_name", tp: mysql.TypeVarchar, size: 64, flag: mysql.NotNullFlag},
	{name: "view_definition", tp: mysql.TypeLongBlob, flag: mysql.NotNullFlag},
	{name: "check_option", tp: mysql.TypeVarchar, size: 8, flag: mysql.NotNullFlag},
	{name: "is_updatable", tp: mysql.TypeVarchar, size: 3, flag: mysql.NotNullFlag},
	{name: "is_insertable_into", tp: mysql.TypeVarchar, size: 64},
	{name: "is_trigger_updatable", tp: mysql.TypeVarchar, size: 64},
	{name: "is_trigger_deletable", tp: mysql.TypeVarchar, size: 64},
	{name: "is_trigger_insertable_into", tp: mysql.TypeVarchar, size: 64},
	{name: "DEFINER", tp: mysql.TypeVarchar, size: 77, flag: mysql.NotNullFlag, comment: "TiDB Table Cols"},
	{name: "SECURITY_TYPE", tp: mysql.TypeVarchar, size: 7, flag: mysql.NotNullFlag, comment: "TiDB Table Cols"},
	{name: "CHARACTER_SET_CLIENT", tp: mysql.TypeVarchar, size: 32, flag: mysql.NotNullFlag, comment: "TiDB Table Cols"},
	{name: "COLLATION_CONNECTION", tp: mysql.TypeVarchar, size: 32, flag: mysql.NotNullFlag, comment: "TiDB Table Cols"},
}

var pgInfoTableNameToColumns = map[string][]columnInfo{
	TablePgInformationsSchemaCatalogName:      pgTableInformationSchemaCatalogNameCols,
	TablePgAdministrableRoleAuthorizations:    pgTableAdministrableRoleAuthorizationsCols,
	TablePgApplicableRole:                     pgTableApplicableRolesCols,
	TablePgAttributes:                         pgTableAttributesCols,
	TablePgCharacterSets:                      pgTableCharacterSetsCols,
	TablePgCheckConstraintRoutineUsage:        pgTableCheckConstraintRoutineUsageCols,
	TablePgCheckConstraints:                   pgTableCheckConstraintsCols,
	TablePgCollations:                         pgTableCollationsCols,
	TablePgCollationCharacterSetApplicability: pgTableCollationCharacterSetApplicabilityCols,
	TablePgColumnColumnUsage:                  pgTableColumnColumnUsageCols,
	TablePgColumnDomainUsage:                  pgTableColumnDomainUsageCols,
	TablePgColumnOptions:                      pgTableColumnOptionsCols,
	TablePgColumnPrivileges:                   pgTableColumnPrivilegesCols,
	TablePgColumnUdtUsage:                     pgTableColumnUdtUsageCols,
	TablePgColumns:                            pgTableColumnsCols,
	TablePgConstraintColumnUsage:              pgTableConstraintColumnUsageCols,
	TablePgConstraintTableUsage:               pgTableConstraintTableUsageCols,
	TablePgDataTypePrivileges:                 pgTableDataTypePrivilegesCols,
	TablePgDomainConstraints:                  pgTableDomainConstraintsCols,
	TablePgDomainUdtUsage:                     pgTableDomainUdtUsageCols,
	TablePgDomains:                            pgTableDomainsCols,
	TablePgElementTypes:                       pgTableElementTypesCols,
	TablePgEnabledRoles:                       pgTableEnabledRolesCols,
	TablePgForeignDataWrapperOptions:          pgTableForeignDataWrapperOptionsCols,
	TablePgForeignDataWrappers:                pgTableForeignDataWrappersCols,
	TablePgForeignServerOptions:               pgTableForeignServerOptionsCols,
	TablePgForeignServers:                     pgTableForeignServers,
	TablePgForeignTableOptions:                pgTableForeignTableOptionsCols,
	TablePgForeignTales:                       pgTableForeignTablesCols,
	TablePgKeyColumnUsage:                     pgTableKeyColumnUsageCols,
	TablePgParameters:                         pgTableParametersCols,
	TablePgReferentialConstraints:             pgTableReferentialConstraintsCols,
	TablePgRoleColumnGrants:                   pgTableRoleColumnGrantsCols,
	TablePgRoleRoutineGrants:                  pgTableRoleRoutineGrantsCols,
	TablePgRoleTableGrants:                    pgTableRoleTableGrantsCols,
	TablePgRoleUdtGrants:                      pgTableRoleUdtGrantsCols,
	TablePgRoleUsageGrants:                    pgTableRoleUsageGrantsCols,
	TablePgRoutinePrivileges:                  pgTableRoutinePrivilegesCols,
	TablePgRoutines:                           pgTableRoutinesCols,
	TablePgSchemata:                           pgTableSchemataCols,
	TablePgSequences:                          pgTableSequencesCols,
	TablePgSQLFeatures:                        pgTableSQLFeaturesCols,
	TablePgSQLImplementationInfo:              pgTableSQLImplementationInfoCols,
	TablePgSQLParts:                           pgTableSQLPartsCols,
	TablePgSQLSizing:                          pgTableSQLSizingCols,
	TablePgTableConstraints:                   pgTableTableConstraintsCols,
	TablePgTablePrivileges:                    pgTableTablePrivilegesCols,
	TablePgTables:                             pgTableTablesCols,
	TablePgTransforms:                         pgTableTransformsCols,
	TablePgTriggeredUpdateColumns:             pgTableTriggeredUpdateColumns,
	TablePgTriggers:                           pgTableTriggersCols,
	TablePgUdtPrivileges:                      pgTableUdtPrivilegesCols,
	TablePgUsagePrivileges:                    pgTableUsagePrivilegesCols,
	TablePgUserDefinedTypes:                   pgTableUserDefinedTypesCols,
	TablePgUserMappingOptions:                 pgTableUserMappingOptionsCols,
	TablePgUserMappings:                       pgTableUserMappingsCols,
	TablePgViewColumnUsage:                    pgTableViewColumnUsageCols,
	TablePgViewRoutineUsage:                   pgTableViewRoutineUsageCols,
	TablePgViewTableUsage:                     pgTableViewTableUsageCols,
	TablePgViews:                              pgTableViewsCols,
}

// addPgInfoTable
func addPgInfoTable() {
	for name, cols := range pgInfoTableNameToColumns {
		tableNameToColumns[name] = cols
	}
	for k, v := range pgTableIDMap {
		tableIDMap[k] = v
	}
}

func init() {
	// Initialize the information shema database and register the driver to `drivers`
	dbID := autoid.InformationSchemaDBID
	addPgInfoTable()
	infoSchemaTables := make([]*model.TableInfo, 0, len(tableNameToColumns))
	for name, cols := range tableNameToColumns {
		tableInfo := buildTableMeta(name, cols)
		infoSchemaTables = append(infoSchemaTables, tableInfo)
		var ok bool
		tableInfo.ID, ok = tableIDMap[tableInfo.Name.O]
		if !ok {
			panic(fmt.Sprintf("get information_schema table id failed, unknown system table `%v`", tableInfo.Name.O))
		}
		for i, c := range tableInfo.Columns {
			c.ID = int64(i) + 1
		}
	}
	infoSchemaDB := &model.DBInfo{
		ID:      dbID,
		Name:    util.InformationSchemaName,
		Charset: mysql.DefaultCharset,
		Collate: mysql.DefaultCollationName,
		Tables:  infoSchemaTables,
	}
	RegisterVirtualTable(infoSchemaDB, createInfoSchemaTable)
	util.GetSequenceByName = func(is interface{}, schema, sequence model.CIStr) (util.SequenceTable, error) {
		return GetSequenceByName(is.(InfoSchema), schema, sequence)
	}
}
