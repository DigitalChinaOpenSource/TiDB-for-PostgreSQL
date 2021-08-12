package aggfuncs_test

import (
	"github.com/DigitalChinaOpenSource/DCParser/ast"
	"github.com/DigitalChinaOpenSource/DCParser/mysql"
	. "github.com/pingcap/check"
)

func (s *testSuite) TestMergePartialResult4Stddevpop(c *C) {
	tests := []aggTest{
		buildAggTester(ast.AggFuncStddevPop, mysql.TypeDouble, 5, 1.4142135623730951, 0.816496580927726, 1.3169567191065923),
	}
	for _, test := range tests {
		s.testMergePartialResult(c, test)
	}
}

func (s *testSuite) TestStddevpop(c *C) {
	tests := []aggTest{
		buildAggTester(ast.AggFuncStddevPop, mysql.TypeDouble, 5, nil, 1.4142135623730951),
	}
	for _, test := range tests {
		s.testAggFunc(c, test)
	}
}
