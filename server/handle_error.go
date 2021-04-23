package server

import (
	"fmt"
	"github.com/jackc/pgproto3/v2"
	"github.com/pingcap/parser/mysql"
	"github.com/pingcap/parser/terror"
	"strings"
)

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
		Code: "42601",
		Detail: "",
		Hint: "",
	}
	setFilePathAndLine(te, errorResp)
	errorResp.Message, errorResp.Position = pgMsg, int32(position)

	return errorResp, nil
}

//handleWrongFieldSpec 列指定符不正确
func handleWrongFieldSpec(m *mysql.SQLError, te *terror.Error, sql string) (*pgproto3.ErrorResponse, error) {
	errorResp := &pgproto3.ErrorResponse{
		Code: "XX0000",
		Severity: "ERROR",
		Message: "有待处理的MySQL 1063 错误",
	}
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
		Code: "tobe",
		Detail: "",
		Hint: "",
	}
	setFilePathAndLine(te, errorResp)
	errorResp.Message = pgMsg

	return errorResp, nil
}

//handleDupKeyName 键名字冲突
func handleDupKeyName(m *mysql.SQLError, te *terror.Error, sql string) (*pgproto3.ErrorResponse, error) {
	errorResp := &pgproto3.ErrorResponse{
		Code: "XX0000",
		Severity: "ERROR",
		Message: "有待处理的MySQL 1061 错误",
	}
	return errorResp, nil
}

//handleDupColumnName 字段名冲突
func handleDupColumnName(m *mysql.SQLError, te *terror.Error, sql string) (*pgproto3.ErrorResponse, error) {
	errorResp := &pgproto3.ErrorResponse{
		Code: "XX0000",
		Severity: "ERROR",
		Message: "有待处理的MySQL 1060 错误",
	}
	return errorResp, nil
}

//handleIdentTooLong 名字太长
func handleIdentTooLong(m *mysql.SQLError, te *terror.Error, sql string) (*pgproto3.ErrorResponse, error) {
	errorResp := &pgproto3.ErrorResponse{
		Code: "XX0000",
		Severity: "ERROR",
		Message: "有待处理的MySQL 1059 错误",
	}
	return errorResp, nil
}

// handleWrongValueCount 解决字段数与值数量不对应的问题
func handleWrongValueCount(m *mysql.SQLError, te *terror.Error, sql string) (*pgproto3.ErrorResponse, error) {
	errorResp := &pgproto3.ErrorResponse{
		Code: "XX0000",
		Severity: "ERROR",
		Message: "有待处理的MySQL 1058 错误",
	}
	return errorResp, nil
}

//handleWrongSumSelect 处理语句有和函数和列在同一语句的错误
func handleWrongSumSelect(m *mysql.SQLError, te *terror.Error, sql string) (*pgproto3.ErrorResponse, error) {
	errorResp := &pgproto3.ErrorResponse{
		Code: "XX0000",
		Severity: "ERROR",
		Message: "有待处理的MySQL 1057 错误",
	}
	return errorResp, nil
}

//handleWrongField 处理无法对字段xx分组的错误
func handleWrongField(m *mysql.SQLError, te *terror.Error, sql string) (*pgproto3.ErrorResponse, error) {
	errorResp := &pgproto3.ErrorResponse{
		Code: "XX0000",
		Severity: "ERROR",
		Message: "有待处理的MySQL 1056 错误",
	}
	return errorResp, nil
}

//handleWrongFieldWithGroup 处理错误的分组列
func handleWrongFieldWithGroup(m *mysql.SQLError, te *terror.Error, sql string) (*pgproto3.ErrorResponse, error) {
	errorResp := &pgproto3.ErrorResponse{
		Code: "XX0000",
		Severity: "ERROR",
		Message: "有待处理的MySQL 1055 错误",
	}
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
	}
	position := strings.Index(sql, columnName) + 3

	errorResp := &pgproto3.ErrorResponse{
		Severity: "ERROR",
		SeverityUnlocalized: "",
		Code: "tobe",
		Detail: "",
		Hint: "",
	}
	setFilePathAndLine(te, errorResp)
	errorResp.Message, errorResp.Position = pgMsg, int32(position)

	return errorResp, nil
}

//handleServerShutDown 处理中途停机错误
func handleServerShutDown(m *mysql.SQLError, te *terror.Error, sql string) (*pgproto3.ErrorResponse, error) {
	errorResp := &pgproto3.ErrorResponse{
		Code: "XX0000",
		Severity: "ERROR",
		Message: "有待处理的MySQL 1053 错误",
	}
	return errorResp, nil
}

