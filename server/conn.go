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
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/binary"
	"fmt"
	"github.com/jackc/pgio"
	"github.com/jackc/pgproto3/v2"
	"github.com/jackc/pgtype"
	"io"
	"io/ioutil"
	"net"
	"runtime"
	"runtime/pprof"
	"runtime/trace"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/DigitalChinaOpenSource/DCParser"
	"github.com/DigitalChinaOpenSource/DCParser/ast"
	"github.com/DigitalChinaOpenSource/DCParser/auth"
	"github.com/DigitalChinaOpenSource/DCParser/mysql"
	"github.com/DigitalChinaOpenSource/DCParser/terror"
	"github.com/opentracing/opentracing-go"
	"github.com/pingcap/errors"
	"github.com/pingcap/failpoint"
	"github.com/pingcap/tidb/config"
	"github.com/pingcap/tidb/executor"
	"github.com/pingcap/tidb/kv"
	"github.com/pingcap/tidb/metrics"
	"github.com/pingcap/tidb/plugin"
	"github.com/pingcap/tidb/session"
	"github.com/pingcap/tidb/sessionctx"
	"github.com/pingcap/tidb/sessionctx/variable"
	"github.com/pingcap/tidb/util/arena"
	"github.com/pingcap/tidb/util/chunk"
	"github.com/pingcap/tidb/util/execdetails"
	"github.com/pingcap/tidb/util/hack"
	"github.com/pingcap/tidb/util/logutil"
	"github.com/pingcap/tidb/util/memory"
	"github.com/pingcap/tidb/util/sqlexec"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

const (
	connStatusDispatching int32 = iota
	connStatusReading
	connStatusShutdown     // Closed by server.
	connStatusWaitShutdown // Notified by server to close.
)

const protocolVersionNumber = 196608 // 3.0
const sslRequestNumber = 80877103
const cancelRequestCode = 80877102
const gssEncReqNumber = 80877104
const protocolSSL = false

var (
	queryTotalCountOk = [...]prometheus.Counter{
		mysql.ComSleep:            metrics.QueryTotalCounter.WithLabelValues("Sleep", "OK"),
		mysql.ComQuit:             metrics.QueryTotalCounter.WithLabelValues("Quit", "OK"),
		mysql.ComInitDB:           metrics.QueryTotalCounter.WithLabelValues("InitDB", "OK"),
		mysql.ComQuery:            metrics.QueryTotalCounter.WithLabelValues("Query", "OK"),
		mysql.ComPing:             metrics.QueryTotalCounter.WithLabelValues("Ping", "OK"),
		mysql.ComFieldList:        metrics.QueryTotalCounter.WithLabelValues("FieldList", "OK"),
		mysql.ComStmtPrepare:      metrics.QueryTotalCounter.WithLabelValues("StmtPrepare", "OK"),
		mysql.ComStmtExecute:      metrics.QueryTotalCounter.WithLabelValues("StmtExecute", "OK"),
		mysql.ComStmtFetch:        metrics.QueryTotalCounter.WithLabelValues("StmtFetch", "OK"),
		mysql.ComStmtClose:        metrics.QueryTotalCounter.WithLabelValues("StmtClose", "OK"),
		mysql.ComStmtSendLongData: metrics.QueryTotalCounter.WithLabelValues("StmtSendLongData", "OK"),
		mysql.ComStmtReset:        metrics.QueryTotalCounter.WithLabelValues("StmtReset", "OK"),
		mysql.ComSetOption:        metrics.QueryTotalCounter.WithLabelValues("SetOption", "OK"),
	}
	queryTotalCountErr = [...]prometheus.Counter{
		mysql.ComSleep:            metrics.QueryTotalCounter.WithLabelValues("Sleep", "Error"),
		mysql.ComQuit:             metrics.QueryTotalCounter.WithLabelValues("Quit", "Error"),
		mysql.ComInitDB:           metrics.QueryTotalCounter.WithLabelValues("InitDB", "Error"),
		mysql.ComQuery:            metrics.QueryTotalCounter.WithLabelValues("Query", "Error"),
		mysql.ComPing:             metrics.QueryTotalCounter.WithLabelValues("Ping", "Error"),
		mysql.ComFieldList:        metrics.QueryTotalCounter.WithLabelValues("FieldList", "Error"),
		mysql.ComStmtPrepare:      metrics.QueryTotalCounter.WithLabelValues("StmtPrepare", "Error"),
		mysql.ComStmtExecute:      metrics.QueryTotalCounter.WithLabelValues("StmtExecute", "Error"),
		mysql.ComStmtFetch:        metrics.QueryTotalCounter.WithLabelValues("StmtFetch", "Error"),
		mysql.ComStmtClose:        metrics.QueryTotalCounter.WithLabelValues("StmtClose", "Error"),
		mysql.ComStmtSendLongData: metrics.QueryTotalCounter.WithLabelValues("StmtSendLongData", "Error"),
		mysql.ComStmtReset:        metrics.QueryTotalCounter.WithLabelValues("StmtReset", "Error"),
		mysql.ComSetOption:        metrics.QueryTotalCounter.WithLabelValues("SetOption", "Error"),
	}

	queryDurationHistogramUse      = metrics.QueryDurationHistogram.WithLabelValues("Use")
	queryDurationHistogramShow     = metrics.QueryDurationHistogram.WithLabelValues("Show")
	queryDurationHistogramBegin    = metrics.QueryDurationHistogram.WithLabelValues("Begin")
	queryDurationHistogramCommit   = metrics.QueryDurationHistogram.WithLabelValues("Commit")
	queryDurationHistogramRollback = metrics.QueryDurationHistogram.WithLabelValues("Rollback")
	queryDurationHistogramInsert   = metrics.QueryDurationHistogram.WithLabelValues("Insert")
	queryDurationHistogramReplace  = metrics.QueryDurationHistogram.WithLabelValues("Replace")
	queryDurationHistogramDelete   = metrics.QueryDurationHistogram.WithLabelValues("Delete")
	queryDurationHistogramUpdate   = metrics.QueryDurationHistogram.WithLabelValues("Update")
	queryDurationHistogramSelect   = metrics.QueryDurationHistogram.WithLabelValues("Select")
	queryDurationHistogramExecute  = metrics.QueryDurationHistogram.WithLabelValues("Execute")
	queryDurationHistogramSet      = metrics.QueryDurationHistogram.WithLabelValues("Set")
	queryDurationHistogramGeneral  = metrics.QueryDurationHistogram.WithLabelValues(metrics.LblGeneral)

	connIdleDurationHistogramNotInTxn = metrics.ConnIdleDurationHistogram.WithLabelValues("0")
	connIdleDurationHistogramInTxn    = metrics.ConnIdleDurationHistogram.WithLabelValues("1")
)

// newClientConn creates a *clientConn object.
func newClientConn(s *Server) *clientConn {
	return &clientConn{
		server:       s,
		connectionID: atomic.AddUint32(&baseConnID, 1),
		collation:    mysql.DefaultCollationID,
		alloc:        arena.NewAllocator(32 * 1024),
		status:       connStatusDispatching,
		lastActive:   time.Now(),
	}
}

// clientConn represents a connection between server and client, it maintains connection specific state,
// handles client query.
type clientConn struct {
	pkt          *packetIO         // a helper to read and write data in packet format.
	bufReadConn  *bufferedReadConn // a buffered-read net.Conn or buffered-read tls.Conn.
	tlsConn      *tls.Conn         // TLS connection, nil if not TLS.
	server       *Server           // a reference of server instance.
	capability   uint32            // client capability affects the way server handles client request.
	connectionID uint32            // atomically allocated by a global variable, unique in process scope.
	user         string            // user of the client.
	dbname       string            // default database name.
	salt         []byte            // random bytes used for authentication.
	alloc        arena.Allocator   // an memory allocator for reducing memory allocation.
	lastPacket   []byte            // latest sql query string, currently used for logging error.
	ctx          QueryCtx          // an interface to execute sql statements.
	attrs        map[string]string // attributes parsed from client handshake response, not used for now.
	peerHost     string            // peer host
	peerPort     string            // peer port
	status       int32             // dispatching/reading/shutdown/waitshutdown
	lastCode     uint16            // last error code
	collation    uint8             // collation used by client, may be different from the collation used by database.
	lastActive   time.Time

	// mu is used for cancelling the execution of current transaction.
	mu struct {
		sync.RWMutex
		cancelFunc context.CancelFunc
	}
}

func (cc *clientConn) String() string {
	collationStr := mysql.Collations[cc.collation]
	return fmt.Sprintf("id:%d, addr:%s status:%b, collation:%s, user:%s",
		cc.connectionID, cc.bufReadConn.RemoteAddr(), cc.ctx.Status(), collationStr, cc.user,
	)
}

// authSwitchRequest is used when the client asked to speak something
// other than mysql_native_password. The server is allowed to ask
// the client to switch, so lets ask for mysql_native_password
// https://dev.mysql.com/doc/internals/en/connection-phase-packets.html#packet-Protocol::AuthSwitchRequest
func (cc *clientConn) authSwitchRequest(ctx context.Context) ([]byte, error) {
	enclen := 1 + len("mysql_native_password") + 1 + len(cc.salt) + 1
	data := cc.alloc.AllocWithLen(4, enclen)
	data = append(data, 0xfe) // switch request
	data = append(data, []byte("mysql_native_password")...)
	data = append(data, byte(0x00)) // requires null
	data = append(data, cc.salt...)
	data = append(data, 0)
	err := cc.writePacket(data)
	if err != nil {
		logutil.Logger(ctx).Debug("write response to client failed", zap.Error(err))
		return nil, err
	}
	err = cc.flush(ctx)
	if err != nil {
		logutil.Logger(ctx).Debug("flush response to client failed", zap.Error(err))
		return nil, err
	}
	resp, err := cc.readPacket()
	if err != nil {
		err = errors.SuspendStack(err)
		if errors.Cause(err) == io.EOF {
			logutil.Logger(ctx).Warn("authSwitchRequest response fail due to connection has be closed by client-side")
		} else {
			logutil.Logger(ctx).Warn("authSwitchRequest response fail", zap.Error(err))
		}
		return nil, err
	}
	return resp, nil
}

func (cc *clientConn) Close() error {
	cc.server.rwlock.Lock()
	delete(cc.server.clients, cc.connectionID)
	connections := len(cc.server.clients)
	cc.server.rwlock.Unlock()
	return closeConn(cc, connections)
}

func closeConn(cc *clientConn, connections int) error {
	metrics.ConnGauge.Set(float64(connections))
	err := cc.bufReadConn.Close()
	terror.Log(err)
	if cc.ctx != nil {
		return cc.ctx.Close()
	}
	return nil
}

func (cc *clientConn) closeWithoutLock() error {
	delete(cc.server.clients, cc.connectionID)
	return closeConn(cc, len(cc.server.clients))
}

// writeInitialHandshake sends server version, connection ID, server capability, collation, server status
// and auth salt to the client.
func (cc *clientConn) writeInitialHandshake(ctx context.Context) error {
	data := make([]byte, 4, 128)

	// min version 10
	data = append(data, 10)
	// server version[00]
	data = append(data, mysql.ServerVersion...)
	data = append(data, 0)
	// connection id
	data = append(data, byte(cc.connectionID), byte(cc.connectionID>>8), byte(cc.connectionID>>16), byte(cc.connectionID>>24))
	// auth-plugin-data-part-1
	data = append(data, cc.salt[0:8]...)
	// filler [00]
	data = append(data, 0)
	// capability flag lower 2 bytes, using default capability here
	data = append(data, byte(cc.server.capability), byte(cc.server.capability>>8))
	// charset
	if cc.collation == 0 {
		cc.collation = uint8(mysql.DefaultCollationID)
	}
	data = append(data, cc.collation)
	// status
	data = dumpUint16(data, mysql.ServerStatusAutocommit)
	// below 13 byte may not be used
	// capability flag upper 2 bytes, using default capability here
	data = append(data, byte(cc.server.capability>>16), byte(cc.server.capability>>24))
	// length of auth-plugin-data
	data = append(data, byte(len(cc.salt)+1))
	// reserved 10 [00]
	data = append(data, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0)
	// auth-plugin-data-part-2
	data = append(data, cc.salt[8:]...)
	data = append(data, 0)
	// auth-plugin name
	data = append(data, []byte("mysql_native_password")...)
	data = append(data, 0)
	err := cc.writePacket(data)
	if err != nil {
		return err
	}
	return cc.flush(ctx)
}

