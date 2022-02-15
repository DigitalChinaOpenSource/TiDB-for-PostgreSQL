package server

import (
	"context"
	"encoding/binary"
	"github.com/jackc/pgproto3/v2"
	pgOID "github.com/lib/pq/oid"
	"github.com/pingcap/errors"
	"github.com/pingcap/tidb/kv"
	"github.com/pingcap/tidb/parser/mysql"
	"github.com/pingcap/tidb/parser/terror"
	plannercore "github.com/pingcap/tidb/planner/core"
	"github.com/pingcap/tidb/sessionctx/stmtctx"
	"github.com/pingcap/tidb/sessionctx/variable"
	storeerr "github.com/pingcap/tidb/store/driver/error"
	"github.com/pingcap/tidb/types"
	"github.com/pingcap/tidb/util/execdetails"
	"github.com/pingcap/tidb/util/hack"
	"github.com/pingcap/tidb/util/topsql"
	"github.com/tikv/client-go/v2/util"
	"math"
	"runtime/trace"
	"strconv"
	"strings"
	"time"
)

// handleStmtPrepare handle prepare message in pgsql's extended query.
// PgSQL Modified
func (cc *clientConn) handleStmtPrepare(ctx context.Context, parse pgproto3.Parse) error {
	//stmt, columns, params, err := cc.ctx.Prepare(parser.Query)
	stmt, _, _, err := cc.ctx.Prepare(parse.Query, parse.Name)

	if err != nil {
		return err
	}

	vars := cc.ctx.GetSessionVars()

	// Get param types in sql plan, and save it in `stmt`.
	var paramTypes []byte
	numberOfParams := stmt.NumParams()

	// parameters' oid sent in by frontend
	oids := parse.ParameterOIDs

	// here we get cacheParams from our prepared plan.
	// if oid is not available, we use our prepared plan.
	cacheStmt := vars.PreparedStmts[uint32(stmt.ID())].(*plannercore.CachedPrepareStmt)
	cacheParams := cacheStmt.PreparedAst.Params

	// If frontend send in OIDs, we save them into stmt
	if len(oids) > 0 {
		stmt.SetOIDs(oids)
	}

	for i := 0; i < numberOfParams; i++ {
		// check if oids are available, there are three situations in which oid is not available:
		// 1. Frontend didn't send them, since it's optional
		// 2. Frontend didn't send enough of them
		// 3. Frontend send in oid of 0, indicate unspecified
		if len(oids) > i && oids[i] != 0 {
			// If frontend gives param OID, we convert it to paramTypes directly
			paramTypes = append(paramTypes, pgOIDToMySQLType(oids[i]))
		} else {
			// If OID is not available, we get it from our prepared statement
			paramTypes = append(paramTypes, cacheParams[i].GetType().Tp)
		}
	}

	stmt.SetParamsType(paramTypes)

	return cc.writeParseComplete()
}

// handleStmtBind handle bind messages in pgsql's extended query.
// PGSQL Modified
func (cc *clientConn) handleStmtBind(ctx context.Context, bind pgproto3.Bind) (err error) {
	vars := cc.ctx.GetSessionVars()

	// When it is a temporary prepared stmt, the default name setting is 0
	if bind.PreparedStatement == "" {
		bind.PreparedStatement = "0"
	}

	// Get stmtID through stmt name.
	stmtID, ok := vars.PreparedStmtNameToID[bind.PreparedStatement]
	if !ok {
		return mysql.NewErr(mysql.ErrUnknownStmtHandler,
			strconv.FormatUint(uint64(stmtID), 10), "stmt_bind")
	}

	stmt := cc.ctx.GetStatement(int(stmtID))
	if stmt == nil {
		return mysql.NewErr(mysql.ErrUnknownStmtHandler,
			strconv.FormatUint(uint64(stmtID), 10), "stmt_bind")
	}

	numParams := stmt.NumParams()
	if numParams != len(bind.Parameters) {
		return mysql.ErrMalformPacket
	}

	if numParams > 0 {
		paramTypes := stmt.GetParamsType()

		args := make([]types.Datum, numParams)
		err = parseBindArgs(cc.ctx.GetSessionVars().StmtCtx, args, paramTypes, bind, stmt.BoundParams(), stmt.GetOIDs())
		stmt.Reset()
		if err != nil {
			return errors.Annotate(err, cc.preparedStmt2String(stmtID))
		}

		stmt.SetArgs(args)
	}

	// When the length of `ResultFormatCodes` equ that the data format of the whole row is set at one time.
	// If the length is greater than 1 and less than column length,
	// it means that there is a problem in the parameter transfer of the client.
	if len(bind.ResultFormatCodes) > 1 && len(bind.ResultFormatCodes) < len(stmt.GetColumnInfo()) {
		return errors.New("the result format code parameter in the bind message is wrong")
	}

	stmt.SetResultFormat(bind.ResultFormatCodes)

	// When create `Portal`, clients will send the portal name.
	// If portal name is empty, it will be set to "0" by default.
	// Notice: there is not a real portal by created,
	// we just put portal name and stmtID in map, then you can get stmtID.
	if bind.DestinationPortal != "" {
		vars.Portal[bind.DestinationPortal] = stmtID
	} else {
		vars.Portal["0"] = stmtID
	}

	return cc.writeBindComplete()
}