//handleColumnAmbiguous 处理字段多义，含糊不清错误
func handleColumnAmbiguous(m *mysql.SQLError, te *terror.Error, sql string) (*pgproto3.ErrorResponse, error) {
	errorResp := &pgproto3.ErrorResponse{
		Code: "XX0000",
		Severity: "ERROR",
		Message: "有待处理的MySQL 1052 错误",
	}
	return errorResp, nil
}

//handleUnknownTable 处理未知的表错误
func handleUnknownTable(m *mysql.SQLError, te *terror.Error, sql string) (*pgproto3.ErrorResponse, error) {
	errorResp := &pgproto3.ErrorResponse{
		Code: "XX0000",
		Severity: "ERROR",
		Message: "有待处理的MySQL 1051 错误",
	}
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
		Code: "tobe",
		Detail: "",
		Hint: "",
	}
	setFilePathAndLine(te, errorResponse)
	errorResponse.Message = pgMsg

	return errorResponse, nil
}

//handleUnknownDB 处理数据库不存在的错误
func handleUnknownDB(m *mysql.SQLError, te *terror.Error, sql string) (*pgproto3.ErrorResponse, error) {
	errorResp := &pgproto3.ErrorResponse{
		Code: "XX0000",
		Severity: "ERROR",
		Message: "有待处理的MySQL 1049 错误",
	}
	return errorResp, nil
}

//handleBadNullColumn 处理字段xxx不能为空的错误
func handleBadNullColumn(m *mysql.SQLError, te *terror.Error, sql string) (*pgproto3.ErrorResponse, error) {
	errorResp := &pgproto3.ErrorResponse{
		Code: "XX0000",
		Severity: "ERROR",
		Message: "有待处理的MySQL 1048 错误",
	}
	return errorResp, nil
}

//handleUnknownCmd 处理未知的cmd命令
func handleUnknownCmd(m *mysql.SQLError, te *terror.Error, sql string) (*pgproto3.ErrorResponse, error) {
	errorResp := &pgproto3.ErrorResponse{
		Code: "XX0000",
		Severity: "ERROR",
		Message: "有待处理的MySQL 1047 错误",
	}
	return errorResp, nil
}

//handleNoDBSelected 处理没有选择数据库的错误
func handleNoDBSelected(m *mysql.SQLError, te *terror.Error, sql string) (*pgproto3.ErrorResponse, error) {
	errorResp := &pgproto3.ErrorResponse{
		Code: "XX0000",
		Severity: "ERROR",
		Message: "有待处理的MySQL 1046 错误",
	}
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
		Code: "tobe",
		Detail: "",
		Hint: "",
	}
	setFilePathAndLine(te, errorResp)
	errorResp.Message = pgMsg

	return errorResp,nil
}

//handleDBAccessDenied 处理连接数据库被拒的错误
func handleDBAccessDenied(m *mysql.SQLError, te *terror.Error, sql string) (*pgproto3.ErrorResponse, error) {
	errorResp := &pgproto3.ErrorResponse{
		Code: "XX0000",
		Severity: "ERROR",
		Message: "有待处理的MySQL 1044 错误",
	}
	return errorResp, nil
}

//handleHandShakeError 处理握手错误
func handleHandShakeError(m *mysql.SQLError, te *terror.Error, sql string) (*pgproto3.ErrorResponse, error) {
	errorResp := &pgproto3.ErrorResponse{
		Code: "XX0000",
		Severity: "ERROR",
		Message: "有待处理的MySQL 1043 错误",
	}
	return errorResp, nil
}

//handleBadHostError 处理主机名错误的信息
func handleBadHostError(m *mysql.SQLError, te *terror.Error, sql string) (*pgproto3.ErrorResponse, error) {
	errorResp := &pgproto3.ErrorResponse{
		Code: "XX0000",
		Severity: "ERROR",
		Message: "有待处理的MySQL 1042 错误",
	}
	return errorResp, nil
}

//handleOutOfResources 处理内存耗尽的错误
func handleOutOfResources(m *mysql.SQLError, te *terror.Error, sql string) (*pgproto3.ErrorResponse, error) {
	errorResp := &pgproto3.ErrorResponse{
		Code: "XX0000",
		Severity: "ERROR",
		Message: "有待处理的MySQL 1041 错误",
	}
	return errorResp, nil
}

