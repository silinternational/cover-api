package api

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// TestSuite establishes a test suite for domain tests
type TestSuite struct {
	*require.Assertions
	suite.Suite
}

// Test_TestSuite runs the test suite
func Test_TestSuite(t *testing.T) {
	suite.Run(t, new(TestSuite))
}

// SetupTest sets the test suite to abort on first failure and sets the session store
func (ts *TestSuite) SetupTest() {
	ts.Assertions = require.New(ts.T())
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
		{
			name: "includes A",
			key:  "ErrorMakeAChoice",
			want: "Make a choice",
		},
		{
			name: "starts with A",
			key:  "ErrorABadChoice",
			want: "A bad choice",
		},
		{
			name: "ends with A",
			key:  "ErrorBadChoiceA",
			want: "Bad choice a",
		},
		{
			name: "includes id",
			key:  "ErrorUserIDField",
			want: "User id field",
		},
		{
			name: "starts with id",
			key:  "ErrorIDNotFound",
			want: "Id not found",
		},
		{
			name: "ends with id",
			key:  "ErrorUserID",
			want: "User id",
		},
		{
			name: "includes url",
			key:  "ErrorUserURLField",
			want: "User url field",
		},
		{
			name: "starts with url",
			key:  "ErrorURLInvalid",
			want: "Url invalid",
		},
		{
			name: "ends with url",
			key:  "ErrorInvalidURL",
			want: "Invalid url",
		},
		{
			name: "trim Error from the front",
			key:  "ErrorKey",
			want: "Key",
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
