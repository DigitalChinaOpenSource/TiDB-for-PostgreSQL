package aggfuncs_test

import (
	"github.com/DigitalChinaOpenSource/DCParser/ast"
	"github.com/DigitalChinaOpenSource/DCParser/mysql"
	. "github.com/pingcap/check"
)

func (s *testSuite) TestMergePartialResult4Varsamp(c *C) {
	tests := []aggTest{
		buildAggTester(ast.AggFuncVarSamp, mysql.TypeDouble, 5, 2.5, 1, 1.9821428571428572),
	}
	for _, test := range tests {
		s.testMergePartialResult(c, test)
	}
}

func (s *testSuite) TestVarsamp(c *C) {
	tests := []aggTest{
		buildAggTester(ast.AggFuncVarSamp, mysql.TypeDouble, 5, nil, 2.5),
	}
	for _, test := range tests {
		s.testAggFunc(c, test)
	}
}
