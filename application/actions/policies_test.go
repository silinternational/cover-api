package actions

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/silinternational/riskman-api/api"

	"github.com/silinternational/riskman-api/models"
	"github.com/stretchr/testify/assert"
)

func (as *ActionSuite) Test_PoliciesList() {
	fixConfig := models.FixturesConfig{
		NumberOfPolicies:    3,
		ItemsPerPolicy:      2,
		UsersPerPolicy:      1,
		DependentsPerPolicy: 0,
	}

	fixtures := models.CreatePolicyFixtures(as.DB, fixConfig)

	for _, p := range fixtures.Policies {
		assert.Nil(as.T(), p.LoadMembers(as.DB, false))
		assert.Nil(as.T(), p.LoadDependents(as.DB, false))
	}

	// alias a couple users
	appAdmin := fixtures.Policies[0].Members[0]
	normalUser := fixtures.Policies[1].Members[0]

	// change user 0 to an admin
	appAdmin.AppRole = models.AppRoleAdmin
	err := appAdmin.Update(as.DB)
	assert.Nil(as.T(), err, "failed to make first policy user an app admin")

	tests := []struct {
		name          string
		actor         models.User
		wantCount     int
		wantStatus    int
		wantInBody    string
		notWantInBody string
	}{
		{
			name:          "unauthenticated",
			actor:         models.User{},
			wantCount:     0,
			wantStatus:    http.StatusUnauthorized,
			wantInBody:    "",
			notWantInBody: fixtures.Policies[0].ID.String(),
		},
		{
			name:          "admin",
			actor:         appAdmin,
			wantCount:     len(fixtures.Policies),
			wantStatus:    http.StatusOK,
			wantInBody:    fixtures.Policies[1].ID.String(),
			notWantInBody: "",
		},
		{
			name:          "user",
			actor:         normalUser,
			wantCount:     1,
			wantStatus:    http.StatusOK,
			wantInBody:    fixtures.Policies[1].ID.String(),
			notWantInBody: fixtures.Policies[0].ID.String(),
		},
	}

	for _, tt := range tests {
		as.T().Run(tt.name, func(t *testing.T) {
			req := as.JSON("/policies")
			req.Headers["Authorization"] = fmt.Sprintf("Bearer %s", tt.actor.Email)
			req.Headers["content-type"] = "application/json"
			res := req.Get()

			body := res.Body.String()
			assert.Equal(as.T(), tt.wantStatus, res.Code, "incorrect status code returned, body: %s", body)
			if tt.wantInBody != "" {
				assert.Contains(as.T(), body, tt.wantInBody)
			}
			if tt.notWantInBody != "" {
				assert.NotContains(as.T(), body, tt.notWantInBody)
			}

			if res.Code != http.StatusOK {
				return
			}
			var policies api.Policies
			err := json.Unmarshal([]byte(body), &policies)
			assert.Nil(t, err)
			assert.Equal(t, tt.wantCount, len(policies))
		})
	}
}