// handleStmtDescription handle Description messages in pgsql's extended query，
// find prepared stmt through `stmtName` or `portal`.
// Return `writeParameterDescription` and `WriteRowDescription` when columnInfo is not empty,
// otherwise return `writeNoData`.
func (cc *clientConn) handleStmtDescription(ctx context.Context, desc pgproto3.Describe) error {
	vars := cc.ctx.GetSessionVars()

	// Whether stmt name or portal name, when it is a temporary statement, the default name is "0".
	if desc.Name == "" {
		desc.Name = "0"
	}

	var stmtID uint32
	var ok bool
	var isPortal bool

	// If it specifies the prepared statement through portal,
	// here can directly find the corresponding stmtID through portal.
	if desc.ObjectType == 'P' {
		stmtID, ok = vars.Portal[desc.Name]
		isPortal = true
	} else {
		// Or get prepared stmtID through stmtName.
		stmtID, ok = vars.PreparedStmtNameToID[desc.Name]
		isPortal = false
	}

	if !ok {
		return mysql.NewErr(mysql.ErrUnknownStmtHandler,
			strconv.FormatUint(uint64(stmtID), 10), "stmt_description")
	}

	// Get prepared stmt through stmtID.
	stmt := cc.ctx.GetStatement(int(stmtID))
	if stmt == nil {
		return mysql.NewErr(mysql.ErrUnknownStmtHandler,
			strconv.FormatUint(uint64(stmtID), 10), "stmt_description")
	}

	// we send parameter description only if this is statement and not a portal
	// https://www.postgresql.org/docs/current/protocol-flow.html#PROTOCOL-FLOW-EXT-QUERY
	if !isPortal {
		// Get param types that analyzed in `handleStmtBind`,

		pgOIDs := stmt.GetOIDs()
		if len(pgOIDs) == 0 {
			paramsType := stmt.GetParamsType()
			pgOIDs = make([]uint32, stmt.NumParams())
			for i := range paramsType {
				pgOIDs[i] = convertMySQLDataTypeToPgSQLDataType(paramsType[i])
			}
		}

		if err := cc.writeParameterDescription(pgOIDs); err != nil {
			return err
		}
	}

	// Return `WriteRowDescription` when columnInfo is not empty,
	// otherwise return `writeNoData`.
	columnInfo := stmt.GetColumnInfo()
	if columnInfo == nil || len(columnInfo) > 0 {
		return cc.WriteRowDescription(columnInfo)
	}
	// If the row description information has been output here,
	// it will not be output when `writeResultset` later.
	return cc.writeNoData()
}

