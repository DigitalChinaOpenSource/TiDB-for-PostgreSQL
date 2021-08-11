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
// +build !race

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
	"fmt"
	"sync/atomic"
	"time"

	tmysql "github.com/DigitalChinaOpenSource/DCParser/mysql"
	. "github.com/pingcap/check"
	"github.com/pingcap/tidb/config"
	"github.com/pingcap/tidb/domain"
	"github.com/pingcap/tidb/kv"
	"github.com/pingcap/tidb/metrics"
	"github.com/pingcap/tidb/session"
	"github.com/pingcap/tidb/sessionctx/variable"
	"github.com/pingcap/tidb/store/mockstore"
	"github.com/pingcap/tidb/util/logutil"
	"github.com/pingcap/tidb/util/testkit"
)

type tidbTestSuite struct {
	*tidbTestSuiteBase
}

type tidbTestSerialSuite struct {
	*tidbTestSuiteBase
}

type tidbTestSuiteBase struct {
	*testServerClient
	tidbdrv *TiDBDriver
	server  *Server
	domain  *domain.Domain
	store   kv.Storage
}

func newTiDBTestSuiteBase() *tidbTestSuiteBase {
	return &tidbTestSuiteBase{
		testServerClient: newTestServerClient(),
	}
}

var _ = Suite(&tidbTestSuite{newTiDBTestSuiteBase()})
var _ = SerialSuites(&tidbTestSerialSuite{newTiDBTestSuiteBase()})

func (ts *tidbTestSuite) SetUpSuite(c *C) {
	metrics.RegisterMetrics()
	ts.tidbTestSuiteBase.SetUpSuite(c)
}

func (ts *tidbTestSuiteBase) SetUpSuite(c *C) {
	var err error
	ts.store, err = mockstore.NewMockTikvStore()
	session.DisableStats4Test()
	c.Assert(err, IsNil)
	ts.domain, err = session.BootstrapSession(ts.store)
	c.Assert(err, IsNil)
	ts.tidbdrv = NewTiDBDriver(ts.store)
	cfg := config.NewConfig()
	cfg.Port = ts.port
	cfg.Status.ReportStatus = true
	cfg.Status.StatusPort = ts.statusPort
	cfg.Performance.TCPKeepAlive = true
	err = logutil.InitLogger(cfg.Log.ToLogConfig())
	c.Assert(err, IsNil)

	server, err := NewServer(cfg, ts.tidbdrv)
	c.Assert(err, IsNil)
	ts.port = getPortFromTCPAddr(server.listener.Addr())
	ts.statusPort = getPortFromTCPAddr(server.statusListener.Addr())
	ts.server = server
	go ts.server.Run()
	ts.waitUntilServerOnline()
}

func (ts *tidbTestSuiteBase) TearDownSuite(c *C) {
	if ts.store != nil {
		ts.store.Close()
	}
	if ts.domain != nil {
		ts.domain.Close()
	}
	if ts.server != nil {
		ts.server.Close()
	}
}

func (ts *tidbTestSuite) TestRegression(c *C) {
	if regression {
		c.Parallel()
		ts.runTestRegression(c, nil, "Regression")
	}
}

func (ts *tidbTestSuite) TestUint64(c *C) {
	ts.runTestPrepareResultFieldType(c)
}

func (ts *tidbTestSuite) TestSpecialType(c *C) {
	c.Parallel()
	ts.runTestSpecialType(c)
}

func (ts *tidbTestSuite) TestPreparedString(c *C) {
	c.Parallel()
	ts.runTestPreparedString(c)
}

func (ts *tidbTestSuite) TestPreparedTimestamp(c *C) {
	c.Parallel()
	ts.runTestPreparedTimestamp(c)
}

// TODO: Add Load Data functionality to TiDB for PG
//// this test will change `kv.TxnTotalSizeLimit` which may affect other test suites,
//// so we must make it running in serial.
//func (ts *tidbTestSerialSuite) TestLoadData(c *C) {
//	ts.runTestLoadData(c, ts.server)
//	ts.runTestLoadDataWithSelectIntoOutfile(c, ts.server)
//	ts.runTestLoadDataForSlowLog(c, ts.server)
//}

