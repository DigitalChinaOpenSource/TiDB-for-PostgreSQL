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

package server

import (
	"fmt"
	"github.com/DigitalChinaOpenSource/DCParser/mysql"
	"github.com/DigitalChinaOpenSource/DCParser/terror"
	"github.com/jackc/pgproto3/v2"
	"strings"
)

//handleTooBigPrecision 处理指定精度过大的错误
func handleTooBigPrecision(m *mysql.SQLError, te *terror.Error, sql string) (*pgproto3.ErrorResponse, error) {
	errorResp := &pgproto3.ErrorResponse {
		Severity: "ERROR",
		SeverityUnlocalized: "",
		//pg不会报这个error
		Code: "XX000",
		Message: m.Message,
		Detail: "",
		Hint: "",
	}
	setFilePathAndLine(te, errorResp)
	return errorResp, nil
}

//handleInvalidUseOfNull 处理设置字段非空时由于原本存在空数据而导致的错误
// eg.Invalid use of NULL value
func handleInvalidUseOfNull(m *mysql.SQLError, te *terror.Error, sql string) (*pgproto3.ErrorResponse, error) {
	upperSql, beforeTable, beforeCol, empty :=strings.ToUpper(sql), "TABLE", "MODIFY", " "
	tableStart := strings.Index(upperSql, beforeTable) + len(beforeTable)
	tableLen := strings.Index(strings.Trim(sql[tableStart : ], empty), empty)
	table := strings.Trim(sql[tableStart : ], empty)[ : tableLen]

	colStart := strings.Index(upperSql, beforeCol) + len(beforeCol)
	colLen := strings.Index(strings.Trim(sql[colStart : ], empty), empty)
	col := strings.Trim(sql[colStart : ], empty)[ : colLen]

	pgMsg := fmt.Sprintf("column \"%s\" of relation \"%s\" contains null values", col, table)

	errorResp := &pgproto3.ErrorResponse {
		Severity: "ERROR",
		SeverityUnlocalized: "",
		// 语法错误
		Code: "42601",
		//pg没有对应的错误，因为pg允许创建表不带任何字段，这里返回mysql原始的错误信息
		Message: pgMsg,
		Detail: "",
		Hint: "",
	}
	setFilePathAndLine(te, errorResp)

	return errorResp, nil
}

//handleTableNoColumn 处理创建表时没有声明字段的错误
func handleTableNoColumn(m *mysql.SQLError, te *terror.Error, sql string) (*pgproto3.ErrorResponse, error) {
	errorResp := &pgproto3.ErrorResponse {
		Severity: "ERROR",
		SeverityUnlocalized: "",
		//internal_error，MySQL无对应错误，因为pg容忍表没有字段
		Code: "XX000",
		//pg没有对应的错误，因为pg允许创建表不带任何字段，这里返回mysql原始的错误信息
		Message: m.Message,
		Detail: "",
		Hint: "",
	}
	setFilePathAndLine(te, errorResp)

	return errorResp, nil
}

//handleInvalidGroupFuncUse 处理错误使用聚合函数的错误
// eg. Invalid use of group function
func handleInvalidGroupFuncUse(m *mysql.SQLError, te *terror.Error, sql string) (*pgproto3.ErrorResponse, error) {
	where ,having := "WHERE","HAVING"
	funcList := [...]string{"AVG(","BIT_AND(","BIT_OR(", "BIT_XOR(", "COUNT(", "GROUP_CONCAT(", "JSON_ARRAYAGG(","JSON_OBJECTAGG(",
		"MAX(","MIN(","STD(","STDDEV(","STDDEV_POP(","STDDEV_SAMP(","SUM(","VAR_POP(","VAR_SAMP(","VARIANCE("}
	upperSql := strings.ToUpper(sql)

	//错误是where子句使用聚合函数。所以检查的起点就是where的位置，终点是having子句
	whereIndex := strings.Index(upperSql, where)
	havingIndex := strings.Index(upperSql, having)

	//在where子句中的第一个聚合函数位置
	firstIndex := len(sql)

	for _,f := range funcList {
		fIndex := strings.Index(upperSql, f)
		if havingIndex == -1 {
			if fIndex != -1 && fIndex > whereIndex && fIndex < firstIndex {
				firstIndex = fIndex
			}
		} else {
			if fIndex != -1 && fIndex < havingIndex && fIndex > whereIndex && fIndex < firstIndex {
				firstIndex = fIndex
			}
		}
	}

	pgMsg := fmt.Sprintf("aggregate functions are not allowed in WHERE")
	errorResp := &pgproto3.ErrorResponse {
		Severity: "ERROR",
		SeverityUnlocalized: "",
		//grouping_error
		Code: "42803",
		Message: pgMsg,
		Position: int32(firstIndex + 1), // Add one because the first character in query string has index of 1 according to document: https://www.postgresql.org/docs/13/protocol-error-fields.html
		Detail: "",
		Hint: "",
	}
	setFilePathAndLine(te, errorResp)

	return errorResp, nil
}

