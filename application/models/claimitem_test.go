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
				} else if len(vErr.Get(tt.errField)) == 0 {
					t.Errorf("Expected an error on field %v, but got none (errors: %+v)", tt.errField, vErr.Errors)
				}
			} else if vErr.HasAny() {
				t.Errorf("Unexpected error: %+v", vErr)
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

	user := CreateAdminUser(db)

	claim := fixtures.Claims[0]
	claim.LoadClaimItems(db, false)
	claimItem := claim.ClaimItems[0]

	tests := []struct {
		name      string
		claimItem *ClaimItem
		newStatus api.ClaimItemStatus
		wantErr   bool
		appError  api.AppError
	}{
		{
			name:      "invalid transition",
			claimItem: &claimItem,
			newStatus: api.ClaimItemStatusDraft,
			appError:  api.AppError{Key: api.ErrorValidation, Category: api.CategoryUser},
			wantErr:   true,
		},
		{
			name:      "ok",
			claimItem: &claimItem,
			newStatus: api.ClaimItemStatusDenied,
			wantErr:   false,
		},
	}
	for _, tt := range tests {
		ms.T().Run(tt.name, func(t *testing.T) {
			claimItemCopy := *tt.claimItem
			err := claimItemCopy.Update(db, tt.newStatus, user)
			var fromDB ClaimItem
			ms.NoError(fromDB.FindByID(db, tt.claimItem.ID))

			if tt.wantErr {
				ms.Error(err)
				ms.EqualAppError(tt.appError, err)
				ms.Equal(tt.claimItem.Status, claimItemCopy.Status, "status should not have changed")
				return
			}
			ms.NoError(err)

			ms.Equal(tt.newStatus, fromDB.Status, "incorrect status")
		})
	}
}
