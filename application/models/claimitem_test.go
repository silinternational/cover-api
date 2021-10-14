package models

import (
	"fmt"
	"testing"
	"time"

	"github.com/gobuffalo/nulls"

	"github.com/silinternational/cover-api/api"
	"github.com/silinternational/cover-api/domain"
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
			name: "approved, but no reviewer",
			claimItem: &ClaimItem{
				Claim: Claim{
					IncidentType: api.ClaimIncidentTypeImpact,
				},
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
				Claim: Claim{
					IncidentType: api.ClaimIncidentTypeImpact,
				},
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

func (ms *ModelSuite) TestClaimItem_ValidateForSubmit() {
	good := ClaimItem{
		IsRepairable:    false,
		RepairEstimate:  100,
		ReplaceEstimate: 1000,
		PayoutOption:    api.PayoutOptionRepair,
		FMV:             1000,
		Claim:           Claim{IncidentType: api.ClaimIncidentTypeTheft},
	}

	missingPayoutOption := good
	missingPayoutOption.PayoutOption = ""

	theftIsNotRepairable := good
	theftIsNotRepairable.IsRepairable = true

	missingReplaceEstimate := good
	missingReplaceEstimate.PayoutOption = api.PayoutOptionReplacement
	missingReplaceEstimate.ReplaceEstimate = 0

	missingFMV := good
	missingFMV.PayoutOption = api.PayoutOptionFMV
	missingFMV.FMV = 0

	missingRepairEstimate := good
	missingRepairEstimate.IsRepairable = true
	missingRepairEstimate.Claim.IncidentType = api.ClaimIncidentTypeImpact
	missingRepairEstimate.RepairEstimate = 0

	missingImpactFMV := good
	missingImpactFMV.IsRepairable = true
	missingImpactFMV.Claim.IncidentType = api.ClaimIncidentTypeImpact
	missingImpactFMV.FMV = 0

	invalidPayoutOption := good
	invalidPayoutOption.IsRepairable = false
	invalidPayoutOption.Claim.IncidentType = api.ClaimIncidentTypeImpact
	invalidPayoutOption.PayoutOption = api.PayoutOptionRepair

	invalidPayoutOptionEvacuation := good
	invalidPayoutOptionEvacuation.Claim.IncidentType = api.ClaimIncidentTypeEvacuation
	invalidPayoutOptionEvacuation.PayoutOption = api.PayoutOptionRepair

	missingReplaceEstimateImpact := good
	missingReplaceEstimateImpact.Claim.IncidentType = api.ClaimIncidentTypeImpact
	missingReplaceEstimateImpact.PayoutOption = api.PayoutOptionReplacement
	missingReplaceEstimateImpact.ReplaceEstimate = 0

	missingFMVImpact := good
	missingFMVImpact.Claim.IncidentType = api.ClaimIncidentTypeImpact
	missingFMVImpact.PayoutOption = api.PayoutOptionFMV
	missingFMVImpact.FMV = 0

	tests := []struct {
		name      string
		claimItem ClaimItem
		want      api.ErrorKey
	}{
		{
			name:      "missing payout option",
			claimItem: missingPayoutOption,
			want:      api.ErrorClaimItemMissingPayoutOption,
		},
		{
			name:      "item is not repairable",
			claimItem: theftIsNotRepairable,
			want:      api.ErrorClaimItemNotRepairable,
		},
		{
			name:      "missing replace estimate",
			claimItem: missingReplaceEstimate,
			want:      api.ErrorClaimItemMissingReplaceEstimate,
		},
		{
			name:      "missing FMV",
			claimItem: missingFMV,
			want:      api.ErrorClaimItemMissingFMV,
		},
		{
			name:      "invalid payout option",
			claimItem: invalidPayoutOption,
			want:      api.ErrorClaimItemInvalidPayoutOption,
		},
		{
			name:      "invalid payout option, evacuation",
			claimItem: invalidPayoutOptionEvacuation,
			want:      api.ErrorClaimItemInvalidPayoutOption,
		},
		{
			name:      "missing repair estimate",
			claimItem: missingRepairEstimate,
			want:      api.ErrorClaimItemMissingRepairEstimate,
		},
		{
			name:      "missing impact FMV",
			claimItem: missingImpactFMV,
			want:      api.ErrorClaimItemMissingFMV,
		},
		{
			name:      "missing replace estimate (impact)",
			claimItem: missingReplaceEstimateImpact,
			want:      api.ErrorClaimItemMissingReplaceEstimate,
		},
		{
			name:      "missing FMV (impact)",
			claimItem: missingFMVImpact,
			want:      api.ErrorClaimItemMissingFMV,
		},
		{
			name:      "good",
			claimItem: good,
			want:      "",
		},
	}
	for _, tt := range tests {
		ms.T().Run(tt.name, func(t *testing.T) {
			if got := tt.claimItem.ValidateForSubmit(ms.DB); got != tt.want {
				t.Errorf("ValidateForSubmit() = %v, want %v", got, tt.want)
			}
		})
	}
}

func (ms *ModelSuite) TestClaimItem_Compare() {
	fixtures := CreateItemFixtures(ms.DB, FixturesConfig{
		ClaimsPerPolicy:    1,
		ClaimItemsPerClaim: 1,
	})
	claim := fixtures.Claims[0]
	claim.LoadClaimItems(ms.DB, false)
	newCItem := claim.ClaimItems[0]

	oldCItem := ClaimItem{
		ItemID:          domain.GetUUID(),
		Status:          api.ClaimItemStatusReview3,
		IsRepairable:    true,
		RepairEstimate:  1111,
		RepairActual:    1112,
		ReplaceEstimate: 2221,
		ReplaceActual:   2222,
		PayoutOption:    api.PayoutOptionReplacement,
		PayoutAmount:    3331,
		FMV:             4441,
		ReviewDate:      nulls.NewTime(time.Date(1991, 1, 1, 1, 1, 1, 1, time.UTC)),
		ReviewerID:      nulls.NewUUID(domain.GetUUID()),
		Location:        "Timbuktu",
	}

	tests := []struct {
		name string
		new  ClaimItem
		old  ClaimItem
		want []FieldUpdate
	}{
		{
			name: "single test case",
			new:  newCItem,
			old:  oldCItem,
			want: []FieldUpdate{
				{
					FieldName: FieldClaimItemItemID,
					OldValue:  oldCItem.ItemID.String(),
					NewValue:  newCItem.ItemID.String(),
				},
				{
					FieldName: FieldClaimItemStatus,
					OldValue:  string(oldCItem.Status),
					NewValue:  string(newCItem.Status),
				},
				{
					FieldName: FieldClaimItemIsRepairable,
					OldValue:  fmt.Sprintf("%t", oldCItem.IsRepairable),
					NewValue:  fmt.Sprintf("%t", newCItem.IsRepairable),
				},
				{
					FieldName: FieldClaimItemRepairEstimate,
					OldValue:  api.Currency(oldCItem.RepairEstimate).String(),
					NewValue:  api.Currency(newCItem.RepairEstimate).String(),
				},
				{
					FieldName: FieldClaimItemRepairActual,
					OldValue:  api.Currency(oldCItem.RepairActual).String(),
					NewValue:  api.Currency(newCItem.RepairActual).String(),
				},
				{
					FieldName: FieldClaimItemReplaceEstimate,
					OldValue:  api.Currency(oldCItem.ReplaceEstimate).String(),
					NewValue:  api.Currency(newCItem.ReplaceEstimate).String(),
				},
				{
					FieldName: FieldClaimItemReplaceActual,
					OldValue:  api.Currency(oldCItem.ReplaceActual).String(),
					NewValue:  api.Currency(newCItem.ReplaceActual).String(),
				},
				{
					FieldName: FieldClaimItemPayoutOption,
					OldValue:  string(oldCItem.PayoutOption),
					NewValue:  string(newCItem.PayoutOption),
				},
				{
					FieldName: FieldClaimItemPayoutAmount,
					OldValue:  api.Currency(oldCItem.PayoutAmount).String(),
					NewValue:  api.Currency(newCItem.PayoutAmount).String(),
				},
				{
					FieldName: FieldClaimItemFMV,
					OldValue:  api.Currency(oldCItem.FMV).String(),
					NewValue:  api.Currency(newCItem.FMV).String(),
				},
				{
					FieldName: FieldClaimItemReviewDate,
					OldValue:  oldCItem.ReviewDate.Time.Format(domain.DateFormat),
					NewValue:  newCItem.ReviewDate.Time.Format(domain.DateFormat),
				},
				{
					FieldName: FieldClaimItemReviewerID,
					OldValue:  oldCItem.ReviewerID.UUID.String(),
					NewValue:  newCItem.ReviewerID.UUID.String(),
				},
				{
					FieldName: FieldClaimItemLocation,
					OldValue:  oldCItem.Location,
					NewValue:  newCItem.Location,
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