//handleFiledSpecifiedTwice 字段声明超过一次，可能两次，三次，都报这个错
// eg.Column 'name' specified twice
func handleFiledSpecifiedTwice(m *mysql.SQLError, te *terror.Error, sql string) (*pgproto3.ErrorResponse, error) {
	msg, beforeColumn, empty, apostrophe := m.Message, "Column ", " ", "'"

	colStart := strings.Index(msg, beforeColumn) + len(beforeColumn)
	colLen := strings.Index(msg[colStart : ], empty)
	col := strings.Trim(msg[colStart : colStart + colLen], apostrophe)

	pgMsg := fmt.Sprintf("column \"%s\" specified more than once",col)

	once := strings.Index(sql, col) + len(col)
	twice := once + strings.Index(sql[once : ], col)

	errorResp := &pgproto3.ErrorResponse{
		Severity: "ERROR",
		SeverityUnlocalized: "",
		//duplicate_column
		Code: "42701",
		Message: pgMsg,
		Position: int32(twice + 3),
		Detail: "",
		Hint: "",
	}
	setFilePathAndLine(te, errorResp)

	return errorResp, nil
}

//handleUnknownTableInDelete 解决delete语法错误导致的错误信息
func handleUnknownTableInDelete(m *mysql.SQLError, te *terror.Error, sql string) (*pgproto3.ErrorResponse, error) {
	del, empty, point,comma := "DELETE", " ", ".",","
	columnStart := strings.Index(strings.ToUpper(sql), del) + len(del)
	cutSql := strings.Trim(sql[columnStart : ], empty)
	// eg1. delete a.name from test
	// eg2. delete id from test
	// eg3 delete id,name from test
	pointLen := strings.Index(cutSql, point)
	commaLen := strings.Index(cutSql, comma)
	emptyLen := strings.Index(cutSql, empty)
	finalLen := len(cutSql)
	if pointLen != -1 && finalLen > pointLen {
		finalLen = pointLen
	}
	if commaLen != -1 && finalLen > commaLen {
		finalLen = commaLen
	}
	if emptyLen != -1 && finalLen > emptyLen {
		finalLen = emptyLen
	}
	column := cutSql[ : finalLen]
	pgMsg := fmt.Sprintf("syntax error at or near \"%s\"", column)

	position := strings.Index(sql, column) + 3

	errorResp := &pgproto3.ErrorResponse{
		Severity: "ERROR",
		SeverityUnlocalized: "",
		//语法错误
		Code: "42601",
		Message: pgMsg,
		Position: int32(position),
		Detail: "",
		Hint: "",
	}
	setFilePathAndLine(te, errorResp)

	return errorResp, nil
}

//handleCantDropFieldOrKey 解决无法删除字段或键的错误
func handleCantDropFieldOrKey(m *mysql.SQLError, te *terror.Error, sql string) (*pgproto3.ErrorResponse, error) {

	msg, beforeColumn, beforeTable, empty := m.Message, "column ", "TABLE", " "
	columnStart := strings.Index(msg, beforeColumn) + len(beforeColumn)
	columnLen := strings.Index(msg[columnStart : ], empty)
	column := msg[columnStart : columnStart + columnLen]

	tableStart := strings.Index(strings.Trim(strings.ToUpper(sql), empty), beforeTable) + len(beforeTable)
	cutSql := strings.Trim(sql[tableStart : ], empty)
	tableLen := strings.Index(cutSql, empty) + 1
	table := cutSql[:tableLen]

	pgMsg := fmt.Sprintf("column \"%s\" of relation \"%s\" does not exist",column, table)

	errorResp := &pgproto3.ErrorResponse{
		Severity: "ERROR",
		SeverityUnlocalized: "",
		//internal_error
		Code: "XX000",
		Message: pgMsg,
		Detail: "",
		Hint: "",
	}
	setFilePathAndLine(te, errorResp)

	return errorResp, nil
}

