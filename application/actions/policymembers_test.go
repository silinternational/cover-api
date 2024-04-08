package actions

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/silinternational/cover-api/models"
)

func (as *ActionSuite) TestPolicyMember_Delete() {
	f := models.CreateItemFixtures(as.DB, models.FixturesConfig{NumberOfPolicies: 4, ItemsPerPolicy: 2, UsersPerPolicy: 2})
	owner := f.Users[0]
	otherUser := f.Users[4]

	policy := f.Policies[0]
	polUserIDs := policy.GetPolicyUserIDs(as.DB, false)

	var polUser models.PolicyUser
	as.NoError(polUser.FindByID(as.DB, polUserIDs[0]), "error fetching PolicyUser fixture")

	tests := []struct {
		name       string
		actor      models.User
		polUserID  string
		wantStatus int
		wantInBody string
	}{
		{
			name:       "unauthorized",
			actor:      otherUser,
			polUserID:  polUser.ID.String(),
			wantStatus: http.StatusNotFound,
			wantInBody: "actor not allowed to perform that action on this resource",
		},
		{
			name:       "good",
			actor:      owner,
			polUserID:  polUser.ID.String(),
			wantStatus: http.StatusNoContent,
		},
	}
	for _, tt := range tests {
		as.T().Run(tt.name, func(t *testing.T) {
			path := fmt.Sprintf("%s/%s", policyMemberPath, tt.polUserID)
			body, status := as.request("DELETE", path, tt.actor.Email, nil)
			as.Equal(tt.wantStatus, status, "incorrect status code returned, body: %s", body)

			if status == http.StatusNoContent {
				return
			}
			as.Contains(string(body), tt.wantInBody, "string is missing from body")
		})
	}
}