//handleConCountError 处理连接数过多的错误
func handleConCountError(m *mysql.SQLError, te *terror.Error, sql string) (*pgproto3.ErrorResponse, error) {
	errorResp := &pgproto3.ErrorResponse{
		Code: "XX0000",
		Severity: "ERROR",
		Message: "有待处理的MySQL 1040 错误",
	}
	return errorResp, nil
}

//handleOutOfSortMemory 处理排序内存溢出
func handleOutOfSortMemory(m *mysql.SQLError, te *terror.Error, sql string) (*pgproto3.ErrorResponse, error) {
	errorResp := &pgproto3.ErrorResponse{
		Code: "XX0000",
		Severity: "ERROR",
		Message: "有待处理的MySQL 1038 错误",
	}
	return errorResp, nil
}

//handleOutOfMemory 处理内存溢出错误
func handleOutOfMemory(m *mysql.SQLError, te *terror.Error, sql string) (*pgproto3.ErrorResponse, error) {
	errorResp := &pgproto3.ErrorResponse{
		Code: "XX0000",
		Severity: "ERROR",
		Message: "有待处理的MySQL 1037 错误",
	}
	return errorResp, nil
}

// handleOpenAsReadonly 处理表只读错误
func handleOpenAsReadonly(m *mysql.SQLError, te *terror.Error, sql string) (*pgproto3.ErrorResponse, error) {
	errorResp := &pgproto3.ErrorResponse{
		Code: "XX0000",
		Severity: "ERROR",
		Message: "有待处理的MySQL 1036 错误",
	}
	return errorResp, nil
}

//handleOldKeyFile 处理过时的key文件错误
func handleOldKeyFile(m *mysql.SQLError, te *terror.Error, sql string) (*pgproto3.ErrorResponse, error) {
	errorResp := &pgproto3.ErrorResponse{
		Code: "XX0000",
		Severity: "ERROR",
		Message: "有待处理的MySQL 1035 错误",
	}
	return errorResp, nil
}

//handleNotKeyFile 不正确的key file对于表xxx
func handleNotKeyFile(m *mysql.SQLError, te *terror.Error, sql string) (*pgproto3.ErrorResponse, error) {
	errorResp := &pgproto3.ErrorResponse{
		Code: "XX0000",
		Severity: "ERROR",
		Message: "有待处理的MySQL 1034 错误",
	}
	return errorResp, nil
}

//handleNotFormFile 处理找不到记录的错误
func handleNotFormFile(m *mysql.SQLError, te *terror.Error, sql string) (*pgproto3.ErrorResponse, error) {
	errorResp := &pgproto3.ErrorResponse{
		Code: "XX0000",
		Severity: "ERROR",
		Message: "有待处理的MySQL 1033 错误",
	}
	return errorResp, nil
}

//handleKeyNotFound 处理找不到键的错误
func handleKeyNotFound(m *mysql.SQLError, te *terror.Error, sql string) (*pgproto3.ErrorResponse, error) {
	errorResp := &pgproto3.ErrorResponse{
		Code: "XX0000",
		Severity: "ERROR",
		Message: "有待处理的MySQL 1032 错误",
	}
	return errorResp, nil
}


//handleIllegalHA 表存储引擎没有这个选项
func handleIllegalHA(m *mysql.SQLError, te *terror.Error, sql string) (*pgproto3.ErrorResponse, error) {
	errorResp := &pgproto3.ErrorResponse{
		Code: "XX0000",
		Severity: "ERROR",
		Message: "有待处理的MySQL 1031 错误",
	}
	return errorResp, nil
}

//handleGotErrno 处理存储引擎错误
func handleGotErrno(m *mysql.SQLError, te *terror.Error, sql string) (*pgproto3.ErrorResponse, error) {
	errorResp := &pgproto3.ErrorResponse{
		Code: "XX0000",
		Severity: "ERROR",
		Message: "有待处理的MySQL 1030 错误",
	}
	return errorResp, nil
}

//handleFilSortAbort 处理排序中止错误
func handleFilSortAbort(m *mysql.SQLError, te *terror.Error, sql string) (*pgproto3.ErrorResponse, error) {
	errorResp := &pgproto3.ErrorResponse{
		Code: "XX0000",
		Severity: "ERROR",
		Message: "有待处理的MySQL 1028 错误",
	}
	return errorResp, nil
}

