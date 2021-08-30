package models

import (
	"errors"
	"testing"
	"time"

	"github.com/silinternational/cover-api/api"
	"github.com/silinternational/cover-api/domain"
)

func (ms *ModelSuite) TestClaim_Validate() {
	tests := []struct {
		name     string
		claim    *Claim
		errField string
		wantErr  bool
	}{
		{
			name:     "empty struct",
			claim:    &Claim{},
			errField: "Claim.Status",
			wantErr:  true,
		},
		{
			name: "valid status",
			claim: &Claim{
				ReferenceNumber:  domain.RandomString(ClaimReferenceNumberLength, ""),
				PolicyID:         domain.GetUUID(),
				EventType:        api.ClaimEventTypeImpact,
				EventDate:        time.Now(),
				EventDescription: "testing123",
				Status:           api.ClaimStatusReview1,
			},
			errField: "",
			wantErr:  false,
		},
	}
	for _, tt := range tests {
		ms.T().Run(tt.name, func(t *testing.T) {
			vErr, _ := tt.claim.Validate(DB)
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

func (ms *ModelSuite) TestClaim_ReferenceNumber() {
	fixtures := CreatePolicyFixtures(ms.DB, FixturesConfig{
		NumberOfPolicies: 1,
	})
	claim := &Claim{
		PolicyID:         fixtures.Policies[0].ID,
		EventDate:        time.Now().UTC(),
		EventType:        api.ClaimEventTypeImpact,
		EventDescription: "fell",
		Status:           api.ClaimStatusReview1,
	}
	ms.NoError(claim.Create(ms.DB))
	ms.Len(claim.ReferenceNumber, ClaimReferenceNumberLength)
}

func (ms *ModelSuite) TestClaim_SubmitForApproval() {
	t := ms.T()

	fixConfig := FixturesConfig{
		NumberOfPolicies:    2,
		UsersPerPolicy:      2,
		DependentsPerPolicy: 2,
		ItemsPerPolicy:      4,
		ClaimsPerPolicy:     4,
		ClaimItemsPerClaim:  1,
	}

	fixtures := CreateItemFixtures(ms.DB, fixConfig)
	policy := fixtures.Policies[0]
	draftClaim := policy.Claims[0]
	revisionClaim := UpdateClaimStatus(ms.DB, policy.Claims[1], api.ClaimStatusRevision)
	reviewClaim := UpdateClaimStatus(ms.DB, policy.Claims[2], api.ClaimStatusReview1)
	emptyClaim := UpdateClaimStatus(ms.DB, policy.Claims[3], api.ClaimStatusDraft)

	tempClaim := emptyClaim
	tempClaim.LoadClaimItems(ms.DB, false)
	ms.NoError(ms.DB.Destroy(&tempClaim.ClaimItems[0]),
		"error trying to destroy ClaimItem fixture for test")

	tests := []struct {
		name            string
		claim           Claim
		wantErrContains string
		wantErrKey      api.ErrorKey
		wantErrCat      api.ErrorCategory
		wantStatus      api.ClaimStatus
	}{
		{
			name:            "bad start status",
			claim:           reviewClaim,
			wantErrKey:      api.ErrorClaimStatus,
			wantErrCat:      api.CategoryUser,
			wantErrContains: "invalid claim status for submit",
		},
		{
			name:            "claim with no ClaimItem",
			claim:           emptyClaim,
			wantErrKey:      api.ErrorClaimMissingClaimItem,
			wantErrCat:      api.CategoryUser,
			wantErrContains: "claim must have a claimItem if no longer in draft",
		},
		{
			name:       "from draft to review1",
			claim:      draftClaim,
			wantStatus: api.ClaimStatusReview1,
		},
		{
			name:       "from revision to review1",
			claim:      revisionClaim,
			wantStatus: api.ClaimStatusReview1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.claim.SubmitForApproval(ms.DB)

			if tt.wantErrContains != "" {
				ms.Error(got, " did not return expected error")
				var appErr *api.AppError
				ms.True(errors.As(got, &appErr), "returned an error that is not an AppError")
				ms.Contains(got.Error(), tt.wantErrContains, "error message is not correct")
				ms.Equal(appErr.Key, tt.wantErrKey, "error key is not correct")
				ms.Equal(appErr.Category, tt.wantErrCat, "error category is not correct")
				return
			}
			ms.NoError(got)

			ms.Equal(tt.wantStatus, tt.claim.Status, "incorrect status")
		})
	}
}

func (ms *ModelSuite) TestClaim_RequestRevision() {
	t := ms.T()

	fixConfig := FixturesConfig{
		NumberOfPolicies:    2,
		UsersPerPolicy:      2,
		DependentsPerPolicy: 2,
		ItemsPerPolicy:      4,
		ClaimsPerPolicy:     4,
		ClaimItemsPerClaim:  1,
	}

	fixtures := CreateItemFixtures(ms.DB, fixConfig)
	policy := fixtures.Policies[0]
	draftClaim := policy.Claims[0]
	review1Claim := UpdateClaimStatus(ms.DB, policy.Claims[2], api.ClaimStatusReview1)
	review3Claim := UpdateClaimStatus(ms.DB, policy.Claims[2], api.ClaimStatusReview3)
	emptyClaim := UpdateClaimStatus(ms.DB, policy.Claims[3], api.ClaimStatusReview1)

	tempClaim := emptyClaim
	tempClaim.LoadClaimItems(ms.DB, false)
	ms.NoError(ms.DB.Destroy(&tempClaim.ClaimItems[0]),
		"error trying to destroy ClaimItem fixture for test")

	tests := []struct {
		name            string
		claim           Claim
		wantErrContains string
		wantErrKey      api.ErrorKey
		wantErrCat      api.ErrorCategory
		wantStatus      api.ClaimStatus
	}{
		{
			name:            "bad start status",
			claim:           draftClaim,
			wantErrKey:      api.ErrorClaimStatus,
			wantErrCat:      api.CategoryUser,
			wantErrContains: "invalid claim status for request revision",
		},
		{
			name:            "claim with no ClaimItem",
			claim:           emptyClaim,
			wantErrKey:      api.ErrorClaimMissingClaimItem,
			wantErrCat:      api.CategoryUser,
			wantErrContains: "claim must have a claimItem if no longer in draft",
		},
		{
			name:       "from review1 to revision",
			claim:      review1Claim,
			wantStatus: api.ClaimStatusRevision,
		},
		{
			name:       "from review3 to revision",
			claim:      review3Claim,
			wantStatus: api.ClaimStatusRevision,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.claim.RequestRevision(ms.DB)

			if tt.wantErrContains != "" {
				ms.Error(got, " did not return expected error")
				var appErr *api.AppError
				ms.True(errors.As(got, &appErr), "returned an error that is not an AppError")
				ms.Contains(got.Error(), tt.wantErrContains, "error message is not correct")
				ms.Equal(appErr.Key, tt.wantErrKey, "error key is not correct")
				ms.Equal(appErr.Category, tt.wantErrCat, "error category is not correct")
				return
			}
			ms.NoError(got)

			ms.Equal(tt.wantStatus, tt.claim.Status, "incorrect status")
		})
	}
}

func (ms *ModelSuite) TestClaim_Preapprove() {
	t := ms.T()

	fixConfig := FixturesConfig{
		NumberOfPolicies:    2,
		UsersPerPolicy:      2,
		DependentsPerPolicy: 2,
		ItemsPerPolicy:      4,
		ClaimsPerPolicy:     4,
		ClaimItemsPerClaim:  1,
	}

	fixtures := CreateItemFixtures(ms.DB, fixConfig)
	policy := fixtures.Policies[0]
	draftClaim := policy.Claims[0]
	review1Claim := UpdateClaimStatus(ms.DB, policy.Claims[2], api.ClaimStatusReview1)
	emptyClaim := UpdateClaimStatus(ms.DB, policy.Claims[3], api.ClaimStatusDraft)

	tempClaim := emptyClaim
	tempClaim.LoadClaimItems(ms.DB, false)
	ms.NoError(ms.DB.Destroy(&tempClaim.ClaimItems[0]),
		"error trying to destroy ClaimItem fixture for test")

	tests := []struct {
		name            string
		claim           Claim
		wantErrContains string
		wantErrKey      api.ErrorKey
		wantErrCat      api.ErrorCategory
		wantStatus      api.ClaimStatus
	}{
		{
			name:            "bad start status",
			claim:           draftClaim,
			wantErrKey:      api.ErrorClaimStatus,
			wantErrCat:      api.CategoryUser,
			wantErrContains: "invalid claim status for preapprove",
		},
		{
			name:            "claim with no ClaimItem",
			claim:           emptyClaim,
			wantErrKey:      api.ErrorClaimMissingClaimItem,
			wantErrCat:      api.CategoryUser,
			wantErrContains: "claim must have a claimItem to preapprove",
		},
		{
			name:       "from review1 to receipt",
			claim:      review1Claim,
			wantStatus: api.ClaimStatusReceipt,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.claim.PreApprove(ms.DB)

			if tt.wantErrContains != "" {
				ms.Error(got, " did not return expected error")
				var appErr *api.AppError
				ms.True(errors.As(got, &appErr), "returned an error that is not an AppError")
				ms.Contains(got.Error(), tt.wantErrContains, "error message is not correct")
				ms.Equal(appErr.Key, tt.wantErrKey, "error key is not correct")
				ms.Equal(appErr.Category, tt.wantErrCat, "error category is not correct")
				return
			}
			ms.NoError(got)

			ms.Equal(tt.wantStatus, tt.claim.Status, "incorrect status")
		})
	}
}
