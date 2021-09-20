package api

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

// TestSuite establishes a test suite for domain tests
type TestSuite struct {
	suite.Suite
}

// Test_TestSuite runs the test suite
func Test_TestSuite(t *testing.T) {
	suite.Run(t, new(TestSuite))
}

func (ts *TestSuite) Test_keyToReadableString() {
	t := ts.T()

	tests := []struct {
		name string
		key  string
		want string
	}{
		{
			name: "all lowercase",
			key:  "lower",
			want: "lower",
		},
		{
			name: "one word",
			key:  "Single",
			want: "Single",
		},
		{
			name: "multiple words",
			key:  "ThisHasManyWords",
			want: "This has many words",
		},
		{
			name: "initial lowercase gets lost",
			key:  "initialLowerGetsLost",
			want: "Lower gets lost",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := keyToReadableString(tt.key)
			ts.Equal(tt.want, got)
		})
	}
}

func (ts *TestSuite) TestCurrency_String() {
	tests := []struct {
		name string
		c    Currency
		want string
	}{
		{
			name: "0",
			c:    0,
			want: "0.00",
		},
		{
			name: "1",
			c:    1,
			want: "0.01",
		},
		{
			name: "10",
			c:    10,
			want: "0.10",
		},
		{
			name: "100",
			c:    100,
			want: "1.00",
		},
		{
			name: "-1",
			c:    -1,
			want: "-0.01",
		},
		{
			name: "-10",
			c:    -10,
			want: "-0.10",
		},
		{
			name: "-100",
			c:    -100,
			want: "-1.00",
		},
	}
	for _, tt := range tests {
		ts.T().Run(tt.name, func(t *testing.T) {
			s := tt.c.String()
			ts.Equal(tt.want, s)
		})
	}
}