// readPacket Read general messages of postgresql protocol
// PostgreSQL modified
func (cc *clientConn) readPacket() ([]byte, error) {
	msgType := make([]byte, 1)
	if _, err := io.ReadFull(cc.bufReadConn, msgType); err != nil {
		return nil, err
	}

	// 后面四个字节为长度，包括自己
	msgLength := make([]byte, 4)

	if _, err := io.ReadFull(cc.bufReadConn, msgLength); err != nil {
		return nil, err
	}

	msgLen := binary.BigEndian.Uint32(msgLength)

	// 获取请求的具体信息
	msg := make([]byte, msgLen-4)

	if _, err := io.ReadFull(cc.bufReadConn, msg); err != nil {
		return nil, err
	}

	data := append(msgType, msg...)
	return data, nil
}

func (cc *clientConn) writePacket(data []byte) error {
	failpoint.Inject("FakeClientConn", func() {
		if cc.pkt == nil {
			failpoint.Return(nil)
		}
	})
	return cc.pkt.writePacket(data)
}

// getSessionVarsWaitTimeout get session variable wait_timeout
func (cc *clientConn) getSessionVarsWaitTimeout(ctx context.Context) uint64 {
	valStr, exists := cc.ctx.GetSessionVars().GetSystemVar(variable.WaitTimeout)
	if !exists {
		return variable.DefWaitTimeout
	}
	waitTimeout, err := strconv.ParseUint(valStr, 10, 64)
	if err != nil {
		logutil.Logger(ctx).Warn("get sysval wait_timeout failed, use default value", zap.Error(err))
		// if get waitTimeout error, use default value
		return variable.DefWaitTimeout
	}
	return waitTimeout
}

type handshakeResponse41 struct {
	Capability uint32
	Collation  uint8
	User       string
	DBName     string
	Auth       []byte
	AuthPlugin string
	Attrs      map[string]string
}

// parseOldHandshakeResponseHeader parses the old version handshake header HandshakeResponse320
func parseOldHandshakeResponseHeader(ctx context.Context, packet *handshakeResponse41, data []byte) (parsedBytes int, err error) {
	// Ensure there are enough data to read:
	// https://dev.mysql.com/doc/internals/en/connection-phase-packets.html#packet-Protocol::HandshakeResponse320
	logutil.Logger(ctx).Debug("try to parse hanshake response as Protocol::HandshakeResponse320", zap.ByteString("packetData", data))
	if len(data) < 2+3 {
		logutil.Logger(ctx).Error("got malformed handshake response", zap.ByteString("packetData", data))
		return 0, mysql.ErrMalformPacket
	}
	offset := 0
	// capability
	capability := binary.LittleEndian.Uint16(data[:2])
	packet.Capability = uint32(capability)

	// be compatible with Protocol::HandshakeResponse41
	packet.Capability = packet.Capability | mysql.ClientProtocol41

	offset += 2
	// skip max packet size
	offset += 3
	// usa default CharsetID
	packet.Collation = mysql.CollationNames["utf8mb4_general_ci"]

	return offset, nil
}

// parseOldHandshakeResponseBody parse the HandshakeResponse for Protocol::HandshakeResponse320 (except the common header part).
func parseOldHandshakeResponseBody(ctx context.Context, packet *handshakeResponse41, data []byte, offset int) (err error) {
	defer func() {
		// Check malformat packet cause out of range is disgusting, but don't panic!
		if r := recover(); r != nil {
			logutil.Logger(ctx).Error("handshake panic", zap.ByteString("packetData", data), zap.Stack("stack"))
			err = mysql.ErrMalformPacket
		}
	}()
	// user name
	packet.User = string(data[offset : offset+bytes.IndexByte(data[offset:], 0)])
	offset += len(packet.User) + 1

	if packet.Capability&mysql.ClientConnectWithDB > 0 {
		if len(data[offset:]) > 0 {
			idx := bytes.IndexByte(data[offset:], 0)
			packet.DBName = string(data[offset : offset+idx])
			offset = offset + idx + 1
		}
		if len(data[offset:]) > 0 {
			packet.Auth = data[offset : offset+bytes.IndexByte(data[offset:], 0)]
		}
	} else {
		packet.Auth = data[offset : offset+bytes.IndexByte(data[offset:], 0)]
	}

	return nil
}

// parseHandshakeResponseHeader parses the common header of SSLRequest and HandshakeResponse41.
func parseHandshakeResponseHeader(ctx context.Context, packet *handshakeResponse41, data []byte) (parsedBytes int, err error) {
	// Ensure there are enough data to read:
	// http://dev.mysql.com/doc/internals/en/connection-phase-packets.html#packet-Protocol::SSLRequest
	if len(data) < 4+4+1+23 {
		logutil.Logger(ctx).Error("got malformed handshake response", zap.ByteString("packetData", data))
		return 0, mysql.ErrMalformPacket
	}

	offset := 0
	// capability
	capability := binary.LittleEndian.Uint32(data[:4])
	packet.Capability = capability
	offset += 4
	// skip max packet size
	offset += 4
	// charset, skip, if you want to use another charset, use set names
	packet.Collation = data[offset]
	offset++
	// skip reserved 23[00]
	offset += 23

	return offset, nil
}

// parseHandshakeResponseBody parse the HandshakeResponse (except the common header part).
func parseHandshakeResponseBody(ctx context.Context, packet *handshakeResponse41, data []byte, offset int) (err error) {
	defer func() {
		// Check malformat packet cause out of range is disgusting, but don't panic!
		if r := recover(); r != nil {
			logutil.Logger(ctx).Error("handshake panic", zap.ByteString("packetData", data))
			err = mysql.ErrMalformPacket
		}
	}()
	// user name
	packet.User = string(data[offset : offset+bytes.IndexByte(data[offset:], 0)])
	offset += len(packet.User) + 1

	if packet.Capability&mysql.ClientPluginAuthLenencClientData > 0 {
		// MySQL client sets the wrong capability, it will set this bit even server doesn't
		// support ClientPluginAuthLenencClientData.
		// https://github.com/mysql/mysql-server/blob/5.7/sql-common/client.c#L3478
		num, null, off := parseLengthEncodedInt(data[offset:])
		offset += off
		if !null {
			packet.Auth = data[offset : offset+int(num)]
			offset += int(num)
		}
	} else if packet.Capability&mysql.ClientSecureConnection > 0 {
		// auth length and auth
		authLen := int(data[offset])
		offset++
		packet.Auth = data[offset : offset+authLen]
		offset += authLen
	} else {
		packet.Auth = data[offset : offset+bytes.IndexByte(data[offset:], 0)]
		offset += len(packet.Auth) + 1
	}

	if packet.Capability&mysql.ClientConnectWithDB > 0 {
		if len(data[offset:]) > 0 {
			idx := bytes.IndexByte(data[offset:], 0)
			packet.DBName = string(data[offset : offset+idx])
			offset += idx + 1
		}
	}

	if packet.Capability&mysql.ClientPluginAuth > 0 {
		idx := bytes.IndexByte(data[offset:], 0)
		s := offset
		f := offset + idx
		if s < f { // handle unexpected bad packets
			packet.AuthPlugin = string(data[s:f])
		}
		offset += idx + 1
	}

	if packet.Capability&mysql.ClientConnectAtts > 0 {
		if len(data[offset:]) == 0 {
			// Defend some ill-formated packet, connection attribute is not important and can be ignored.
			return nil
		}
		if num, null, off := parseLengthEncodedInt(data[offset:]); !null {
			offset += off
			row := data[offset : offset+int(num)]
			attrs, err := parseAttrs(row)
			if err != nil {
				logutil.Logger(ctx).Warn("parse attrs failed", zap.Error(err))
				return nil
			}
			packet.Attrs = attrs
		}
	}

	return nil
}

func parseAttrs(data []byte) (map[string]string, error) {
	attrs := make(map[string]string)
	pos := 0
	for pos < len(data) {
		key, _, off, err := parseLengthEncodedBytes(data[pos:])
		if err != nil {
			return attrs, err
		}
		pos += off
		value, _, off, err := parseLengthEncodedBytes(data[pos:])
		if err != nil {
			return attrs, err
		}
		pos += off

		attrs[string(key)] = string(value)
	}
	return attrs, nil
}

func (cc *clientConn) readOptionalSSLRequestAndHandshakeResponse(ctx context.Context) error {
	// Read a packet. It may be a SSLRequest or HandshakeResponse.
	data, err := cc.readPacket()
	if err != nil {
		err = errors.SuspendStack(err)
		if errors.Cause(err) == io.EOF {
			logutil.Logger(ctx).Debug("wait handshake response fail due to connection has be closed by client-side")
		} else {
			logutil.Logger(ctx).Debug("wait handshake response fail", zap.Error(err))
		}
		return err
	}

	isOldVersion := false

	var resp handshakeResponse41
	var pos int

	if len(data) < 2 {
		logutil.Logger(ctx).Error("got malformed handshake response", zap.ByteString("packetData", data))
		return mysql.ErrMalformPacket
	}

	capability := uint32(binary.LittleEndian.Uint16(data[:2]))
	if capability&mysql.ClientProtocol41 > 0 {
		pos, err = parseHandshakeResponseHeader(ctx, &resp, data)
	} else {
		pos, err = parseOldHandshakeResponseHeader(ctx, &resp, data)
		isOldVersion = true
	}

	if err != nil {
		terror.Log(err)
		return err
	}

	if resp.Capability&mysql.ClientSSL > 0 {
		tlsConfig := (*tls.Config)(atomic.LoadPointer(&cc.server.tlsConfig))
		if tlsConfig != nil {
			// The packet is a SSLRequest, let's switch to TLS.
			if err = cc.upgradeToTLS(tlsConfig); err != nil {
				return err
			}
			// Read the following HandshakeResponse packet.
			data, err = cc.readPacket()
			if err != nil {
				logutil.Logger(ctx).Warn("read handshake response failure after upgrade to TLS", zap.Error(err))
				return err
			}
			if isOldVersion {
				pos, err = parseOldHandshakeResponseHeader(ctx, &resp, data)
			} else {
				pos, err = parseHandshakeResponseHeader(ctx, &resp, data)
			}
			if err != nil {
				terror.Log(err)
				return err
			}
		}
	} else if config.GetGlobalConfig().Security.RequireSecureTransport {
		err := errSecureTransportRequired.FastGenByArgs()
		terror.Log(err)
		return err
	}

	// Read the remaining part of the packet.
	if isOldVersion {
		err = parseOldHandshakeResponseBody(ctx, &resp, data, pos)
	} else {
		err = parseHandshakeResponseBody(ctx, &resp, data, pos)
	}
	if err != nil {
		terror.Log(err)
		return err
	}

	// switching from other methods should work, but not tested
	if resp.AuthPlugin == "caching_sha2_password" {
		resp.Auth, err = cc.authSwitchRequest(ctx)
		if err != nil {
			logutil.Logger(ctx).Warn("attempt to send auth switch request packet failed", zap.Error(err))
			return err
		}
	}
	cc.capability = resp.Capability & cc.server.capability
	cc.user = resp.User
	cc.dbname = resp.DBName
	cc.collation = resp.Collation
	cc.attrs = resp.Attrs

	err = cc.openSessionAndDoAuth(resp.Auth)
	if err != nil {
		logutil.Logger(ctx).Warn("open new session failure", zap.Error(err))
	}
	return err
}