//handleMultiplePKDefined 处理多主键被定义的错误
// eg. Multiple primary key defined
func handleMultiplePKDefined(m *mysql.SQLError, te *terror.Error, sql string) (*pgproto3.ErrorResponse, error) {

	msg, beforeTable, brackets, pk := m.Message, "TABLE ", "(", "PRIMARY KEY"
	tableStart := strings.Index(strings.ToUpper(sql), beforeTable) + len(beforeTable)
	tableLen := strings.Index(sql, brackets)
	table := msg[tableStart : tableStart + tableLen]

	firstPKSpec := strings.Index(strings.ToUpper(sql), pk) + len(pk)
	position := firstPKSpec + strings.Index(strings.ToUpper(sql[firstPKSpec : ]), pk)
	pgMsg := fmt.Sprintf("multiple primary keys for table \"%s\" are not allowed", table)

	errorResp := &pgproto3.ErrorResponse{
		//invalid_table_definition
		Code: "42P16",
		Severity: "ERROR",
		Message: pgMsg,
		Position: int32(position),
	}
	setFilePathAndLine(te,errorResp)

	return errorResp, nil
}


// handleParseError 处理编译sql出错的信息
// eg. You have an error in your SQL syntax; check the manual that corresponds to your TiDB version for the right syntax to use line 1 column 5 near "creat table student(id int);"
func handleParseError(m *mysql.SQLError, te *terror.Error, sql string) (*pgproto3.ErrorResponse, error) {
	msg, nearDelimiter, wrap, quotes, emptyStr := m.Message, "near ", "\n","\""," "

	nearStart := strings.Index(msg, nearDelimiter) + len(nearDelimiter)
	nearlen := strings.Index(strings.Trim(msg[nearStart : ], emptyStr), emptyStr)
	lineEnd := strings.Index(strings.Trim(msg[nearStart : ], emptyStr), wrap)

	var finalLen int
	var behindNear string
	if nearlen == -1 && lineEnd == -1 {
		behindNear = strings.Trim(msg[nearStart : ], emptyStr)
	} else if nearlen == -1 && lineEnd != -1 {
		behindNear = strings.Trim(msg[nearStart : nearStart + lineEnd], emptyStr)
	} else {
		if lineEnd < nearlen && lineEnd != -1 {
			finalLen = nearStart + lineEnd
		} else {
			finalLen = nearStart + nearlen
		}
		behindNear = strings.Trim(msg[nearStart : finalLen], emptyStr)
	}

	if !strings.HasSuffix(behindNear, quotes) {
		behindNear += quotes
	}

	pgMsg := fmt.Sprintf("syntax error at or near %s",behindNear)

	// 为什么加3? 在提示信息第二行开头回提示是第几行sql的错误，他也会占偏移量，所以要将其加上。
	position := strings.Index(sql,strings.Trim(behindNear, quotes)) + 3

	errorResp := &pgproto3.ErrorResponse{
		Severity: "ERROR",
		SeverityUnlocalized: "",
		//语法错误
		Code: "42601",
		Message: pgMsg,
		Position: int32(position),
		Detail: "",
		Hint: "",
	}
	setFilePathAndLine(te, errorResp)

	return errorResp, nil
}

