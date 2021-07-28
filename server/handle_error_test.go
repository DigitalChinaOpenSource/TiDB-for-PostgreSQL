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
	"encoding/hex"
	"github.com/DigitalChinaOpenSource/DCParser/mysql"
	"github.com/DigitalChinaOpenSource/DCParser/terror"
	"github.com/jackc/pgproto3/v2"
	. "github.com/pingcap/check"
	"github.com/pingcap/errors"
	"github.com/pingcap/tidb/config"
	"github.com/pingcap/tidb/domain"
	"github.com/pingcap/tidb/kv"
	"github.com/pingcap/tidb/session"
	"github.com/pingcap/tidb/store/mockstore"
)

type HandleErrorTestSuite struct {
	dom   *domain.Domain
	store kv.Storage
}

var _ = Suite(&HandleErrorTestSuite{})

// Here is the basic info a single error conversion test needed
type testCase struct {
	setupSQLs           []string // a list of sql to execute to setup the error
	triggerSQL          string   // the sql that should trigger the error
	expectedErrorPacket string   // the hex stream dump error packet captured using pgsql
}

// Test option controls what to compare during a test
type testOption struct {
	compareSeverity bool
	compareCode     bool
	compareMessage  bool
	compareDetail   bool
	compareHint     bool
	comparePosition bool
}

// default test option compares the following
var defaultTestOption = testOption{
	compareSeverity: true,
	compareCode:     true,
	compareMessage:  true,
	compareDetail:   true,
	compareHint:     true,
	comparePosition: true,
}

/*
	Skipped tests:
	handleNoPermissionToCreateUser:		Permission module related
	handleTableAccessDenied: 			Permission module related
	handleColumnAccessDenied:			Permission module related
	handleAccessDenied: 				Permission module related

	handleTableNoColumn: 				Postgres allows table with no column
*/

func (ts *HandleErrorTestSuite) TestHandleUndefinedTable(c *C) {
	c.Parallel()
	testcase := testCase{
		setupSQLs: []string{
			"drop table if exists testundefinedtable;",
		},
		triggerSQL:
		"drop table testundefinedtable;",
		expectedErrorPacket:
		"4500000071534552524f5200564552524f5200433432503031004d7461626c65202274657374756e646566696e65647461626c652220646f6573206e6f7420657869737400467461626c65636d64732e63004c31323136005244726f704572726f724d73674e6f6e4578697374656e740000",
	}

	ts.testErrorConversion(c, testcase, defaultTestOption)
}

func (ts *HandleErrorTestSuite) TestHandleInvalidGroupFuncUse(c *C) {
	c.Parallel()
	testcase := testCase{
		setupSQLs: []string{
			"drop table if exists testhandleinvalidgroupfuncuse;",
			"create table testhandleinvalidgroupfuncuse(a int);",
		},
		triggerSQL:
		"select * from testhandleinvalidgroupfuncuse where sum(a) > 1000;",
		expectedErrorPacket:
		"450000007f534552524f5200564552524f5200433432383033004d6167677265676174652066756e6374696f6e7320617265206e6f7420616c6c6f77656420696e20574845524500503531004670617273655f6167672e63004c3537360052636865636b5f6167676c6576656c735f616e645f636f6e73747261696e74730000",
	}

	ts.testErrorConversion(c, testcase, defaultTestOption)
}

func (ts *HandleErrorTestSuite) TestHandleFiledSpecifiedTwice(c *C) {
	c.Parallel()
	testcase := testCase{
		setupSQLs: []string{
			"drop table if exists testfieldspecifiedtwice;",
			"create table testfieldspecifiedtwice(a int);",
		},
		triggerSQL:
		"insert into testfieldspecifiedtwice(a, a) values(10, 10);",
		expectedErrorPacket:
		"450000006d534552524f5200564552524f5200433432373031004d636f6c756d6e2022612220737065636966696564206d6f7265207468616e206f6e636500503430004670617273655f7461726765742e63004c313035340052636865636b496e73657274546172676574730000",
	}

	ts.testErrorConversion(c, testcase, defaultTestOption)
}

