package actions

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/gobuffalo/nulls"

	"github.com/silinternational/cover-api/api"
	"github.com/silinternational/cover-api/models"
)

func (as *ActionSuite) Test_ClaimItemsUpdate() {
	db := as.DB
	fixConfig := models.FixturesConfig{
		NumberOfPolicies:    2,
		UsersPerPolicy:      1,
		ClaimsPerPolicy:     4,
		ClaimItemsPerClaim:  3,
		DependentsPerPolicy: 0,
		ItemsPerPolicy:      5,
	}

	fixtures := models.CreateItemFixtures(as.DB, fixConfig)
	policy := fixtures.Policies[0]

	// alias a couple users
	authorizedUser := policy.Members[0]
	unauthorizedUser := fixtures.Policies[1].Members[0]

	// make an admin
	appAdmin := models.CreateAdminUsers(as.DB)[models.AppRoleAdmin]

	draftClaim := policy.Claims[0]
	draftClaim.LoadClaimItems(as.DB, false)
	draftClaimItem := draftClaim.ClaimItems[0]

	review1Claim := models.UpdateClaimStatus(db, policy.Claims[1], api.ClaimStatusReview1, "")
	review1Claim.LoadClaimItems(as.DB, false)
	review1ClaimItem := review1Claim.ClaimItems[0]

	approvedClaim := models.UpdateClaimStatus(db, policy.Claims[2], api.ClaimStatusApproved, "")
	approvedClaim.LoadClaimItems(as.DB, false)
	approvedClaimItem := approvedClaim.ClaimItems[0]
	approvedClaimItem.ReviewerID = nulls.NewUUID(appAdmin.ID)
	approvedClaimItem.ReviewDate = nulls.NewTime(time.Now().UTC())

	ctx := models.CreateTestContext(appAdmin)
	approvedClaimItem.Update(ctx)

	input := api.ClaimItemUpdateInput{
		IsRepairable:    true,
		RepairEstimate:  110,
		RepairActual:    100,
		ReplaceEstimate: 220,
		ReplaceActual:   200,
		PayoutOption:    api.PayoutOptionRepair,
		FMV:             199,
	}

	tests := []struct {
		name        string
		actor       models.User
		claimItem   models.ClaimItem
		input       interface{}
		addReviewer bool
		wantStatus  int
		wantInBody  string
	}{
		{
			name:       "bad input",
			actor:      authorizedUser,
			claimItem:  draftClaimItem,
			input:      api.Claim{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "unauthorized user",
			actor:      unauthorizedUser,
			claimItem:  draftClaimItem,
			input:      input,
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "authorized user but bad claim status",
			actor:      authorizedUser,
			claimItem:  approvedClaimItem,
			input:      input,
			wantStatus: http.StatusBadRequest,
			wantInBody: string(api.ErrorClaimStatus),
		},
		{
			name:       "authorized user draft",
			actor:      authorizedUser,
			claimItem:  draftClaimItem,
			input:      input,
			wantStatus: http.StatusOK,
			wantInBody: draftClaim.ID.String(),
		},
		{
			name:       "authorized user review1 to draft",
			actor:      authorizedUser,
			claimItem:  review1ClaimItem,
			input:      input,
			wantStatus: http.StatusOK,
			wantInBody: review1Claim.ID.String(),
		},
		{
			name:       "admin user",
			actor:      appAdmin,
			claimItem:  approvedClaimItem,
			input:      input,
			wantStatus: http.StatusOK,
			wantInBody: approvedClaim.ID.String(),
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