// handleStmtExecute handle execute messages in pgsql's extended query.
// PGSQL Modified
func (cc *clientConn) handleStmtExecute(ctx context.Context, execute pgproto3.Execute) (err error) {
	defer trace.StartRegion(ctx, "HandleStmtExecute").End()

	if execute.Portal == "" {
		execute.Portal = "0"
	}

	stmtID, ok := cc.preparedPortal2StmtID(execute.Portal)
	if !ok {
		return mysql.NewErr(mysql.ErrUnknownStmtHandler,
			strconv.FormatUint(uint64(stmtID), 10), "stmt_description")
	}

	if variable.TopSQLEnabled() {
		preparedStmt, _ := cc.preparedStmtID2CachePreparedStmt(stmtID)
		//preparedStmt, _ := cc.preparedStmtID2CachePreparedStmt(execute.Portal)
		if preparedStmt != nil && preparedStmt.SQLDigest != nil {
			ctx = topsql.AttachSQLInfo(ctx, preparedStmt.NormalizedSQL, preparedStmt.SQLDigest, "", nil, false)
		}
	}

	stmt := cc.ctx.GetStatement(int(stmtID))
	if stmt == nil {
		return mysql.NewErr(mysql.ErrUnknownStmtHandler,
			strconv.FormatUint(uint64(stmtID), 10), "stmt_execute")
	}

	ctx = context.WithValue(ctx, execdetails.StmtExecDetailKey, &execdetails.StmtExecDetails{})
	ctx = context.WithValue(ctx, util.ExecDetailsKey, &util.ExecDetails{})
	retryable, err := cc.executePreparedStmtAndWriteResult(ctx, stmt, false)
	_, allowTiFlashFallback := cc.ctx.GetSessionVars().AllowFallbackToTiKV[kv.TiFlash]
	if allowTiFlashFallback && err != nil && errors.ErrorEqual(err, storeerr.ErrTiFlashServerTimeout) && retryable {
		// When the TiFlash server seems down, we append a warning to remind the user to check the status of the TiFlash
		// server and fallback to TiKV.
		prevErr := err
		delete(cc.ctx.GetSessionVars().IsolationReadEngines, kv.TiFlash)
		defer func() {
			cc.ctx.GetSessionVars().IsolationReadEngines[kv.TiFlash] = struct{}{}
		}()
		_, err = cc.executePreparedStmtAndWriteResult(ctx, stmt, false)
		// We append warning after the retry because `ResetContextOfStmt` may be called during the retry, which clears warnings.
		cc.ctx.GetSessionVars().StmtCtx.AppendError(prevErr)
	}
	return err
}

// The first return value indicates whether the call of executePreparedStmtAndWriteResult has no side effect and can be retried.
// Currently the first return value is used to fallback to TiKV when TiFlash is down.
func (cc *clientConn) executePreparedStmtAndWriteResult(ctx context.Context, stmt PreparedStatement, useCursor bool) (bool, error) {
	args := stmt.GetArgs()
	rs, err := stmt.Execute(ctx, args)
	if err != nil {
		return true, errors.Annotate(err, cc.preparedStmt2String(uint32(stmt.ID())))
	}
	if rs == nil {
		return false, cc.writeCommandComplete()
	}

	// if the client wants to use cursor
	// we should hold the ResultSet in PreparedStatement for next stmt_fetch, and only send back ColumnInfo.
	// Tell the client cursor exists in server by setting proper serverStatus.
	if useCursor {
		cc.initResultEncoder(ctx)
		defer cc.rsEncoder.clean()
		stmt.StoreResultSet(rs)
		err = cc.writeColumnInfo(rs.Columns(), mysql.ServerStatusCursorExists)
		if err != nil {
			return false, err
		}
		if cl, ok := rs.(fetchNotifier); ok {
			cl.OnFetchReturned()
		}
		// explicitly flush columnInfo to client.
		return false, cc.flush(ctx)
	}
	defer terror.Call(rs.Close)
	retryable, err := cc.writeResultset(ctx, rs, stmt.GetResultFormat(), 0, 0)
	if err != nil {
		return retryable, errors.Annotate(err, cc.preparedStmt2String(uint32(stmt.ID())))
	}
	return false, nil
}

