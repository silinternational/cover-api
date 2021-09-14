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
			name: "empty revision message",
			claim: &Claim{
				ReferenceNumber:     domain.RandomString(ClaimReferenceNumberLength, ""),
				PolicyID:            domain.GetUUID(),
				IncidentType:        api.ClaimIncidentTypeImpact,
				IncidentDate:        time.Now(),
				IncidentDescription: "testing123",
				Status:              api.ClaimStatusRevision,
			},
			errField: "Claim.RevisionMessage",
			wantErr:  true,
		},
		{
			name: "valid status",
			claim: &Claim{
				ReferenceNumber:     domain.RandomString(ClaimReferenceNumberLength, ""),
				PolicyID:            domain.GetUUID(),
				IncidentType:        api.ClaimIncidentTypeImpact,
				IncidentDate:        time.Now(),
				IncidentDescription: "testing123",
				Status:              api.ClaimStatusReview1,
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
		PolicyID:            fixtures.Policies[0].ID,
		IncidentDate:        time.Now().UTC(),
		IncidentType:        api.ClaimIncidentTypeImpact,
		IncidentDescription: "fell",
		Status:              api.ClaimStatusReview1,
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
			const message = "change all the things"
			got := tt.claim.RequestRevision(ms.DB, message)

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
			ms.Equal(message, tt.claim.RevisionMessage, "incorrect revision message")
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
			wantErrContains: "invalid claim status for request receipt",
		},
		{
			name:            "claim with no ClaimItem",
			claim:           emptyClaim,
			wantErrKey:      api.ErrorClaimMissingClaimItem,
			wantErrCat:      api.CategoryUser,
			wantErrContains: "claim must have a claimItem if no longer in draft",
		},
		{
			name:       "from review1 to receipt",
			claim:      review1Claim,
			wantStatus: api.ClaimStatusReceipt,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.claim.RequestReceipt(ms.DB)

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

func (ms *ModelSuite) TestClaim_Approve() {
	t := ms.T()

	fixConfig := FixturesConfig{
		NumberOfPolicies:    2,
		UsersPerPolicy:      2,
		DependentsPerPolicy: 2,
		ItemsPerPolicy:      4,
		ClaimsPerPolicy:     5,
		ClaimItemsPerClaim:  1,
	}

	fixtures := CreateItemFixtures(ms.DB, fixConfig)

	appAdmin := CreateAdminUsers(ms.DB)[AppRoleAdmin]

	policy := fixtures.Policies[0]
	draftClaim := policy.Claims[0]
	review1Claim := UpdateClaimStatus(ms.DB, policy.Claims[1], api.ClaimStatusReview1)
	review2Claim := UpdateClaimStatus(ms.DB, policy.Claims[2], api.ClaimStatusReview2)
	review3Claim := UpdateClaimStatus(ms.DB, policy.Claims[3], api.ClaimStatusReview3)
	emptyClaim := UpdateClaimStatus(ms.DB, policy.Claims[4], api.ClaimStatusReview1)

	tempClaim := emptyClaim
	tempClaim.LoadClaimItems(ms.DB, false)
	ms.NoError(ms.DB.Destroy(&tempClaim.ClaimItems[0]),
		"error trying to destroy ClaimItem fixture for test")

	tests := []struct {
		name            string
		claim           Claim
		actor           User
		wantErrContains string
		wantErrKey      api.ErrorKey
		wantErrCat      api.ErrorCategory
		wantStatus      api.ClaimStatus
	}{
		{
			name:            "bad start status",
			claim:           draftClaim,
			actor:           appAdmin,
			wantErrKey:      api.ErrorClaimStatus,
			wantErrCat:      api.CategoryUser,
			wantErrContains: "invalid claim status for approve",
		},
		{
			name:            "claim with no ClaimItem",
			claim:           emptyClaim,
			actor:           appAdmin,
			wantErrKey:      api.ErrorClaimMissingClaimItem,
			wantErrCat:      api.CategoryUser,
			wantErrContains: "claim must have a claimItem if no longer in draft",
		},
		{
			name:       "from review1 to review3",
			claim:      review1Claim,
			actor:      appAdmin,
			wantStatus: api.ClaimStatusReview3,
		},
		{
			name:       "from review2 to review3",
			claim:      review2Claim,
			actor:      appAdmin,
			wantStatus: api.ClaimStatusReview3,
		},
		{
			name:       "from review3 to approved",
			claim:      review3Claim,
			actor:      appAdmin,
			wantStatus: api.ClaimStatusApproved,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.claim.Approve(ms.DB, tt.actor)

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
			ms.Equal(tt.actor.ID.String(), tt.claim.ReviewerID.UUID.String(), "incorrect reviewer id")
			ms.WithinDuration(time.Now().UTC(), tt.claim.ReviewDate.Time, time.Second*2, "incorrect reviewer date id")
		})
	}
}

func (ms *ModelSuite) TestClaim_Deny() {
	t := ms.T()

	fixConfig := FixturesConfig{
		NumberOfPolicies:    2,
		UsersPerPolicy:      2,
		DependentsPerPolicy: 2,
		ItemsPerPolicy:      4,
		ClaimsPerPolicy:     5,
		ClaimItemsPerClaim:  1,
	}

	fixtures := CreateItemFixtures(ms.DB, fixConfig)

	appAdmin := CreateAdminUsers(ms.DB)[AppRoleAdmin]

	policy := fixtures.Policies[0]
	draftClaim := policy.Claims[0]
	review1Claim := UpdateClaimStatus(ms.DB, policy.Claims[1], api.ClaimStatusReview1)
	review2Claim := UpdateClaimStatus(ms.DB, policy.Claims[2], api.ClaimStatusReview2)
	review3Claim := UpdateClaimStatus(ms.DB, policy.Claims[3], api.ClaimStatusReview3)
	emptyClaim := UpdateClaimStatus(ms.DB, policy.Claims[4], api.ClaimStatusReview1)

	tempClaim := emptyClaim
	tempClaim.LoadClaimItems(ms.DB, false)
	ms.NoError(ms.DB.Destroy(&tempClaim.ClaimItems[0]),
		"error trying to destroy ClaimItem fixture for test")

	tests := []struct {
		name            string
		claim           Claim
		actor           User
		wantErrContains string
		wantErrKey      api.ErrorKey
		wantErrCat      api.ErrorCategory
		wantStatus      api.ClaimStatus
	}{
		{
			name:            "bad start status",
			claim:           draftClaim,
			actor:           appAdmin,
			wantErrKey:      api.ErrorClaimStatus,
			wantErrCat:      api.CategoryUser,
			wantErrContains: "invalid claim status for deny",
		},
		{
			name:            "claim with no ClaimItem",
			claim:           emptyClaim,
			actor:           appAdmin,
			wantErrKey:      api.ErrorClaimMissingClaimItem,
			wantErrCat:      api.CategoryUser,
			wantErrContains: "claim must have a claimItem if no longer in draft",
		},
		{
			name:       "from review1 to denied",
			claim:      review1Claim,
			actor:      appAdmin,
			wantStatus: api.ClaimStatusDenied,
		},
		{
			name:       "from review2 to denied",
			claim:      review2Claim,
			actor:      appAdmin,
			wantStatus: api.ClaimStatusDenied,
		},
		{
			name:       "from review3 to denied",
			claim:      review3Claim,
			actor:      appAdmin,
			wantStatus: api.ClaimStatusDenied,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.claim.Deny(ms.DB, tt.actor)

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
			ms.Equal(tt.actor.ID.String(), tt.claim.ReviewerID.UUID.String(), "incorrect reviewer id")
			ms.WithinDuration(time.Now().UTC(), tt.claim.ReviewDate.Time, time.Second*2, "incorrect reviewer date id")
		})
	}
}

