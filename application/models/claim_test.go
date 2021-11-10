package models

import (
	"errors"
	"testing"
	"time"

	"github.com/gobuffalo/nulls"
	"github.com/gofrs/uuid"

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
			name: "empty revision message - status = Revision",
			claim: &Claim{
				ReferenceNumber:     domain.RandomString(ClaimReferenceNumberLength, ""),
				PolicyID:            domain.GetUUID(),
				IncidentType:        api.ClaimIncidentTypeImpact,
				IncidentDate:        time.Now(),
				IncidentDescription: "testing123",
				Status:              api.ClaimStatusRevision,
			},
			errField: "Claim.StatusReason",
			wantErr:  true,
		},
		{
			name: "empty revision message - status = Denied",
			claim: &Claim{
				ReferenceNumber:     domain.RandomString(ClaimReferenceNumberLength, ""),
				PolicyID:            domain.GetUUID(),
				IncidentType:        api.ClaimIncidentTypeImpact,
				IncidentDate:        time.Now(),
				IncidentDescription: "testing123",
				Status:              api.ClaimStatusDenied,
			},
			errField: "Claim.StatusReason",
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
		ClaimsPerPolicy:     5,
		ClaimItemsPerClaim:  1,
	}

	fixtures := CreateItemFixtures(ms.DB, fixConfig)
	policy := fixtures.Policies[0]
	draftClaim := policy.Claims[0]
	revisionClaim := UpdateClaimStatus(ms.DB, policy.Claims[1], api.ClaimStatusRevision, "")
	reviewClaim := UpdateClaimStatus(ms.DB, policy.Claims[2], api.ClaimStatusReview1, "")
	emptyClaim := UpdateClaimStatus(ms.DB, policy.Claims[3], api.ClaimStatusDraft, "")
	itemNotReadyClaim := policy.Claims[4]

	goodParams := UpdateClaimItemsParams{
		PayoutOption:   api.PayoutOptionRepair,
		IsRepairable:   true,
		RepairEstimate: 1000,
		FMV:            2000,
	}
	UpdateClaimItems(ms.DB, draftClaim, goodParams)
	UpdateClaimItems(ms.DB, revisionClaim, goodParams)
	UpdateClaimItems(ms.DB, reviewClaim, goodParams)

	badParams := goodParams
	badParams.RepairEstimate = 0
	UpdateClaimItems(ms.DB, itemNotReadyClaim, badParams)

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
			name:            "from draft to review1, item not ready",
			claim:           itemNotReadyClaim,
			wantErrKey:      api.ErrorClaimItemMissingRepairEstimate,
			wantErrCat:      api.CategoryUser,
			wantErrContains: "not valid for claim submission",
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
			ctx := CreateTestContext(fixtures.Users[0])
			got := tt.claim.SubmitForApproval(ctx)

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
			ms.Greater(tt.claim.TotalPayout, 0, "total payout was not set")
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
	review1Claim := UpdateClaimStatus(ms.DB, policy.Claims[2], api.ClaimStatusReview1, "")
	review3Claim := UpdateClaimStatus(ms.DB, policy.Claims[2], api.ClaimStatusReview3, "")
	emptyClaim := UpdateClaimStatus(ms.DB, policy.Claims[3], api.ClaimStatusReview1, "")

	tempClaim := emptyClaim
	tempClaim.LoadClaimItems(ms.DB, false)
	ms.NoError(ms.DB.Destroy(&tempClaim.ClaimItems[0]),
		"error trying to destroy ClaimItem fixture for test")

	admin := CreateAdminUsers(ms.DB)[AppRoleSteward]

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
			wantErrKey:      api.ErrorValidation,
			wantErrCat:      api.CategoryUser,
			wantErrContains: "invalid claim status transition from Draft to Revision",
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
			ctx := CreateTestContext(admin)
			got := tt.claim.RequestRevision(ctx, message)

			if tt.wantErrContains != "" {
				ms.Error(got, " did not return expected error")
				var appErr *api.AppError
				ms.True(errors.As(got, &appErr), "returned an error that is not an AppError")
				ms.Contains(got.Error(), tt.wantErrContains, "error message is not correct")
				ms.Equal(tt.wantErrKey, appErr.Key, "error key is not correct")
				ms.Equal(tt.wantErrCat, appErr.Category, "error category is not correct")
				return
			}
			ms.NoError(got)

			ms.Equal(tt.wantStatus, tt.claim.Status, "incorrect status")
			ms.Equal(message, tt.claim.StatusReason, "incorrect status reason message")
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
	review1Claim := UpdateClaimStatus(ms.DB, policy.Claims[2], api.ClaimStatusReview1, "")
	emptyClaim := UpdateClaimStatus(ms.DB, policy.Claims[3], api.ClaimStatusReview1, "")

	tempClaim := emptyClaim
	tempClaim.LoadClaimItems(ms.DB, false)
	ms.NoError(ms.DB.Destroy(&tempClaim.ClaimItems[0]),
		"error trying to destroy ClaimItem fixture for test")

	admin := CreateAdminUsers(ms.DB)[AppRoleSteward]

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
			ctx := CreateTestContext(admin)
			got := tt.claim.RequestReceipt(ctx, "")

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

	admin := CreateAdminUsers(ms.DB)[AppRoleAdmin]

	policy := fixtures.Policies[0]
	draftClaim := policy.Claims[0]
	review1Claim := UpdateClaimStatus(ms.DB, policy.Claims[1], api.ClaimStatusReview1, "")
	review2Claim := UpdateClaimStatus(ms.DB, policy.Claims[2], api.ClaimStatusReview2, "")
	review3Claim := UpdateClaimStatus(ms.DB, policy.Claims[3], api.ClaimStatusReview3, "")
	emptyClaim := UpdateClaimStatus(ms.DB, policy.Claims[4], api.ClaimStatusReview1, "")

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
			wantErrContains: "invalid claim status for approve",
		},
		{
			name:            "claim with no ClaimItem",
			claim:           emptyClaim,
			wantErrKey:      api.ErrorClaimMissingClaimItem,
			wantErrCat:      api.CategoryUser,
			wantErrContains: "claim must have a claimItem if no longer in draft",
		},
		{
			name:       "from review1 to review3",
			claim:      review1Claim,
			wantStatus: api.ClaimStatusReview3,
		},
		{
			name:       "from review2 to review3",
			claim:      review2Claim,
			wantStatus: api.ClaimStatusReview3,
		},
		{
			name:       "from review3 to approved",
			claim:      review3Claim,
			wantStatus: api.ClaimStatusApproved,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := CreateTestContext(admin)
			got := tt.claim.Approve(ctx)

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
			ms.Equal(admin.ID.String(), tt.claim.ReviewerID.UUID.String(), "incorrect reviewer id")
			ms.WithinDuration(time.Now().UTC(), tt.claim.ReviewDate.Time, time.Second*2, "incorrect reviewer date id")
			ms.Equal("", tt.claim.StatusReason, "StatusReason should be empty after approval")
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

	admin := CreateAdminUsers(ms.DB)[AppRoleAdmin]

	policy := fixtures.Policies[0]
	draftClaim := policy.Claims[0]
	review1Claim := UpdateClaimStatus(ms.DB, policy.Claims[1], api.ClaimStatusReview1, "")
	review2Claim := UpdateClaimStatus(ms.DB, policy.Claims[2], api.ClaimStatusReview2, "")
	review3Claim := UpdateClaimStatus(ms.DB, policy.Claims[3], api.ClaimStatusReview3, "")
	emptyClaim := UpdateClaimStatus(ms.DB, policy.Claims[4], api.ClaimStatusReview1, "")

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
			wantErrContains: "invalid claim status for deny",
		},
		{
			name:            "claim with no ClaimItem",
			claim:           emptyClaim,
			wantErrKey:      api.ErrorClaimMissingClaimItem,
			wantErrCat:      api.CategoryUser,
			wantErrContains: "claim must have a claimItem if no longer in draft",
		},
		{
			name:       "from review1 to denied",
			claim:      review1Claim,
			wantStatus: api.ClaimStatusDenied,
		},
		{
			name:       "from review2 to denied",
			claim:      review2Claim,
			wantStatus: api.ClaimStatusDenied,
		},
		{
			name:       "from review3 to denied",
			claim:      review3Claim,
			wantStatus: api.ClaimStatusDenied,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			const message = "change all the things"
			ctx := CreateTestContext(admin)
			got := tt.claim.Deny(ctx, message)

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
			ms.Equal(admin.ID.String(), tt.claim.ReviewerID.UUID.String(), "incorrect reviewer id")
			ms.WithinDuration(time.Now().UTC(), tt.claim.ReviewDate.Time, time.Second*2, "incorrect reviewer date id")
			ms.Equal(message, tt.claim.StatusReason, "incorrect status reason message")
		})
	}
}

