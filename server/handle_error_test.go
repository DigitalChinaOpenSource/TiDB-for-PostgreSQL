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
	expectedErrorPacket []byte   // the error packet captured using pgsql
}

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
	_, err = se.Execute(context.Background(), "create table t(a int)")
	c.Assert(err, IsNil)
	_, err = se.Execute(context.Background(), "insert into t values (1)")
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
	c.Assert(compareError, nil) // error during comparison must be nil

	c.Assert(isSameError, IsTrue)
}

// compare if the tidb error converts to the expected errorPacket captured from pgsql
func sameError(sql string, tidbError error, expectedErrorPacket []byte) (bool, error) {
	m, te := unpackError(tidbError)
	convertedPGError, err := convertMysqlErrorToPgError(m, te, sql)
	if err != nil {
		return false, err
	}

	expectedPGError := &pgproto3.ErrorResponse{}
	err = expectedPGError.Decode(expectedErrorPacket)
	if err != nil {
		return false, err
	}

	return samePGError(convertedPGError, expectedPGError), nil
}

// unpack a wrapped error into a mysql error and a terror.error
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
// SeverityUnlocalized
// Code
// Message
// Detail
// Hint
func samePGError(e1, e2 *pgproto3.ErrorResponse) bool {
	sameSeverity := sameString(e1.Severity, e2.Severity)
	sameSeverityUnlocalized := sameString(e1.SeverityUnlocalized, e2.SeverityUnlocalized)
	sameCode := sameString(e1.Code, e2.Code)
	sameMessage := sameString(e1.Message, e2.Message)
	sameDetail := sameString(e1.Detail, e2.Detail)
	sameHint := sameString(e1.Hint, e2.Hint)
	samePosition := e1.Position == e2.Position

	return sameSeverity && sameSeverityUnlocalized && sameCode && sameMessage && sameDetail && sameHint && samePosition
}

func sameString(s1, s2 string) bool {
	if strings.Compare(s1, s2) == 0 {
		return true
	} else {
		return false
	}
}

// TODO: handleTooBigPrecision
// TODO: handleInvalidUseOfNull
// TODO: handleTableNoColumn
// TODO: handleInvalidGroupFuncUse
// TODO: handleFiledSpecifiedTwice
// TODO: handleUnknownTableInDelete
// TODO: handleCantDropFieldOrKey
// TODO: handleMultiplePKDefined
// TODO: handleParseError
// TODO: handleDuplicateKey
// TODO: handleUnknownColumn
// TODO: handleTableExists
// TODO: handleUndefinedTable
// TODO: handleUnknownDB
// TODO: handleAccessDenied
// TODO: handleDropDBFail
// TODO: handleCreateDBFail
// TODO: handleDataOutOfRange
// TODO: handleDataTooLong
// TODO: handleWrongNumberOfColsInSelect
// TODO: handleDerivedMustHaveAlias
// TODO: handleSubqueryNo1Row
// TODO: handleNoPermissionToCreateUser
// TODO: handleTableAccessDenied
// TODO: handleNoDefaultValue
// TODO: handleColumnMisMatch
// TODO: handleRelationNotExists
// TODO: handleTypeError
