package actions

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/silinternational/riskman-api/api"
	"github.com/silinternational/riskman-api/models"
)

func (as *ActionSuite) Test_ClaimsCreate() {
	fixConfig := models.FixturesConfig{
		NumberOfPolicies:    3,
		UsersPerPolicy:      1,
		DependentsPerPolicy: 0,
		ItemsPerPolicy:      2,
	}

	fixtures := models.CreateItemFixtures(as.DB, fixConfig)

	// alias a couple users
	appAdmin := fixtures.Policies[0].Members[0]
	normalUser := fixtures.Policies[1].Members[0]

	// make an admin
	appAdmin.AppRole = models.AppRoleAdmin
	err := appAdmin.Update(as.DB)
	as.NoError(err, "failed to make an app admin")

	input := api.ClaimCreateInput{
		EventDate:        time.Now(),
		EventType:        api.ClaimEventTypeTheft,
		EventDescription: "a description",
	}

	tests := []struct {
		name          string
		actor         models.User
		policy        models.Policy
		input         api.ClaimCreateInput
		wantStatus    int
		wantInBody    string
		notWantInBody string
	}{
		{
			name:          "incomplete input",
			actor:         normalUser,
			policy:        fixtures.Policies[1],
			input:         api.ClaimCreateInput{},
			wantStatus:    http.StatusBadRequest,
			notWantInBody: fixtures.Policies[1].ID.String(),
		},
		{
			name:       "valid input",
			actor:      normalUser,
			policy:     fixtures.Policies[1],
			input:      input,
			wantStatus: http.StatusOK,
			wantInBody: fixtures.Policies[1].ID.String(),
		},
		{
			name:          "other person's policy",
			actor:         normalUser,
			policy:        fixtures.Policies[0],
			input:         input,
			wantStatus:    http.StatusNotFound,
			notWantInBody: fixtures.Policies[0].ID.String(),
		},
		{
			name:       "admin operation on other person's policy",
			actor:      appAdmin,
			policy:     fixtures.Policies[2],
			input:      input,
			wantStatus: http.StatusOK,
			wantInBody: fixtures.Policies[2].ID.String(),
		},
	}

	for _, tt := range tests {
		as.T().Run(tt.name, func(t *testing.T) {
			req := as.JSON(fmt.Sprintf("/policies/%s/claims", tt.policy.ID))
			req.Headers["Authorization"] = fmt.Sprintf("Bearer %s", tt.actor.Email)
			req.Headers["content-type"] = "application/json"
			res := req.Post(tt.input)

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
			var responseObject api.Policy
			as.NoError(json.Unmarshal([]byte(body), &responseObject))
			as.Len(responseObject.Claims, 1, "incorrect # of claims on policy, %+v", responseObject)
			as.Equal(tt.input.EventDescription, responseObject.Claims[0].EventDescription,
				"response object is not correct, %+v", responseObject)
		})
	}
}