func (cc *clientConn) SessionStatusToString() string {
	status := cc.ctx.Status()
	inTxn, autoCommit := 0, 0
	if status&mysql.ServerStatusInTrans > 0 {
		inTxn = 1
	}
	if status&mysql.ServerStatusAutocommit > 0 {
		autoCommit = 1
	}
	return fmt.Sprintf("inTxn:%d, autocommit:%d",
		inTxn, autoCommit,
	)
}

func (cc *clientConn) openSessionAndDoAuth(authData []byte) error {
	var tlsStatePtr *tls.ConnectionState
	if cc.tlsConn != nil {
		tlsState := cc.tlsConn.ConnectionState()
		tlsStatePtr = &tlsState
	}
	var err error
	cc.ctx, err = cc.server.driver.OpenCtx(uint64(cc.connectionID), cc.capability, cc.collation, cc.dbname, tlsStatePtr)
	if err != nil {
		return err
	}

	if err = cc.server.checkConnectionCount(); err != nil {
		return err
	}
	hasPassword := "YES"
	if len(authData) == 0 {
		hasPassword = "NO"
	}
	host, err := cc.PeerHost(hasPassword)
	if err != nil {
		return err
	}
	if !cc.ctx.Auth(&auth.UserIdentity{Username: cc.user, Hostname: host}, authData, cc.salt) {
		return errAccessDenied.FastGenByArgs(cc.user, host, hasPassword)
	}
	if cc.dbname != "" {
		err = cc.useDB(context.Background(), cc.dbname)
		if err != nil {
			return err
		}
	}
	cc.ctx.SetSessionManager(cc.server)
	return nil
}

func (cc *clientConn) PeerHost(hasPassword string) (host string, err error) {
	if len(cc.peerHost) > 0 {
		return cc.peerHost, nil
	}
	host = variable.DefHostname
	if cc.server.isUnixSocket() {
		cc.peerHost = host
		return
	}
	addr := cc.bufReadConn.RemoteAddr().String()
	var port string
	host, port, err = net.SplitHostPort(addr)
	if err != nil {
		err = errAccessDenied.GenWithStackByArgs(cc.user, addr, hasPassword)
		return
	}
	cc.peerHost = host
	cc.peerPort = port
	return
}

// Run reads client query and writes query result to client in for loop, if there is a panic during query handling,
// it will be recovered and log the panic error.
// This function returns and the connection is closed if there is an IO error or there is a panic.
func (cc *clientConn) Run(ctx context.Context) {
	const size = 4096
	defer func() {
		r := recover()
		if r != nil {
			buf := make([]byte, size)
			stackSize := runtime.Stack(buf, false)
			buf = buf[:stackSize]
			logutil.Logger(ctx).Error("connection running loop panic",
				zap.Stringer("lastSQL", getLastStmtInConn{cc}),
				zap.String("err", fmt.Sprintf("%v", r)),
				zap.String("stack", string(buf)),
			)
			err := cc.writeError(ctx, errors.New(fmt.Sprintf("%v", r)))
			terror.Log(err)
			metrics.PanicCounter.WithLabelValues(metrics.LabelSession).Inc()
		}
		if atomic.LoadInt32(&cc.status) != connStatusShutdown {
			err := cc.Close()
			terror.Log(err)
		}
	}()
	// Usually, client connection status changes between [dispatching] <=> [reading].
	// When some event happens, server may notify this client connection by setting
	// the status to special values, for example: kill or graceful shutdown.
	// The client connection would detect the events when it fails to change status
	// by CAS operation, it would then take some actions accordingly.
	for {
		if !atomic.CompareAndSwapInt32(&cc.status, connStatusDispatching, connStatusReading) ||
			// The judge below will not be hit by all means,
			// But keep it stayed as a reminder and for the code reference for connStatusWaitShutdown.
			atomic.LoadInt32(&cc.status) == connStatusWaitShutdown {
			return
		}

		cc.alloc.Reset()
		// close connection when idle time is more than wait_timeout
		waitTimeout := cc.getSessionVarsWaitTimeout(ctx)
		cc.pkt.setReadTimeout(time.Duration(waitTimeout) * time.Second)
		start := time.Now()
		data, err := cc.readPacket()
		if err != nil {
			if terror.ErrorNotEqual(err, io.EOF) {
				if netErr, isNetErr := errors.Cause(err).(net.Error); isNetErr && netErr.Timeout() {
					idleTime := time.Since(start)
					logutil.Logger(ctx).Info("read packet timeout, close this connection",
						zap.Duration("idle", idleTime),
						zap.Uint64("waitTimeout", waitTimeout),
						zap.Error(err),
					)
				} else {
					errStack := errors.ErrorStack(err)
					if !strings.Contains(errStack, "use of closed network connection") {
						logutil.Logger(ctx).Warn("read packet failed, close this connection",
							zap.Error(errors.SuspendStack(err)))
					}
				}
			}
			return
		}

		if !atomic.CompareAndSwapInt32(&cc.status, connStatusReading, connStatusDispatching) {
			return
		}

		startTime := time.Now()
		if err = cc.dispatch(ctx, data); err != nil {
			if terror.ErrorEqual(err, io.EOF) {
				cc.addMetrics(data[0], startTime, nil)
				return
			} else if terror.ErrResultUndetermined.Equal(err) {
				logutil.Logger(ctx).Error("result undetermined, close this connection", zap.Error(err))
				return
			} else if terror.ErrCritical.Equal(err) {
				metrics.CriticalErrorCounter.Add(1)
				logutil.Logger(ctx).Fatal("critical error, stop the server", zap.Error(err))
			}
			var txnMode string
			if cc.ctx != nil {
				txnMode = cc.ctx.GetSessionVars().GetReadableTxnMode()
			}
			logutil.Logger(ctx).Info("command dispatched failed",
				zap.String("connInfo", cc.String()),
				zap.String("command", mysql.Command2Str[data[0]]),
				zap.String("status", cc.SessionStatusToString()),
				zap.Stringer("sql", getLastStmtInConn{cc}),
				zap.String("txn_mode", txnMode),
				zap.String("err", errStrForLog(err, cc.ctx.GetSessionVars().EnableRedactLog)),
			)
			err1 := cc.writeError(ctx, err)
			terror.Log(err1)
		}
		cc.addMetrics(data[0], startTime, err)
		cc.pkt.sequence = 0
	}
}

// ShutdownOrNotify will Shutdown this client connection, or do its best to notify.
func (cc *clientConn) ShutdownOrNotify() bool {
	if (cc.ctx.Status() & mysql.ServerStatusInTrans) > 0 {
		return false
	}
	// If the client connection status is reading, it's safe to shutdown it.
	if atomic.CompareAndSwapInt32(&cc.status, connStatusReading, connStatusShutdown) {
		return true
	}
	// If the client connection status is dispatching, we can't shutdown it immediately,
	// so set the status to WaitShutdown as a notification, the loop in clientConn.Run
	// will detect it and then exit.
	atomic.StoreInt32(&cc.status, connStatusWaitShutdown)
	return false
}

func queryStrForLog(query string) string {
	const size = 4096
	if len(query) > size {
		return query[:size] + fmt.Sprintf("(len: %d)", len(query))
	}
	return query
}

func errStrForLog(err error, enableRedactLog bool) string {
	if enableRedactLog {
		// currently, only ErrParse is considered when enableRedactLog because it may contain sensitive information like
		// password or accesskey
		if parser.ErrParse.Equal(err) {
			return "fail to parse SQL and can't redact when enable log redaction"
		}
	}
	if kv.ErrKeyExists.Equal(err) || parser.ErrParse.Equal(err) {
		// Do not log stack for duplicated entry error.
		return err.Error()
	}
	return errors.ErrorStack(err)
}

func (cc *clientConn) addMetrics(cmd byte, startTime time.Time, err error) {
	if cmd == mysql.ComQuery && cc.ctx.Value(sessionctx.LastExecuteDDL) != nil {
		// Don't take DDL execute time into account.
		// It's already recorded by other metrics in ddl package.
		return
	}

	var counter prometheus.Counter
	if err != nil && int(cmd) < len(queryTotalCountErr) {
		counter = queryTotalCountErr[cmd]
	} else if err == nil && int(cmd) < len(queryTotalCountOk) {
		counter = queryTotalCountOk[cmd]
	}
	if counter != nil {
		counter.Inc()
	} else {
		label := strconv.Itoa(int(cmd))
		if err != nil {
			metrics.QueryTotalCounter.WithLabelValues(label, "ERROR").Inc()
		} else {
			metrics.QueryTotalCounter.WithLabelValues(label, "OK").Inc()
		}
	}

	stmtType := cc.ctx.GetSessionVars().StmtCtx.StmtType
	sqlType := metrics.LblGeneral
	if stmtType != "" {
		sqlType = stmtType
	}

	switch sqlType {
	case "Use":
		queryDurationHistogramUse.Observe(time.Since(startTime).Seconds())
	case "Show":
		queryDurationHistogramShow.Observe(time.Since(startTime).Seconds())
	case "Begin":
		queryDurationHistogramBegin.Observe(time.Since(startTime).Seconds())
	case "Commit":
		queryDurationHistogramCommit.Observe(time.Since(startTime).Seconds())
	case "Rollback":
		queryDurationHistogramRollback.Observe(time.Since(startTime).Seconds())
	case "Insert":
		queryDurationHistogramInsert.Observe(time.Since(startTime).Seconds())
	case "Replace":
		queryDurationHistogramReplace.Observe(time.Since(startTime).Seconds())
	case "Delete":
		queryDurationHistogramDelete.Observe(time.Since(startTime).Seconds())
	case "Update":
		queryDurationHistogramUpdate.Observe(time.Since(startTime).Seconds())
	case "Select":
		queryDurationHistogramSelect.Observe(time.Since(startTime).Seconds())
	case "Execute":
		queryDurationHistogramExecute.Observe(time.Since(startTime).Seconds())
	case "Set":
		queryDurationHistogramSet.Observe(time.Since(startTime).Seconds())
	case metrics.LblGeneral:
		queryDurationHistogramGeneral.Observe(time.Since(startTime).Seconds())
	default:
		metrics.QueryDurationHistogram.WithLabelValues(sqlType).Observe(time.Since(startTime).Seconds())
	}
}