func (ms *ModelSuite) TestClaim_HasReceiptFile() {
	db := ms.DB
	config := FixturesConfig{
		NumberOfPolicies: 3,
		ItemsPerPolicy:   1,
		ClaimsPerPolicy:  1,
	}
	fixtures := CreateItemFixtures(db, config)

	files := CreateFileFixtures(db, 2, CreateAdminUsers(db)[AppRoleAdmin].ID).Files

	policies := fixtures.Policies

	claimNoFile := UpdateClaimStatus(db, policies[0].Claims[0], api.ClaimStatusReceipt, "")
	claimNoReceiptFile := UpdateClaimStatus(db, policies[1].Claims[0], api.ClaimStatusReceipt, "")
	claimWithReceipt := UpdateClaimStatus(db, policies[2].Claims[0], api.ClaimStatusReceipt, "")

	ms.NoError(NewClaimFile(claimNoReceiptFile.ID, files[0].ID, api.ClaimFilePurposeRepairEstimate).Create(db))
	ms.NoError(NewClaimFile(claimWithReceipt.ID, files[1].ID, api.ClaimFilePurposeReceipt).Create(db))

	tests := []struct {
		name  string
		claim Claim
		want  bool
	}{
		{
			name:  "has no file at all",
			claim: claimNoFile,
			want:  false,
		},
		{
			name:  "has only a non-receipt file",
			claim: claimNoReceiptFile,
			want:  false,
		},
		{
			name:  "has a receipt file",
			claim: claimWithReceipt,
			want:  true,
		},
	}
	for _, tt := range tests {
		ms.T().Run(tt.name, func(t *testing.T) {
			got := tt.claim.HasReceiptFile(db)
			ms.Equal(tt.want, got, "incorrect value for whether claim has a receipt file")
		})
	}
}