func (ts *HandleErrorTestSuite) TestHandleUnknownTableInDelete(c *C) {
	c.Parallel()
	testcase := testCase{
		setupSQLs: []string{
			"drop table if exists test_table;",
			"create table test_table(a int);",
		},
		triggerSQL:
		"delete a from test_table;",
		expectedErrorPacket:
		"4500000059534552524f5200564552524f5200433432363031004d73796e746178206572726f72206174206f72206e6561722022612200503800467363616e2e6c004c3131383000527363616e6e65725f79796572726f720000",
	}

	ts.testErrorConversion(c, testcase, defaultTestOption)
}

func (ts *HandleErrorTestSuite) TestHandleCantDropFieldOrKey(c *C) {
	c.Parallel()
	testcase := testCase{
		setupSQLs: []string{
			"drop table if exists test_table;",
			"create table test_table(a int);",
		},
		triggerSQL:
		"alter table test_table drop column b;",
		expectedErrorPacket:
		"4500000073534552524f5200564552524f5200433432373033004d636f6c756d6e20226222206f662072656c6174696f6e2022746573745f7461626c652220646f6573206e6f7420657869737400467461626c65636d64732e63004c37373930005241544578656344726f70436f6c756d6e0000",
	}

	ts.testErrorConversion(c, testcase, defaultTestOption)
}

func (ts *HandleErrorTestSuite) TestHandleMultiplePKDefined(c *C) {
	c.Parallel()
	testcase := testCase{
		setupSQLs: []string{
			"drop table if exists test_table;",
		},
		triggerSQL:
		"create table test_table(a int primary key, b int primary key);",
		expectedErrorPacket:
		"450000008d534552524f5200564552524f5200433432503136004d6d756c7469706c65207072696d617279206b65797320666f72207461626c652022746573745f7461626c652220617265206e6f7420616c6c6f77656400503530004670617273655f7574696c636d642e63004c3231333300527472616e73666f726d496e646578436f6e73747261696e740000",
	}

	ts.testErrorConversion(c, testcase, defaultTestOption)
}

func (ts *HandleErrorTestSuite) TestHandleParseError(c *C) {
	c.Parallel()
	testcase := testCase{
		setupSQLs: []string{
			"drop table if exists test_table;",
		},
		triggerSQL:
		"creat table test_table(a int);", //create spelled wrong intentionally
		expectedErrorPacket:
		"450000005d534552524f5200564552524f5200433432363031004d73796e746178206572726f72206174206f72206e656172202263726561742200503100467363616e2e6c004c3131383000527363616e6e65725f79796572726f720000",
	}

	ts.testErrorConversion(c, testcase, defaultTestOption)
}

// TODO: Change the test method so it doesn't compare message
func (ts *HandleErrorTestSuite) TestHandleDuplicateKey(c *C) {
	c.Parallel()
	testcase := testCase{
		setupSQLs: []string{
			"drop table if exists test_table;",
			"create table test_table(a int, primary key (a));",
			"insert into test_table values(1);",
		},
		triggerSQL:
		"insert into test_table values(1);",
		expectedErrorPacket:
		"45000000c2534552524f5200564552524f5200433233353035004d6475706c6963617465206b65792076616c75652076696f6c6174657320756e6971756520636f6e73747261696e742022746573745f7461626c655f706b65792200444b6579202861293d28312920616c7265616479206578697374732e00737075626c69630074746573745f7461626c65006e746573745f7461626c655f706b657900466e6274696e736572742e63004c36353600525f62745f636865636b5f756e697175650000",
	}
	ts.testErrorConversion(c, testcase, defaultTestOption)
}

func (ts *HandleErrorTestSuite) TestHandleUnknownColumn(c *C) {
	c.Parallel()
	testcase := testCase{
		setupSQLs: []string{
			"drop table if exists test_table;",
			"create table test_table(a int);",
		},
		triggerSQL:
		"insert into test_table(bbb) values(1);",
		expectedErrorPacket:
		"450000007e534552524f5200564552524f5200433432373033004d636f6c756d6e202262626222206f662072656c6174696f6e2022746573745f7461626c652220646f6573206e6f7420657869737400503234004670617273655f7461726765742e63004c313033390052636865636b496e73657274546172676574730000",
	}

	ts.testErrorConversion(c, testcase, defaultTestOption)
}