// handleDuplicateKey 处理键冲突的问题
func handleDuplicateKey(m *mysql.SQLError, te *terror.Error, sql string) (*pgproto3.ErrorResponse, error) {
	msg, entry, key, apostrophe, empty := m.Message,"entry ", "key ", "'", " "

	keyStart := strings.Index(msg, key) + len(key) + 1
	keyName := strings.Trim(msg[keyStart : ], apostrophe)

	valueStart := strings.Index(msg, entry) + len(entry)
	valueLen := strings.Index(msg[valueStart : ], empty)
	value := msg[valueStart : valueStart + valueLen]
	pgMsg := fmt.Sprintf("duplicate key value violates unique constraint \"%s\"\nDescription:  Key value %v already exists.", keyName, value)

	errorResp := &pgproto3.ErrorResponse{
		Severity: "ERROR",
		SeverityUnlocalized: "",
		//unique_violation
		Code: "23505",
		Message: pgMsg,
		Detail: "",
		Hint: "",
	}
	setFilePathAndLine(te, errorResp)

	return errorResp, nil
}

//handleUnknownColumn 处理未知字段名的错误信息
func handleUnknownColumn(m *mysql.SQLError, te *terror.Error, sql string) (*pgproto3.ErrorResponse, error) {
	msg, column, into, apostrophe, bracket,empty := m.Message, "column ", "into ","'", "(", " "

	sel, del, insert, update := "SELECT","DELETE","INSERT","UPDATE"

	columnNameStart := strings.Index(msg, column) + len(column)
	columnNameLen := strings.Index(msg[columnNameStart : ], empty) + 1
	columnName := strings.Trim(strings.Trim(msg[columnNameStart : columnNameStart + columnNameLen], empty), apostrophe)

	var relation,pgMsg string
	if strings.HasPrefix(strings.ToUpper(strings.Trim(sql, empty)), insert) {
		relationStart := strings.Index(sql, into) + len(into)
		relationLen := strings.Index(sql[relationStart : ], empty)
		leftBracketLen := strings.Index(sql[relationStart : ], bracket)

		if leftBracketLen != -1 && leftBracketLen < relationLen {
			relation = sql[relationStart : relationStart + leftBracketLen]
		} else {
			relation = sql[relationStart : relationStart + relationLen]
		}
		pgMsg = fmt.Sprintf("column \"%s\" of relation \"%s\" does not exist", columnName, relation)
	} else if strings.HasPrefix(strings.ToUpper(strings.Trim(sql, empty)), del) {
		pgMsg = fmt.Sprintf("column \"%s\" does not exist", columnName)
	} else if strings.HasPrefix(strings.ToUpper(strings.Trim(sql, empty)), sel) {
		pgMsg = fmt.Sprintf("column \"%s\" does not exist", columnName)
	} else if strings.HasPrefix(strings.ToUpper(strings.Trim(sql, empty)), update) {
		relationStart := strings.Index(strings.ToUpper(sql), update)
		relationLen := strings.Index(strings.Trim(sql[relationStart : ], empty), empty)
		relation = sql[relationStart : relationStart + relationLen]
		pgMsg = fmt.Sprintf("column \"%s\" of relation \"%s\" does not exist", columnName, relation)
	} else {
		pgMsg = m.Message
	}
	position := strings.Index(sql, columnName) + 3

	errorResp := &pgproto3.ErrorResponse{
		Severity: "ERROR",
		SeverityUnlocalized: "",
		//undefined_column
		Code: "42703",
		Message: pgMsg,
		Position: int32(position),
		Detail: "",
		Hint: "",
	}
	setFilePathAndLine(te, errorResp)

	return errorResp, nil
}

//handleTableExists 处理表已经存在，重复创建表的错误信息
func handleTableExists(m *mysql.SQLError, te *terror.Error, sql string) (*pgproto3.ErrorResponse, error) {
	msg, point, apostrophe := m.Message, ".", "'"

	relationStart := strings.Index(msg, point) + 1
	relationLen := strings.Index(msg[relationStart : ], apostrophe)
	relation := msg[relationStart : relationStart + relationLen]
	pgMsg := fmt.Sprintf("relation \"%s\" already exists", relation)

	errorResponse := &pgproto3.ErrorResponse{
		Severity: "ERROR",
		SeverityUnlocalized: "",
		//duplicate_table
		Code: "42P07",
		Message: pgMsg,
		Detail: "",
		Hint: "",
	}
	setFilePathAndLine(te, errorResponse)

	return errorResponse, nil
}
// return the name of the table from MySQL error message
func getTableName(msg string, immediateBefore string, immediateAfter string) string {
	tableStart := strings.Index(msg, immediateBefore) + 1
	tableLen := strings.Index(msg[tableStart:], immediateAfter)
	tableName := msg[tableStart : tableStart+tableLen]

	return tableName
}

