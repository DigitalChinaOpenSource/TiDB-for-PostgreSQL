// Copyright 2013 The Go-MySQL-Driver Authors. All rights reserved.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at http://mozilla.org/MPL/2.0/.

// The MIT License (MIT)
//
// Copyright (c) 2014 wandoulabs
// Copyright (c) 2014 siddontang
//
// Permission is hereby granted, free of charge, to any person obtaining a copy of
// this software and associated documentation files (the "Software"), to deal in
// the Software without restriction, including without limitation the rights to
// use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of
// the Software, and to permit persons to whom the Software is furnished to do so,
// subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.

// Copyright 2015 PingCAP, Inc.
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
	"context"
	"encoding/binary"
	"fmt"
	"github.com/jackc/pgproto3/v2"
	"github.com/pingcap/tidb/util/execdetails"
	"math"
	"runtime/trace"
	"strconv"
	"strings"
	"time"

	"github.com/DigitalChinaOpenSource/DCParser/mysql"
	pgOID "github.com/lib/pq/oid"
	"github.com/pingcap/errors"
	plannercore "github.com/pingcap/tidb/planner/core"
	"github.com/pingcap/tidb/sessionctx/stmtctx"
	"github.com/pingcap/tidb/types"
	"github.com/pingcap/tidb/util/hack"
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
func (cc *clientConn) handleStmtExecute(ctx context.Context, execute pgproto3.Execute) error {
	defer trace.StartRegion(ctx, "HandleStmtExecute").End()

	// When it is a temporary prepared stmt, the default name setting is "0".
	if execute.Portal == "" {
		execute.Portal = "0"
	}

	vars := cc.ctx.GetSessionVars()

	stmtID, ok := vars.Portal[execute.Portal]
	if !ok {
		return mysql.NewErr(mysql.ErrUnknownStmtHandler,
			strconv.FormatUint(uint64(stmtID), 10), "stmt_description")
	}

	stmt := cc.ctx.GetStatement(int(stmtID))
	args := stmt.GetArgs()

	ctx = context.WithValue(ctx, execdetails.StmtExecDetailKey, &execdetails.StmtExecDetails{})
	rs, err := stmt.Execute(ctx, args)
	if err != nil {
		return errors.Annotate(err, cc.preparedStmt2String(stmtID))
	}

	if rs == nil {
		return cc.writeCommandComplete()
	}
	err = cc.writeResultset(ctx, rs, stmt.GetResultFormat(), 0, 0)
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

// maxFetchSize constants
const (
	maxFetchSize = 1024
)

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

	err = cc.writeResultset(ctx, rs, []int16{1}, mysql.ServerStatusCursorExists, int(fetchSize))
	if err != nil {
		return errors.Annotate(err, cc.preparedStmt2String(stmtID))
	}
	return nil
}

func parseStmtFetchCmd(data []byte) (uint32, uint32, error) {
	if len(data) != 8 {
		return 0, 0, mysql.ErrMalformPacket
	}
	// Please refer to https://dev.mysql.com/doc/internals/en/com-stmt-fetch.html
	stmtID := binary.LittleEndian.Uint32(data[0:4])
	fetchSize := binary.LittleEndian.Uint32(data[4:8])
	if fetchSize > maxFetchSize {
		fetchSize = maxFetchSize
	}
	return stmtID, fetchSize, nil
}

// getFormatCode decode the formatCodes passed in from Bind struct
// it will return 0 if the format is Text, 1 if Binary
// Note that this will handle empty formatCodes and single format Codes gracefully
func getFormatCode(formatCodes []int16, index int) int16 {
	// default format is Text
	if len(formatCodes) == 0 {
		return 0
	}

	// if length is one, use that for all arguments
	if len(formatCodes) == 1 {
		return formatCodes[0]
	}

	return formatCodes[index]
}