func (ts *HandleErrorTestSuite) TestHandleTableExists(c *C) {
	c.Parallel()
	testcase := testCase{
		setupSQLs: []string{
			"drop table if exists test_table;",
			"create table test_table(a int);",
		},
		triggerSQL:
		"create table test_table(a int);",
		expectedErrorPacket:
		"4500000068534552524f5200564552524f5200433432503037004d72656c6174696f6e2022746573745f7461626c652220616c7265616479206578697374730046686561702e63004c313136340052686561705f6372656174655f776974685f636174616c6f670000",
	}

	ts.testErrorConversion(c, testcase, defaultTestOption)
}
func (ts *HandleErrorTestSuite) TestHandleUnknownDB(c *C) {
	c.Parallel()
	testcase := testCase{
		setupSQLs: []string{
			"drop database if exists test_db;",
		},
		triggerSQL:
		"use test_db;",
		expectedErrorPacket:
		"450000005c53464154414c0056464154414c00433344303030004d64617461626173652022746573745f64622220646f6573206e6f742065786973740046706f7374696e69742e63004c3837370052496e6974506f7374677265730000",
	}

	ts.testErrorConversion(c, testcase, defaultTestOption)
}

func (ts *HandleErrorTestSuite) TestHandleDropDBFail(c *C) {
	c.Parallel()
	testcase := testCase{
		setupSQLs: []string{
			"drop database if exists test_db;",
		},
		triggerSQL:
		"drop database test_db;",
		expectedErrorPacket:
		"4500000058534552524f5200564552524f5200433344303030004d64617461626173652022746573745f64622220646f6573206e6f7420657869737400466462636f6d6d616e64732e63004c383431005264726f7064620000",
	}

	ts.testErrorConversion(c, testcase, defaultTestOption)
}
func (ts *HandleErrorTestSuite) TestHandleCreateDBFail(c *C) {
	c.Parallel()
	testcase := testCase{
		setupSQLs: []string{
			"drop database if exists test_db;",
			"create database test_db;",
		},
		triggerSQL:
		"create database test_db;",
		expectedErrorPacket:
		"450000005a534552524f5200564552524f5200433432503034004d64617461626173652022746573745f64622220616c72656164792065786973747300466462636f6d6d616e64732e63004c353132005263726561746564620000",
	}

	ts.testErrorConversion(c, testcase, defaultTestOption)
}

// TODO: Change this so it doesn't compare string
//... obtained string = "column 'a' value out of range"
//... expected string = "integer out of range"
func (ts *HandleErrorTestSuite) TestHandleDataOutOfRange(c *C) {
	c.Parallel()
	testcase := testCase{
		setupSQLs: []string{
			"drop table if exists test_table;",
			"create table test_table(a int);",
		},
		triggerSQL:
		"insert into test_table values(2147483647 + 1);", //max for signed INT + 1
		expectedErrorPacket:
		"4500000044534552524f5200564552524f5200433232303033004d696e7465676572206f7574206f662072616e67650046696e742e63004c3738310052696e7434706c0000",
	}

	ts.testErrorConversion(c, testcase, defaultTestOption)
}

// TODO: Change test function so it doen't compare message
func (ts *HandleErrorTestSuite) TestHandleDataTooLong(c *C) {
	c.Parallel()
	testcase := testCase{
		setupSQLs: []string{
			"drop table if exists test_table;",
			"create table test_table(a bit(3));", // don't use char type to test this, it will result in stirng right truncation error, error code 22001
		},
		triggerSQL:
		"insert into test_table values('1000000');",
		expectedErrorPacket:
		"450000005e534552524f5200564552524f5200433232303236004d62697420737472696e67206c656e677468203720646f6573206e6f74206d6174636820747970652062697428332900467661726269742e63004c34303600526269740000",
	}

	ts.testErrorConversion(c, testcase, defaultTestOption)
}

