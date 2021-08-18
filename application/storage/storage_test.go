package storage

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

// TestSuite establishes a test suite
type TestSuite struct {
	suite.Suite
}

// Test_TestSuite runs the test suite
func Test_TestSuite(t *testing.T) {
	suite.Run(t, new(TestSuite))
}
