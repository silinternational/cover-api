package actions

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/silinternational/cover-api/api"
	"github.com/silinternational/cover-api/domain"
	"github.com/silinternational/cover-api/models"
)

func (as *ActionSuite) Test_DependentsList() {
	config := models.FixturesConfig{
		NumberOfPolicies:    3,
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
		Name:           "dependent name",
		Relationship:   api.PolicyDependentRelationshipChild,
		Country:        "Bahamas",
		ChildBirthYear: 1999,
	}

	goodRequest2 := api.PolicyDependentInput{
		Name:           "dependent name2",
		Relationship:   api.PolicyDependentRelationshipChild,
		Country:        "USA",
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

func (as *ActionSuite) Test_DependentsUpdate() {
	db := as.DB
	config := models.FixturesConfig{
		NumberOfPolicies:    2,
		UsersPerPolicy:      1,
		DependentsPerPolicy: 2,
	}

	fixtures := models.CreatePolicyFixtures(db, config)

	// alias a couple users
	goodActor := fixtures.Policies[0].Members[0]
	wrongActor := fixtures.Policies[1].Members[0]

	dependent := fixtures.Policies[0].Dependents[1]

	goodDep := api.PolicyDependentInput{
		Name:           "New-" + dependent.Name,
		Relationship:   dependent.Relationship,
		Country:        "New-" + dependent.Country,
		ChildBirthYear: dependent.ChildBirthYear - 10,
	}

	badDep := goodDep
	badDep.Name = ""

	tests := []struct {
		name       string
		actor      models.User
		oldDep     models.PolicyDependent
		input      api.PolicyDependentInput
		wantStatus int
		wantInBody []string
	}{
		{
			name:       "unauthenticated",
			actor:      models.User{},
			oldDep:     dependent,
			input:      goodDep,
			wantStatus: http.StatusUnauthorized,
			wantInBody: []string{
				api.ErrorNotAuthorized.String(),
				"no bearer token provided",
			},
		},
		{
			name:       "unauthorized",
			actor:      wrongActor,
			oldDep:     dependent,
			input:      goodDep,
			wantStatus: http.StatusNotFound,
			wantInBody: []string{"actor not allowed to perform that action on this resource"},
		},
		{
			name:       "bad input",
			actor:      goodActor,
			oldDep:     dependent,
			input:      badDep,
			wantStatus: http.StatusBadRequest,
			wantInBody: []string{"PolicyDependent.Name"},
		},
		{
			name:       "good input",
			actor:      goodActor,
			oldDep:     dependent,
			input:      goodDep,
			wantStatus: http.StatusOK,
			wantInBody: []string{
				`"id":"` + dependent.ID.String(),
				`"name":"` + goodDep.Name,
				`"relationship":"` + string(goodDep.Relationship),
				`"country":"` + goodDep.Country,
				`"child_birth_year":` + fmt.Sprintf("%d", goodDep.ChildBirthYear),
			},
		},
	}

	for _, tt := range tests {
		as.T().Run(tt.name, func(t *testing.T) {
			req := as.JSON("/%s/%s", domain.TypePolicyDependent, tt.oldDep.ID)
			req.Headers["Authorization"] = fmt.Sprintf("Bearer %s", tt.actor.Email)
			req.Headers["content-type"] = "application/json"
			res := req.Put(tt.input)

			body := res.Body.String()
			as.Equal(tt.wantStatus, res.Code, "incorrect status code returned, body: %s", body)

			as.verifyResponseData(tt.wantInBody, body, "")

			if res.Code != http.StatusOK {
				return
			}

			var apiDep api.PolicyDependent
			err := json.Unmarshal([]byte(body), &apiDep)
			as.NoError(err)

			var dependent models.PolicyDependent
			as.NoError(as.DB.Find(&dependent, tt.oldDep.ID), "error finding newly updated dependent.")
			as.Equal(tt.input.Name, dependent.Name, "incorrect Name")
			as.Equal(tt.input.Relationship, dependent.Relationship, "incorrect Relationship")
			as.Equal(tt.input.Country, dependent.Country, "incorrect Country")
			as.Equal(tt.input.ChildBirthYear, dependent.ChildBirthYear, "incorrect ChildBirthYear")
		})
	}
}
