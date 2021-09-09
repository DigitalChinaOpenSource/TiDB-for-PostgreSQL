// Copyright 2016 PingCAP, Inc.
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

package infoschema_test

import (
	"github.com/DigitalChinaOpenSource/DCParser/model"
	. "github.com/pingcap/check"
	"github.com/pingcap/tidb/infoschema"
	"github.com/pingcap/tidb/store/mockstore"
	"github.com/pingcap/tidb/util"
	"github.com/pingcap/tidb/util/testleak"
)

var _ = Suite(&testSuite{})

// TestInfoTables makes sure that all tables of pg_catalog could be found in infoschema handle.
func (*testSuite) TestCatalogTables(c *C) {
	defer testleak.AfterTest(c)()
	store, err := mockstore.NewMockTikvStore()
	c.Assert(err, IsNil)
	defer store.Close()
	handle := infoschema.NewHandle(store)
	builder, err := infoschema.NewBuilder(handle).InitWithDBInfos(nil, 0)
	c.Assert(err, IsNil)
	builder.Build()
	is := handle.Get()
	c.Assert(is, NotNil)

	infoTables := []string{
		"pg_aggregate",
		"pg_class",
		"pg_inherits",
		"pg_namespace",
		"pg_partitioned_table",
	}
	for _, t := range infoTables {
		tb, err1 := is.TableByName(util.PgCatalogName, model.NewCIStr(t))
		c.Assert(err1, IsNil)
		c.Assert(tb, NotNil)
	}
}