func (ms *ModelSuite) TestClaim_ConvertToAPI() {
	fixtures := CreateItemFixtures(ms.DB, FixturesConfig{
		ClaimsPerPolicy:    1,
		ClaimItemsPerClaim: 2,
		ClaimFilesPerClaim: 3,
	})
	claim := fixtures.Claims[0]

	claim.StatusReason = "change request " + domain.RandomString(8, "0123456789")

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
	ms.Equal(claim.StatusReason, got.StatusReason, "StatusReason is not correct")

	ms.Greater(len(claim.ClaimItems), 0, "test should be revised, fixture has no ClaimItems")
	ms.Len(got.Items, len(claim.ClaimItems), "Items is not correct length")

	ms.Greater(len(claim.ClaimFiles), 0, "test should be revised, fixture has no ClaimFiles")
	ms.Len(got.Files, len(claim.ClaimFiles), "Files is not correct length")
}

func (ms *ModelSuite) TestClaim_Compare() {
	e := EntityCode{
		Code: randStr(3),
		Name: "Acme, Inc.",
	}
	MustCreate(ms.DB, &e)

	f := CreateItemFixtures(ms.DB, FixturesConfig{ClaimsPerPolicy: 1})
	oldClaim := f.Claims[0]
	newClaim := Claim{
		ReviewDate:   nulls.NewTime(time.Now().UTC().Add(-1 * time.Hour)),
		ReviewerID:   nulls.NewUUID(f.Users[0].ID),
		PaymentDate:  nulls.NewTime(time.Now().UTC()),
		TotalPayout:  10000,
		StatusReason: "because",
	}

	tests := []struct {
		name string
		new  Claim
		old  Claim
		want []FieldUpdate
	}{
		{
			name: "1",
			new:  newClaim,
			old:  oldClaim,
			want: []FieldUpdate{
				{
					FieldName: FieldClaimPolicyID,
					OldValue:  oldClaim.PolicyID.String(),
					NewValue:  newClaim.PolicyID.String(),
				},
				{
					FieldName: FieldClaimReferenceNumber,
					OldValue:  oldClaim.ReferenceNumber,
					NewValue:  newClaim.ReferenceNumber,
				},
				{
					FieldName: FieldClaimIncidentDate,
					OldValue:  oldClaim.IncidentDate.String(),
					NewValue:  newClaim.IncidentDate.String(),
				},
				{
					FieldName: FieldClaimIncidentType,
					OldValue:  string(oldClaim.IncidentType),
					NewValue:  string(newClaim.IncidentType),
				},
				{
					FieldName: FieldClaimIncidentDescription,
					OldValue:  oldClaim.IncidentDescription,
					NewValue:  newClaim.IncidentDescription,
				},
				{
					FieldName: FieldClaimStatus,
					OldValue:  string(oldClaim.Status),
					NewValue:  string(newClaim.Status),
				},
				{
					FieldName: FieldClaimReviewDate,
					OldValue:  oldClaim.ReviewDate.Time.String(),
					NewValue:  newClaim.ReviewDate.Time.String(),
				},
				{
					FieldName: FieldClaimReviewerID,
					OldValue:  oldClaim.ReviewerID.UUID.String(),
					NewValue:  newClaim.ReviewerID.UUID.String(),
				},
				{
					FieldName: FieldClaimPaymentDate,
					OldValue:  oldClaim.PaymentDate.Time.String(),
					NewValue:  newClaim.PaymentDate.Time.String(),
				},
				{
					FieldName: FieldClaimTotalPayout,
					OldValue:  oldClaim.TotalPayout.String(),
					NewValue:  newClaim.TotalPayout.String(),
				},
				{
					FieldName: FieldClaimStatusReason,
					OldValue:  oldClaim.StatusReason,
					NewValue:  newClaim.StatusReason,
				},
			},
		},
	}
	for _, tt := range tests {
		ms.T().Run(tt.name, func(t *testing.T) {
			got := tt.new.Compare(tt.old)
			ms.ElementsMatch(tt.want, got)
		})
	}
}

