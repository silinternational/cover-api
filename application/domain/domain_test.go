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
			input:     366, // this looks like days per year just to make the calculations easy to figure out
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

func (ts *TestSuite) Test_BeginningOfLastMonth() {
	tests := []struct {
		name string
		time time.Time
		want time.Time
	}{
		{
			name: "span year",
			time: time.Date(2020, 1, 6, 0, 0, 0, 0, time.UTC),
			want: time.Date(2019, 12, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			name: "first of month",
			time: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
			want: time.Date(2019, 12, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			name: "last of month",
			time: time.Date(2020, 1, 31, 0, 0, 0, 0, time.UTC),
			want: time.Date(2019, 12, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			name: "30-day month",
			time: time.Date(2020, 12, 31, 0, 0, 0, 0, time.UTC),
			want: time.Date(2020, 11, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			name: "29-day month",
			time: time.Date(2020, 3, 31, 0, 0, 0, 0, time.UTC),
			want: time.Date(2020, 2, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			name: "28-day month",
			time: time.Date(2021, 3, 31, 0, 0, 0, 0, time.UTC),
			want: time.Date(2021, 2, 1, 0, 0, 0, 0, time.UTC),
		},
	}
	for _, tt := range tests {
		ts.T().Run(tt.name, func(t *testing.T) {
			ts.Equal(tt.want, BeginningOfLastMonth(tt.time))
		})
	}
}

func (ts *TestSuite) Test_EndOfMonth() {
	tests := []struct {
		name string
		time time.Time
		want time.Time
	}{
		{
			name: "first of month",
			time: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
			want: time.Date(2020, 1, 31, 0, 0, 0, 0, time.UTC),
		},
		{
			name: "last of month",
			time: time.Date(2020, 1, 31, 0, 0, 0, 0, time.UTC),
			want: time.Date(2020, 1, 31, 0, 0, 0, 0, time.UTC),
		},
		{
			name: "30-day month",
			time: time.Date(2020, 4, 1, 0, 0, 0, 0, time.UTC),
			want: time.Date(2020, 4, 30, 0, 0, 0, 0, time.UTC),
		},
		{
			name: "29-day month",
			time: time.Date(2020, 2, 1, 0, 0, 0, 0, time.UTC),
			want: time.Date(2020, 2, 29, 0, 0, 0, 0, time.UTC),
		},
		{
			name: "28-day month",
			time: time.Date(2021, 2, 1, 0, 0, 0, 0, time.UTC),
			want: time.Date(2021, 2, 28, 0, 0, 0, 0, time.UTC),
		},
	}
	for _, tt := range tests {
		ts.T().Run(tt.name, func(t *testing.T) {
			ts.Equal(tt.want, EndOfMonth(tt.time))
		})
	}
}

func (ts *TestSuite) TestIsLeapYear() {
	tests := []struct {
		year int
		want bool
	}{
		{year: 1900, want: false},
		{year: 2000, want: true},
		{year: 2100, want: false},
		{year: 2400, want: true},
	}

	for _, tt := range tests {
		ts.T().Run(strconv.Itoa(tt.year), func(t *testing.T) {
			ts.Equal(tt.want, IsLeapYear(time.Date(tt.year, 1, 1, 0, 0, 0, 0, time.UTC)))
		})
	}
}
