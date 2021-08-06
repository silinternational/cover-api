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
		p.LoadMembers(as.DB, false)
		p.LoadDependents(as.DB, false)
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

func (as *ActionSuite) Test_PoliciesUpdate() {
	fixConfig := models.FixturesConfig{
		NumberOfPolicies:    3,
		ItemsPerPolicy:      2,
		UsersPerPolicy:      1,
		DependentsPerPolicy: 0,
	}

	fixtures := models.CreatePolicyFixtures(as.DB, fixConfig)

	for _, p := range fixtures.Policies {
		p.LoadMembers(as.DB, false)
		p.LoadDependents(as.DB, false)
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
		policy        models.Policy
		update        api.PolicyUpdate
		wantStatus    int
		wantInBody    string
		notWantInBody string
	}{
		{
			name:          "empty household id",
			actor:         normalUser,
			policy:        fixtures.Policies[1],
			update:        api.PolicyUpdate{},
			wantStatus:    http.StatusBadRequest,
			notWantInBody: fixtures.Policies[1].ID.String(),
		},
		{
			name:   "valid household id",
			actor:  normalUser,
			policy: fixtures.Policies[1],
			update: api.PolicyUpdate{
				HouseholdID: "654978",
			},
			wantStatus: http.StatusOK,
			wantInBody: fixtures.Policies[1].ID.String(),
		},
		{
			name:   "other person's policy",
			actor:  normalUser,
			policy: fixtures.Policies[0],
			update: api.PolicyUpdate{
				HouseholdID: "09876",
			},
			wantStatus:    http.StatusNotFound,
			notWantInBody: fixtures.Policies[0].ID.String(),
		},
		{
			name:   "admin update other person's policy",
			actor:  appAdmin,
			policy: fixtures.Policies[1],
			update: api.PolicyUpdate{
				HouseholdID: "998877",
			},
			wantStatus: http.StatusOK,
			wantInBody: fixtures.Policies[1].ID.String(),
		},
	}

	for _, tt := range tests {
		as.T().Run(tt.name, func(t *testing.T) {
			req := as.JSON("/policies/" + tt.policy.ID.String())
			req.Headers["Authorization"] = fmt.Sprintf("Bearer %s", tt.actor.Email)
			req.Headers["content-type"] = "application/json"
			res := req.Put(tt.update)

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
			var policy api.Policy
			as.NoError(json.Unmarshal([]byte(body), &policy))
			as.Equal(tt.update.HouseholdID, policy.HouseholdID)
		})
	}
}

func (as *ActionSuite) Test_PoliciesListMembers() {
	fixConfig := models.FixturesConfig{
		NumberOfPolicies:    3,
		ItemsPerPolicy:      0,
		UsersPerPolicy:      2,
		DependentsPerPolicy: 0,
	}

	fixtures := models.CreatePolicyFixtures(as.DB, fixConfig)

	for _, p := range fixtures.Policies {
		p.LoadMembers(as.DB, false)
		p.LoadDependents(as.DB, false)
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
		policyID      string
		wantCount     int
		wantStatus    int
		wantInBody    string
		notWantInBody string
	}{
		{
			name:          "unauthenticated",
			actor:         models.User{},
			policyID:      fixtures.Policies[0].ID.String(),
			wantCount:     0,
			wantStatus:    http.StatusUnauthorized,
			wantInBody:    "",
			notWantInBody: fixtures.Policies[0].Members[0].ID.String(),
		},
		{
			name:          "admin",
			actor:         appAdmin,
			policyID:      fixtures.Policies[0].ID.String(),
			wantCount:     fixConfig.UsersPerPolicy,
			wantStatus:    http.StatusOK,
			wantInBody:    appAdmin.ID.String(),
			notWantInBody: normalUser.ID.String(),
		},
		{
			name:          "admin - other user's policy",
			actor:         appAdmin,
			policyID:      fixtures.Policies[1].ID.String(),
			wantCount:     fixConfig.UsersPerPolicy,
			wantStatus:    http.StatusOK,
			wantInBody:    normalUser.ID.String(),
			notWantInBody: appAdmin.ID.String(),
		},
		{
			name:          "user",
			actor:         normalUser,
			policyID:      fixtures.Policies[1].ID.String(),
			wantCount:     fixConfig.UsersPerPolicy,
			wantStatus:    http.StatusOK,
			wantInBody:    normalUser.ID.String(),
			notWantInBody: appAdmin.ID.String(),
		},
		{
			name:          "normal user, other user's policy",
			actor:         normalUser,
			policyID:      fixtures.Policies[0].ID.String(),
			wantCount:     fixConfig.UsersPerPolicy,
			wantStatus:    http.StatusNotFound,
			wantInBody:    "",
			notWantInBody: appAdmin.ID.String(),
		},
		{
			name:          "invalid ID",
			actor:         normalUser,
			policyID:      "abc123",
			wantCount:     fixConfig.UsersPerPolicy,
			wantStatus:    http.StatusNotFound,
			wantInBody:    "",
			notWantInBody: normalUser.ID.String(),
		},
	}

	for _, tt := range tests {
		as.T().Run(tt.name, func(t *testing.T) {
			req := as.JSON("/policies/" + tt.policyID + "/members")
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
			var members api.PolicyMembers
			err := json.Unmarshal([]byte(body), &members)
			as.NoError(err)
			as.Equal(tt.wantCount, len(members))
		})
	}
}
