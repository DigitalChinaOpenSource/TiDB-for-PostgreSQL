package variable

// PostgreSQL system variable
const (
	//PgExtraFloatDigits is the database system variable
	PgExtraFloatDigits = "extra_float_digits"

	//PgSearchPath is the database system variable
	PgSearchPath = "search_path"

	//PgDateStyle is the pg database system variable
	PgDateStyle = "datestyle"

	//PgClientMinMessage is the pg database system variable
	PgClientMinMessage = "client_min_messages"

	//PgClientEncoding is the pg database system_variable
	PgClientEncoding = "client_encoding"

	//PgByteaOutput is the pg database system_variable
	PgByteaOutput = "bytea_output"
)

// Default Pg system variable values.
const (
	DefPgClientEncoding   = "UTF8"
	DefPgDateStyle        = "ISO"
	DefPgClientMinMessage = "notice"
	DefPgByteaOutput      = "hex"
	DefPgSearchPath       = "public"
	DefPgExtraFloatDigits = 1
)