//handleUndefinedTable 处理表不存在的错误
func handleUndefinedTable(m *mysql.SQLError, te *terror.Error, sql string) (*pgproto3.ErrorResponse, error) {
	tableName := getTableName(m.Message, ".", "'")
	pgMsg := fmt.Sprintf("table \"%s\" does not exist", tableName)

	errorResponse := &pgproto3.ErrorResponse{
		Severity: "ERROR",
		SeverityUnlocalized: "",
		//undefined_table
		Code: "42P01",
		Message: pgMsg,
		Detail: "",
		Hint: "",
	}
	setFilePathAndLine(te, errorResponse)
	return errorResponse, nil
}

//handleUnknownDB 处理数据库不存在的错误
// eg. Unknown database 'testDB'
func handleUnknownDB(m *mysql.SQLError, te *terror.Error, sql string) (*pgproto3.ErrorResponse, error) {
	msg, beforeDB, apostrophe,empty := m.Message, "database ", "'"," "
	dbStart := strings.Index(msg, beforeDB) + len(beforeDB)
	db := strings.Trim(strings.Trim(msg[dbStart : ], empty), apostrophe)
	pgMsg := fmt.Sprintf("database \"%s\" does not exist\nkeep last connection",db)

	errorResp := &pgproto3.ErrorResponse{
		Code: "3D000",
		Severity: "FATAL",
		Message: pgMsg,
	}
	setFilePathAndLine(te,errorResp)
	return errorResp, nil
}

//handleAccessDenied 处理登录失败信息 todo 写这里的时候dctidb没有密码，也没法测试，需要加上密码之后测试
//eg.Access denied for user 'root'@'localhost' (using password: YES)
func handleAccessDenied(m *mysql.SQLError, te *terror.Error, sql string) (*pgproto3.ErrorResponse, error) {
	msg, beforeUser, at, apostrophe := m.Message, "user ", "@", "'"

	userStart := strings.Index(msg, beforeUser) + len(beforeUser)
	userLen := strings.Index(msg[userStart : ], at)
	user := strings.Trim(msg[userStart : userStart + userLen], apostrophe)
	pgMsg := fmt.Sprintf("password authentication failed for user \"%s\"",user)

	errorResp := &pgproto3.ErrorResponse{
		Severity: "ERROR",
		SeverityUnlocalized: "",
		//privilege_not_granted
		Code: "01007",
		Message: pgMsg,
		Detail: "",
		Hint: "",
	}
	setFilePathAndLine(te, errorResp)

	return errorResp,nil
}

//handleDropDBFail 处理数据库不存在，删除数据库失败的错误信息
func handleDropDBFail(m *mysql.SQLError, te *terror.Error, sql string) (*pgproto3.ErrorResponse, error) {
	msg, beforeDatabase,semicolon, apostrophe := m.Message, "database ", ";", "'"

	databaseStart := strings.Index(msg, beforeDatabase) + len(beforeDatabase)
	databaseLen := strings.Index(msg[databaseStart : ], semicolon)
	database := strings.Trim(msg[databaseStart : databaseStart + databaseLen], apostrophe)
	pgMsg := fmt.Sprintf("database \"%s\" does not exist", database)

	errorResponse := &pgproto3.ErrorResponse{
		Severity: "ERROR",
		SeverityUnlocalized: "",
		//invalid_catalog_name
		Code: "3D000",
		Message: pgMsg,
		Detail: "",
		Hint: "",
	}
	setFilePathAndLine(te,errorResponse)

	return errorResponse, nil
}

