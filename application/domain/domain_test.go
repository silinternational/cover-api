package domain

import (
	"fmt"
	"testing"

	"github.com/gofrs/uuid"
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