func (ms *ModelSuite) TestClaim_ConvertToAPI() {
	policy := CreatePolicyFixtures(ms.DB, FixturesConfig{}).Policies[0]
	claim := createClaimFixture(ms.DB, policy, FixturesConfig{
		ClaimItemsPerClaim: 2,
		ClaimFilesPerClaim: 3,
	})
	claim.RevisionMessage = "change request " + domain.RandomString(8, "0123456789")

	got := claim.ConvertToAPI(ms.DB)

	ms.Equal(claim.ID, got.ID, "ID is not correct")
	ms.Equal(claim.PolicyID, got.PolicyID, "PolicyID is not correct")
	ms.Equal(claim.ReferenceNumber, got.ReferenceNumber, "ReferenceNumber is not correct")
	ms.Equal(claim.IncidentDate, got.IncidentDate, "IncidentDate is not correct")
	ms.Equal(claim.IncidentType, got.IncidentType, "IncidentType is not correct")
	ms.Equal(claim.IncidentDescription, got.IncidentDescription, "IncidentDescription is not correct")
	ms.Equal(claim.Status, got.Status, "Status is not correct")
	ms.Equal(claim.ReviewDate, got.ReviewDate, "ReviewDate is not correct")
	ms.Equal(claim.ReviewerID, got.ReviewerID, "ReviewerID is not correct")
	ms.Equal(claim.PaymentDate, got.PaymentDate, "PaymentDate is not correct")
	ms.Equal(claim.TotalPayout, got.TotalPayout, "TotalPayout is not correct")
	ms.Equal(claim.RevisionMessage, got.RevisionMessage, "RevisionMessage is not correct")

	ms.Greater(len(claim.ClaimItems), 0, "test should be revised, fixture has no ClaimItems")
	ms.Len(got.Items, len(claim.ClaimItems), "Items is not correct length")

	ms.Greater(len(claim.ClaimFiles), 0, "test should be revised, fixture has no ClaimFiles")
	ms.Len(got.Files, len(claim.ClaimFiles), "Files is not correct length")
}