//handleCreateDBFail 数据库已存在，创建数据库失败
func handleCreateDBFail(m *mysql.SQLError, te *terror.Error, sql string) (*pgproto3.ErrorResponse, error) {
	msg, beforeDatabase,semicolon, apostrophe := m.Message, "database ", ";", "'"

	databaesStart := strings.Index(msg, beforeDatabase) + len(beforeDatabase)
	databaseLen := strings.Index(msg[databaesStart : ], semicolon)
	database := strings.Trim(msg[databaesStart : databaesStart + databaseLen], apostrophe)
	pgMsg := fmt.Sprintf("database \"%s\" already exists", database)

	errorResponse := &pgproto3.ErrorResponse{
		Severity: "ERROR",
		SeverityUnlocalized: "",
		//duplicate_database
		Code: "42P04",
		Message: pgMsg,
		Detail: "",
		Hint: "",
	}
	setFilePathAndLine(te, errorResponse)

	return errorResponse, nil
}

//handleDataOutOfRange 处理超过字段范围的错误
// eg. Out of range value for column 'id' at row 1
func handleDataOutOfRange(m *mysql.SQLError, te *terror.Error, sql string) (*pgproto3.ErrorResponse, error) {
	msg, beforeColumn, empty, apostrophe := m.Message, "column ", " ", "'"

	colunmStart := strings.Index(msg, beforeColumn) + len(beforeColumn)
	columnLen := strings.Index(msg[colunmStart : ], empty)
	column := strings.Trim(msg[colunmStart : colunmStart + columnLen], apostrophe)

	//eg.integer out of range， mysql的错误信息不足以判断类型，于是新创了一个错误信息
	pgMsg := fmt.Sprintf("column '%s' value out of range",column)

	errorResp := &pgproto3.ErrorResponse{
		Severity: "ERROR",
		SeverityUnlocalized: "",
		//numeric_value_out_of_range
		Code: "22003",
		Message: pgMsg,
		Detail: "",
		Hint: "",
	}
	setFilePathAndLine(te,errorResp)

	return errorResp,nil
}

//handleDataTooLong 处理数据超长错误
//eg. Data too long for column 'name' at row 1
func handleDataTooLong(m *mysql.SQLError, te *terror.Error, sql string) (*pgproto3.ErrorResponse, error) {
	msg, beforeColumn, empty, apostrophe := m.Message, "column ", " ", "'"

	columnStart := strings.Index(msg, beforeColumn) + len(beforeColumn)
	columnLen := strings.Index(msg[columnStart : ], empty)
	column := strings.Trim(msg[columnStart : columnStart + columnLen], apostrophe)

	//eg. value too long for type character(5)
	//二者信息差距太多，综合二者重新创建一条错误信息语句
	pgMsg := fmt.Sprintf("value too long for column '%s'",column)

	errorResp := &pgproto3.ErrorResponse{
		Severity: "ERROR",
		SeverityUnlocalized: "",
		//string_data_length_mismatch
		Code: "22026",
		Message: pgMsg,
		Detail: "",
		Hint: "",
	}
	setFilePathAndLine(te, errorResp)

	return errorResp, nil
}

//handleWrongNumberOfColsInSelect 处理union时两表字段不对应的情况
//eg. The used SELECT statements have a different number of columns
func handleWrongNumberOfColsInSelect(m *mysql.SQLError, te *terror.Error, sql string) (*pgproto3.ErrorResponse, error) {
	pgMsg := "each UNION query must have the same number of columns"

	// mysql 的错徐信息不足以支持计算出position，尤其是在多个union同时使用时.
	errorResp := &pgproto3.ErrorResponse{
		Severity: "ERROR",
		SeverityUnlocalized: "",
		//语法错误
		Code: "42601",
		Message: pgMsg,
		Detail: "",
		Hint: "",
	}
	setFilePathAndLine(te, errorResp)

	return errorResp, nil
}

//handleDerivedMustHaveAlias 处理派生表缺少别名的错误
// eg. Every derived table must have its own alias
func handleDerivedMustHaveAlias(m *mysql.SQLError, te *terror.Error, sql string) (*pgproto3.ErrorResponse, error) {
	brackets := "("

	pgMsg := "subquery in FROM must have an alias"
	position := strings.Index(sql, brackets) + 3

	errorResp := &pgproto3.ErrorResponse{
		Severity: "ERROR",
		SeverityUnlocalized: "",
		//语法错误
		Code: "42601",
		Message: pgMsg,
		Position: int32(position),
		Detail: "",
		Hint: "",
	}
	setFilePathAndLine(te, errorResp)

	return errorResp, nil
}

