package variable

import (
	"strconv"
)

func init() {
	sysVars = make(map[string]*SysVar)
	for _, v := range defaultSysVars {
		RegisterSysVar(v)
	}
	for _, v := range postgresSysVars {
		RegisterSysVar(v)
	}
	for _, v := range noopSysVars {
		v.IsNoop = true
		RegisterSysVar(v)
	}
}

var postgresSysVars = []*SysVar{
	{Scope: ScopeGlobal | ScopeSession, Name: PgExtraFloatDigits, Value: strconv.FormatInt(DefPgExtraFloatDigits, 10)},
	{Scope: ScopeGlobal | ScopeSession, Name: PgSearchPath, Value: DefPgSearchPath},
	{Scope: ScopeGlobal | ScopeSession, Name: PgDateStyle, Value: DefPgDateStyle},
	{Scope: ScopeGlobal | ScopeSession, Name: PgClientEncoding, Value: DefPgClientEncoding},
	{Scope: ScopeGlobal | ScopeSession, Name: PgClientMinMessage, Value: DefPgClientMinMessage},
	{Scope: ScopeGlobal | ScopeSession, Name: PgByteaOutput, Value: DefPgByteaOutput},
}
