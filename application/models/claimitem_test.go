package models

import (
	"testing"
	"time"

	"github.com/gobuffalo/nulls"

	"github.com/silinternational/cover-api/api"
)

func (ms *ModelSuite) TestClaimItem_Validate() {
	user := CreateUserFixtures(ms.DB, 1).Users[0]

	tests := []struct {
		name      string
		claimItem *ClaimItem
		errField  string
		wantErr   bool
	}{
		{
			name:      "empty struct",
			claimItem: &ClaimItem{},
			errField:  "ClaimItem.Status",
			wantErr:   true,
		},
		{
			name: "valid status, not approved",
			claimItem: &ClaimItem{
				Claim: Claim{
					IncidentType: api.ClaimIncidentTypeImpact,
				},
				Status:       api.ClaimItemStatusReview1,
				PayoutOption: api.PayoutOptionRepair,
			},
			errField: "",
			wantErr:  false,
		},
		{
			name: "valid status, missing claim incident type",
			claimItem: &ClaimItem{
				Status:       api.ClaimItemStatusReview1,
				PayoutOption: api.PayoutOptionRepair,
			},
			errField: "ClaimItem.IncidentType",
			wantErr:  true,
		},
		{
			name: "invalid payout option",
			claimItem: &ClaimItem{
				Claim: Claim{
					IncidentType: api.ClaimIncidentTypeImpact,
				},
				Status:       api.ClaimItemStatusDenied,
				PayoutOption: api.PayoutOption("bitcoin"),
				ReviewerID:   nulls.NewUUID(user.ID),
				ReviewDate:   nulls.NewTime(time.Now()),
			},
			errField: "ClaimItem.PayoutOption",
			wantErr:  true,
		},
		{
			name: "invalid payout option for Evacuation",
			claimItem: &ClaimItem{
				Claim: Claim{
					IncidentType: api.ClaimIncidentTypeEvacuation,
				},
				Status:       api.ClaimItemStatusDraft,
				PayoutOption: api.PayoutOptionFMV,
				ReviewerID:   nulls.NewUUID(user.ID),
				ReviewDate:   nulls.NewTime(time.Now()),
			},
			errField: "ClaimItem.PayoutOption",
			wantErr:  true,
		},
		{
			name: "valid payout option for Evacuation",
			claimItem: &ClaimItem{
				Claim: Claim{
					IncidentType: api.ClaimIncidentTypeEvacuation,
				},
				Status:       api.ClaimItemStatusDraft,
				PayoutOption: api.PayoutOptionFixedFraction,
				ReviewerID:   nulls.NewUUID(user.ID),
				ReviewDate:   nulls.NewTime(time.Now()),
			},
			wantErr: false,
		},
		{
			name: "invalid payout option for Theft",
			claimItem: &ClaimItem{
				Claim: Claim{
					IncidentType: api.ClaimIncidentTypeTheft,
				},
				Status:       api.ClaimItemStatusDraft,
				PayoutOption: api.PayoutOptionFixedFraction,
				ReviewerID:   nulls.NewUUID(user.ID),
				ReviewDate:   nulls.NewTime(time.Now()),
			},
			errField: "ClaimItem.PayoutOption",
			wantErr:  true,
		},
		{
			name: "valid payout option for Theft",
			claimItem: &ClaimItem{
				Claim: Claim{
					IncidentType: api.ClaimIncidentTypeTheft,
				},
				Status:       api.ClaimItemStatusDraft,
				PayoutOption: api.PayoutOptionFMV,
				ReviewerID:   nulls.NewUUID(user.ID),
				ReviewDate:   nulls.NewTime(time.Now()),
			},
			wantErr: false,
		},
		{
			name: "invalid payout option for Impact",
			claimItem: &ClaimItem{
				Claim: Claim{
					IncidentType: api.ClaimIncidentTypeImpact,
				},
				Status:       api.ClaimItemStatusDraft,
				PayoutOption: api.PayoutOptionFixedFraction,
				ReviewerID:   nulls.NewUUID(user.ID),
				ReviewDate:   nulls.NewTime(time.Now()),
			},
			errField: "ClaimItem.PayoutOption",
			wantErr:  true,
		},
		{
			name: "valid payout option for Impact",
			claimItem: &ClaimItem{
				Claim: Claim{
					IncidentType: api.ClaimIncidentTypeImpact,
				},
				Status:       api.ClaimItemStatusDraft,
				PayoutOption: api.PayoutOptionRepair,
				ReviewerID:   nulls.NewUUID(user.ID),
				ReviewDate:   nulls.NewTime(time.Now()),
			},
			wantErr: false,
		},
		{
			name: "valid status, approved",
			claimItem: &ClaimItem{
				Status:       api.ClaimItemStatusApproved,
				PayoutOption: api.PayoutOptionRepair,
				ReviewerID:   nulls.NewUUID(user.ID),
				ReviewDate:   nulls.NewTime(time.Now()),
			},
			errField: "",
			wantErr:  false,
		},
	}
	mt := ms.T()
	for _, tt := range tests {
		mt.Run(tt.name, func(t *testing.T) {
			vErr, _ := tt.claimItem.Validate(ms.DB)
			if tt.wantErr {
				if vErr.Count() == 0 {
					mt.Fatal("Expected an error, but did not get one")
				} else if len(vErr.Get(tt.errField)) == 0 {
					mt.Fatalf("Expected an error on field %v, but got none (errors: %+v)", tt.errField, vErr.Errors)
				}
			} else if vErr.HasAny() {
				mt.Fatalf("Unexpected error: %+v", vErr)
			}
		})
	}
}