//handleSubqueryNo1Row 处理子查询返回数据行数超过一行的错误
//eg. Subquery returns more than 1 row
func handleSubqueryNo1Row(m *mysql.SQLError, te *terror.Error, sql string) (*pgproto3.ErrorResponse, error) {
	pgMsg := "more than one row returned by a subquery used as an expression"

	errorResp := &pgproto3.ErrorResponse{
		Severity: "ERROR",
		SeverityUnlocalized: "",
		//cardinality_violation
		Code: "21000",
		Message: pgMsg,
		Detail: "",
		Hint: "",
	}
	setFilePathAndLine(te, errorResp)

	return errorResp, nil
}

//handleNoPermissionToCreateUser 处理无权创建用户的错误信息
//eg. 'u'@'192.168.2.111' is not allowed to create new users
func handleNoPermissionToCreateUser(m *mysql.SQLError, te *terror.Error, sql string) (*pgproto3.ErrorResponse, error) {
	pgMsg := "permission denied to create role"

	errorResp := &pgproto3.ErrorResponse{
		Severity: "ERROR",
		SeverityUnlocalized: "",
		//privilege_not_granted
		Code: "01007",
		Message: pgMsg,
		Detail: "",
		Hint: "",
	}
	setFilePathAndLine(te, errorResp)

	return errorResp, nil
}

//handleTableAccessDenied 处理数据表访问被拒的错误信息 todo 有待权限完善后测试
//eg. SELECT command denied to user 'XXX'@'localhost' for table 'db'
func handleTableAccessDenied(m *mysql.SQLError, te *terror.Error, sql string) (*pgproto3.ErrorResponse, error) {
	msg, beforeTable, apostrophe := m.Message, "table ", "'"

	tableStart := strings.Index(msg, beforeTable) + len(beforeTable)
	tableLen := strings.Index(msg, apostrophe)
	table := strings.Trim(msg[tableStart : tableStart + tableLen], apostrophe)
	pgMsg := fmt.Sprintf("permission denied for table %s", table)

	errorResp := &pgproto3.ErrorResponse{
		Severity: "ERROR",
		SeverityUnlocalized: "",
		Code: "2F004",
		Message: pgMsg,
		Detail: "",
		Hint: "",
	}
	setFilePathAndLine(te,errorResp)

	return errorResp, nil
}

//handleColumnAccessDenied 处理字段访问拒绝信息 todo 写这里的时候dctidb权限模块还没做好，所以没有测试，后续需要测试一下。
// pgsql权限粒度没有达到字段级别，所以pg的错误信息中不会指出哪个字段访问被拒,只会指明表访问被拒，这里与pgsql保持一致、
func handleColumnAccessDenied(m *mysql.SQLError, te *terror.Error, sql string) (*pgproto3.ErrorResponse, error) {
	// eg. SELECT command denied to user 'e1534'@'192.168.176.100' for column 'id' in table 't1'
	msg, beforeTable, apostrophe := m.Message, "table ", "'"

	tableStart := strings.Index(msg, beforeTable) + len(beforeTable)
	tableLen := strings.Index(msg, apostrophe)
	table := strings.Trim(msg[tableStart : tableStart + tableLen], apostrophe)
	pgMsg := fmt.Sprintf("permission denied for table %s", table)

	errorResp := &pgproto3.ErrorResponse{
		Severity: "ERROR",
		SeverityUnlocalized: "",
		Code: "2F004",
		Message: pgMsg,
		Detail: "",
		Hint: "",
	}
	setFilePathAndLine(te, errorResp)

	return errorResp, nil
}

