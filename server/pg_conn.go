package server

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/binary"
	"fmt"
	"github.com/jackc/pgio"
	"github.com/jackc/pgproto3/v2"
	"github.com/jackc/pgtype"
	"github.com/opentracing/opentracing-go"
	"github.com/pingcap/errors"
	"github.com/pingcap/failpoint"
	"github.com/pingcap/tidb/executor"
	"github.com/pingcap/tidb/parser/ast"
	"github.com/pingcap/tidb/parser/auth"
	"github.com/pingcap/tidb/parser/mysql"
	"github.com/pingcap/tidb/parser/terror"
	"github.com/pingcap/tidb/plugin"
	"github.com/pingcap/tidb/session"
	"github.com/pingcap/tidb/sessionctx/stmtctx"
	"github.com/pingcap/tidb/sessionctx/variable"
	storeerr "github.com/pingcap/tidb/store/driver/error"
	"github.com/pingcap/tidb/util/execdetails"
	"github.com/pingcap/tidb/util/hack"
	"github.com/pingcap/tidb/util/logutil"
	"github.com/pingcap/tidb/util/memory"
	"github.com/tikv/client-go/v2/util"
	"go.uber.org/zap"
	"io"
	"io/ioutil"
	"runtime"
	"runtime/pprof"
	"runtime/trace"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
	"unsafe"
)

const protocolVersionNumber = 196608 // 3.0
const sslRequestNumber = 80877103
const cancelRequestCode = 80877102
const gssEncReqNumber = 80877104
const protocolSSL = false

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
func (cc *clientConn) DoAuth(ctx context.Context, user *auth.UserIdentity, auth []byte, salt []byte) error {
	hasPassword := "NO"
	host, post, err := cc.PeerHost(hasPassword)
	user.Hostname = host
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

		cc.ctx.SetPort(post)
		return nil
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

	cc.ctx.SetPort(post)

	if err = cc.writeAuthenticationOK(ctx); err != nil {
		return err
	}

	return nil
}

// The first return value indicates whether the call of handleStmt has no side effect and can be retried.
// Currently the first return value is used to fallback to TiKV when TiFlash is down.
func (cc *clientConn) handleStmt(ctx context.Context, stmt ast.StmtNode, warns []stmtctx.SQLWarn, lastStmt bool) (bool, error) {
	ctx = context.WithValue(ctx, execdetails.StmtExecDetailKey, &execdetails.StmtExecDetails{})
	ctx = context.WithValue(ctx, util.ExecDetailsKey, &util.ExecDetails{})
	reg := trace.StartRegion(ctx, "ExecuteStmt")
	cc.audit(plugin.Starting)
	rs, err := cc.ctx.ExecuteStmt(ctx, stmt)
	reg.End()
	// The session tracker detachment from global tracker is solved in the `rs.Close` in most cases.
	// If the rs is nil, the detachment will be done in the `handleNoDelay`.
	if rs != nil {
		defer terror.Call(rs.Close)
	}
	if err != nil {
		return true, err
	}

	if lastStmt {
		cc.ctx.GetSessionVars().StmtCtx.AppendWarnings(warns)
	}

	status := cc.ctx.Status()
	if !lastStmt {
		status |= mysql.ServerMoreResultsExists
	}

	if rs != nil {
		connStatus := atomic.LoadInt32(&cc.status)
		if connStatus == connStatusShutdown {
			return false, executor.ErrQueryInterrupted
		}

		retryable, err := cc.writeResultset(ctx, rs, nil, status, 0)
		if err != nil {
			return retryable, err
		}
	} else {
		handled, err := cc.handleQuerySpecial(ctx, status)
		if handled {
			execStmt := cc.ctx.Value(session.ExecStmtVarKey)
			if execStmt != nil {
				execStmt.(*executor.ExecStmt).FinishExecuteStmt(0, err, false)
			}
		}
		if err != nil {
			return false, err
		}
	}
	return false, nil
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

// writeResultset writes data into a resultset and uses rs.Next to get row data back.
// If binary is true, the data would be encoded in BINARY format.
// serverStatus, a flag bit represents server information.
// fetchSize, the desired number of rows to be fetched each time when client uses cursor.
// retryable indicates whether the call of writeResultset has no side effect and can be retried to correct error. The call
// has side effect in cursor mode or once data has been sent to client. Currently retryable is used to fallback to TiKV when
// TiFlash is down.
// resultFormat Specifies the return format for each field, `nil` indicates return text format for all fields, []int16{1} indicates return binary format for all fields.
func (cc *clientConn) writeResultset(ctx context.Context, rs ResultSet, resultFormat []int16, serverStatus uint16, fetchSize int) (retryable bool, runErr error) {
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
	cc.initResultEncoder(ctx)
	defer cc.rsEncoder.clean()
	var err error
	if mysql.HasCursorExistsFlag(serverStatus) {
		// err = cc.writeChunksWithFetchSize(ctx, rs, serverStatus, fetchSize)
		return false, errors.New("TiDB4Pg does not support cursor for now")
	} else {
		retryable, err = cc.writeChunks(ctx, rs, resultFormat, serverStatus)
	}
	if err != nil {
		return retryable, err
	}

	return false, cc.flush(ctx)
}

// writeChunks writes data from a Chunk, which filled data by a ResultSet, into a connection.
// binary specifies the way to dump data. It throws any error while dumping data.
// serverStatus, a flag bit represents server information
// The first return value indicates whether error occurs at the first call of ResultSet.Next.
func (cc *clientConn) writeChunks(ctx context.Context, rs ResultSet,  rf []int16, serverStatus uint16) (bool, error) {
	data := cc.alloc.AllocWithLen(4, 1024)
	req := rs.NewChunk()

	// 当为预处理查询执行时，在执行完成后不需要返回 RowDescription
	gotColumnInfo := rs.IsPrepareStmt()

	firstNext := true
	var stmtDetail *execdetails.StmtExecDetails
	stmtDetailRaw := ctx.Value(execdetails.StmtExecDetailKey)
	if stmtDetailRaw != nil {
		stmtDetail = stmtDetailRaw.(*execdetails.StmtExecDetails)
	}
	for {
		failpoint.Inject("fetchNextErr", func(value failpoint.Value) {
			switch value.(string) {
			case "firstNext":
				failpoint.Return(firstNext, storeerr.ErrTiFlashServerTimeout)
			case "secondNext":
				if !firstNext {
					failpoint.Return(firstNext, storeerr.ErrTiFlashServerTimeout)
				}
			}
		})
		// Here server.tidbResultSet implements Next method.
		err := rs.Next(ctx, req)
		if err != nil {
			return firstNext, err
		}
		firstNext = false
		if !gotColumnInfo {
			// We need to call Next before we get columns.
			// Otherwise, we will get incorrect columns info.
			columns := rs.Columns()
			// err = cc.writeColumnInfo(columns, serverStatus)
			err = cc.WriteRowDescription(columns)
			if err != nil {
				return false, err
			}
			gotColumnInfo = true
		}
		rowCount := req.NumRows()
		if rowCount == 0 {
			break
		}
		reg := trace.StartRegion(ctx, "WriteClientConn")
		start := time.Now()

		// row data : 'D' + len(msg) + len(columns) + for(len(val) + val)
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
				return false, err
			}

			pgio.SetInt32(data[1:], int32(len(data[1:])))
			if err = cc.WriteData(data); err != nil {
				return false, err
			}
		}

		reg.End()
		if stmtDetail != nil {
			stmtDetail.WriteSQLRespDuration += time.Since(start)
		}
	}
	return false, cc.writeEOF(serverStatus)
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