func (ts *tidbTestSerialSuite) TestExplainFor(c *C) {
	ts.runTestExplainForConn(c)
}

func (ts *tidbTestSerialSuite) TestStmtCount(c *C) {
	ts.runTestStmtCount(c)
}

func (ts *tidbTestSuite) TestConcurrentUpdate(c *C) {
	c.Parallel()
	ts.runTestConcurrentUpdate(c)
}

func (ts *tidbTestSuite) TestAuth(c *C) {
	c.Parallel()
	ts.runTestAuth(c)
	ts.runTestIssue3682(c)
}

func (ts *tidbTestSuite) TestIssues(c *C) {
	c.Parallel()
	ts.runTestIssue3662(c)
	ts.runTestIssue3680(c)
	ts.runTestIssue22646(c)
}

func (ts *tidbTestSuite) TestDBNameEscape(c *C) {
	c.Parallel()
	ts.runTestDBNameEscape(c)
}

func (ts *tidbTestSuite) TestResultFieldTableIsNull(c *C) {
	c.Parallel()
	ts.runTestResultFieldTableIsNull(c)
}

func (ts *tidbTestSuite) TestStatusAPI(c *C) {
	c.Parallel()
	ts.runTestStatusAPI(c)
}

func (ts *tidbTestSuite) TestStatusPort(c *C) {
	var err error
	ts.store, err = mockstore.NewMockTikvStore()
	session.DisableStats4Test()
	c.Assert(err, IsNil)
	ts.domain, err = session.BootstrapSession(ts.store)
	c.Assert(err, IsNil)
	ts.tidbdrv = NewTiDBDriver(ts.store)
	cfg := config.NewConfig()
	cfg.Port = 0
	cfg.Status.ReportStatus = true
	cfg.Status.StatusPort = ts.statusPort
	cfg.Performance.TCPKeepAlive = true

	server, err := NewServer(cfg, ts.tidbdrv)
	c.Assert(err, NotNil)
	c.Assert(err.Error(), Equals,
		fmt.Sprintf("listen tcp 0.0.0.0:%d: bind: address already in use", ts.statusPort))
	c.Assert(server, IsNil)
}

func (ts *tidbTestSuite) TestSystemTimeZone(c *C) {
	tk := testkit.NewTestKit(c, ts.store)
	cfg := config.NewConfig()
	cfg.Port, cfg.Status.StatusPort = 0, 0
	cfg.Status.ReportStatus = false
	server, err := NewServer(cfg, ts.tidbdrv)
	c.Assert(err, IsNil)
	defer server.Close()

	tz1 := tk.MustQuery("select variable_value from mysql.tidb where variable_name = 'system_tz'").Rows()
	tk.MustQuery("select @@system_time_zone").Check(tz1)
}