func (cc *clientConn) handleStmtFetch(ctx context.Context, data []byte) (err error) {
	cc.ctx.GetSessionVars().StartTime = time.Now()

	stmtID, fetchSize, err := parseStmtFetchCmd(data)
	if err != nil {
		return err
	}

	stmt := cc.ctx.GetStatement(int(stmtID))
	if stmt == nil {
		return errors.Annotate(mysql.NewErr(mysql.ErrUnknownStmtHandler,
			strconv.FormatUint(uint64(stmtID), 10), "stmt_fetch"), cc.preparedStmt2String(stmtID))
	}
	if variable.TopSQLEnabled() {
		prepareObj, _ := cc.preparedStmtID2CachePreparedStmt(stmtID)
		if prepareObj != nil && prepareObj.SQLDigest != nil {
			ctx = topsql.AttachSQLInfo(ctx, prepareObj.NormalizedSQL, prepareObj.SQLDigest, "", nil, false)
		}
	}
	sql := ""
	if prepared, ok := cc.ctx.GetStatement(int(stmtID)).(*TiDBStatement); ok {
		sql = prepared.sql
	}
	cc.ctx.SetProcessInfo(sql, time.Now(), mysql.ComStmtExecute, 0)
	rs := stmt.GetResultSet()
	if rs == nil {
		return errors.Annotate(mysql.NewErr(mysql.ErrUnknownStmtHandler,
			strconv.FormatUint(uint64(stmtID), 10), "stmt_fetch_rs"), cc.preparedStmt2String(stmtID))
	}

	_, err = cc.writeResultset(ctx, rs, []int16{1}, mysql.ServerStatusCursorExists, int(fetchSize))
	if err != nil {
		return errors.Annotate(err, cc.preparedStmt2String(stmtID))
	}
	return nil
}

// handleStmtClose handle close messages in pgsql's extended query.
func (cc *clientConn) handleStmtClose(ctx context.Context, close pgproto3.Close) error {
	vars := cc.ctx.GetSessionVars()
	var stmtID uint32
	if close.ObjectType == 'S' {
		stmtID = vars.PreparedStmtNameToID[close.Name]
	} else {
		stmtID = vars.Portal[close.Name]
	}

	stmt := cc.ctx.GetStatement(int(stmtID))
	if stmt != nil {
		return stmt.Close()
	}
	if err := cc.writeCloseComplete(); err != nil {
		return err
	}
	return cc.flush(ctx)
}

func (cc *clientConn) preparedPortal2StmtID(portal string) (uint32, bool) {
	sv := cc.ctx.GetSessionVars()
	if sv == nil {
		return 0, false
	}

	stmtID, ok := sv.Portal[portal]
	if !ok {
		return 0, false
	}

	return stmtID, true
}