// TODO: Change this so it doesn't compare position
func (ts *HandleErrorTestSuite) TestHandleWrongNumberOfColsInSelect(c *C) {
	c.Parallel()
	testcase := testCase{
		setupSQLs: []string{
			"drop table if exists test_table;",
			"drop table if exists test_table2;",
			"create table test_table(a int);",
			"create table test_table2(a int, b int);",
		},
		triggerSQL:
		"select * from test_table UNION select * from test_table2;",
		expectedErrorPacket:
		"4500000081534552524f5200564552524f5200433432363031004d6561636820554e494f4e207175657279206d7573742068617665207468652073616d65206e756d626572206f6620636f6c756d6e73005033390046616e616c797a652e63004c3230303700527472616e73666f726d5365744f7065726174696f6e547265650000",
	}

	ts.testErrorConversion(c, testcase, defaultTestOption)
}
func (ts *HandleErrorTestSuite) TestHandleDerivedMustHaveAlias(c *C) {
	c.Parallel()
	testcase := testCase{
		setupSQLs: []string{
			"drop table if exists test_table;",
			"create table test_table(a int);",
		},
		triggerSQL:
		"select * from (select * from test_table);",
		expectedErrorPacket:
		"450000008a534552524f5200564552524f5200433432363031004d737562717565727920696e2046524f4d206d757374206861766520616e20616c6961730048466f72206578616d706c652c2046524f4d202853454c454354202e2e2e29205b41535d20666f6f2e0050313500466772616d2e79004c31323131350052626173655f797970617273650000",
	}

	ts.testErrorConversion(c, testcase, defaultTestOption)
}

func (ts *HandleErrorTestSuite) TestHandleSubqueryNo1Row(c *C) {
	c.Parallel()
	testcase := testCase{
		setupSQLs: []string{
			"drop table if exists test_table;",
			"create table test_table(a int);",
			"insert into test_table values(1);",
			"insert into test_table values(2);",
		},
		triggerSQL:
		"select * from test_table where a = (select a from test_table);",
		expectedErrorPacket:
		"4500000081534552524f5200564552524f5200433231303030004d6d6f7265207468616e206f6e6520726f772072657475726e65642062792061207375627175657279207573656420617320616e2065787072657373696f6e00466e6f6465537562706c616e2e63004c31313539005245786563536574506172616d506c616e0000",
	}

	ts.testErrorConversion(c, testcase, defaultTestOption)
}

// TODO: Change test method so this one doesnt compare detail information
func (ts *HandleErrorTestSuite) TestHandleNoDefaultValue(c *C) {
	c.Parallel()
	testcase := testCase{
		setupSQLs: []string{
			"drop table if exists test_table;",
			"create table test_table(a int, b varchar(1) not null);",
		},
		triggerSQL:
		"insert into test_table(a) values (1);",
		expectedErrorPacket:
		"45000000c5534552524f5200564552524f5200433233353032004d6e756c6c2076616c756520696e20636f6c756d6e20226222206f662072656c6174696f6e2022746573745f7461626c65222076696f6c61746573206e6f742d6e756c6c20636f6e73747261696e7400444661696c696e6720726f7720636f6e7461696e732028312c206e756c6c292e00737075626c69630074746573745f7461626c650063620046657865634d61696e2e63004c31393533005245786563436f6e73747261696e74730000",
	}

	ts.testErrorConversion(c, testcase, defaultTestOption)
}

// TODO: Change this so it doesn't compare position
func (ts *HandleErrorTestSuite) TestHandleColumnMisMatch(c *C) {
	c.Parallel()
	testcase := testCase{
		setupSQLs: []string{
			"drop table if exists test_table;",
			"create table test_table(a int);",
		},
		triggerSQL:
		"insert into test_table values (1,2);",
		expectedErrorPacket:
		"4500000073534552524f5200564552524f5200433432363031004d494e5345525420686173206d6f72652065787072657373696f6e73207468616e2074617267657420636f6c756d6e73005033340046616e616c797a652e63004c39303700527472616e73666f726d496e73657274526f770000",
	}

	ts.testErrorConversion(c, testcase, defaultTestOption)
}