func (ts *tidbTestSuite) TestCreateTableFlen(c *C) {
	// issue #4540
	qctx, err := ts.tidbdrv.OpenCtx(uint64(0), 0, uint8(tmysql.DefaultCollationID), "test", nil)
	c.Assert(err, IsNil)
	_, err = Execute(context.Background(), qctx, "use test;")
	c.Assert(err, IsNil)

	ctx := context.Background()
	testSQL := "CREATE TABLE `t1` (" +
		"`a` char(36) NOT NULL," +
		"`b` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP," +
		"`c` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP," +
		"`d` varchar(50) DEFAULT ''," +
		"`e` char(36) NOT NULL DEFAULT ''," +
		"`f` char(36) NOT NULL DEFAULT ''," +
		"`g` char(1) NOT NULL DEFAULT 'N'," +
		"`h` varchar(100) NOT NULL," +
		"`i` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP," +
		"`j` varchar(10) DEFAULT ''," +
		"`k` varchar(10) DEFAULT ''," +
		"`l` varchar(20) DEFAULT ''," +
		"`m` varchar(20) DEFAULT ''," +
		"`n` varchar(30) DEFAULT ''," +
		"`o` varchar(100) DEFAULT ''," +
		"`p` varchar(50) DEFAULT ''," +
		"`q` varchar(50) DEFAULT ''," +
		"`r` varchar(100) DEFAULT ''," +
		"`s` varchar(20) DEFAULT ''," +
		"`t` varchar(50) DEFAULT ''," +
		"`u` varchar(100) DEFAULT ''," +
		"`v` varchar(50) DEFAULT ''," +
		"`w` varchar(300) NOT NULL," +
		"`x` varchar(250) DEFAULT ''," +
		"`y` decimal(20)," +
		"`z` decimal(20, 4)," +
		"PRIMARY KEY (`a`)" +
		") ENGINE=InnoDB DEFAULT CHARSET=utf8 COLLATE=utf8_bin"
	_, err = Execute(ctx, qctx, testSQL)
	c.Assert(err, IsNil)
	rs, err := Execute(ctx, qctx, "show create table t1")
	c.Assert(err, IsNil)
	req := rs.NewChunk()
	err = rs.Next(ctx, req)
	c.Assert(err, IsNil)
	cols := rs.Columns()
	c.Assert(err, IsNil)
	c.Assert(len(cols), Equals, 2)
	c.Assert(int(cols[0].ColumnLength), Equals, 5*tmysql.MaxBytesOfCharacter)
	c.Assert(int(cols[1].ColumnLength), Equals, len(req.GetRow(0).GetString(1))*tmysql.MaxBytesOfCharacter)

	// for issue#5246
	rs, err = Execute(ctx, qctx, "select y, z from t1")
	c.Assert(err, IsNil)
	cols = rs.Columns()
	c.Assert(len(cols), Equals, 2)
	c.Assert(int(cols[0].ColumnLength), Equals, 21)
	c.Assert(int(cols[1].ColumnLength), Equals, 22)
}

func Execute(ctx context.Context, qc QueryCtx, sql string) (ResultSet, error) {
	stmts, err := qc.Parse(ctx, sql)
	if err != nil {
		return nil, err
	}
	if len(stmts) != 1 {
		panic("wrong input for Execute: " + sql)
	}
	return qc.ExecuteStmt(ctx, stmts[0])
}

func (ts *tidbTestSuite) TestShowTablesFlen(c *C) {
	qctx, err := ts.tidbdrv.OpenCtx(uint64(0), 0, uint8(tmysql.DefaultCollationID), "test", nil)
	c.Assert(err, IsNil)
	ctx := context.Background()
	_, err = Execute(ctx, qctx, "use test;")
	c.Assert(err, IsNil)

	testSQL := "create table abcdefghijklmnopqrstuvwxyz (i int)"
	_, err = Execute(ctx, qctx, testSQL)
	c.Assert(err, IsNil)
	rs, err := Execute(ctx, qctx, "show tables")
	c.Assert(err, IsNil)
	req := rs.NewChunk()
	err = rs.Next(ctx, req)
	c.Assert(err, IsNil)
	cols := rs.Columns()
	c.Assert(err, IsNil)
	c.Assert(len(cols), Equals, 1)
	c.Assert(int(cols[0].ColumnLength), Equals, 26*tmysql.MaxBytesOfCharacter)
}

func checkColNames(c *C, columns []*ColumnInfo, names ...string) {
	for i, name := range names {
		c.Assert(columns[i].Name, Equals, name)
		c.Assert(columns[i].OrgName, Equals, name)
	}
}

