package models

import (
	"errors"
	"testing"
	"time"

	"github.com/gobuffalo/buffalo"
	"github.com/gobuffalo/pop/v5"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/silinternational/cover-api/api"
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

// EqualAppError verifies that the actual error contains an AppError and that a subset of the fields match
func (ms *ModelSuite) EqualAppError(expected api.AppError, actual error) {
	var appErr *api.AppError
	ms.True(errors.As(actual, &appErr), "error does not contain an api.AppError")
	ms.Equal(appErr.Key, expected.Key, "error key does not match")
	ms.Equal(appErr.Category, expected.Category, "error category does not match")
}

func (ms *ModelSuite) Test_sortIDTimes() {
	itemID0 := domain.GetUUID()
	itemID1 := domain.GetUUID()
	itemID2 := domain.GetUUID()
	itemID3 := domain.GetUUID()

	time0 := time.Date(2000, 1, 1, 1, 0, 0, 0, time.UTC)
	time1 := time.Date(2001, 1, 1, 1, 0, 0, 0, time.UTC)
	time2 := time.Date(2002, 1, 1, 1, 0, 0, 0, time.UTC)
	time3 := time.Date(2003, 1, 1, 1, 0, 0, 0, time.UTC)

	idTimes := map[string]time.Time{
		itemID2.String(): time2,
		itemID3.String(): time3,
		itemID0.String(): time0,
		itemID1.String(): time1,
	}
	got := sortIDTimes(idTimes)

	want := []idTime{
		{ID: itemID3.String(), UpdatedAt: time3},
		{ID: itemID2.String(), UpdatedAt: time2},
		{ID: itemID1.String(), UpdatedAt: time1},
		{ID: itemID0.String(), UpdatedAt: time0},
	}

	ms.ElementsMatch(want, got, "incorrect results")
}
