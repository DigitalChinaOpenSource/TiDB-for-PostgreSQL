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