func (ms *ModelSuite) TestClaim_NewHistory() {
	f := CreateItemFixtures(ms.DB, FixturesConfig{ClaimsPerPolicy: 1})
	claim := f.Claims[0]
	user := f.Users[0]

	const newStatus = api.ClaimStatusApproved
	newIncidentDate := time.Now().UTC()

	tests := []struct {
		name   string
		claim  Claim
		user   User
		update FieldUpdate
		want   ClaimHistory
	}{
		{
			name:  "Status",
			claim: claim,
			user:  user,
			update: FieldUpdate{
				FieldName: "Status",
				OldValue:  string(claim.Status),
				NewValue:  string(newStatus),
			},
			want: ClaimHistory{
				ClaimID:   claim.ID,
				UserID:    user.ID,
				Action:    api.HistoryActionUpdate,
				FieldName: "Status",
				OldValue:  string(claim.Status),
				NewValue:  string(newStatus),
			},
		},
		{
			name:  "IncidentDate",
			claim: claim,
			user:  user,
			update: FieldUpdate{
				FieldName: "IncidentDate",
				OldValue:  claim.IncidentDate.String(),
				NewValue:  newIncidentDate.String(),
			},
			want: ClaimHistory{
				ClaimID:   claim.ID,
				UserID:    user.ID,
				Action:    api.HistoryActionUpdate,
				FieldName: "IncidentDate",
				OldValue:  claim.IncidentDate.String(),
				NewValue:  newIncidentDate.String(),
			},
		},
	}
	for _, tt := range tests {
		ms.T().Run(tt.name, func(t *testing.T) {
			got := tt.claim.NewHistory(CreateTestContext(tt.user), api.HistoryActionUpdate, tt.update)
			ms.False(tt.want.NewValue == tt.want.OldValue, "test isn't correctly checking a field update")
			ms.Equal(tt.want.ClaimID, got.ClaimID, "ClaimID is not correct")
			ms.Equal(tt.want.UserID, got.UserID, "UserID is not correct")
			ms.Equal(tt.want.Action, got.Action, "Action is not correct")
			ms.Equal(tt.want.FieldName, got.FieldName, "FieldName is not correct")
			ms.Equal(tt.want.OldValue, got.OldValue, "OldValue is not correct")
			ms.Equal(tt.want.NewValue, got.NewValue, "NewValue is not correct")
		})
	}
}

func (ms *ModelSuite) Test_ClaimsWithRecentStatusChanges() {
	fixtures := CreateClaimHistoryFixtures_RecentClaimStatusChanges(ms.DB)
	chFixes := fixtures.ClaimHistories

	gotRaw, gotErr := ClaimsWithRecentStatusChanges(ms.DB)
	ms.NoError(gotErr)

	const tmFmt = "Jan _2 15:04:05.00"

	got := make([][2]string, len(gotRaw))
	for i, g := range gotRaw {
		got[i] = [2]string{g.Claim.ID.String(), g.StatusUpdatedAt.Format(tmFmt)}
	}

	want := [][2]string{
		{chFixes[3].ClaimID.String(), chFixes[3].UpdatedAt.Format(tmFmt)},
		{chFixes[7].ClaimID.String(), chFixes[7].UpdatedAt.Format(tmFmt)},
	}

	ms.ElementsMatch(want, got, "incorrect results")
}

