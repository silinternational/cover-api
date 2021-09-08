package domain

import (
	"fmt"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// TestSuite establishes a test suite for domain tests
type TestSuite struct {
	suite.Suite
	*require.Assertions
}

func (ts *TestSuite) SetupTest() {
	ts.Assertions = require.New(ts.T())
}

// Test_TestSuite runs the test suite
func Test_TestSuite(t *testing.T) {
	suite.Run(t, new(TestSuite))
}

func (ts *TestSuite) Test_emptyUUIDValue() {
	val := uuid.UUID{}
	ts.Equal("00000000-0000-0000-0000-000000000000", val.String(), "incorrect empty uuid value")
}

func (ts *TestSuite) Test_RandomString() {
	for i := 1; i < 30; i++ {
		ts.Len(RandomString(i, ""), i)
	}
}

func (ts *TestSuite) TestEmailFromAddress() {
	nickname := "nickname"

	tests := []struct {
		name string
		arg  *string
		want string
	}{
		{
			name: "name given",
			arg:  &nickname,
			want: fmt.Sprintf("nickname via %s <%s>", Env.AppName, Env.EmailFromAddress),
		},
		{
			name: "no name given",
			arg:  nil,
			want: fmt.Sprintf("%s <%s>", Env.AppName, Env.EmailFromAddress),
		},
	}
	for _, tt := range tests {
		ts.T().Run(tt.name, func(t *testing.T) {
			if got := EmailFromAddress(tt.arg); got != tt.want {
				t.Errorf("EmailFromAddress() = %v, want %v", got, tt.want)
			}
		})
	}
}

func (ts *TestSuite) TestCalculatePartialYearValue() {
	tests := []struct {
		name      string
		input     int
		startDate time.Time
		want      int
	}{
		{
			name:      "whole year",
			input:     10,
			startDate: time.Date(2021, 1, 1, 23, 0, 0, 0, time.UTC),
			want:      10,
		},
		{
			name:      "whole year minus one day",
			input:     3650,
			startDate: time.Date(2021, 1, 2, 10, 0, 0, 0, time.UTC),
			want:      3640,
		},
		{
			name:      "leap year minus one day",
			input:     365, // this looks like days per year just to make the calculations easy to figure out
			startDate: time.Date(2020, 1, 2, 10, 0, 0, 0, time.UTC),
			want:      365,
		},
		{
			name:      "a month and a day",
			input:     365,
			startDate: time.Date(2021, 11, 30, 20, 0, 0, 0, time.UTC),
			want:      32,
		},
		{
			name:      "a day",
			input:     365,
			startDate: time.Date(2021, 12, 31, 0, 0, 0, 0, time.UTC),
			want:      1,
		},
		{
			name:      "a complicated day",
			input:     365 * 40,
			startDate: time.Date(2021, 12, 31, 0, 0, 0, 0, time.UTC),
			want:      40,
		},
	}
	for _, tt := range tests {
		ts.T().Run(tt.name, func(t *testing.T) {
			got := CalculatePartialYearValue(tt.input, tt.startDate)
			ts.Equal(tt.want, got, "incorrect output value")
		})
	}
}