// dispatch handles client request based on command which is the first byte of the data.
// It also gets a token from server which is used to limit the concurrently handling clients.
// The most frequently used command is ComQuery.
// PostgreSQL Modified
func (cc *clientConn) dispatch(ctx context.Context, data []byte) error {
	defer func() {
		// reset killed for each request
		atomic.StoreUint32(&cc.ctx.GetSessionVars().Killed, 0)
	}()
	t := time.Now()
	if (cc.ctx.Status() & mysql.ServerStatusInTrans) > 0 {
		connIdleDurationHistogramInTxn.Observe(t.Sub(cc.lastActive).Seconds())
	} else {
		connIdleDurationHistogramNotInTxn.Observe(t.Sub(cc.lastActive).Seconds())
	}

	span := opentracing.StartSpan("server.dispatch")

	var cancelFunc context.CancelFunc
	ctx, cancelFunc = context.WithCancel(ctx)
	cc.mu.Lock()
	cc.mu.cancelFunc = cancelFunc
	cc.mu.Unlock()

	cc.lastPacket = data
	cmd := data[0]
	data = data[1:]
	if variable.EnablePProfSQLCPU.Load() {
		label := getLastStmtInConn{cc}.PProfLabel()
		if len(label) > 0 {
			defer pprof.SetGoroutineLabels(ctx)
			ctx = pprof.WithLabels(ctx, pprof.Labels("sql", label))
			pprof.SetGoroutineLabels(ctx)
		}
	}
	if trace.IsEnabled() {
		lc := getLastStmtInConn{cc}
		sqlType := lc.PProfLabel()
		if len(sqlType) > 0 {
			var task *trace.Task
			ctx, task = trace.NewTask(ctx, sqlType)
			defer task.End()

			trace.Log(ctx, "sql", lc.String())
			ctx = logutil.WithTraceLogger(ctx, cc.connectionID)

			taskID := *(*uint64)(unsafe.Pointer(task))
			ctx = pprof.WithLabels(ctx, pprof.Labels("trace", strconv.FormatUint(taskID, 10)))
			pprof.SetGoroutineLabels(ctx)
		}
	}
	token := cc.server.getToken()
	defer func() {
		// if handleChangeUser failed, cc.ctx may be nil
		if cc.ctx != nil {
			cc.ctx.SetProcessInfo("", t, mysql.ComSleep, 0)
		}

		cc.server.releaseToken(token)
		span.Finish()
		cc.lastActive = time.Now()
	}()

	vars := cc.ctx.GetSessionVars()
	// reset killed for each request
	atomic.StoreUint32(&vars.Killed, 0)
	if cmd < mysql.ComEnd {
		cc.ctx.SetCommandValue(cmd)
	}

	dataStr := string(hack.String(data))
	switch cmd {
	case 'Q': /* simple query */
		if len(data) > 0 && data[len(data)-1] == 0 {
			data = data[:len(data)-1]
			dataStr = string(hack.String(data))
		}
		return cc.handleQuery(ctx, dataStr)
	case 'P': /* parse */
		parse := pgproto3.Parse{}
		if err := parse.Decode(data); err != nil {
			return err
		}
		return cc.handleStmtPrepare(ctx, parse)
	case 'B': /* bind */
		bind := pgproto3.Bind{}
		if err := bind.Decode(data); err != nil {
			return err
		}
		return cc.handleStmtBind(ctx, bind)
	case 'E': /* execute */
		execute := pgproto3.Execute{}
		if err := execute.Decode(data); err != nil {
			return err
		}
		return cc.handleStmtExecute(ctx, execute)
	case 'F': /* fastpath function call */
	case 'C': /* close */
		c := pgproto3.Close{}
		if err := c.Decode(data); err != nil {
			return err
		}
		return cc.handleStmtClose(ctx, c)
	case 'D': /* describe */
		desc := pgproto3.Describe{}
		if err := desc.Decode(data); err != nil {
			return err
		}
		return cc.handleStmtDescription(ctx, desc)
	case 'H': /* flush */
		return cc.flush(ctx)
	case 'S': /* sync */
		return cc.writeReadyForQuery(ctx, cc.ctx.Status())
	case 'X':
		return io.EOF
	case 'd': /* copy data */
	case 'c': /* copy done */
	case 'f': /* copy fail */
	default:
		return mysql.NewErrf(mysql.ErrUnknown, "command %d not supported now", nil, cmd)
	}
	return mysql.NewErrf(mysql.ErrUnknown, "command %d not supported now", nil, cmd)
}

// useDB
func (cc *clientConn) useDB(ctx context.Context, db string) (err error) {
	// if input is "use `SELECT`", mysql client just send "SELECT"
	// so we add `` around db.
	stmts, err := cc.ctx.Parse(ctx, "use `"+db+"`")
	if err != nil {
		return err
	}
	_, err = cc.ctx.ExecuteStmt(ctx, stmts[0])
	if err != nil {
		return err
	}
	cc.dbname = db
	return
}

// flush
func (cc *clientConn) flush(ctx context.Context) error {
	defer trace.StartRegion(ctx, "FlushClientConn").End()
	failpoint.Inject("FakeClientConn", func() {
		if cc.pkt == nil {
			failpoint.Return(nil)
		}
	})
	return cc.pkt.flush()
}

// writeOK You can choose this method when you need to return complete and readyforquery directly to the client
func (cc *clientConn) writeOK(ctx context.Context) error {
	//msg := cc.ctx.LastMessage()
	if err := cc.writeCommandComplete(); err != nil {
		return err
	}
	return cc.writeReadyForQuery(ctx, cc.ctx.Status())
}

// writeOkWith 这个方法没什么用,后面可以考虑删除
func (cc *clientConn) writeOkWith(ctx context.Context, msg string, affectedRows, lastInsertID uint64, status, warnCnt uint16) error {
	return cc.writeCommandComplete()
}

// writeError 向客户端写回错误信息
// 这里需要将MySQL错误信息转换为PgSQL格式
// PgSQL 错误信息报文格式: https://www.postgresql.org/docs/13/protocol-error-fields.html
// MySLQ 错误信息报文格式: https://dev.mysql.com/doc/mysql-errors/8.0/en/server-error-reference.html
// PostgreSQL Modified
func (cc *clientConn) writeError(ctx context.Context, e error) error {
	var (
		m  *mysql.SQLError
		te *terror.Error
		ok bool
	)
	originErr := errors.Cause(e)
	if te, ok = originErr.(*terror.Error); ok {
		m = terror.ToSQLError(te)
	} else {
		e := errors.Cause(originErr)
		switch y := e.(type) {
		case *terror.Error:
			m = terror.ToSQLError(y)
		default:
			m = mysql.NewErrf(mysql.ErrUnknown, "%s", nil, e.Error())
		}
	}

	// todo 处理某些情况下从lastPacket获取不到sql的情况，比如命令行prepare语句， 它是分段提交的，第一阶段prepare不出错，第二阶段绑定出错，此时获取packet中的数据不是sql语句

	//读包获取sql，去除第一位的类型，
	var sql string
	if cc.lastPacket != nil {
		sql = string(cc.lastPacket)[1:]
	}

	// todo 完成MySQL错误与PgSQL错误的转换和返回
	// https://www.postgresql.org/docs/13/errcodes-appendix.html
	errorResponse, err := convertMysqlErrorToPgError(m, te, sql)
	if err != nil {
		return err
	}
	cc.lastCode = m.Code

	if err := cc.WriteData(errorResponse.Encode(nil)); err != nil {
		return err
	}

	// 发送错误后需要发送 ReadyForQuery 通知客户端可以继续执行命令
	// 所以这里需要获取到事务状态是否处于空闲阶段
	// todo 获取事务状态

	return cc.writeReadyForQuery(ctx, cc.ctx.Status())
}

// writeEOF writes an EOF packet.
// Note this function won't flush the stream because maybe there are more
// packets following it.
// serverStatus, a flag bit represents server information
// in the packet.
func (cc *clientConn) writeEOF(serverStatus uint16) error {
	data := cc.alloc.AllocWithLen(4, 9)

	data = append(data, mysql.EOFHeader)
	if cc.capability&mysql.ClientProtocol41 > 0 {
		data = dumpUint16(data, cc.ctx.WarningCount())
		status := cc.ctx.Status()
		status |= serverStatus
		data = dumpUint16(data, status)
	}

	err := cc.writePacket(data)
	return err
}

func (cc *clientConn) writeReq(ctx context.Context, filePath string) error {
	data := cc.alloc.AllocWithLen(4, 5+len(filePath))
	data = append(data, mysql.LocalInFileHeader)
	data = append(data, filePath...)

	err := cc.writePacket(data)
	if err != nil {
		return err
	}

	return cc.flush(ctx)
}

func insertDataWithCommit(ctx context.Context, prevData,
	curData []byte, loadDataInfo *executor.LoadDataInfo) ([]byte, error) {
	var err error
	var reachLimit bool
	for {
		prevData, reachLimit, err = loadDataInfo.InsertData(ctx, prevData, curData)
		if err != nil {
			return nil, err
		}
		if !reachLimit {
			break
		}
		// push into commit task queue
		err = loadDataInfo.EnqOneTask(ctx)
		if err != nil {
			return prevData, err
		}
		curData = prevData
		prevData = nil
	}
	return prevData, nil
}

// processStream process input stream from network
func processStream(ctx context.Context, cc *clientConn, loadDataInfo *executor.LoadDataInfo, wg *sync.WaitGroup) {
	var err error
	var shouldBreak bool
	var prevData, curData []byte
	defer func() {
		r := recover()
		if r != nil {
			logutil.Logger(ctx).Error("process routine panicked",
				zap.Reflect("r", r),
				zap.Stack("stack"))
		}
		if err != nil || r != nil {
			loadDataInfo.ForceQuit()
		} else {
			loadDataInfo.CloseTaskQueue()
		}
		wg.Done()
	}()
	for {
		curData, err = cc.readPacket()
		if err != nil {
			if terror.ErrorNotEqual(err, io.EOF) {
				logutil.Logger(ctx).Error("read packet failed", zap.Error(err))
				break
			}
		}
		if len(curData) == 0 {
			loadDataInfo.Drained = true
			shouldBreak = true
			if len(prevData) == 0 {
				break
			}
		}
		select {
		case <-loadDataInfo.QuitCh:
			err = errors.New("processStream forced to quit")
		default:
		}
		if err != nil {
			break
		}
		// prepare batch and enqueue task
		prevData, err = insertDataWithCommit(ctx, prevData, curData, loadDataInfo)
		if err != nil {
			break
		}
		if shouldBreak {
			break
		}
	}
	if err != nil {
		logutil.Logger(ctx).Error("load data process stream error", zap.Error(err))
	} else {
		err = loadDataInfo.EnqOneTask(ctx)
		if err != nil {
			logutil.Logger(ctx).Error("load data process stream error", zap.Error(err))
		}
	}
}

// handleLoadData does the additional work after processing the 'load data' query.
// It sends client a file path, then reads the file content from client, inserts data into database.
func (cc *clientConn) handleLoadData(ctx context.Context, loadDataInfo *executor.LoadDataInfo) error {
	// If the server handles the load data request, the client has to set the ClientLocalFiles capability.
	if cc.capability&mysql.ClientLocalFiles == 0 {
		return errNotAllowedCommand
	}
	if loadDataInfo == nil {
		return errors.New("load data info is empty")
	}
	if loadDataInfo.Table.Meta().IsView() || loadDataInfo.Table.Meta().IsSequence() {
		return errors.New("can only load data into base tables")
	}
	err := cc.writeReq(ctx, loadDataInfo.Path)
	if err != nil {
		return err
	}

	loadDataInfo.InitQueues()
	loadDataInfo.SetMaxRowsInBatch(uint64(loadDataInfo.Ctx.GetSessionVars().DMLBatchSize))
	loadDataInfo.StartStopWatcher()
	// let stop watcher goroutine quit
	defer loadDataInfo.ForceQuit()
	err = loadDataInfo.Ctx.NewTxn(ctx)
	if err != nil {
		return err
	}
	// processStream process input data, enqueue commit task
	wg := new(sync.WaitGroup)
	wg.Add(1)
	go processStream(ctx, cc, loadDataInfo, wg)
	err = loadDataInfo.CommitWork(ctx)
	wg.Wait()
	if err != nil {
		if !loadDataInfo.Drained {
			logutil.Logger(ctx).Info("not drained yet, try reading left data from client connection")
		}
		// drain the data from client conn util empty packet received, otherwise the connection will be reset
		for !loadDataInfo.Drained {
			// check kill flag again, let the draining loop could quit if empty packet could not be received
			if atomic.CompareAndSwapUint32(&loadDataInfo.Ctx.GetSessionVars().Killed, 1, 0) {
				logutil.Logger(ctx).Warn("receiving kill, stop draining data, connection may be reset")
				return executor.ErrQueryInterrupted
			}
			curData, err1 := cc.readPacket()
			if err1 != nil {
				logutil.Logger(ctx).Error("drain reading left data encounter errors", zap.Error(err1))
				break
			}
			if len(curData) == 0 {
				loadDataInfo.Drained = true
				logutil.Logger(ctx).Info("draining finished for error", zap.Error(err))
				break
			}
		}
	}
	loadDataInfo.SetMessage()
	return err
}

