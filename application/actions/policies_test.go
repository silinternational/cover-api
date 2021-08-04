package actions

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/silinternational/riskman-api/api"
	"github.com/silinternational/riskman-api/models"
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
		as.NoError(p.LoadMembers(as.DB, false))
		as.NoError(p.LoadDependents(as.DB, false))
	}

	// alias a couple users
	appAdmin := fixtures.Policies[0].Members[0]
	normalUser := fixtures.Policies[1].Members[0]

	// change user 0 to an admin
	appAdmin.AppRole = models.AppRoleAdmin
	err := appAdmin.Update(as.DB)
	as.NoError(err, "failed to make first policy user an app admin")

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
			as.Equal(tt.wantStatus, res.Code, "incorrect status code returned, body: %s", body)
			if tt.wantInBody != "" {
				as.Contains(body, tt.wantInBody)
			}
			if tt.notWantInBody != "" {
				as.NotContains(body, tt.notWantInBody)
			}

			if res.Code != http.StatusOK {
				return
			}
			var policies api.Policies
			err := json.Unmarshal([]byte(body), &policies)
			as.NoError(err)
			as.Equal(tt.wantCount, len(policies))
		})
	}
}
