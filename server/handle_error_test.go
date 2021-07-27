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
	"strings"
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

	ts.testErrorConversion(c, testcase)
}

/*
	Skipped Testing handleTableNoColumn since this is not a error in postgresql as pgsql allows table with no column
 */


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

	ts.testErrorConversion(c, testcase)
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

	ts.testErrorConversion(c, testcase)
}


// testErrorConversion does the actual comparison, will be called by the various tests
func (ts *HandleErrorTestSuite) testErrorConversion(c *C, inputCase testCase) {
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

	isSameError, compareError := sameError(inputCase.triggerSQL, err, inputCase.expectedErrorPacket)
	c.Assert(compareError, IsNil) // error during comparison must be nil

	c.Assert(isSameError, IsTrue)
}

// sameError compare if the tidb error converts to the expected errorPacket captured from pgsql
func sameError(sql string, tidbError error, expectedErrorPacket string) (bool, error) {
	m, te := unpackError(tidbError)
	convertedPGError, err := convertMysqlErrorToPgError(m, te, sql)
	if err != nil {
		return false, err
	}

	expectedPGError := &pgproto3.ErrorResponse{}
	// convert the hex stream to byte stream
	expected, _ := hex.DecodeString(expectedErrorPacket)
	// remove the first 5 bytes: 4bytes for error, 1 bytes for length
	expected = expected[5:]
	err = expectedPGError.Decode(expected)
	if err != nil {
		return false, err
	}
	return samePGError(convertedPGError, expectedPGError), nil
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

// samePGError will check if two pgproto3 Error response are functionally the same
// Note that this will not compare every field as TiDB and PG server implement differently, it will compare:
// Severity
// Code
// Message
// Detail
// Hint
func samePGError(e1, e2 *pgproto3.ErrorResponse) bool {
	sameSeverity := sameString(e1.Severity, e2.Severity)
	sameCode := sameString(e1.Code, e2.Code)
	sameMessage := sameString(e1.Message, e2.Message)
	sameDetail := sameString(e1.Detail, e2.Detail)
	sameHint := sameString(e1.Hint, e2.Hint)
	samePosition := e1.Position == e2.Position

	return sameSeverity && sameCode && sameMessage && sameDetail && sameHint && samePosition
}

// sameString check if two string are lexically the same, return true if the same, false otherwise
func sameString(s1, s2 string) bool {
	if strings.Compare(s1, s2) == 0 {
		return true
	} else {
		return false
	}
}