// getDataFromPath gets file contents from file path.
func (cc *clientConn) getDataFromPath(ctx context.Context, path string) ([]byte, error) {
	err := cc.writeReq(ctx, path)
	if err != nil {
		return nil, err
	}
	var prevData, curData []byte
	for {
		curData, err = cc.readPacket()
		if err != nil && terror.ErrorNotEqual(err, io.EOF) {
			return nil, err
		}
		if len(curData) == 0 {
			break
		}
		prevData = append(prevData, curData...)
	}
	return prevData, nil
}

// handleLoadStats does the additional work after processing the 'load stats' query.
// It sends client a file path, then reads the file content from client, loads it into the storage.
func (cc *clientConn) handleLoadStats(ctx context.Context, loadStatsInfo *executor.LoadStatsInfo) error {
	// If the server handles the load data request, the client has to set the ClientLocalFiles capability.
	if cc.capability&mysql.ClientLocalFiles == 0 {
		return errNotAllowedCommand
	}
	if loadStatsInfo == nil {
		return errors.New("load stats: info is empty")
	}
	data, err := cc.getDataFromPath(ctx, loadStatsInfo.Path)
	if err != nil {
		return err
	}
	if len(data) == 0 {
		return nil
	}
	return loadStatsInfo.Update(data)
}

// handleIndexAdvise does the index advise work and returns the advise result for index.
func (cc *clientConn) handleIndexAdvise(ctx context.Context, indexAdviseInfo *executor.IndexAdviseInfo) error {
	if cc.capability&mysql.ClientLocalFiles == 0 {
		return errNotAllowedCommand
	}
	if indexAdviseInfo == nil {
		return errors.New("Index Advise: info is empty")
	}

	data, err := cc.getDataFromPath(ctx, indexAdviseInfo.Path)
	if err != nil {
		return err
	}
	if len(data) == 0 {
		return errors.New("Index Advise: infile is empty")
	}

	if err := indexAdviseInfo.GetIndexAdvice(ctx, data); err != nil {
		return err
	}

	// TODO: Write the rss []ResultSet. It will be done in another PR.
	return nil
}

// handleQuery executes the sql query string and writes result set or result ok to the client.
// As the execution time of this function represents the performance of TiDB, we do time log and metrics here.
// There is a special query `load data` that does not return result, which is handled differently.
// Query `load stats` does not return result either.
func (cc *clientConn) handleQuery(ctx context.Context, sql string) (err error) {
	ctx = context.WithValue(ctx, execdetails.StmtExecDetailKey, &execdetails.StmtExecDetails{})
	defer trace.StartRegion(ctx, "handleQuery").End()
	stmts, err := cc.ctx.Parse(ctx, sql)
	if err != nil {
		metrics.ExecuteErrorCounter.WithLabelValues(metrics.ExecuteErrorToLabel(err)).Inc()
		return err
	}
	if len(stmts) == 0 {
		return cc.writeOK(ctx)
	}

	var appendMultiStmtWarning bool

	if len(stmts) > 1 {
		// The client gets to choose if it allows multi-statements, and
		// probably defaults OFF. This helps prevent against SQL injection attacks
		// by early terminating the first statement, and then running an entirely
		// new statement.
		capabilities := cc.ctx.GetSessionVars().ClientCapability
		if capabilities&mysql.ClientMultiStatements < 1 {
			// The client does not have multi-statement enabled. We now need to determine
			// how to handle an unsafe sitution based on the multiStmt sysvar.
			switch cc.ctx.GetSessionVars().MultiStatementMode {
			case variable.OffInt:
				err = errMultiStatementDisabled
				metrics.ExecuteErrorCounter.WithLabelValues(metrics.ExecuteErrorToLabel(err)).Inc()
				return err
			case variable.OnInt:
				// multi statement is fully permitted, do nothing
			default:
				appendMultiStmtWarning = true
			}
		}
	}

	for i, stmt := range stmts {
		if err = cc.handleStmt(ctx, stmt, i == len(stmts)-1, appendMultiStmtWarning); err != nil {
			break
		}
	}
	if err != nil {
		metrics.ExecuteErrorCounter.WithLabelValues(metrics.ExecuteErrorToLabel(err)).Inc()
	}
	return err
}

func (cc *clientConn) handleStmt(ctx context.Context, stmt ast.StmtNode, lastStmt bool, appendMultiStmtWarning bool) error {
	rs, err := cc.ctx.ExecuteStmt(ctx, stmt)
	if rs != nil {
		defer terror.Call(rs.Close)
	}
	if err != nil {
		return err
	}

	if lastStmt && appendMultiStmtWarning {
		cc.ctx.GetSessionVars().StmtCtx.AppendWarning(errMultiStatementDisabled)
	}

	status := cc.ctx.Status()
	if !lastStmt {
		status |= mysql.ServerMoreResultsExists
	}

	if rs != nil {
		connStatus := atomic.LoadInt32(&cc.status)
		if connStatus == connStatusShutdown {
			return executor.ErrQueryInterrupted
		}

		err = cc.writeResultset(ctx, rs, nil, status, 0)
		if err != nil {
			return err
		}

		// 如果是预处理查询,则不需要返回ReadyForQuery
		isPrepareStmt := rs.IsPrepareStmt()
		if isPrepareStmt {
			return nil
		}
	} else {
		var handled bool
		// 对stmt做一个特殊的处理
		err = cc.handleQueryWithStmt(stmt)
		if err != nil {
			return err
		}
		handled, err = cc.handleQuerySpecial(ctx)
		if handled {
			execStmt := cc.ctx.Value(session.ExecStmtVarKey)
			if execStmt != nil {
				execStmt.(*executor.ExecStmt).FinishExecuteStmt(0, err == nil, false)
			}
		}
		if err != nil {
			return err
		}
	}

	// 如果是最后一个查询,则需要返回ReadyForQuery
	if lastStmt {
		if err := cc.writeReadyForQuery(ctx, status); err != nil {
			return err
		}
	}
	return nil
}

func (cc *clientConn) handleQuerySpecial(ctx context.Context) (bool, error) {
	handled := false
	loadDataInfo := cc.ctx.Value(executor.LoadDataVarKey)
	if loadDataInfo != nil {
		handled = true
		defer cc.ctx.SetValue(executor.LoadDataVarKey, nil)
		if err := cc.handleLoadData(ctx, loadDataInfo.(*executor.LoadDataInfo)); err != nil {
			return handled, err
		}
	}

	loadStats := cc.ctx.Value(executor.LoadStatsVarKey)
	if loadStats != nil {
		handled = true
		defer cc.ctx.SetValue(executor.LoadStatsVarKey, nil)
		if err := cc.handleLoadStats(ctx, loadStats.(*executor.LoadStatsInfo)); err != nil {
			return handled, err
		}
	}

	indexAdvise := cc.ctx.Value(executor.IndexAdviseVarKey)
	if indexAdvise != nil {
		handled = true
		defer cc.ctx.SetValue(executor.IndexAdviseVarKey, nil)
		if err := cc.handleIndexAdvise(ctx, indexAdvise.(*executor.IndexAdviseInfo)); err != nil {
			return handled, err
		}
	}
	return handled, cc.writeCommandComplete()
}

func (cc *clientConn) handleQueryWithStmt(stmt ast.StmtNode) error {
	// 判断 stmt 的类型
	// 如果是 SET 类型的语句
	// 其余的暂不做处理
	if set, OK := stmt.(*ast.SetStmt); OK {
		key := set.Variables[0].Name
		value := set.Variables[0].Value.(ast.ValueExpr).GetDatumString()
		par := make(map[string]string)
		par[key] = value
		if err := cc.writeParameterStatus(par); err != nil {
			return err
		}
	}

	return nil
}

// handleFieldList returns the field list for a table.
// The sql string is composed of a table name and a terminating character \x00.
func (cc *clientConn) handleFieldList(ctx context.Context, sql string) (err error) {
	parts := strings.Split(sql, "\x00")
	columns, err := cc.ctx.FieldList(parts[0])
	if err != nil {
		return err
	}
	data := cc.alloc.AllocWithLen(4, 1024)
	for _, column := range columns {
		// Current we doesn't output defaultValue but reserve defaultValue length byte to make mariadb client happy.
		// https://dev.mysql.com/doc/internals/en/com-query-response.html#column-definition
		// TODO: fill the right DefaultValues.
		column.DefaultValueLength = 0
		column.DefaultValue = []byte{}

		data = data[0:4]
		data = column.Dump(data)
		if err := cc.writePacket(data); err != nil {
			return err
		}
	}
	if err := cc.writeEOF(0); err != nil {
		return err
	}
	return cc.flush(ctx)
}

// writeResultset writes data into a resultset and uses rs.Next to get row data back.
// If resultFormat is nil, the data would be encoded in Text format.
// If resultFormat just one value, the data would be encoded in Text(0) or Binary(1) format
// If resultFormat have many values, each column would be encoded in Text(0) or Binary(1) format.
// serverStatus, a flag bit represents server information.
// fetchSize, the desired number of rows to be fetched each time when client uses cursor.
func (cc *clientConn) writeResultset(ctx context.Context, rs ResultSet, resultFormat []int16, serverStatus uint16, fetchSize int) (runErr error) {
	defer func() {
		// close ResultSet when cursor doesn't exist
		r := recover()
		if r == nil {
			return
		}
		if str, ok := r.(string); !ok || !strings.HasPrefix(str, memory.PanicMemoryExceed) {
			panic(r)
		}
		// TODO(jianzhang.zj: add metrics here)
		runErr = errors.Errorf("%v", r)
		buf := make([]byte, 4096)
		stackSize := runtime.Stack(buf, false)
		buf = buf[:stackSize]
		logutil.Logger(ctx).Error("write query result panic", zap.Stringer("lastSQL", getLastStmtInConn{cc}), zap.String("stack", string(buf)))
	}()
	var err error
	if mysql.HasCursorExistsFlag(serverStatus) {
		// todo writeChunksWithFetchSize
		//err = cc.writeChunksWithFetchSize(ctx, rs, serverStatus, fetchSize)
		err = nil
	} else {
		err = cc.writeChunks(ctx, rs, serverStatus, resultFormat)
	}

	return err
}

func (cc *clientConn) writeColumnInfo(columns []*ColumnInfo, serverStatus uint16) error {
	data := cc.alloc.AllocWithLen(4, 1024)
	data = dumpLengthEncodedInt(data, uint64(len(columns)))
	if err := cc.writePacket(data); err != nil {
		return err
	}
	for _, v := range columns {
		data = data[0:4]
		data = v.Dump(data)
		if err := cc.writePacket(data); err != nil {
			return err
		}
	}
	return cc.writeEOF(serverStatus)
}

