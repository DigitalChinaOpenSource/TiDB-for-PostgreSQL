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

package expression

import (
	"github.com/DigitalChinaOpenSource/DCParser/ast"
	. "github.com/pingcap/check"
	"github.com/pingcap/tidb/types"
	"github.com/pingcap/tidb/util/chunk"
	"github.com/pingcap/tidb/util/mock"
)

// TestCurrentDatabase test PostgreSQL system function current_database().
func (s *testEvaluatorSuite) TestCurrentDatabase(c *C) {
	fc := funcs[ast.PgFuncCurrentDatabase]
	ctx := mock.NewContext()
	f, err := fc.getFunction(ctx, nil)
	c.Assert(err, IsNil)
	d, err := evalBuiltinFunc(f, chunk.Row{})
	c.Assert(err, IsNil)
	c.Assert(d.Kind(), Equals, types.KindNull)
	ctx.GetSessionVars().CurrentDB = "test"
	d, err = evalBuiltinFunc(f, chunk.Row{})
	c.Assert(err, IsNil)
	c.Assert(d.GetString(), Equals, "test")
	c.Assert(f.Clone().PbCode(), Equals, f.PbCode())
}

// TestCurrentSchema test PostgreSQL system function current_schema()
func (s *testEvaluatorSuite) TestCurrentSchema(c *C) {
	fc := funcs[ast.PgFuncCurrentSchema]
	ctx := mock.NewContext()
	f, err := fc.getFunction(ctx, nil)
	c.Assert(err, IsNil)
	d, err := evalBuiltinFunc(f, chunk.Row{})
	c.Assert(err, IsNil)
	c.Assert(d.GetString(), Equals, "public")
}
