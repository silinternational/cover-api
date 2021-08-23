package models

import (
	"testing"

	"github.com/gobuffalo/buffalo"
	"github.com/gobuffalo/pop/v5"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/silinternational/cover-api/domain"
)

// ModelSuite doesn't contain a buffalo suite.Model and can be used for tests that don't need access to the database
// or don't need the buffalo test runner to refresh the database
type ModelSuite struct {
	suite.Suite
	*require.Assertions
	DB *pop.Connection
}

func (ms *ModelSuite) SetupTest() {
	ms.Assertions = require.New(ms.T())
	DestroyAll()
}

// Test_ModelSuite runs the test suite
func Test_ModelSuite(t *testing.T) {
	ms := &ModelSuite{}
	c, err := pop.Connect(domain.Env.GoEnv)
	if err == nil {
		ms.DB = c
	}
	suite.Run(t, ms)
}

func (ms *ModelSuite) Test_CurrentUser() {
	// setup
	user := CreateUserFixtures(ms.DB, 1).Users[0]
	ctx := CreateTestContext(user)

	tests := []struct {
		name     string
		context  buffalo.Context
		wantUser User
	}{
		{
			name:     "buffalo context",
			context:  ctx,
			wantUser: user,
		},
		{
			name:     "empty context",
			context:  &TestBuffaloContext{params: map[interface{}]interface{}{}},
			wantUser: User{},
		},
	}

	for _, tt := range tests {
		ms.T().Run(tt.name, func(t *testing.T) {
			// execute
			got := CurrentUser(tt.context)

			// verify
			ms.Equal(tt.wantUser.ID, got.ID)
		})
	}
}