//handleFileUsed 处理文件被锁导致的冲突
func handleFileUsed(m *mysql.SQLError, te *terror.Error, sql string) (*pgproto3.ErrorResponse, error) {
	errorResp := &pgproto3.ErrorResponse{
		Code: "XX0000",
		Severity: "ERROR",
		Message: "有待处理的MySQL 1027 错误",
	}
	return errorResp, nil
}

//handleErrorOnWrite 处理写文件错误
func handleErrorOnWrite(m *mysql.SQLError, te *terror.Error, sql string) (*pgproto3.ErrorResponse, error) {
	errorResp := &pgproto3.ErrorResponse{
		Code: "XX0000",
		Severity: "ERROR",
		Message: "有待处理的MySQL 1026 错误",
	}
	return errorResp, nil
}

//handleErrorOnRename 处理重命名错误
func handleErrorOnRename(m *mysql.SQLError, te *terror.Error, sql string) (*pgproto3.ErrorResponse, error) {
	errorResp := &pgproto3.ErrorResponse{
		Code: "XX0000",
		Severity: "ERROR",
		Message: "有待处理的MySQL 1025 错误",
	}
	return errorResp, nil
}

//handleErrorOnRead 处理读文件出错信息
func handleErrorOnRead(m *mysql.SQLError, te *terror.Error, sql string) (*pgproto3.ErrorResponse, error) {
	errorResp := &pgproto3.ErrorResponse{
		Code: "XX0000",
		Severity: "ERROR",
		Message: "有待处理的MySQL 1024 错误",
	}
	return errorResp, nil
}

// handleDupKey 处理外键重名导致的错误
func handleDupKey(m *mysql.SQLError, te *terror.Error, sql string) (*pgproto3.ErrorResponse, error) {
	errorResp := &pgproto3.ErrorResponse{
		Code: "XX0000",
		Severity: "ERROR",
		Message: "有待处理的MySQL 1022 错误",
	}
	return errorResp, nil
}

//handleCheckRead 处理读后记录变更错误
func handleCheckRead(m *mysql.SQLError, te *terror.Error, sql string) (*pgproto3.ErrorResponse, error) {
	errorResp := &pgproto3.ErrorResponse{
		Code: "XX0000",
		Severity: "ERROR",
		Message: "有待处理的MySQL 1020 错误",
	}
	return errorResp, nil
}

//handleCantReadDir
func handleCantReadDir(m *mysql.SQLError, te *terror.Error, sql string) (*pgproto3.ErrorResponse, error) {
	errorResp := &pgproto3.ErrorResponse{
		Code: "XX0000",
		Severity: "ERROR",
		Message: "有待处理的MySQL 1018 错误",
	}
	return errorResp, nil
}

//handleFileNotFound 处理找不到文件错误
func handleFileNotFound(m *mysql.SQLError, te *terror.Error, sql string) (*pgproto3.ErrorResponse, error) {
	errorResp := &pgproto3.ErrorResponse{
		Code: "XX0000",
		Severity: "ERROR",
		Message: "有待处理的MySQL 1017 错误",
	}
	return errorResp, nil
}

//handleCantOpenFile 处理无法打开文件的错误
func handleCantOpenFile(m *mysql.SQLError, te *terror.Error, sql string) (*pgproto3.ErrorResponse, error) {
	errorResp := &pgproto3.ErrorResponse{
		Code: "XX0000",
		Severity: "ERROR",
		Message: "有待处理的MySQL 1016 错误",
	}
	return errorResp, nil
}

//handleCantLock 处理无法锁定文件的错误
func handleCantLock(m *mysql.SQLError, te *terror.Error, sql string) (*pgproto3.ErrorResponse, error) {
	errorResp := &pgproto3.ErrorResponse{
		Code: "XX0000",
		Severity: "ERROR",
		Message: "有待处理的MySQL 1015 错误",
	}
	return errorResp, nil
}

//handleCantGetStat 处理获取不到状态的错误
func handleCantGetStat(m *mysql.SQLError, te *terror.Error, sql string) (*pgproto3.ErrorResponse, error) {
	errorResp := &pgproto3.ErrorResponse{
		Code: "XX0000",
		Severity: "ERROR",
		Message: "有待处理的MySQL 1013 错误",
	}
	return errorResp, nil
}

//handleCantFindSystemRec 处理读取不到系统表记录的错误
func handleCantFindSystemRec(m *mysql.SQLError, te *terror.Error, sql string) (*pgproto3.ErrorResponse, error) {
	errorResp := &pgproto3.ErrorResponse{
		Code: "XX0000",
		Severity: "ERROR",
		Message: "有待处理的MySQL 1012 错误",
	}
	return errorResp, nil
}

