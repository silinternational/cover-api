package actions

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/silinternational/cover-api/api"
	"github.com/silinternational/cover-api/models"
)

func (as *ActionSuite) Test_ClaimItemsUpdate() {
	fixConfig := models.FixturesConfig{
		NumberOfPolicies:    2,
		UsersPerPolicy:      1,
		ClaimsPerPolicy:     4,
		ClaimItemsPerClaim:  3,
		DependentsPerPolicy: 0,
		ItemsPerPolicy:      5,
	}

	fixtures := models.CreateItemFixtures(as.DB, fixConfig)

	// alias a couple users
	authorizedUser := fixtures.Policies[0].Members[0]
	unauthorizedUser := fixtures.Policies[1].Members[0]

	// make an admin
	appAdmin := models.CreateAdminUser(as.DB)

	claim := fixtures.Claims[0]
	claim.LoadClaimItems(as.DB, false)
	claimItem := claim.ClaimItems[0]

	input := api.ClaimItemUpdateInput{
		RepairActual:  100,
		ReplaceActual: 200,
	}

	tests := []struct {
		name       string
		actor      models.User
		claimItem  models.ClaimItem
		input      interface{}
		wantStatus int
		wantInBody string
	}{
		{
			name:       "bad input",
			actor:      authorizedUser,
			claimItem:  claimItem,
			input:      api.Claim{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "unauthorized user",
			actor:      unauthorizedUser,
			claimItem:  claimItem,
			input:      input,
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "authorized user",
			actor:      authorizedUser,
			claimItem:  claimItem,
			input:      input,
			wantStatus: http.StatusOK,
			wantInBody: claim.ID.String(),
		},
		{
			name:       "admin user",
			actor:      appAdmin,
			claimItem:  claimItem,
			input:      input,
			wantStatus: http.StatusOK,
			wantInBody: claim.ID.String(),
		},
	}

	for _, tt := range tests {
		as.T().Run(tt.name, func(t *testing.T) {
			req := as.JSON(claimItemsPath + "/" + tt.claimItem.ID.String())
			req.Headers["Authorization"] = fmt.Sprintf("Bearer %s", tt.actor.Email)
			req.Headers["content-type"] = "application/json"
			res := req.Put(tt.input)

			body := res.Body.String()
			as.Equal(tt.wantStatus, res.Code, "incorrect status code returned, body: %s", body)
			if tt.wantInBody != "" {
				as.Contains(body, tt.wantInBody, "did not find expected string")
			}

			if res.Code != http.StatusOK {
				return
			}

			var responseObject api.ClaimItem
			as.NoError(json.Unmarshal([]byte(body), &responseObject))
			as.Equal(tt.claimItem.ItemID, responseObject.ItemID, "incorrect object in response: %v", responseObject)

			updatedClaimItem := models.ClaimItem{}
			as.NoError(as.DB.Find(&updatedClaimItem, tt.claimItem.ID))
			as.Equal(updatedClaimItem.ReplaceActual, tt.input.(api.ClaimItemUpdateInput).ReplaceActual,
				"ReplaceActual is not correct")
			as.Equal(updatedClaimItem.RepairActual, tt.input.(api.ClaimItemUpdateInput).RepairActual,
				"RepairActual is not correct")
		})
	}
}
