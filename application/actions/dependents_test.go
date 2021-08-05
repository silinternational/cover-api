package actions

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/silinternational/riskman-api/api"
	"github.com/silinternational/riskman-api/models"
)

func (as *ActionSuite) Test_DependentsList() {
	config := models.FixturesConfig{
		NumberOfPolicies:    3,
		ItemsPerPolicy:      2,
		UsersPerPolicy:      1,
		DependentsPerPolicy: 1,
	}

	fixtures := models.CreatePolicyFixtures(as.DB, config)

	// alias a couple users
	normalUser := fixtures.Policies[0].Members[0]
	appAdmin := fixtures.Policies[1].Members[0]

	// change user to an admin
	appAdmin.AppRole = models.AppRoleAdmin
	as.NoError(appAdmin.Update(as.DB), "failed to make first policy user an app admin")

	tests := []struct {
		name          string
		actor         models.User
		policy        models.Policy
		wantCount     int
		wantStatus    int
		wantInBody    string
		notWantInBody string
	}{
		{
			name:          "unauthenticated",
			actor:         models.User{},
			policy:        fixtures.Policies[0],
			wantCount:     0,
			wantStatus:    http.StatusUnauthorized,
			wantInBody:    "",
			notWantInBody: fixtures.Policies[0].Dependents[0].ID.String(),
		},
		{
			name:          "admin",
			actor:         appAdmin,
			policy:        fixtures.Policies[0],
			wantCount:     len(fixtures.Policies[0].Dependents),
			wantStatus:    http.StatusOK,
			wantInBody:    fixtures.Policies[0].Dependents[0].ID.String(),
			notWantInBody: fixtures.Policies[1].Dependents[0].ID.String(),
		},
		{
			name:          "user",
			actor:         normalUser,
			policy:        fixtures.Policies[0],
			wantCount:     len(fixtures.Policies[0].Dependents),
			wantStatus:    http.StatusOK,
			wantInBody:    fixtures.Policies[0].Dependents[0].ID.String(),
			notWantInBody: fixtures.Policies[1].Dependents[0].ID.String(),
		},
	}

	for _, tt := range tests {
		as.T().Run(tt.name, func(t *testing.T) {
			req := as.JSON("/policies/" + tt.policy.ID.String() + "/dependents")
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
			var dependents api.PolicyDependents
			err := json.Unmarshal([]byte(body), &dependents)
			as.NoError(err)
			as.Equal(tt.wantCount, len(dependents))
		})
	}
}

func (as *ActionSuite) Test_DependentsCreate() {
	config := models.FixturesConfig{
		NumberOfPolicies:    3,
		ItemsPerPolicy:      2,
		UsersPerPolicy:      1,
		DependentsPerPolicy: 1,
	}

	fixtures := models.CreatePolicyFixtures(as.DB, config)

	// alias a couple users
	normalUser := fixtures.Policies[0].Members[0]
	appAdmin := fixtures.Policies[1].Members[0]

	// change user 0 to an admin
	appAdmin.AppRole = models.AppRoleAdmin
	as.NoError(appAdmin.Update(as.DB), "failed to make first policy user an app admin")

	incompleteRequest := api.PolicyDependentInput{
		Name:           "",
		ChildBirthYear: 1999,
	}

	goodRequest := api.PolicyDependentInput{
		Name:           "dependent name2",
		Relationship:   api.PolicyDependentRelationshipChild,
		Location:       "bahamas",
		ChildBirthYear: 1999,
	}

	goodRequest2 := api.PolicyDependentInput{
		Name:           "dependent name2",
		Relationship:   api.PolicyDependentRelationshipChild,
		Location:       "bahamas",
		ChildBirthYear: 2001,
	}

	tests := []struct {
		name          string
		actor         models.User
		reqBody       interface{}
		policy        models.Policy
		wantCount     int
		wantStatus    int
		wantInBody    string
		notWantInBody string
	}{
		{
			name:          "unauthenticated",
			actor:         models.User{},
			reqBody:       goodRequest,
			policy:        fixtures.Policies[0],
			wantCount:     0,
			wantStatus:    http.StatusUnauthorized,
			wantInBody:    "",
			notWantInBody: fixtures.Policies[0].Dependents[0].ID.String(),
		},
		{
			name:          "bad request",
			actor:         appAdmin,
			reqBody:       "{}",
			policy:        fixtures.Policies[0],
			wantCount:     0,
			wantStatus:    http.StatusBadRequest,
			wantInBody:    "",
			notWantInBody: fixtures.Policies[0].Dependents[0].ID.String(),
		},
		{
			name:          "incomplete request",
			actor:         appAdmin,
			reqBody:       incompleteRequest,
			policy:        fixtures.Policies[0],
			wantCount:     0,
			wantStatus:    http.StatusBadRequest,
			wantInBody:    "",
			notWantInBody: fixtures.Policies[0].Dependents[0].ID.String(),
		},
		{
			name:          "admin",
			actor:         appAdmin,
			reqBody:       goodRequest,
			policy:        fixtures.Policies[0],
			wantCount:     1 + len(fixtures.Policies[0].Dependents),
			wantStatus:    http.StatusOK,
			wantInBody:    goodRequest.Name,
			notWantInBody: fixtures.Policies[1].Dependents[0].ID.String(),
		},
		{
			name:          "user",
			actor:         normalUser,
			reqBody:       goodRequest2,
			policy:        fixtures.Policies[0],
			wantCount:     2 + len(fixtures.Policies[0].Dependents),
			wantStatus:    http.StatusOK,
			wantInBody:    goodRequest2.Name,
			notWantInBody: fixtures.Policies[1].Dependents[0].ID.String(),
		},
	}

	for _, tt := range tests {
		as.T().Run(tt.name, func(t *testing.T) {
			req := as.JSON("/policies/" + tt.policy.ID.String() + "/dependents")
			req.Headers["Authorization"] = fmt.Sprintf("Bearer %s", tt.actor.Email)
			req.Headers["content-type"] = "application/json"
			res := req.Post(tt.reqBody)

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
			var dependents api.PolicyDependents
			err := json.Unmarshal([]byte(body), &dependents)
			as.NoError(err)
			as.Equal(tt.wantCount, len(dependents))
		})
	}
}