func (ms *ModelSuite) TestClaim_CreateLedgerEntry() {
	f := CreateItemFixtures(ms.DB, FixturesConfig{ClaimsPerPolicy: 1, ClaimItemsPerClaim: 1})
	item := f.Claims[0].ClaimItems[0].Item

	user := f.Users[0]
	ctx := CreateTestContext(user)
	ms.NoError(item.setAccountablePerson(ms.DB, user.ID))
	ms.NoError(item.Update(ctx))

	var claim Claim
	ms.NoError(ms.DB.Find(&claim, f.Claims[0].ID))

	ms.Error(claim.CreateLedgerEntry(ms.DB), "expected an error, claim isn't approved yet")

	claim.Status = api.ClaimStatusApproved
	claim.TotalPayout = 12345
	ms.NoError(ms.DB.Update(&claim), "unable to update claim test fixture")

	ms.NoError(claim.CreateLedgerEntry(ms.DB), "claim is approved now, it shouldn't be a problem")

	var le LedgerEntry
	ms.NoError(ms.DB.Where("claim_id = ?", claim.ID).First(&le))

	ms.Equal(LedgerEntryTypeClaim, le.Type, "Type is incorrect")
	ms.Equal(item.PolicyID, le.PolicyID, "PolicyID is incorrect")
	ms.Equal(item.ID, le.ItemID.UUID, "ItemID is incorrect")
	ms.Equal(claim.ID, le.ClaimID.UUID, "ClaimID is incorrect")
	ms.Equal(api.Currency(-12345), le.Amount, "Amount is incorrect")
	ms.Equal(user.FirstName, le.FirstName, "FirstName is incorrect")
	ms.Equal(user.LastName, le.LastName, "LastName is incorrect")
}

func (ms *ModelSuite) TestClaims_ByStatus() {
	f := CreateItemFixtures(ms.DB, FixturesConfig{
		NumberOfPolicies: 9,
		ClaimsPerPolicy:  1,
	})

	f.Claims[0].Status = api.ClaimStatusDraft
	f.Claims[1].Status = api.ClaimStatusReview1
	f.Claims[2].Status = api.ClaimStatusReview2
	f.Claims[3].Status = api.ClaimStatusReview3
	f.Claims[4].Status = api.ClaimStatusRevision
	f.Claims[5].Status = api.ClaimStatusReceipt
	f.Claims[6].Status = api.ClaimStatusApproved
	f.Claims[7].Status = api.ClaimStatusPaid
	f.Claims[8].Status = api.ClaimStatusDenied

	ms.NoError(ms.DB.Update(&f.Claims))

	tests := []struct {
		name         string
		statuses     []api.ClaimStatus
		wantClaimIDs []uuid.UUID
		wantErr      bool
	}{
		{
			name:         "default",
			wantClaimIDs: []uuid.UUID{f.Claims[1].ID, f.Claims[2].ID, f.Claims[3].ID},
			wantErr:      false,
		},
		{
			name:         "approved and paid",
			statuses:     []api.ClaimStatus{api.ClaimStatusApproved, api.ClaimStatusPaid},
			wantClaimIDs: []uuid.UUID{f.Claims[6].ID, f.Claims[7].ID},
			wantErr:      false,
		},
	}
	for _, tt := range tests {
		ms.T().Run(tt.name, func(t *testing.T) {
			var claims Claims
			err := claims.ByStatus(ms.DB, tt.statuses)
			ms.NoError(err)

			gotIDs := make([]uuid.UUID, len(claims))
			for i := range claims {
				gotIDs[i] = claims[i].ID
			}

			ms.Equal(len(tt.wantClaimIDs), len(gotIDs))
			ms.ElementsMatch(tt.wantClaimIDs, gotIDs)
		})
	}
}

func (ms *ModelSuite) TestClaim_calculatePayoutAmount() {
	fixtures := CreateItemFixtures(ms.DB, FixturesConfig{ClaimsPerPolicy: 1, ClaimItemsPerClaim: 1})
	fixtures.Claims[0].ClaimItems[0].RepairEstimate = 100
	ms.NoError(ms.DB.Update(&fixtures.Claims[0].ClaimItems[0]))

	// Get a fresh copy of the claim to ensure the UUT hydrates it as necessary
	var claim Claim
	ms.NoError(claim.FindByID(ms.DB, fixtures.Claims[0].ID))

	before := claim.TotalPayout

	ms.NoError(claim.calculatePayoutAmount(CreateTestContext(fixtures.Users[0])))

	// The claim item test will check the actual amount. Just make sure it changed.
	ms.False(claim.TotalPayout == before, "payout was not updated")
}
