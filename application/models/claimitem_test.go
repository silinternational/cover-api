package models

import (
	"testing"
	"time"

	"github.com/gobuffalo/nulls"

	"github.com/silinternational/riskman-api/api"
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
				Status: api.ClaimItemStatusPending,
			},
			errField: "",
			wantErr:  false,
		},
		{
			name: "approved, but no reviewer",
			claimItem: &ClaimItem{
				Status:     api.ClaimItemStatusApproved,
				ReviewDate: nulls.NewTime(time.Now()),
			},
			errField: "ClaimItem.ReviewerID",
			wantErr:  true,
		},
		{
			name: "denied, but no review date",
			claimItem: &ClaimItem{
				Status:     api.ClaimItemStatusDenied,
				ReviewerID: nulls.NewUUID(user.ID),
			},
			errField: "ClaimItem.ReviewDate",
			wantErr:  true,
		},
		{
			name: "valid status, approved",
			claimItem: &ClaimItem{
				Status:     api.ClaimItemStatusApproved,
				ReviewerID: nulls.NewUUID(user.ID),
				ReviewDate: nulls.NewTime(time.Now()),
			},
			errField: "",
			wantErr:  false,
		},
	}
	for _, tt := range tests {
		ms.T().Run(tt.name, func(t *testing.T) {
			vErr, _ := tt.claimItem.Validate(DB)
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