func (ts *tidbTestSuite) TestFieldList(c *C) {
	qctx, err := ts.tidbdrv.OpenCtx(uint64(0), 0, uint8(tmysql.DefaultCollationID), "test", nil)
	c.Assert(err, IsNil)
	_, err = Execute(context.Background(), qctx, "use test;")
	c.Assert(err, IsNil)

	ctx := context.Background()
	testSQL := `create table t (
		c_bit bit(10),
		c_int_d int,
		c_bigint_d bigint,
		c_float_d float,
		c_double_d double,
		c_decimal decimal(6, 3),
		c_datetime datetime(2),
		c_time time(3),
		c_date date,
		c_timestamp timestamp(4) DEFAULT CURRENT_TIMESTAMP(4),
		c_char char(20),
		c_varchar varchar(20),
		c_text_d text,
		c_binary binary(20),
		c_blob_d blob,
		c_set set('a', 'b', 'c'),
		c_enum enum('a', 'b', 'c'),
		c_json JSON,
		c_year year
	)`
	_, err = Execute(ctx, qctx, testSQL)
	c.Assert(err, IsNil)
	colInfos, err := qctx.FieldList("t")
	c.Assert(err, IsNil)
	c.Assert(len(colInfos), Equals, 19)

	checkColNames(c, colInfos, "c_bit", "c_int_d", "c_bigint_d", "c_float_d",
		"c_double_d", "c_decimal", "c_datetime", "c_time", "c_date", "c_timestamp",
		"c_char", "c_varchar", "c_text_d", "c_binary", "c_blob_d", "c_set", "c_enum",
		"c_json", "c_year")

	for _, cols := range colInfos {
		c.Assert(cols.Schema, Equals, "test")
	}

	for _, cols := range colInfos {
		c.Assert(cols.Table, Equals, "t")
	}

	for i, col := range colInfos {
		switch i {
		case 10, 11, 12, 15, 16:
			// c_char char(20), c_varchar varchar(20), c_text_d text,
			// c_set set('a', 'b', 'c'), c_enum enum('a', 'b', 'c')
			c.Assert(col.Charset, Equals, uint16(tmysql.CharsetNameToID(tmysql.DefaultCharset)), Commentf("index %d", i))
			continue
		}

		c.Assert(col.Charset, Equals, uint16(tmysql.CharsetNameToID("binary")), Commentf("index %d", i))
	}

	// c_decimal decimal(6, 3)
	c.Assert(colInfos[5].Decimal, Equals, uint8(3))

	// for issue#10513
	tooLongColumnAsName := "COALESCE(0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0)"
	columnAsName := tooLongColumnAsName[:tmysql.MaxAliasIdentifierLen]

	rs, err := Execute(ctx, qctx, "select "+tooLongColumnAsName)
	c.Assert(err, IsNil)
	cols := rs.Columns()
	c.Assert(cols[0].OrgName, Equals, tooLongColumnAsName)
	c.Assert(cols[0].Name, Equals, columnAsName)

	rs, err = Execute(ctx, qctx, "select c_bit as '"+tooLongColumnAsName+"' from t")
	c.Assert(err, IsNil)
	cols = rs.Columns()
	c.Assert(cols[0].OrgName, Equals, "c_bit")
	c.Assert(cols[0].Name, Equals, columnAsName)
}

func (ts *tidbTestSuite) TestSumAvg(c *C) {
	c.Parallel()
	ts.runTestSumAvg(c)
}

func (ts *tidbTestSuite) TestNullFlag(c *C) {
	// issue #9689
	qctx, err := ts.tidbdrv.OpenCtx(uint64(0), 0, uint8(tmysql.DefaultCollationID), "test", nil)
	c.Assert(err, IsNil)

	ctx := context.Background()
	rs, err := Execute(ctx, qctx, "select 1")
	c.Assert(err, IsNil)
	cols := rs.Columns()
	c.Assert(len(cols), Equals, 1)
	expectFlag := uint16(tmysql.NotNullFlag | tmysql.BinaryFlag)
	c.Assert(dumpFlag(cols[0].Type, cols[0].Flag), Equals, expectFlag)
}

func (ts *tidbTestSuite) TestNO_DEFAULT_VALUEFlag(c *C) {
	// issue #21465
	qctx, err := ts.tidbdrv.OpenCtx(uint64(0), 0, uint8(tmysql.DefaultCollationID), "test", nil)
	c.Assert(err, IsNil)

	ctx := context.Background()
	_, err = Execute(ctx, qctx, "use test")
	c.Assert(err, IsNil)
	_, err = Execute(ctx, qctx, "drop table if exists t")
	c.Assert(err, IsNil)
	_, err = Execute(ctx, qctx, "create table t(c1 int key, c2 int);")
	c.Assert(err, IsNil)
	rs, err := Execute(ctx, qctx, "select c1 from t;")
	c.Assert(err, IsNil)
	cols := rs.Columns()
	c.Assert(len(cols), Equals, 1)
	expectFlag := uint16(tmysql.NotNullFlag | tmysql.PriKeyFlag | tmysql.NoDefaultValueFlag)
	c.Assert(dumpFlag(cols[0].Type, cols[0].Flag), Equals, expectFlag)
}