//handleDBDropRmDir 处理MySQL data文件夹下有其他文件导致的报错
func handleDBDropRmDir(m *mysql.SQLError, te *terror.Error, sql string) (*pgproto3.ErrorResponse, error) {

	errorResp := &pgproto3.ErrorResponse{
		Code: "XX0000",
		Severity: "ERROR",
		Message: "有待处理的MySQL 1010 错误",
	}
	return errorResp, nil
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
		Code: "tobe",
		Detail: "",
		Hint: "",
	}
	setFilePathAndLine(te,errorResponse)
	errorResponse.Message = pgMsg

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
		Code: "tobe",
		Detail: "",
		Hint: "",
	}
	setFilePathAndLine(te, errorResponse)
	errorResponse.Message = pgMsg

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
		Code: "tobe",
		Detail: "",
		Hint: "",
	}
	setFilePathAndLine(te,errorResp)
	errorResp.Message = pgMsg

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
		Code: "tobe",
		Detail: "",
		Hint: "",
	}
	setFilePathAndLine(te, errorResp)
	errorResp.Message = pgMsg

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
		Code: "tobe",
		Detail: "",
		Hint: "",
	}
	setFilePathAndLine(te, errorResp)
	errorResp.Message = pgMsg

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
		Code: "tobe",
		Detail: "",
		Hint: "",
	}
	setFilePathAndLine(te, errorResp)
	errorResp.Message, errorResp.Position = pgMsg, int32(position)

	return errorResp, nil
}

//handleSubqueryNo1Row 处理子查询返回数据行数超过一行的错误
//eg. Subquery returns more than 1 row
func handleSubqueryNo1Row(m *mysql.SQLError, te *terror.Error, sql string) (*pgproto3.ErrorResponse, error) {
	pgMsg := "more than one row returned by a subquery used as an expression"

	errorResp := &pgproto3.ErrorResponse{
		Severity: "ERROR",
		SeverityUnlocalized: "",
		Code: "tobe",
		Detail: "",
		Hint: "",
	}
	setFilePathAndLine(te, errorResp)
	errorResp.Message =  pgMsg

	return errorResp, nil
}

//handleNoPermissionToCreateUser 处理无权创建用户的错误信息
//eg. 'u'@'192.168.2.111' is not allowed to create new users
func handleNoPermissionToCreateUser(m *mysql.SQLError, te *terror.Error, sql string) (*pgproto3.ErrorResponse, error) {
	pgMsg := "permission denied to create role"

	errorResp := &pgproto3.ErrorResponse{
		Severity: "ERROR",
		SeverityUnlocalized: "",
		Code: "tobe",
		Detail: "",
		Hint: "",
	}
	setFilePathAndLine(te, errorResp)
	errorResp.Message = pgMsg

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
		Detail: "",
		Hint: "",
	}
	setFilePathAndLine(te,errorResp)
	errorResp.Message = pgMsg

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
		Detail: "",
		Hint: "",
	}
	setFilePathAndLine(te, errorResp)
	errorResp.Message = pgMsg

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
		Code: "tobe",
		Detail: "",
		Hint: "",
	}
	setFilePathAndLine(te, errorResp)
	errorResp.Message = pgMsg

	return errorResp, nil
}



//handeleColumnMisMatch 处理字段与值的数量匹配不上的情况
func handeleColumnMisMatch(m *mysql.SQLError, te *terror.Error, sql string) (*pgproto3.ErrorResponse, error) {
	values := "values"

	valueStart := strings.Index(sql, values) + len(values)
	//这里的游标只能指到values后面的括号，mysql的错误信息里拿不到哪些 value 是多出来的。
	position := valueStart + 3
	pgMsg := fmt.Sprintf("INSERT has more expressions than target columns")

	errorResp := &pgproto3.ErrorResponse{
		Severity: "ERROR",
		SeverityUnlocalized: "",
		Code: "tobe",
		Detail: "",
		Hint: "",
	}
	setFilePathAndLine(te, errorResp)
	errorResp.Message, errorResp.Position = pgMsg, int32(position)

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
		Code: "tobe",
		Detail: "",
		Hint: "",
	}
	setFilePathAndLine(te, errorResp)
	errorResp.Message, errorResp.Position = pgMsg, int32(position)

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
		Code: "tobe",
		Detail: "",
		Hint: "",
	}
	setFilePathAndLine(te, errorResp)
	errorResp.Message, errorResp.Position = pgMsg, int32(position)

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