func (ts *HandleErrorTestSuite) TestHandleRelationNotExists(c *C) {
	c.Parallel()
	testcase := testCase{
		setupSQLs: []string{
			"drop table if exists test_table;",
		},
		triggerSQL:
		"insert into test_table values(1);",
		expectedErrorPacket:
		"450000006d534552524f5200564552524f5200433432503031004d72656c6174696f6e2022746573745f7461626c652220646f6573206e6f7420657869737400503133004670617273655f72656c6174696f6e2e63004c3133373600527061727365724f70656e5461626c650000",
	}

	ts.testErrorConversion(c, testcase, defaultTestOption)
}

// testErrorConversion does the actual comparison, will be called by the various tests
func (ts *HandleErrorTestSuite) testErrorConversion(c *C, inputCase testCase, option testOption) {
	store, err := mockstore.NewMockTikvStore()
	c.Assert(err, IsNil)
	defer store.Close()
	dom, err := session.BootstrapSession(store)
	c.Assert(err, IsNil)
	defer dom.Close()

	se, err := session.CreateSession4Test(store)
	c.Assert(err, IsNil)
	_, err = se.Execute(context.Background(), "use test")
	c.Assert(err, IsNil)

	tidbdrv := NewTiDBDriver(ts.store)
	cfg := config.NewConfig()
	cfg.Port = 0
	cfg.Status.ReportStatus = false
	server, err := NewServer(cfg, tidbdrv)

	c.Assert(err, IsNil)
	defer server.Close()

	// execute the setup SQLs
	for _, setupSQL := range inputCase.setupSQLs {
		_, err = se.Execute(context.Background(), setupSQL)
		c.Assert(err, IsNil) //error must be nil during setup
	}

	// execute the trigger SQL
	_, err = se.Execute(context.Background(), inputCase.triggerSQL)

	compareError := sameError(inputCase.triggerSQL, err, inputCase.expectedErrorPacket, option, c)
	c.Assert(compareError, IsNil) // error during comparison must be nil
}

// sameError compare if the tidb error converts to the expected errorPacket captured from pgsql
func sameError(sql string, tidbError error, expectedErrorPacket string, option testOption, c *C) error {
	m, te := unpackError(tidbError)
	convertedPGError, err := convertMysqlErrorToPgError(m, te, sql)
	if err != nil {
		return err
	}

	expectedPGError := &pgproto3.ErrorResponse{}
	// convert the hex stream to byte stream
	expected, _ := hex.DecodeString(expectedErrorPacket)
	// remove the first 5 bytes: 4bytes for error, 1 bytes for length
	expected = expected[5:]
	err = expectedPGError.Decode(expected)
	if err != nil {
		return err
	}
	samePGError(convertedPGError, expectedPGError, option, c)
	return nil
}

// unpackError unpack a wrapped error into a mysql error and a terror.error
func unpackError(e error) (*mysql.SQLError, *terror.Error) {
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
	return m, te
}

// samePGError will check if two pgproto3 Error response are functionally the same according to the test option given
func samePGError(e1, e2 *pgproto3.ErrorResponse, option testOption, c *C) {
	if option.compareSeverity {
		c.Assert(e1.Severity, DeepEquals, e2.Severity)
	}
	if option.compareCode {
		c.Assert(e1.Code, DeepEquals, e2.Code)
	}
	if option.compareMessage {
		c.Assert(e1.Message, DeepEquals, e2.Message)
	}
	if option.compareDetail {
		c.Assert(e1.Detail, DeepEquals, e2.Detail)
	}
	if option.compareHint {
		c.Assert(e1.Hint, DeepEquals, e2.Hint)
	}
	if option.comparePosition {
		c.Assert(e1.Position, Equals, e2.Position)
	}
}
