package actions

import (
	"net/http"
	"testing"

	"github.com/silinternational/cover-api/api"
	"github.com/silinternational/cover-api/domain"
	"github.com/silinternational/cover-api/models"
)

func (as *ActionSuite) Test_StewardListRecent() {
	fixtures := models.CreatePolicyHistoryFixtures_RecentItemStatusChanges(as.DB)
	phFixes := fixtures.PolicyHistories

	fixtures = models.CreateClaimHistoryFixtures_RecentClaimStatusChanges(as.DB)
	chFixes := fixtures.ClaimHistories

	const tmFmt = "Jan _2 15:04:05.00"

	// alias a couple users
	steward := models.CreateAdminUsers(as.DB)[models.AppRoleSteward]
	normalUser := fixtures.Policies[0].Members[0]

	tests := []struct {
		name          string
		actor         models.User
		wantCount     int
		wantStatus    int
		wantInBody    []string
		notWantInBody string
	}{
		{
			name:          "unauthenticated",
			actor:         models.User{},
			wantCount:     0,
			wantStatus:    http.StatusUnauthorized,
			notWantInBody: "Items",
		},
		{
			name:          "user",
			actor:         normalUser,
			wantCount:     0,
			wantStatus:    http.StatusNotFound,
			notWantInBody: "Items",
		},
		{
			name:       "steward",
			actor:      steward,
			wantCount:  len(fixtures.Policies),
			wantStatus: http.StatusOK,
			wantInBody: []string{
				`"Items":[`,
				`"Item":{"id":"` + phFixes[7].ItemID.UUID.String(),
				`"Item":{"id":"` + phFixes[3].ItemID.UUID.String(),
				`"Claims":[`,
				`"Claim":{"id":"` + chFixes[7].ClaimID.String(),
				`"Claim":{"id":"` + chFixes[3].ClaimID.String(),
				`"StatusUpdatedAt":"` + chFixes[3].UpdatedAt.Format(domain.DateFormat),
			},
		},
	}

	for _, tt := range tests {
		as.T().Run(tt.name, func(t *testing.T) {
			path := stewardPath + "/" + api.ResourceRecent
			body, status := as.request("GET", path, tt.actor.Email, nil)
			as.Equal(tt.wantStatus, status, "incorrect status code returned, body: %s", body)

			if tt.notWantInBody != "" {
				as.NotContains(body, tt.notWantInBody)
			}

			if status != http.StatusOK {
				return
			}

			as.verifyResponseData(tt.wantInBody, body, "Recent Object fields")
		})
	}
}