func (ms *ModelSuite) TestClaimItem_UpdateByUser() {
	fixConfig := FixturesConfig{
		NumberOfPolicies:    2,
		UsersPerPolicy:      1,
		ClaimsPerPolicy:     4,
		ClaimItemsPerClaim:  3,
		DependentsPerPolicy: 0,
		ItemsPerPolicy:      5,
	}

	db := ms.DB
	fixtures := CreateItemFixtures(db, fixConfig)

	user := fixtures.Policies[0].Members[0]
	adminUser := CreateAdminUsers(db)[AppRoleAdmin]

	draftClaim := fixtures.Policies[0].Claims[0]
	draftClaim.LoadClaimItems(db, false)

	review2Claim := UpdateClaimStatus(db, fixtures.Policies[0].Claims[1], api.ClaimStatusReview2, "")
	review2Claim.LoadClaimItems(db, false)

	tests := []struct {
		name        string
		claimItem   ClaimItem
		input       api.ClaimItemUpdateInput
		addReviewer bool
		user        User
		appError    *api.AppError
		wantStatus  api.ClaimItemStatus
	}{
		{
			name:      "not allowed on review2 ClaimItem",
			claimItem: review2Claim.ClaimItems[0],
			input: api.ClaimItemUpdateInput{
				ReplaceEstimate: 999,
				ReplaceActual:   999,
			},
			user:     user,
			appError: &api.AppError{Key: api.ErrorClaimStatus, Category: api.CategoryUser},
		},
		{
			name:      "ok on review2 ClaimItem for admin",
			claimItem: review2Claim.ClaimItems[0],
			input: api.ClaimItemUpdateInput{
				ReplaceEstimate: 999,
				ReplaceActual:   999,
			},
			user:        adminUser,
			addReviewer: true,
			wantStatus:  api.ClaimItemStatusReview2,
		},
		{
			name:      "ok on draft",
			claimItem: draftClaim.ClaimItems[0],
			input: api.ClaimItemUpdateInput{
				ReplaceEstimate: 1333,
				ReplaceActual:   1330,
			},
			user:       user,
			wantStatus: api.ClaimItemStatusDraft,
		},
	}
	for _, tt := range tests {
		ms.T().Run(tt.name, func(t *testing.T) {
			ctx := CreateTestContext(fixtures.Users[0])
			claimItem := tt.claimItem
			oldStatus := claimItem.Status
			claimItem.ReplaceEstimate = tt.input.ReplaceEstimate
			claimItem.ReplaceActual = tt.input.ReplaceActual

			if tt.addReviewer {
				claimItem.ReviewerID = nulls.NewUUID(tt.user.ID)
				claimItem.ReviewDate = nulls.NewTime(time.Now().UTC())
			}

			err := claimItem.UpdateByUser(ctx, oldStatus, tt.user)

			var fromDB ClaimItem
			ms.NoError(fromDB.FindByID(db, claimItem.ID))

			if tt.appError != nil {
				ms.EqualAppError(*tt.appError, err)
				return
			}
			ms.NoError(err)

			ms.Equal(tt.input.ReplaceEstimate, fromDB.ReplaceEstimate, "incorrect status")
			ms.Equal(tt.input.ReplaceActual, fromDB.ReplaceActual, "incorrect status")
			ms.Equal(tt.wantStatus, fromDB.Status, "incorrect status")
		})
	}
}