// parseBindArgs 将客户端传来的参数值解析为 Datum 结构
// PgSQL Modified
func parseBindArgs(sc *stmtctx.StatementContext, args []types.Datum, paramTypes []byte, bind pgproto3.Bind, boundParams [][]byte, pgOIDs []uint32) error {
	// todo 传参为文本 text 格式时候的处理

	hasOID := len(pgOIDs) > 0
	for i := 0; i < len(args); i++ {

		// todo 使用boundParams

		if bind.Parameters[i] == nil {
			var nilDatum types.Datum
			nilDatum.SetNull()
			args[i] = nilDatum
			continue
		}

		// todo isUnsigned
		// isUnsigned 暂时Pg无法判断, 默认为有符号
		isUnsigned := false

		switch paramTypes[i] {
		case mysql.TypeNull:
			var nilDatum types.Datum
			nilDatum.SetNull()
			args[i] = nilDatum
			continue
		case mysql.TypeTiny:
			{
				if isUnsigned {
					args[i] = types.NewUintDatum(uint64(uint8(bind.Parameters[i][0])))
				} else {
					args[i] = types.NewIntDatum(int64(int8(bind.Parameters[i][0])))
				}
				continue
			}

		case mysql.TypeShort, mysql.TypeYear:
			valInt, err := strconv.Atoi(string(bind.Parameters[i]))
			if err != nil {
				return err
			}
			if isUnsigned {
				args[i] = types.NewUintDatum(uint64(uint16(valInt)))
			} else {
				args[i] = types.NewIntDatum(int64(int16(valInt)))
			}
			continue

		case mysql.TypeInt24, mysql.TypeLong:
			if bind.ParameterFormatCodes[i] == 1 { // The data passed in is in binary format
				var b [8]byte
				copy(b[8-len(bind.Parameters[i]):], bind.Parameters[i])
				val := binary.BigEndian.Uint64(b[:])
				args[i] = types.NewUintDatum(val)
				continue
			}
			valInt, err := strconv.Atoi(string(bind.Parameters[i]))
			if err != nil {
				return err
			}
			if isUnsigned {
				args[i] = types.NewUintDatum(uint64(uint32(valInt)))
			} else {
				args[i] = types.NewIntDatum(int64(int32(valInt)))
			}
			continue

		case mysql.TypeLonglong:
			valInt, err := strconv.Atoi(string(bind.Parameters[i]))
			if err != nil {
				return err
			}
			if isUnsigned {
				args[i] = types.NewUintDatum(uint64(valInt))
			} else {
				args[i] = types.NewIntDatum(int64(valInt))
			}
			continue

		case mysql.TypeFloat:
			valFloat, err := strconv.ParseFloat(string(bind.Parameters[i]), 32)
			if err != nil {
				return err
			}
			args[i] = types.NewFloat32Datum(float32(valFloat))
			continue

		case mysql.TypeDouble:
			valFloat, err := strconv.ParseFloat(string(bind.Parameters[i]), 64)
			if err != nil {
				return err
			}
			args[i] = types.NewFloat64Datum(valFloat)
			continue
		case mysql.TypeTimestamp:
			// we ignore timezone here
			timeStr := string(bind.Parameters[i])
			tzIndex := strings.Index(timeStr, " +")
			if tzIndex == -1 {
				args[i] = types.NewDatum(timeStr)
				continue
			}
			noTzStr := timeStr[:tzIndex]
			args[i] = types.NewDatum(noTzStr)
			continue
		case mysql.TypeDate, mysql.TypeDatetime:
			// fixme 日期待测试 待修复
			args[i] = types.NewDatum(string(bind.Parameters[i]))
			continue

		case mysql.TypeDuration:
			// fixme 日期待测试 待修复
			args[i] = types.NewDatum(string(bind.Parameters[i]))
			continue
		case mysql.TypeNewDecimal:
			// fixme decimal 待测试 待修复
			var dec types.MyDecimal
			if bind.ParameterFormatCodes[i] == 1 {
				bits := binary.BigEndian.Uint64(bind.Parameters[i])
				f64 := math.Float64frombits(bits)
				args[i] = types.NewFloat64Datum(f64)
				continue
			}
			err := sc.HandleTruncate(dec.FromString(bind.Parameters[i]))
			if err != nil {
				return err
			}
			args[i] = types.NewDecimalDatum(&dec)
			continue

		case mysql.TypeBlob, mysql.TypeTinyBlob, mysql.TypeMediumBlob, mysql.TypeLongBlob:
			// fixme 待测试 待修复
			args[i] = types.NewBytesDatum(bind.Parameters[i])
			continue

		case mysql.TypeVarchar, mysql.TypeVarString, mysql.TypeString,
			mysql.TypeEnum, mysql.TypeSet, mysql.TypeGeometry, mysql.TypeBit:
			tmp := string(hack.String(bind.Parameters[i]))
			args[i] = types.NewDatum(tmp)
			continue
		case mysql.TypeUnspecified:
			if hasOID {
				if bind.ParameterFormatCodes[i] == 1 && pgOIDs[i] == 23 { // The data passed in is in binary format
					args[i] = types.NewBinaryLiteralDatum(bind.Parameters[i])
					continue
				}

				if bind.ParameterFormatCodes[i] == 1 && pgOIDs[i] == 701 { // The data passed in is in binary format
					bits := binary.BigEndian.Uint64(bind.Parameters[i])
					f64 := math.Float64frombits(bits)
					args[i] = types.NewFloat64Datum(f64)
					continue
				}
			}
			tmp := string(hack.String(bind.Parameters[i]))
			args[i] = types.NewDatum(tmp)
			continue
		default:
			err := errUnknownFieldType.GenWithStack("stmt unknown field type %d", paramTypes[i])
			return err
		}
	}

	return nil
}

// pgOIDToMySQLType converts postgres OID into mysql type
func pgOIDToMySQLType(oid uint32) byte {
	switch pgOID.Oid(oid) {
	case pgOID.T_int8:
		return mysql.TypeLonglong
	case pgOID.T_int4:
		return mysql.TypeLong
	case pgOID.T_int2:
		return mysql.TypeShort
	case pgOID.T_float4:
		return mysql.TypeFloat
	case pgOID.T_float8:
		return mysql.TypeDouble
	case pgOID.T_timestamp:
		return mysql.TypeTimestamp
	case pgOID.T_date:
		return mysql.TypeNewDate
	case pgOID.T_numeric:
		return mysql.TypeNewDecimal
	case pgOID.T_bytea:
		return mysql.TypeBlob
	case pgOID.T_text, pgOID.T_varchar:
		return mysql.TypeVarchar
	default:
		return mysql.TypeUnspecified
	}
}