// parseBindArgs 将客户端传来的参数值解析为 Datum 结构
// PgSQL Modified
func parseBindArgs(sc *stmtctx.StatementContext, args []types.Datum, paramTypes []byte, bind pgproto3.Bind, boundParams [][]byte, pgOIDs []uint32) error {

	var (
		formatCode int16
		isUnsigned bool
	)

	for i := 0; i < len(args); i++ {

		// todo BoundParams
		formatCode = getFormatCode(bind.ParameterFormatCodes, i)
		// todo Check If variable should be signed, currently we are assuming all signed
		isUnsigned = false

		if bind.Parameters[i] == nil {
			var nilDatum types.Datum
			nilDatum.SetNull()
			args[i] = nilDatum
			continue
		}

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
			if formatCode == 1 { // The data passed in is in binary format
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
			if formatCode == 1 {
				bits := binary.BigEndian.Uint32(bind.Parameters[i])
				f32 := math.Float32frombits(bits)
				args[i] = types.NewFloat32Datum(f32)
				continue
			}
			valFloat, err := strconv.ParseFloat(string(bind.Parameters[i]), 32)
			if err != nil {
				return err
			}
			args[i] = types.NewFloat32Datum(float32(valFloat))
			continue

		case mysql.TypeDouble:
			if formatCode == 1 {
				bits := binary.BigEndian.Uint64(bind.Parameters[i])
				f64 := math.Float64frombits(bits)
				args[i] = types.NewFloat64Datum(f64)
				continue
			}
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
			if formatCode == 1 {
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

func parseExecArgs(sc *stmtctx.StatementContext, args []types.Datum, boundParams [][]byte, nullBitmap, paramTypes, paramValues []byte) (err error) {
	pos := 0
	var (
		tmp    interface{}
		v      []byte
		n      int
		isNull bool
	)

	for i := 0; i < len(args); i++ {
		// if params had received via ComStmtSendLongData, use them directly.
		// ref https://dev.mysql.com/doc/internals/en/com-stmt-send-long-data.html
		// see clientConn#handleStmtSendLongData
		if boundParams[i] != nil {
			args[i] = types.NewBytesDatum(boundParams[i])
			continue
		}

		// check nullBitMap to determine the NULL arguments.
		// ref https://dev.mysql.com/doc/internals/en/com-stmt-execute.html
		// notice: some client(e.g. mariadb) will set nullBitMap even if data had be sent via ComStmtSendLongData,
		// so this check need place after boundParam's check.
		if nullBitmap[i>>3]&(1<<(uint(i)%8)) > 0 {
			var nilDatum types.Datum
			nilDatum.SetNull()
			args[i] = nilDatum
			continue
		}

		if (i<<1)+1 >= len(paramTypes) {
			return mysql.ErrMalformPacket
		}

		tp := paramTypes[i<<1]
		isUnsigned := (paramTypes[(i<<1)+1] & 0x80) > 0

		switch tp {
		case mysql.TypeNull:
			var nilDatum types.Datum
			nilDatum.SetNull()
			args[i] = nilDatum
			continue

		case mysql.TypeTiny:
			if len(paramValues) < (pos + 1) {
				err = mysql.ErrMalformPacket
				return
			}

			if isUnsigned {
				args[i] = types.NewUintDatum(uint64(uint8(paramValues[pos])))
			} else {
				args[i] = types.NewIntDatum(int64(int8(paramValues[pos])))
			}

			pos++
			continue

		case mysql.TypeShort, mysql.TypeYear:
			if len(paramValues) < (pos + 2) {
				err = mysql.ErrMalformPacket
				return
			}
			valU16 := binary.LittleEndian.Uint16(paramValues[pos : pos+2])
			if isUnsigned {
				args[i] = types.NewUintDatum(uint64(uint16(valU16)))
			} else {
				args[i] = types.NewIntDatum(int64(int16(valU16)))
			}
			pos += 2
			continue

		case mysql.TypeInt24, mysql.TypeLong:
			if len(paramValues) < (pos + 4) {
				err = mysql.ErrMalformPacket
				return
			}
			valU32 := binary.LittleEndian.Uint32(paramValues[pos : pos+4])
			if isUnsigned {
				args[i] = types.NewUintDatum(uint64(uint32(valU32)))
			} else {
				args[i] = types.NewIntDatum(int64(int32(valU32)))
			}
			pos += 4
			continue

		case mysql.TypeLonglong:
			if len(paramValues) < (pos + 8) {
				err = mysql.ErrMalformPacket
				return
			}
			valU64 := binary.LittleEndian.Uint64(paramValues[pos : pos+8])
			if isUnsigned {
				args[i] = types.NewUintDatum(valU64)
			} else {
				args[i] = types.NewIntDatum(int64(valU64))
			}
			pos += 8
			continue

		case mysql.TypeFloat:
			if len(paramValues) < (pos + 4) {
				err = mysql.ErrMalformPacket
				return
			}

			args[i] = types.NewFloat32Datum(math.Float32frombits(binary.LittleEndian.Uint32(paramValues[pos : pos+4])))
			pos += 4
			continue

		case mysql.TypeDouble:
			if len(paramValues) < (pos + 8) {
				err = mysql.ErrMalformPacket
				return
			}

			args[i] = types.NewFloat64Datum(math.Float64frombits(binary.LittleEndian.Uint64(paramValues[pos : pos+8])))
			pos += 8
			continue

		case mysql.TypeDate, mysql.TypeTimestamp, mysql.TypeDatetime:
			if len(paramValues) < (pos + 1) {
				err = mysql.ErrMalformPacket
				return
			}
			// See https://dev.mysql.com/doc/internals/en/binary-protocol-value.html
			// for more details.
			length := uint8(paramValues[pos])
			pos++
			switch length {
			case 0:
				tmp = types.ZeroDatetimeStr
			case 4:
				pos, tmp = parseBinaryDate(pos, paramValues)
			case 7:
				pos, tmp = parseBinaryDateTime(pos, paramValues)
			case 11:
				pos, tmp = parseBinaryTimestamp(pos, paramValues)
			default:
				err = mysql.ErrMalformPacket
				return
			}
			args[i] = types.NewDatum(tmp) // FIXME: After check works!!!!!!
			continue

		case mysql.TypeDuration:
			if len(paramValues) < (pos + 1) {
				err = mysql.ErrMalformPacket
				return
			}
			// See https://dev.mysql.com/doc/internals/en/binary-protocol-value.html
			// for more details.
			length := uint8(paramValues[pos])
			pos++
			switch length {
			case 0:
				tmp = "0"
			case 8:
				isNegative := uint8(paramValues[pos])
				if isNegative > 1 {
					err = mysql.ErrMalformPacket
					return
				}
				pos++
				pos, tmp = parseBinaryDuration(pos, paramValues, isNegative)
			case 12:
				isNegative := uint8(paramValues[pos])
				if isNegative > 1 {
					err = mysql.ErrMalformPacket
					return
				}
				pos++
				pos, tmp = parseBinaryDurationWithMS(pos, paramValues, isNegative)
			default:
				err = mysql.ErrMalformPacket
				return
			}
			args[i] = types.NewDatum(tmp)
			continue
		case mysql.TypeNewDecimal:
			if len(paramValues) < (pos + 1) {
				err = mysql.ErrMalformPacket
				return
			}

			v, isNull, n, err = parseLengthEncodedBytes(paramValues[pos:])
			pos += n
			if err != nil {
				return
			}

			if isNull {
				args[i] = types.NewDecimalDatum(nil)
			} else {
				var dec types.MyDecimal
				err = sc.HandleTruncate(dec.FromString(v))
				if err != nil {
					return err
				}
				args[i] = types.NewDecimalDatum(&dec)
			}
			continue
		case mysql.TypeBlob, mysql.TypeTinyBlob, mysql.TypeMediumBlob, mysql.TypeLongBlob:
			if len(paramValues) < (pos + 1) {
				err = mysql.ErrMalformPacket
				return
			}
			v, isNull, n, err = parseLengthEncodedBytes(paramValues[pos:])
			pos += n
			if err != nil {
				return
			}

			if isNull {
				args[i] = types.NewBytesDatum(nil)
			} else {
				args[i] = types.NewBytesDatum(v)
			}
			continue
		case mysql.TypeUnspecified, mysql.TypeVarchar, mysql.TypeVarString, mysql.TypeString,
			mysql.TypeEnum, mysql.TypeSet, mysql.TypeGeometry, mysql.TypeBit:
			if len(paramValues) < (pos + 1) {
				err = mysql.ErrMalformPacket
				return
			}

			v, isNull, n, err = parseLengthEncodedBytes(paramValues[pos:])
			pos += n
			if err != nil {
				return
			}

			if !isNull {
				tmp = string(hack.String(v))
			} else {
				tmp = nil
			}
			args[i] = types.NewDatum(tmp)
			continue
		default:
			err = errUnknownFieldType.GenWithStack("stmt unknown field type %d", tp)
			return
		}
	}
	return
}

func parseBinaryDate(pos int, paramValues []byte) (int, string) {
	year := binary.LittleEndian.Uint16(paramValues[pos : pos+2])
	pos += 2
	month := uint8(paramValues[pos])
	pos++
	day := uint8(paramValues[pos])
	pos++
	return pos, fmt.Sprintf("%04d-%02d-%02d", year, month, day)
}

func parseBinaryDateTime(pos int, paramValues []byte) (int, string) {
	pos, date := parseBinaryDate(pos, paramValues)
	hour := uint8(paramValues[pos])
	pos++
	minute := uint8(paramValues[pos])
	pos++
	second := uint8(paramValues[pos])
	pos++
	return pos, fmt.Sprintf("%s %02d:%02d:%02d", date, hour, minute, second)
}

func parseBinaryTimestamp(pos int, paramValues []byte) (int, string) {
	pos, dateTime := parseBinaryDateTime(pos, paramValues)
	microSecond := binary.LittleEndian.Uint32(paramValues[pos : pos+4])
	pos += 4
	return pos, fmt.Sprintf("%s.%06d", dateTime, microSecond)
}

func parseBinaryDuration(pos int, paramValues []byte, isNegative uint8) (int, string) {
	sign := ""
	if isNegative == 1 {
		sign = "-"
	}
	days := binary.LittleEndian.Uint32(paramValues[pos : pos+4])
	pos += 4
	hours := uint8(paramValues[pos])
	pos++
	minutes := uint8(paramValues[pos])
	pos++
	seconds := uint8(paramValues[pos])
	pos++
	return pos, fmt.Sprintf("%s%d %02d:%02d:%02d", sign, days, hours, minutes, seconds)
}

func parseBinaryDurationWithMS(pos int, paramValues []byte,
	isNegative uint8) (int, string) {
	pos, dur := parseBinaryDuration(pos, paramValues, isNegative)
	microSecond := binary.LittleEndian.Uint32(paramValues[pos : pos+4])
	pos += 4
	return pos, fmt.Sprintf("%s.%06d", dur, microSecond)
}

func (cc *clientConn) handleStmtSendLongData(data []byte) (err error) {
	if len(data) < 6 {
		return mysql.ErrMalformPacket
	}

	stmtID := int(binary.LittleEndian.Uint32(data[0:4]))

	stmt := cc.ctx.GetStatement(stmtID)
	if stmt == nil {
		return mysql.NewErr(mysql.ErrUnknownStmtHandler,
			strconv.Itoa(stmtID), "stmt_send_longdata")
	}

	paramID := int(binary.LittleEndian.Uint16(data[4:6]))
	return stmt.AppendParam(paramID, data[6:])
}

func (cc *clientConn) handleStmtReset(ctx context.Context, data []byte) (err error) {
	if len(data) < 4 {
		return mysql.ErrMalformPacket
	}

	stmtID := int(binary.LittleEndian.Uint32(data[0:4]))
	stmt := cc.ctx.GetStatement(stmtID)
	if stmt == nil {
		return mysql.NewErr(mysql.ErrUnknownStmtHandler,
			strconv.Itoa(stmtID), "stmt_reset")
	}
	stmt.Reset()
	stmt.StoreResultSet(nil)
	return cc.writeOK(ctx)
}

// handleSetOption refer to https://dev.mysql.com/doc/internals/en/com-set-option.html
func (cc *clientConn) handleSetOption(ctx context.Context, data []byte) (err error) {
	if len(data) < 2 {
		return mysql.ErrMalformPacket
	}

	switch binary.LittleEndian.Uint16(data[:2]) {
	case 0:
		cc.capability |= mysql.ClientMultiStatements
		cc.ctx.SetClientCapability(cc.capability)
	case 1:
		cc.capability &^= mysql.ClientMultiStatements
		cc.ctx.SetClientCapability(cc.capability)
	default:
		return mysql.ErrMalformPacket
	}
	if err = cc.writeEOF(0); err != nil {
		return err
	}

	return cc.flush(ctx)
}

func (cc *clientConn) preparedStmt2String(stmtID uint32) string {
	sv := cc.ctx.GetSessionVars()
	if sv == nil {
		return ""
	}
	if sv.EnableRedactLog {
		return cc.preparedStmt2StringNoArgs(stmtID)
	}
	return cc.preparedStmt2StringNoArgs(stmtID) + sv.PreparedParams.String()
}

func (cc *clientConn) preparedStmt2StringNoArgs(stmtID uint32) string {
	sv := cc.ctx.GetSessionVars()
	if sv == nil {
		return ""
	}
	preparedPointer, ok := sv.PreparedStmts[stmtID]
	if !ok {
		return "prepared statement not found, ID: " + strconv.FormatUint(uint64(stmtID), 10)
	}
	preparedObj, ok := preparedPointer.(*plannercore.CachedPrepareStmt)
	if !ok {
		return "invalidate CachedPrepareStmt type, ID: " + strconv.FormatUint(uint64(stmtID), 10)
	}
	preparedAst := preparedObj.PreparedAst
	return preparedAst.Stmt.Text()
}