//handleNoDefaultValue 处理非空字段缺少默认值的情况
func handleNoDefaultValue(m *mysql.SQLError, te *terror.Error, sql string) (*pgproto3.ErrorResponse, error) {
	msg, beforeColumn, empty, apostrophe, into, brackets := m.Message, "Field ", " ", "'", "INTO ", "("

	columnStart := strings.Index(msg, beforeColumn) + len(beforeColumn)
	columnLen := strings.Index(msg[columnStart : ], empty)
	column := strings.Trim(msg[columnStart : columnStart + columnLen], apostrophe)

	relationStart := strings.Index(strings.ToUpper(sql), into) + len(into)
	relationLen := strings.Index(sql[relationStart : ], empty)
	bracketsLen := strings.Index(sql[relationStart : ], brackets)

	var relation string
	if bracketsLen < relationLen && bracketsLen != -1 {
		relation = sql[relationStart : relationStart + bracketsLen]
	}else {
		relation = sql[relationStart : relationStart + relationLen]
	}
	//原版pg对应错误还有一行错误信息，eg.描述:  Failing row contains (null, xxx    , null).，但是MySQL的错误信息不足以凑出第二行的信息。
	pgMsg := fmt.Sprintf("null value in column \"%s\" of relation \"%s\" violates not-null constraint", column, relation)

	errorResp := &pgproto3.ErrorResponse{
		Severity: "ERROR",
		SeverityUnlocalized: "",
		Code: "42601",
		Message: pgMsg,
		Detail: "",
		Hint: "",
	}
	setFilePathAndLine(te, errorResp)

	return errorResp, nil
}



//handleColumnMisMatch 处理字段与值的数量匹配不上的情况
func handleColumnMisMatch(m *mysql.SQLError, te *terror.Error, sql string) (*pgproto3.ErrorResponse, error) {
	values := "values"

	valueStart := strings.Index(sql, values) + len(values)
	//这里的游标只能指到values后面的括号，mysql的错误信息里拿不到哪些 value 是多出来的。
	position := valueStart + 3
	pgMsg := fmt.Sprintf("INSERT has more expressions than target columns")

	errorResp := &pgproto3.ErrorResponse{
		Severity: "ERROR",
		SeverityUnlocalized: "",
		//语法错误
		Code: "42601",
		Message: pgMsg,
		Position: int32(position),
		Detail: "",
		Hint: "",
	}
	setFilePathAndLine(te, errorResp)

	return errorResp, nil
}


// handleRelationNotExists 处理表（关系）不存在的错误信息
func handleRelationNotExists(m *mysql.SQLError, te *terror.Error, sql string) (*pgproto3.ErrorResponse, error) {
	msg, point, apostrophe := m.Message, ".", "'"

	tableNameStart := strings.Index(msg,point) + 1
	nameLen := strings.Index(msg[tableNameStart : ], apostrophe)
	tableName := msg[tableNameStart : tableNameStart + nameLen]
	pgMsg := fmt.Sprintf("relation \"%s\" does not exist", tableName)
	position := strings.Index(sql, tableName) + 3

	errorResp := &pgproto3.ErrorResponse{
		Severity: "ERROR",
		SeverityUnlocalized: "",
		Code: "42P01",
		Message: pgMsg,
		Position: int32(position),
		Detail: "",
		Hint: "",
	}
	setFilePathAndLine(te, errorResp)

	return errorResp,nil
}
// handleTypeError 处理类型转换错误信息
func handleTypeError(m *mysql.SQLError, te *terror.Error, sql string) (*pgproto3.ErrorResponse, error) {
	msg, quotes, beforeInput := m.Message, "\"", "parsing "
	inputStart := strings.Index(msg, beforeInput) + len(beforeInput)
	inputLen := strings.Index(msg[inputStart + 1 : ], quotes) + 1
	input := msg[inputStart : inputStart + inputLen]
	if !strings.HasSuffix(input, quotes) {
		input += quotes
	}
	pgMsg := fmt.Sprintf("invalid input syntax for : %s", input)
	position := strings.Index(sql, input)

	errorResp := &pgproto3.ErrorResponse{
		Severity: "ERROR",
		SeverityUnlocalized: "",
		//datatype_mismatch
		Code: "42804",
		Message: pgMsg,
		Position: int32(position),
		Detail: "",
		Hint: "",
	}
	setFilePathAndLine(te, errorResp)

	return errorResp,nil
}

//setFilePathAndLine 通过terror.Error对象获取出错文件和行位置
func setFilePathAndLine(te *terror.Error, errorResponse *pgproto3.ErrorResponse){
	//Get the file location and line number where the error occurred
	var filePath string
	var line int
	if te != nil {
		filePath, line = te.Location()
	}
	errorResponse.File, errorResponse.Line = filePath, int32(line)
}