// writeChunks writes data from a Chunk, which filled data by a ResultSet, into a connection.
// serverStatus, a flag bit represents server information
// PostgreSQL Modified
// 判断是否为预处理查询,决定是否返回readyQuery,不在这儿做
func (cc *clientConn) writeChunks(ctx context.Context, rs ResultSet, serverStatus uint16, rf []int16) error {
	data := cc.alloc.AllocWithLen(0, 1024)
	req := rs.NewChunk()

	// 当为预处理查询执行时，在执行完成后不需要返回 RowDescription
	gotColumnInfo := rs.IsPrepareStmt()

	var stmtDetail *execdetails.StmtExecDetails
	stmtDetailRaw := ctx.Value(execdetails.StmtExecDetailKey)
	if stmtDetailRaw != nil {
		stmtDetail = stmtDetailRaw.(*execdetails.StmtExecDetails)
	}
	for {
		// Here server.tidbResultSet implements Next method.
		err := rs.Next(ctx, req)
		if err != nil {
			return err
		}
		if !gotColumnInfo {
			// We need to call Next before we get columns.
			// Otherwise, we will get incorrect columns info.
			columns := rs.Columns()
			// err = cc.writeColumnInfo(columns, serverStatus)
			err = cc.WriteRowDescription(columns)
			if err != nil {
				return err
			}
			gotColumnInfo = true
		}
		rowCount := req.NumRows()
		if rowCount == 0 {
			break
		}
		start := time.Now()
		reg := trace.StartRegion(ctx, "WriteClientConn")

		// rowdata : 'D' + len(msg) + len(columns) + for(len(val) + val)
		data = append(data, 'D')
		data = pgio.AppendInt32(data, -1)
		for i := 0; i < rowCount; i++ {
			data = data[0:5]
			if len(rf) > 0 {
				data, err = dumpRowData(data, rs.Columns(), req.GetRow(i), rf)
			} else {
				data, err = dumpTextRowData(data, rs.Columns(), req.GetRow(i))
			}
			if err != nil {
				return err
			}

			pgio.SetInt32(data[1:], int32(len(data[1:])))
			if err = cc.WriteData(data); err != nil {
				return err
			}
		}
		if stmtDetail != nil {
			stmtDetail.WriteSQLRespDuration += time.Since(start)
		}
		reg.End()
	}

	return cc.writeCommandComplete()
}

// writeChunksWithFetchSize writes data from a Chunk, which filled data by a ResultSet, into a connection.
// binary specifies the way to dump data. It throws any error while dumping data.
// serverStatus, a flag bit represents server information.
// fetchSize, the desired number of rows to be fetched each time when client uses cursor.
func (cc *clientConn) writeChunksWithFetchSize(ctx context.Context, rs ResultSet, serverStatus uint16, fetchSize int) error {
	fetchedRows := rs.GetFetchedRows()

	// if fetchedRows is not enough, getting data from recordSet.
	req := rs.NewChunk()
	for len(fetchedRows) < fetchSize {
		// Here server.tidbResultSet implements Next method.
		err := rs.Next(ctx, req)
		if err != nil {
			return err
		}
		rowCount := req.NumRows()
		if rowCount == 0 {
			break
		}
		// filling fetchedRows with chunk
		for i := 0; i < rowCount; i++ {
			fetchedRows = append(fetchedRows, req.GetRow(i))
		}
		req = chunk.Renew(req, cc.ctx.GetSessionVars().MaxChunkSize)
	}

	// tell the client COM_STMT_FETCH has finished by setting proper serverStatus,
	// and close ResultSet.
	if len(fetchedRows) == 0 {
		serverStatus &^= mysql.ServerStatusCursorExists
		serverStatus |= mysql.ServerStatusLastRowSend
		terror.Call(rs.Close)
		return cc.writeEOF(serverStatus)
	}

	// construct the rows sent to the client according to fetchSize.
	var curRows []chunk.Row
	if fetchSize < len(fetchedRows) {
		curRows = fetchedRows[:fetchSize]
		fetchedRows = fetchedRows[fetchSize:]
	} else {
		curRows = fetchedRows[:]
		fetchedRows = fetchedRows[:0]
	}
	rs.StoreFetchedRows(fetchedRows)

	data := cc.alloc.AllocWithLen(4, 1024)
	var stmtDetail *execdetails.StmtExecDetails
	stmtDetailRaw := ctx.Value(execdetails.StmtExecDetailKey)
	if stmtDetailRaw != nil {
		stmtDetail = stmtDetailRaw.(*execdetails.StmtExecDetails)
	}
	start := time.Now()
	var err error
	for _, row := range curRows {
		data = data[0:4]
		data, err = dumpBinaryRow(data, rs.Columns(), row)
		if err != nil {
			return err
		}
		if err = cc.writePacket(data); err != nil {
			return err
		}
	}
	if stmtDetail != nil {
		stmtDetail.WriteSQLRespDuration += time.Since(start)
	}
	if cl, ok := rs.(fetchNotifier); ok {
		cl.OnFetchReturned()
	}
	return cc.writeEOF(serverStatus)
}

func (cc *clientConn) writeMultiResultset(ctx context.Context, rss []ResultSet, resultFormat []int16) error {
	for i, rs := range rss {
		lastRs := i == len(rss)-1
		if r, ok := rs.(*tidbResultSet).recordSet.(sqlexec.MultiQueryNoDelayResult); ok {
			status := r.Status()
			if !lastRs {
				status |= mysql.ServerMoreResultsExists
			}
			if err := cc.writeOkWith(ctx, r.LastMessage(), r.AffectedRows(), r.LastInsertID(), status, r.WarnCount()); err != nil {
				return err
			}
			continue
		}
		status := uint16(0)
		if !lastRs {
			status |= mysql.ServerMoreResultsExists
		}
		if err := cc.writeResultset(ctx, rs, resultFormat, status, 0); err != nil {
			return err
		}
	}
	return nil
}

func (cc *clientConn) setConn(conn net.Conn) {
	cc.bufReadConn = newBufferedReadConn(conn)
	if cc.pkt == nil {
		cc.pkt = newPacketIO(cc.bufReadConn)
	} else {
		// Preserve current sequence number.
		cc.pkt.setBufferedReadConn(cc.bufReadConn)
	}
}

func (cc *clientConn) upgradeToTLS(tlsConfig *tls.Config) error {
	// Important: read from buffered reader instead of the original net.Conn because it may contain data we need.
	tlsConn := tls.Server(cc.bufReadConn, tlsConfig)
	if err := tlsConn.Handshake(); err != nil {
		return err
	}
	cc.setConn(tlsConn)
	cc.tlsConn = tlsConn
	return nil
}

