package models

import (
	"testing"

	"github.com/silinternational/riskman-api/api"
)

func (ms *ModelSuite) TestClaimItem_Validate() {
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
			name: "valid status",
			claimItem: &ClaimItem{
				Status: api.ClaimItemStatusApproved,
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
