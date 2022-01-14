package models

import (
	"errors"
	"testing"
	"time"

	"github.com/gobuffalo/buffalo"
	"github.com/gobuffalo/nulls"
	"github.com/gobuffalo/pop/v5"
	"github.com/gofrs/uuid"
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

func (ms *ModelSuite) EqualNullTime(expected nulls.Time, actual *time.Time, msgAndArgs ...interface{}) {
	if actual == nil {
		ms.False(expected.Valid, msgAndArgs...)
	} else {
		ms.Equal(expected, *actual, msgAndArgs...)
	}
}

func (ms *ModelSuite) EqualNullUUID(expected nulls.UUID, actual *uuid.UUID, msgAndArgs ...interface{}) {
	if actual == nil {
		ms.False(expected.Valid, msgAndArgs...)
	} else {
		ms.Equal(expected, *actual, msgAndArgs...)
	}
}

func (ms *ModelSuite) TestGetHHID() {
	if domain.Env.HouseholdIDLookupURL == "" {
		ms.T().Skip("skipping test because no HOUSEHOLD_ID_LOOKUP_URL was provided")
	}

	tests := []struct {
		name    string
		staffID string
		want    string
	}{
		{
			name:    "good",
			staffID: "32329",
			want:    "232329",
		},
		{
			name:    "not found",
			staffID: "9999999",
			want:    "",
		},
	}
	for _, tt := range tests {
		ms.T().Run(tt.name, func(t *testing.T) {
			got := GetHHID(tt.staffID)
			ms.Equal(tt.want, got)
		})
	}
}
