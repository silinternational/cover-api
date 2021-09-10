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
				Status:       api.ClaimItemStatusPending,
				PayoutOption: api.PayoutOptionRepair,
			},
			errField: "",
			wantErr:  false,
		},
		{
			name: "approved, but no reviewer",
			claimItem: &ClaimItem{
				Status:       api.ClaimItemStatusApproved,
				PayoutOption: api.PayoutOptionRepair,
				ReviewDate:   nulls.NewTime(time.Now()),
			},
			errField: "ClaimItem.ReviewerID",
			wantErr:  true,
		},
		{
			name: "denied, but no review date",
			claimItem: &ClaimItem{
				Status:       api.ClaimItemStatusDenied,
				PayoutOption: api.PayoutOptionRepair,
				ReviewerID:   nulls.NewUUID(user.ID),
			},
			errField: "ClaimItem.ReviewDate",
			wantErr:  true,
		},
		{
			name: "invalid payout option",
			claimItem: &ClaimItem{
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
					EventType: api.ClaimEventTypeEvacuation,
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
					EventType: api.ClaimEventTypeEvacuation,
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
					EventType: api.ClaimEventTypeTheft,
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
					EventType: api.ClaimEventTypeTheft,
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
					EventType: api.ClaimEventTypeImpact,
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
					EventType: api.ClaimEventTypeImpact,
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
	for _, tt := range tests {
		ms.T().Run(tt.name, func(t *testing.T) {
			vErr, _ := tt.claimItem.Validate(ms.DB)
			if tt.wantErr {
				if vErr.Count() == 0 {
					t.Errorf("Expected an error, but did not get one")
					return
				} else if len(vErr.Get(tt.errField)) == 0 {
					t.Errorf("Expected an error on field %v, but got none (errors: %+v)", tt.errField, vErr.Errors)
					return
				}
			} else if vErr.HasAny() {
				t.Errorf("Unexpected error: %+v", vErr)
				return
			}
		})
	}
}

func (ms *ModelSuite) TestClaimItem_Update() {
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

	user := CreateAdminUsers(db)[AppRoleAdmin]

	claim := fixtures.Claims[0]
	claim.LoadClaimItems(db, false)

	tests := []struct {
		name      string
		newStatus api.ClaimItemStatus
		wantErr   bool
		appError  api.AppError
	}{
		{
			name:      "invalid transition",
			newStatus: api.ClaimItemStatusDraft,
			appError:  api.AppError{Key: api.ErrorValidation, Category: api.CategoryUser},
			wantErr:   true,
		},
		{
			name:      "ok",
			newStatus: api.ClaimItemStatusDenied,
			wantErr:   false,
		},
	}
	for _, tt := range tests {
		ms.T().Run(tt.name, func(t *testing.T) {
			claimItem := claim.ClaimItems[0]
			oldStatus := claimItem.Status
			claimItem.Status = tt.newStatus

			err := claimItem.Update(db, oldStatus, user)

			var fromDB ClaimItem
			ms.NoError(fromDB.FindByID(db, claimItem.ID))

			if tt.wantErr {
				ms.Error(err)
				ms.EqualAppError(tt.appError, err)
				return
			}
			ms.NoError(err)

			ms.Equal(tt.newStatus, fromDB.Status, "incorrect status")
		})
	}
}