func (ts *tidbTestSuite) TestGracefulShutdown(c *C) {
	var err error
	ts.store, err = mockstore.NewMockTikvStore()
	session.DisableStats4Test()
	c.Assert(err, IsNil)
	ts.domain, err = session.BootstrapSession(ts.store)
	c.Assert(err, IsNil)
	ts.tidbdrv = NewTiDBDriver(ts.store)
	cli := newTestServerClient()
	cfg := config.NewConfig()
	cfg.GracefulWaitBeforeShutdown = 2 // wait before shutdown
	cfg.Port = 0
	cfg.Status.StatusPort = 0
	cfg.Status.ReportStatus = true
	cfg.Performance.TCPKeepAlive = true
	server, err := NewServer(cfg, ts.tidbdrv)
	c.Assert(err, IsNil)
	c.Assert(server, NotNil)
	cli.port = getPortFromTCPAddr(server.listener.Addr())
	cli.statusPort = getPortFromTCPAddr(server.statusListener.Addr())
	go server.Run()
	time.Sleep(time.Millisecond * 100)

	_, err = cli.fetchStatus("/status") // server is up
	c.Assert(err, IsNil)

	go server.Close()
	time.Sleep(time.Millisecond * 500)

	resp, _ := cli.fetchStatus("/status") // should return 5xx code
	c.Assert(resp.StatusCode, Equals, 500)

	time.Sleep(time.Second * 2)

	_, err = cli.fetchStatus("/status") // status is gone
	c.Assert(err, ErrorMatches, ".*connect: connection refused")
}

func (ts *tidbTestSuite) TestPessimisticInsertSelectForUpdate(c *C) {
	qctx, err := ts.tidbdrv.OpenCtx(uint64(0), 0, uint8(tmysql.DefaultCollationID), "test", nil)
	c.Assert(err, IsNil)
	ctx := context.Background()
	_, err = Execute(ctx, qctx, "use test;")
	c.Assert(err, IsNil)
	_, err = Execute(ctx, qctx, "drop table if exists t1, t2")
	c.Assert(err, IsNil)
	_, err = Execute(ctx, qctx, "create table t1 (id int)")
	c.Assert(err, IsNil)
	_, err = Execute(ctx, qctx, "create table t2 (id int)")
	c.Assert(err, IsNil)
	_, err = Execute(ctx, qctx, "insert into t1 select 1")
	c.Assert(err, IsNil)
	_, err = Execute(ctx, qctx, "begin pessimistic")
	c.Assert(err, IsNil)
	rs, err := Execute(ctx, qctx, "INSERT INTO t2 (id) select id from t1 where id = 1 for update")
	c.Assert(err, IsNil)
	c.Assert(rs, IsNil) // should be no delay
}

func (ts *tidbTestSerialSuite) TestPrepareCount(c *C) {
	qctx, err := ts.tidbdrv.OpenCtx(uint64(0), 0, uint8(tmysql.DefaultCollationID), "test", nil)
	c.Assert(err, IsNil)
	prepareCnt := atomic.LoadInt64(&variable.PreparedStmtCount)
	ctx := context.Background()
	_, err = Execute(ctx, qctx, "use test;")
	c.Assert(err, IsNil)
	_, err = Execute(ctx, qctx, "drop table if exists t1")
	c.Assert(err, IsNil)
	_, err = Execute(ctx, qctx, "create table t1 (id int)")
	c.Assert(err, IsNil)
	stmt, _, _, err := qctx.Prepare("insert into t1 values (?)", "")
	c.Assert(err, IsNil)
	c.Assert(atomic.LoadInt64(&variable.PreparedStmtCount), Equals, prepareCnt+1)
	c.Assert(err, IsNil)
	err = qctx.GetStatement(stmt.ID()).Close()
	c.Assert(err, IsNil)
	c.Assert(atomic.LoadInt64(&variable.PreparedStmtCount), Equals, prepareCnt)
}