func (cc *clientConn) handleChangeUser(ctx context.Context, data []byte) error {
	user, data := parseNullTermString(data)
	cc.user = string(hack.String(user))
	if len(data) < 1 {
		return mysql.ErrMalformPacket
	}
	passLen := int(data[0])
	data = data[1:]
	if passLen > len(data) {
		return mysql.ErrMalformPacket
	}
	pass := data[:passLen]
	data = data[passLen:]
	dbName, _ := parseNullTermString(data)
	cc.dbname = string(hack.String(dbName))
	err := cc.ctx.Close()
	if err != nil {
		logutil.Logger(ctx).Debug("close old context failed", zap.Error(err))
	}
	err = cc.openSessionAndDoAuth(pass)
	if err != nil {
		return err
	}

	if plugin.IsEnable(plugin.Audit) {
		cc.ctx.GetSessionVars().ConnectionInfo = cc.connectInfo()
	}

	err = plugin.ForeachPlugin(plugin.Audit, func(p *plugin.Plugin) error {
		authPlugin := plugin.DeclareAuditManifest(p.Manifest)
		if authPlugin.OnConnectionEvent != nil {
			connInfo := cc.ctx.GetSessionVars().ConnectionInfo
			err = authPlugin.OnConnectionEvent(context.Background(), plugin.ChangeUser, connInfo)
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}

	return cc.writeOK(ctx)
}

var _ fmt.Stringer = getLastStmtInConn{}

type getLastStmtInConn struct {
	*clientConn
}

func (cc getLastStmtInConn) String() string {
	if len(cc.lastPacket) == 0 {
		return ""
	}
	cmd, data := cc.lastPacket[0], cc.lastPacket[1:]
	switch cmd {
	case mysql.ComInitDB:
		return "Use " + string(data)
	case mysql.ComFieldList:
		return "ListFields " + string(data)
	case mysql.ComQuery, mysql.ComStmtPrepare:
		sql := string(hack.String(data))
		if cc.ctx.GetSessionVars().EnableRedactLog {
			sql = parser.Normalize(sql)
		}
		return queryStrForLog(sql)
	case mysql.ComStmtExecute, mysql.ComStmtFetch:
		stmtID := binary.LittleEndian.Uint32(data[0:4])
		return queryStrForLog(cc.preparedStmt2String(stmtID))
	case mysql.ComStmtClose, mysql.ComStmtReset:
		stmtID := binary.LittleEndian.Uint32(data[0:4])
		return mysql.Command2Str[cmd] + " " + strconv.Itoa(int(stmtID))
	default:
		if cmdStr, ok := mysql.Command2Str[cmd]; ok {
			return cmdStr
		}
		return string(hack.String(data))
	}
}

// PProfLabel return sql label used to tag pprof.
func (cc getLastStmtInConn) PProfLabel() string {
	if len(cc.lastPacket) == 0 {
		return ""
	}
	cmd, data := cc.lastPacket[0], cc.lastPacket[1:]
	switch cmd {
	case mysql.ComInitDB:
		return "UseDB"
	case mysql.ComFieldList:
		return "ListFields"
	case mysql.ComStmtClose:
		return "CloseStmt"
	case mysql.ComStmtReset:
		return "ResetStmt"
	case mysql.ComQuery, mysql.ComStmtPrepare:
		return parser.Normalize(queryStrForLog(string(hack.String(data))))
	case mysql.ComStmtExecute, mysql.ComStmtFetch:
		stmtID := binary.LittleEndian.Uint32(data[0:4])
		return queryStrForLog(cc.preparedStmt2StringNoArgs(stmtID))
	default:
		return ""
	}
}

//-----------------------------------------------------------------
// Handshake PostgreSQL 握手流程
// 1. postgresql SQL 会先发送一个 SSLRequest报文 询问是否开启SSL加密
// 2. 如果需要开启则服务端回复一个 'S' 否则回复一个 'N'
// 3. 当回复为 'S' 时，客户端则进行SSL起始握手（先不做实现），后面所有通信都使用SSL加密
// 4. 完成SSL起始握手或者在2中直接回复 'N'后, 客户端向服务端发送 StartupMessage 启动包
// 5. 服务端接收到启动包判断是否需要授权信息，若需要则发送 AuthenticationRequest
// 6. 在认证过程中可以采用多种认证方式，不同方式交流不同，最终结束于服务端发送的 AuthenticationOk 或者 ErrorResponse
// 7. 服务端验证完成后发送一些参数信息，即 ParameterStatus ，包括 server_version ， client_encoding 和 DateStyle 等。
// 8. 服务端最后发送一个 ReadyForQuery 表示一切准备就绪，可以进行创建连接成功
// 9. 从 7 发送完 AuthenticationOk 和 ErrorResponse 到最后发送 ReadyForQuery 客户端不会发送信息，会一直等待
// 10. 客户端接收到 ReadyForQuery 后开始与服务端进行正常通信
func (cc *clientConn) handshake(ctx context.Context) error {
	// 第一次接收的启动包并不是真正的启动包
	// 一般会先接收 SSLRequest 报文，决定是否开启 SSL 后，才会再次接收 StartUpMessage
	// 如果接收的第一个包就为启动包，则为危险连接（可处理）
	m, err := cc.ReceiveStartupMessage()
	if err != nil {
		logutil.Logger(ctx).Debug(err.Error())
		return err
	}

	switch m.(type) {
	case *pgproto3.CancelRequest:
		// todo 结束连接
		return err
	case *pgproto3.SSLRequest:
		return cc.handleSSLRequest(ctx)
	case *pgproto3.StartupMessage:
		return cc.handleStartupMessage(ctx, m.(*pgproto3.StartupMessage))
	default:
		err = errors.New("received is not a expected packet")
		logutil.Logger(ctx).Debug(err.Error())
		return err
	}
}

//接收到来自客户端的SSLRequest之后，服务端需要对其进行一定的处理
func (cc *clientConn) handleSSLRequest(ctx context.Context) error {
	if protocolSSL {
		tlsConfig, err := loadSSLCertificates()
		if err != nil {
			logutil.Logger(ctx).Debug(err.Error())
			return err
		}

		// 写回 'S' 表示使用 SSL 连接
		if err := cc.writeSSLRequest(ctx, 'S'); err != nil {
			logutil.Logger(ctx).Debug(err.Error())
			return err
		}

		//将现有的连接升级为SSL连接
		if err := cc.upgradeToTLS(tlsConfig); err != nil {
			logutil.Logger(ctx).Debug(err.Error())
			return err
		}
	} else {
		// 写回 'N' 表示不使用 SSL 连接
		if err := cc.writeSSLRequest(ctx, 'N'); err != nil {
			logutil.Logger(ctx).Debug(err.Error())
			return err
		}
	}

	// 完成 SSL 确认后需要正式接收 StartupMessage
	m, err := cc.ReceiveStartupMessage()
	if err != nil {
		logutil.Logger(ctx).Debug(err.Error())
		return err
	}

	msg, ok := m.(*pgproto3.StartupMessage)

	// 如果接收到的包不为启动包则报错
	if !ok {
		err := errors.New("received is not a StartupMessage")
		logutil.Logger(ctx).Debug(err.Error())
		return err
	}

	// 接收完 SSLRequest 包后接收 StartupMessage
	if err := cc.handleStartupMessage(ctx, msg); err != nil {
		logutil.Logger(ctx).Debug(err.Error())
		return err
	}

	return nil
}

// handleStartupMessage 处理 StartupMessage 请求
// 接收客户端StartupMessage，获取客户端信息，初始化Session并进行用户认证
// 初始化Session时，由于TiDB中存在MySQL协议使用，会有许多参数和 PgSQL 有差异，这里会自定义部分参数直接赋值给Session
// 用户认证时，客服端可能会断开连接，等待用户输入密码后再重新建立连接，需要注意
// 最后发送 AuthenticationOK 或 ErrorResponse 来表式认证成功或失败
func (cc *clientConn) handleStartupMessage(ctx context.Context, startupMessage *pgproto3.StartupMessage) error {

	// 这里获取到的启动包后会发现只有部分参数
	// 而MySQL协议会需求更多的参数
	// 所以添加部分常量作为参数 参照客户端为 Mysql 5.6.47
	resp := handshakeResponse41{
		Capability: uint32(8365701),
		Collation:  uint8(28), //gbk_chinese_ci
		User:       startupMessage.Parameters["user"],
		DBName:     startupMessage.Parameters["database"],
		Attrs: map[string]string{
			"_os":             "Win64",
			"_client_name":    "libmysql",
			"_pid":            "3692",
			"_thread":         "480",
			"_platform":       "x86_64",
			"program_name":    "mysql",
			"_client_version": "5.6.47",
		},
	}

	cc.capability = resp.Capability & cc.server.capability
	// 获取用户名和数据库名称
	cc.user = resp.User
	cc.dbname = resp.DBName
	cc.collation = resp.Collation
	cc.attrs = resp.Attrs

	// 初始化Session并进行用户验证
	if err := cc.PgOpenSessionAndDoAuth(ctx); err != nil {
		logutil.Logger(ctx).Warn("open new session failure", zap.Error(err))
		return err
	}

	parameters := map[string]string{
		"client_encoding":   "UTF8",
		"DateStyle":         "ISO, YMD",
		"integer_datetimes": "on",
		"is_superuser":      "on",
		"server_encoding":   "UTF8",
		"server_version":    "version: TiDB-v4.0.11 TiDB Server (Apache License 2.0) Community Edition, PostgreSQL 12 compatible",
		"TimeZone":          "PRC",
	}

	// 发送 ParameterStatus
	if err := cc.writeParameterStatus(parameters); err != nil {
		logutil.Logger(ctx).Debug(err.Error())
		return err
	}

	// 发送 ReadyForQuery 表示一切准备就绪。"I"表示空闲(没有在事务中)
	if err := cc.writeReadyForQuery(ctx, mysql.ServerStatusAutocommit); err != nil {
		logutil.Logger(ctx).Debug(err.Error())
		return err
	}

	return nil
}

// ReceiveStartupMessage receives the initial connection message. This method is used of the normal Receive method
// because the initial connection message is "special" and does not include the message type as the first byte. This
// will return either a StartupMessage, SSLRequest, GSSEncRequest, or CancelRequest.
func (cc *clientConn) ReceiveStartupMessage() (pgproto3.FrontendMessage, error) {
	header := make([]byte, 4)
	if _, err := io.ReadFull(cc.bufReadConn, header); err != nil {
		return nil, err
	}
	msgLen := int(binary.BigEndian.Uint32(header) - 4)

	msg := make([]byte, msgLen)
	if _, err := io.ReadFull(cc.bufReadConn, msg); err != nil {
		return nil, err
	}

	code := binary.BigEndian.Uint32(msg)

	switch code {
	case protocolVersionNumber:
		startMessage := &pgproto3.StartupMessage{}
		if err := startMessage.Decode(msg); err != nil {
			return nil, err
		}
		return startMessage, nil
	case sslRequestNumber:
		sslRequest := &pgproto3.SSLRequest{}
		if err := sslRequest.Decode(msg); err != nil {
			return nil, err
		}
		return sslRequest, nil
	case cancelRequestCode:
		cancelRequest := &pgproto3.CancelRequest{}
		if err := cancelRequest.Decode(msg); err != nil {
			return nil, err
		}
		return cancelRequest, nil
	case gssEncReqNumber:
		gssEncRequest := &pgproto3.GSSEncRequest{}
		if err := gssEncRequest.Decode(msg); err != nil {
			return nil, err
		}
		return gssEncRequest, nil
	default:
		return nil, errors.New("unknown startup message code: " + fmt.Sprint(code))
	}
}

// PgOpenSessionAndDoAuth 对应 openSessionAndDoAuth
// 初始化 Session 并进行用户认证
// PgSQL 和 MySQL 存在差异，PgSQL客户端只有在收到服务端的 Auth Request 才会将密码发送过来
// 而 Mysql 客户端一开始如果存在密码则会将密码直接一起发送到服务端
func (cc *clientConn) PgOpenSessionAndDoAuth(ctx context.Context) error {
	var tlsStatePtr *tls.ConnectionState
	if cc.tlsConn != nil {
		tlsState := cc.tlsConn.ConnectionState()
		tlsStatePtr = &tlsState
	}
	var err error
	cc.ctx, err = cc.server.driver.OpenCtx(uint64(cc.connectionID), cc.capability, cc.collation, cc.dbname, tlsStatePtr)
	if err != nil {
		return err
	}

	if err = cc.server.checkConnectionCount(); err != nil {
		return err
	}

	authData := make([]byte, 0)
	err = cc.DoAuth(ctx, &auth.UserIdentity{Username: cc.user, Hostname: ""}, authData, cc.salt)
	if err != nil {
		return err
	}

	if cc.dbname != "" {
		err = cc.useDB(context.Background(), cc.dbname)
		if err != nil {
			return err
		}
	}
	cc.ctx.SetSessionManager(cc.server)
	return nil
}

// DoAuth 进行用户认证
// PostgreSQL 认证时，客户端是不会主动发送认证密码的，需要服务端发出不同类型的密码认证请求，客户端才会返回相应的认证信息
func (cc *clientConn) DoAuth(ctx context.Context, user *auth.UserIdentity, auth []byte, salt []byte) (err error) {
	hasPassword := "NO"
	user.Hostname, err = cc.PeerHost(hasPassword)
	if err != nil {
		return err
	}
	// 认证需要先发送一个 auth request 给客户端请求密码
	// 认证形式： AuthenticationKerberosV5 AuthenticationCleartextPassword AuthenticationCryptPassword AuthenticationMD5Password
	// AuthenticationSCMCredential AuthenticationGSS AuthenticationSSPI AuthenticationGSSContinue
	// 当客户端发送密码后进行验证
	// 最后认证成功

	// 判断用户是否需要密码登录
	if cc.ctx.NeedPassword(user) {
		hasPassword = "YES"

		// 发送一个 authRequest, 这里使用 MD5 加密要求
		// 前端必须返回一个 MD5 加密的密码进行验证
		// salt 为随机生成的 4 个字节，这里直接取 cc 中生成是 salt 前四位
		salt := [4]byte{cc.salt[0], cc.salt[1], cc.salt[2], cc.salt[3]}
		authRequest := pgproto3.AuthenticationMD5Password{Salt: salt}
		if err = cc.WriteData(authRequest.Encode(nil)); err != nil {
			return errors.New("write AuthenticationMD5Password to client failed: " + err.Error())
		}

		if err = cc.flush(ctx); err != nil {
			return err
		}
	} else {
		// 不需要密码则直接空密码验证
		if !cc.ctx.Auth(user, auth, cc.salt) {
			return errAccessDenied.FastGenByArgs(cc.user, user.Hostname, hasPassword)
		}

		if err = cc.writeAuthenticationOK(ctx); err != nil {
			return err
		}

		return
	}

	// 当发送完成 AuthRequest 后，客户端（SQL Shell）会退出该连接
	// 然后等待用户重新输入了密码后再次建立连接
	// 当前仅在 SQL Shell 上运行，至少在 SQL Shell 中存在该问题
	// 所以在这里进行一个字节的读取时需要判断，当错误为 EOF 时直接正常结束即可
	// 读取客户端发来的密码
	// 格式： 'p' + len + 'password' + '0'
	// 长度 = len + password + 1
	auth, err = cc.readPacket()
	if err != nil {
		return err
	}

	if auth[0] != 'p' {
		return errors.New("received is not a password packet" + string(auth[0]))
	}

	// 去掉第一个 'p' 和最后一个结束符，中间的为认证信息
	auth = auth[1 : len(auth)-1]

	if !cc.ctx.Auth(user, auth, cc.salt) {
		return errAccessDenied.FastGenByArgs(cc.user, user.Hostname, hasPassword)
	}

	if err = cc.writeAuthenticationOK(ctx); err != nil {
		return err
	}

	return
}

// writeSSLRequest
// 服务端回复 'S'，表示同意SSL握手
// 服务端回复 'N'，表示不使用SSL握手
func (cc *clientConn) writeSSLRequest(ctx context.Context, pgRequestSSL byte) error {
	if err := cc.WriteData([]byte{pgRequestSSL}); err != nil {
		return err
	}

	return cc.flush(ctx)
}

// writeParameterStatus 用于写回一些参数给客户端 ParameterStatus
// pgAdmin 需要 client_encoding 这个参数
func (cc *clientConn) writeParameterStatus(parameters map[string]string) error {
	for k, v := range parameters {
		parameterStatus := &pgproto3.ParameterStatus{Name: k, Value: v}
		if err := cc.WriteData(parameterStatus.Encode(nil)); err != nil {
			return errors.New("write ParameterStatus to client failed: " + err.Error())
		}
	}
	return nil
}

// writeAuthenticationOK 向客户端写回 AuthenticationOK
func (cc *clientConn) writeAuthenticationOK(ctx context.Context) error {
	authOK := &pgproto3.AuthenticationOk{}
	if err := cc.WriteData(authOK.Encode(nil)); err != nil {
		logutil.Logger(ctx).Debug(err.Error())
		return err
	}
	return cc.flush(ctx)
}

// WriteRowDescription 向客户端写回 RowDescription
// 需要将ColumnInfo 转换为 RowDescription
// 这里只写向缓存，并不发送
func (cc *clientConn) WriteRowDescription(columns []*ColumnInfo) error {
	if len(columns) <= 0 || columns == nil {
		return errors.New("columnInfos is empty")
	}
	rowDescription := convertColumnInfoToRowDescription(columns)

	return cc.WriteData(rowDescription.Encode(nil))
}

// writeCommandComplete 向客户端写回 CommandComplete 表示一次查询已经完成
// 在 writeCommandComplete 之后需要向客户端写回 writeReadyForQuery 发送服务端状态
// 这里只写向缓存，并不发送
func (cc *clientConn) writeCommandComplete() error {
	stmtType := strings.ToUpper(cc.ctx.GetSessionVars().StmtCtx.StmtType)
	affectedRows := strconv.FormatUint(cc.ctx.AffectedRows(), 10)
	var msg string

	/*
		The following block of code handles tag field of command completion message based on query type
		Note that:
		For Insert statements, TiDB4PG will return "INSERT 0 {affected rows}" as there is no table oid in mysql
	*/
	switch stmtType {
	case "INSERT":
		msg = stmtType + " 0 " + affectedRows
	case "SET":
		msg = stmtType
	case "BEGIN":
		msg = stmtType
	default:
		msg = stmtType + " " + affectedRows
	}
	commandComplete := pgproto3.CommandComplete{CommandTag: []byte(msg)}
	return cc.WriteData(commandComplete.Encode(nil))
}

// writeParseComplete 向客户端写回 ParseComplete 表示解析命令完成
func (cc *clientConn) writeParseComplete() error {
	parseComplete := pgproto3.ParseComplete{}
	return cc.WriteData(parseComplete.Encode(nil))
}

// writeParameterDescription 向客户端写回 ParameterDescription
func (cc *clientConn) writeParameterDescription(paramOid []uint32) error {
	parameterDescription := pgproto3.ParameterDescription{
		ParameterOIDs: paramOid,
	}
	return cc.WriteData(parameterDescription.Encode(nil))
}

// writeNoData 向客户端写回 NoData 表示语句无数据返回
func (cc *clientConn) writeNoData() error {
	noData := pgproto3.NoData{}
	return cc.WriteData(noData.Encode(nil))
}

// writeBindComplete 向客户端写回 BindComplete 表示绑定命令完成
func (cc *clientConn) writeBindComplete() error {
	bindComplete := pgproto3.BindComplete{}
	return cc.WriteData(bindComplete.Encode(nil))
}

// writeCloseComplete 向客户端写回 BindComplete 表示绑定命令完成
func (cc *clientConn) writeCloseComplete() error {
	closeComplete := pgproto3.CloseComplete{}
	return cc.WriteData(closeComplete.Encode(nil))
}

// writeReadyForQuery 向客户端写回 ReadyForQuery
// status 为当前后端事务状态码。"I"表示空闲(没有在事务中)、"T"表示在事务中；"E"表示在失败的事务中(事务块结束前查询都回被驳回)
// 调用该方法后会清空缓存，发送缓存中的所有报文
func (cc *clientConn) writeReadyForQuery(ctx context.Context, status uint16) error {
	pgStatus, err := convertMySQLServerStatusToPgSQLServerStatus(status)
	if err != nil {
		return err
	}

	readyForQuery := &pgproto3.ReadyForQuery{TxStatus: pgStatus}
	if err := cc.WriteData(readyForQuery.Encode(nil)); err != nil {
		return err
	}

	return cc.flush(ctx)
}

func (cc *clientConn) WriteData(data []byte) error {
	if n, err := cc.pkt.bufWriter.Write(data); err != nil {
		terror.Log(errors.Trace(err))
		return errors.Trace(mysql.ErrBadConn)
	} else if n != len(data) {
		return errors.Trace(mysql.ErrBadConn)
	} else {
		return nil
	}
}

// convertColumnInfoToRowDescription 将 MySQL 的字段结构信息 转换为 PgSQL 的字段结构信息
// 将 ColumnInfo 转换为 RowDescription
func convertColumnInfoToRowDescription(columns []*ColumnInfo) pgproto3.RowDescription {
	// todo 完善两者字段结构信息的转换

	if strings.Contains(columns[0].Name, "(") {
		columns[0].Name = strings.Split(columns[0].Name, "(")[0]
	}
	fieldDescriptions := make([]pgproto3.FieldDescription, len(columns))
	for index := range columns {
		fieldDescriptions[index] = pgproto3.FieldDescription{
			Name:                 []byte(columns[index].Name),
			TableOID:             0,
			TableAttributeNumber: 0,
			DataTypeOID:          convertMySQLDataTypeToPgSQLDataType(columns[index].Type),
			DataTypeSize:         int16(columns[index].ColumnLength),
			TypeModifier:         0,
			Format:               0,
		}
	}

	return pgproto3.RowDescription{Fields: fieldDescriptions}
}

// convertMySQLDataTypeToPgSQLDataType 将 MySQL 数据类型 转换为 PgSQL 数据类型
// MySQL 数据类型为 Uint8 PgSQL数据类型为OID Uint32
// MySQL 数据类型查看 D:/GoRepository/pkg/mod/github.com/DigitalChinaOpenSource/DCParser@v0.0.0-20210108074737-814a888e05e2/mysql/type.go:17
// PgSQL 数据类型查看 D:/GoRepository/pkg/mod/github.com/jackc/pgtype@v1.6.2/pgtype.go:16
func convertMySQLDataTypeToPgSQLDataType(mysqlType uint8) uint32 {
	// todo 完善 mysql 和 pgsql 对应的数据类型

	switch mysqlType {
	case mysql.TypeUnspecified:
		return pgtype.UnknownOID
	case mysql.TypeTiny, mysql.TypeShort:
		return pgtype.Int2OID
	case mysql.TypeLong, mysql.TypeInt24:
		return pgtype.Int4OID
	case mysql.TypeFloat:
		return pgtype.Float4OID
	case mysql.TypeDouble:
		return pgtype.Float8OID
	case mysql.TypeNull: //未找到对应类型
		return pgtype.UnknownOID
	case mysql.TypeTimestamp:
		return pgtype.TimestampOID
	case mysql.TypeLonglong:
		return pgtype.Int8OID
	case mysql.TypeDate:
		return pgtype.DateOID
	case mysql.TypeDuration:
		return pgtype.TimeOID //与Time并不完全想对应
	case mysql.TypeDatetime:
		return pgtype.TimestampOID
	case mysql.TypeYear:
		return pgtype.UnknownOID //未找到对应类型
	case mysql.TypeNewDate:
		return pgtype.DateOID
	case mysql.TypeVarchar:
		return pgtype.VarcharOID
	case mysql.TypeBit:
		return pgtype.BitOID
	case mysql.TypeJSON:
		return pgtype.JSONOID
	case mysql.TypeNewDecimal:
		return pgtype.NumericOID
	case mysql.TypeEnum:
		return pgtype.UnknownOID //未找到对应类型dumpLengthEncodedStringByEndian
	case mysql.TypeSet:
		return pgtype.UnknownOID //未找到对应类型
	case mysql.TypeTinyBlob, mysql.TypeMediumBlob, mysql.TypeLongBlob, mysql.TypeBlob:
		return pgtype.ByteaOID
	case mysql.TypeVarString, mysql.TypeString:
		return pgtype.TextOID
	case mysql.TypeGeometry:
		return pgtype.UnknownOID //未找到对应类型
	default:
		return pgtype.UnknownOID
	}
}

// convertMySQLServerStatusToPgSQLServerStatus converts MySQL's server status into postgreSQL's
// In mysql, server status is a bit map of different server states, and server can be in multiple state at the same time.
// In postgres, server is in on of the following three mutually exclusive states:
// "T" for in transaction
// "I" for idle
// "E" for error
func convertMySQLServerStatusToPgSQLServerStatus(mysqlStatus uint16) (byte, error) {
	// TODO: Find error indicator and complete server status translation
	switch true {
	case mysqlStatus&mysql.ServerStatusInTrans > 0:
		return 'T', nil
	default:
		return 'I', nil
	}
}

// loadSSLCertificates 服务端加载证书
func loadSSLCertificates() (tlsConfig *tls.Config, err error) {
	tlsCert, err := tls.LoadX509KeyPair("server/certs/pgserver.crt", "server/certs/pgserver.key")
	if err != nil {
		println("Load X509 failed" + err.Error())
	}
	clientAutoPolicy := tls.RequireAndVerifyClientCert

	//建立受信任的CA池
	caCert, err := ioutil.ReadFile("server/certs/pgroot.crt")
	if err != nil {
		println("read ca file filed" + err.Error())
		return nil, nil
	}
	certPool := x509.NewCertPool()
	certPool.AppendCertsFromPEM(caCert)
	tlsConfig = &tls.Config{
		ClientAuth:   clientAutoPolicy,
		ClientCAs:    certPool,
		Certificates: []tls.Certificate{tlsCert},
	}
	return tlsConfig, nil
}

// convertMysqlErrorToPgError 从mysql消息体中获取信息转换出pgsql所需要的信息
func convertMysqlErrorToPgError(m *mysql.SQLError, te *terror.Error, sql string) (response *pgproto3.ErrorResponse, err error) {
	switch m.Code {
	case 1007:
		return handleCreateDBFail(m, te, sql)
	case 1008:
		return handleDropDBFail(m, te, sql)
	case 1045:
		return handleAccessDenied(m, te, sql)
	case 1049:
		return handleUnknownDB(m, te, sql)
	case 1050:
		return handleTableExists(m, te, sql)
	case 1051:
		return handleUndefinedTable(m, te, sql)
	case 1054:
		return handleUnknownColumn(m, te, sql)
	case 1062:
		return handleDuplicateKey(m, te, sql)
	case 1064:
		return handleParseError(m, te, sql)
	case 1068:
		return handleMultiplePKDefined(m, te, sql)
	case 1091:
		return handleCantDropFieldOrKey(m, te, sql)
	case 1109:
		return handleUnknownTableInDelete(m, te, sql)
	case 1110:
		return handleFiledSpecifiedTwice(m, te, sql)
	case 1111:
		return handleInvalidGroupFuncUse(m, te, sql)
	case 1113:
		return handleTableNoColumn(m, te, sql)
	case 1136:
		return handleColumnMisMatch(m, te, sql)
	case 1138:
		return handleInvalidUseOfNull(m, te, sql)
	case 1142:
		return handleTableAccessDenied(m, te, sql)
	case 1143:
		return handleColumnAccessDenied(m, te, sql)
	case 1146:
		return handleRelationNotExists(m, te, sql)
	case 1211:
		return handleNoPermissionToCreateUser(m, te, sql)
	case 1222:
		return handleWrongNumberOfColsInSelect(m, te, sql)
	case 1242:
		return handleSubqueryNo1Row(m, te, sql)
	case 1248:
		return handleDerivedMustHaveAlias(m, te, sql)
	case 1264:
		return handleDataOutOfRange(m, te, sql)
	case 1364:
		return handleNoDefaultValue(m, te, sql)
	case 1406:
		return handleDataTooLong(m, te, sql)
	case 1426:
		return handleTooBigPrecision(m, te, sql)
	default:
		return &pgproto3.ErrorResponse{
			Code:     "MySQL" + strconv.Itoa(int(m.Code)),
			Severity: "ERROR",
			Message:  "Unknown Error: " + m.Message,
		}, err
	}
}
