package notifications

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// TestSuite establishes a test suite
type TestSuite struct {
	suite.Suite
	*require.Assertions
}

func (m *TestSuite) SetupTest() {
	m.Assertions = require.New(m.T())
}

// Test_TestSuite runs the test suite
func Test_TestSuite(t *testing.T) {
	suite.Run(t, new(TestSuite))
}
