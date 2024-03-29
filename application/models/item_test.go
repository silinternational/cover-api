package models

import (
	"fmt"
	"testing"
	"time"

	"github.com/gofrs/uuid"

	"github.com/silinternational/cover-api/domain"

	"github.com/gobuffalo/nulls"

	"github.com/silinternational/cover-api/api"
)

func (ms *ModelSuite) Test_isItemActionAllowed() {
	t := ms.T()
	tests := []struct {
		name         string
		actorIsAdmin bool
		startStatus  api.ItemCoverageStatus
		permission   Permission
		subRes       SubResource
		want         bool
	}{
		{
			name:         "draft with create and no sub resource - NO",
			actorIsAdmin: false,
			startStatus:  api.ItemCoverageStatusDraft,
			permission:   PermissionCreate,
			subRes:       "",
			want:         false,
		},
		{
			name:         "draft with update and no sub resource - YES",
			actorIsAdmin: false,
			startStatus:  api.ItemCoverageStatusDraft,
			permission:   PermissionUpdate,
			subRes:       "",
			want:         true,
		},
		{
			name:         "draft with update and submit sub resource - NO",
			actorIsAdmin: false,
			startStatus:  api.ItemCoverageStatusDraft,
			permission:   PermissionUpdate,
			subRes:       api.ResourceSubmit,
			want:         false,
		},
		{
			name:         "draft with create and wrong sub resource - NO",
			actorIsAdmin: false,
			startStatus:  api.ItemCoverageStatusDraft,
			permission:   PermissionCreate,
			subRes:       api.ResourceApprove,
			want:         false,
		},
		{
			name:         "draft with create and submit sub resource - YES",
			actorIsAdmin: false,
			startStatus:  api.ItemCoverageStatusDraft,
			permission:   PermissionCreate,
			subRes:       api.ResourceSubmit,
			want:         true,
		},
		{
			name:         "draft with delete and no sub resource - YES",
			actorIsAdmin: false,
			startStatus:  api.ItemCoverageStatusDraft,
			permission:   PermissionDelete,
			subRes:       "",
			want:         true,
		},
		{
			name:         "draft with delete and submit sub resource - NO",
			actorIsAdmin: false,
			startStatus:  api.ItemCoverageStatusDraft,
			permission:   PermissionDelete,
			subRes:       api.ResourceSubmit,
			want:         false,
		},
		{
			name:         "revision with create and no sub resource - NO",
			actorIsAdmin: false,
			startStatus:  api.ItemCoverageStatusRevision,
			permission:   PermissionCreate,
			subRes:       "",
			want:         false,
		},
		{
			name:         "revision with update and no sub resource - YES",
			actorIsAdmin: false,
			startStatus:  api.ItemCoverageStatusRevision,
			permission:   PermissionUpdate,
			subRes:       "",
			want:         true,
		},
		{
			name:         "revision with update and submit sub resource - NO",
			actorIsAdmin: false,
			startStatus:  api.ItemCoverageStatusRevision,
			permission:   PermissionUpdate,
			subRes:       api.ResourceSubmit,
			want:         false,
		},
		{
			name:         "revision with create and wrong sub resource - NO",
			actorIsAdmin: false,
			startStatus:  api.ItemCoverageStatusRevision,
			permission:   PermissionCreate,
			subRes:       api.ResourceApprove,
			want:         false,
		},
		{
			name:         "revision with create and submit sub resource - YES",
			actorIsAdmin: false,
			startStatus:  api.ItemCoverageStatusRevision,
			permission:   PermissionCreate,
			subRes:       api.ResourceSubmit,
			want:         true,
		},
		{
			name:         "revision with delete and no sub resource - YES",
			actorIsAdmin: false,
			startStatus:  api.ItemCoverageStatusRevision,
			permission:   PermissionDelete,
			subRes:       "",
			want:         true,
		},
		{
			name:         "revision with delete and submit sub resource - NO",
			actorIsAdmin: false,
			startStatus:  api.ItemCoverageStatusRevision,
			permission:   PermissionDelete,
			subRes:       api.ResourceSubmit,
			want:         false,
		},
		{
			name:         "pending with create and no sub resource - NO",
			actorIsAdmin: true,
			startStatus:  api.ItemCoverageStatusPending,
			permission:   PermissionCreate,
			subRes:       "",
			want:         false,
		},
		{
			name:         "pending with create and revision sub resource - YES",
			actorIsAdmin: true,
			startStatus:  api.ItemCoverageStatusPending,
			permission:   PermissionCreate,
			subRes:       api.ResourceRevision,
			want:         true,
		},
		{
			name:         "pending with create and approve sub resource - YES",
			actorIsAdmin: true,
			startStatus:  api.ItemCoverageStatusPending,
			permission:   PermissionCreate,
			subRes:       api.ResourceApprove,
			want:         true,
		},
		{
			name:         "pending with create and deny sub resource - YES",
			actorIsAdmin: true,
			startStatus:  api.ItemCoverageStatusPending,
			permission:   PermissionCreate,
			subRes:       api.ResourceDeny,
			want:         true,
		},
		{
			name:         "pending with create and revision sub resource but non-admin - NO",
			actorIsAdmin: false,
			startStatus:  api.ItemCoverageStatusPending,
			permission:   PermissionCreate,
			subRes:       api.ResourceRevision,
			want:         false,
		},
		{
			name:         "pending with delete and no sub resource - YES",
			actorIsAdmin: false,
			startStatus:  api.ItemCoverageStatusPending,
			permission:   PermissionDelete,
			subRes:       "",
			want:         true,
		},
		{
			name:         "pending with delete and submit sub resource - NO",
			actorIsAdmin: false,
			startStatus:  api.ItemCoverageStatusPending,
			permission:   PermissionDelete,
			subRes:       api.ResourceSubmit,
			want:         false,
		},
		{
			name:         "approved with create and no sub resource - NO",
			actorIsAdmin: true,
			startStatus:  api.ItemCoverageStatusApproved,
			permission:   PermissionCreate,
			subRes:       "",
			want:         false,
		},
		{
			name:         "approved with create and deny sub resource - NO",
			actorIsAdmin: true,
			startStatus:  api.ItemCoverageStatusApproved,
			permission:   PermissionCreate,
			subRes:       api.ResourceDeny,
			want:         false,
		},
		{
			name:         "approved with update and no sub resource - YES",
			actorIsAdmin: true,
			startStatus:  api.ItemCoverageStatusApproved,
			permission:   PermissionUpdate,
			subRes:       "",
			want:         true,
		},
		{
			name:         "approved with delete and no sub resource - YES",
			actorIsAdmin: false,
			startStatus:  api.ItemCoverageStatusApproved,
			permission:   PermissionDelete,
			subRes:       "",
			want:         true,
		},
		{
			name:         "approved with delete and deny sub resource - NO",
			actorIsAdmin: true,
			startStatus:  api.ItemCoverageStatusApproved,
			permission:   PermissionDelete,
			subRes:       api.ResourceDeny,
			want:         false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isItemActionAllowed(tt.actorIsAdmin, tt.startStatus, tt.permission, tt.subRes)
			ms.Equal(tt.want, got)
		})
	}
}

func (ms *ModelSuite) TestItem_Create() {
	t := ms.T()

	fixConfig := FixturesConfig{
		NumberOfPolicies:    2,
		UsersPerPolicy:      2,
		DependentsPerPolicy: 2,
		ItemsPerPolicy:      3,
	}

	fixtures := CreateItemFixtures(ms.DB, fixConfig)
	policy := fixtures.Policies[0]
	policy.LoadItems(ms.DB, false)
	items := policy.Items

	badPolicy := fixtures.Policies[1]
	badPolicy.HouseholdID = nulls.String{}
	ms.NoError(ms.DB.Update(&badPolicy))

	// give two items a dependent and calculate expected values
	dependent := policy.Dependents[0]
	coverageForPolicy := 0
	coverageForDep := 0
	for i, item := range items {
		if item.CoverageStatus != api.ItemCoverageStatusApproved {
			continue
		}
		if i == 2 {
			items[i].PolicyDependentID = nulls.NewUUID(dependent.ID)
			ms.NoError(ms.DB.Update(&items[i]), "error trying to change item DependentID")
			coverageForDep += items[i].CoverageAmount
		}
		coverageForPolicy += items[i].CoverageAmount
	}

	iCat := fixtures.ItemCategories[0]

	goodItem := Item{
		Name:              "Good Item",
		CategoryID:        iCat.ID,
		RiskCategoryID:    RiskCategoryStationaryID(),
		PolicyID:          policy.ID,
		InStorage:         true,
		Country:           "Thailand",
		Description:       "camera",
		Make:              "Minolta",
		Model:             "Max",
		SerialNumber:      "MM1234",
		CoverageAmount:    200,
		CoverageStartDate: time.Now().UTC().Add(time.Hour * 48),
	}

	noHouseholdID := goodItem
	noHouseholdID.PolicyID = badPolicy.ID

	tests := []struct {
		name     string
		item     Item
		appError *api.AppError
	}{
		{
			name:     "no household ID",
			item:     noHouseholdID,
			appError: &api.AppError{Key: api.ErrorPolicyHasNoHouseholdID, Category: api.CategoryUser},
		},
		{
			name: "good item",
			item: goodItem,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.item.Create(ms.DB)

			if tt.appError != nil {
				ms.Error(err, "test should have produced an error")
				ms.EqualAppError(*tt.appError, err)
				return
			}
			ms.NoError(err)

			ms.NotEqual(uuid.Nil, tt.item.ID, "expected item to have been given an ID")
			ms.Equal(api.ItemCoverageStatusDraft, tt.item.CoverageStatus, "incorrect status")
		})
	}
}

func (ms *ModelSuite) TestItem_SubmitForApproval() {
	t := ms.T()

	fixConfig := FixturesConfig{
		NumberOfPolicies:    2,
		UsersPerPolicy:      2,
		DependentsPerPolicy: 2,
		ItemsPerPolicy:      14,
	}

	fixtures := CreateItemFixtures(ms.DB, fixConfig)
	policy := fixtures.Policies[0]
	policy.LoadItems(ms.DB, false)
	items := fixtures.Items
	dependent := policy.Dependents[0]

	// first set the PolicyDependentID and CoverageAmount on an approved item
	itemDependent := items[1]
	itemDependent.PolicyDependentID = nulls.NewUUID(dependent.ID)
	itemDependent.CoverageAmount = 2000 * domain.CurrencyFactor // $2000
	itemDependent.CoverageStatus = api.ItemCoverageStatusApproved
	ms.NoError(ms.DB.Update(&itemDependent), "error updating item fixture for test")

	// specify other items for testing
	itemAutoApprove := fixtures.Policies[1].Items[0] // stationary
	itemManualApprove := items[2]                    // stationary
	itemManualApproveDependent := items[4]           // stationary
	itemAutoApproveDependent := items[6]             // stationary
	itemExceedsMax := items[8]                       // stationary
	itemStationaryMissingFields := items[10]         // stationary
	itemMobileMissingMake := items[11]               // mobile
	itemMobileMissingModel := items[13]              // mobile

	// set them all to Draft status
	itemAutoApprove.CoverageStatus = api.ItemCoverageStatusDraft
	itemManualApprove.CoverageStatus = api.ItemCoverageStatusDraft
	itemManualApproveDependent.CoverageStatus = api.ItemCoverageStatusDraft
	itemAutoApproveDependent.CoverageStatus = api.ItemCoverageStatusDraft
	itemExceedsMax.CoverageStatus = api.ItemCoverageStatusDraft
	itemStationaryMissingFields.CoverageStatus = api.ItemCoverageStatusDraft
	itemMobileMissingMake.CoverageStatus = api.ItemCoverageStatusDraft
	itemMobileMissingModel.CoverageStatus = api.ItemCoverageStatusDraft

	// set their coverage amounts to be helpful for the tests and set the dependent as needed
	itemAutoApprove.LoadCategory(ms.DB, false)
	itemAutoApprove.CoverageAmount = itemAutoApprove.Category.AutoApproveMax - 1
	ms.NoError(ms.DB.Update(&itemAutoApprove), "error updating item fixture for test")

	itemManualApprove.LoadCategory(ms.DB, false)
	itemManualApprove.CoverageAmount = itemManualApprove.Category.AutoApproveMax + 1
	ms.NoError(ms.DB.Update(&itemManualApprove), "error updating item fixture for test")

	itemManualApproveDependent.CoverageAmount = domain.Env.DependentAutoApproveMax - itemDependent.CoverageAmount + 1
	itemManualApproveDependent.PolicyDependentID = nulls.NewUUID(dependent.ID)
	ms.NoError(ms.DB.Update(&itemManualApproveDependent), "error updating item fixture for test")

	itemAutoApproveDependent.CoverageAmount = domain.Env.DependentAutoApproveMax - itemDependent.CoverageAmount - 1
	itemAutoApproveDependent.PolicyDependentID = nulls.NewUUID(dependent.ID)
	ms.NoError(ms.DB.Update(&itemAutoApproveDependent), "error updating item fixture for test")

	itemExceedsMax.CoverageAmount = domain.Env.PolicyMaxCoverage
	ms.NoError(ms.DB.Update(&itemExceedsMax), "error updating item fixture for test")

	itemStationaryMissingFields.CoverageAmount = 500
	itemStationaryMissingFields.Make = ""
	itemStationaryMissingFields.Model = ""
	ms.NoError(ms.DB.Update(&itemStationaryMissingFields), "error updating item fixture for test")

	itemMobileMissingMake.CoverageAmount = 500
	itemMobileMissingMake.Make = ""
	ms.NoError(ms.DB.Update(&itemMobileMissingMake), "error updating item fixture for test")

	itemMobileMissingModel.CoverageAmount = 500
	itemMobileMissingModel.Model = ""
	ms.NoError(ms.DB.Update(&itemMobileMissingModel), "error updating item fixture for test")

	corpFixtures := CreateItemFixtures(ms.DB, FixturesConfig{ItemsPerPolicy: 2})

	corpPolicy := corpFixtures.Policies[0]
	ConvertPolicyType(ms.DB, corpPolicy)
	corpItemAutoApprove := corpPolicy.Items[0]
	corpItemAutoApprove.LoadCategory(ms.DB, true)
	corpItemAutoApprove.CoverageAmount = corpItemAutoApprove.Category.AutoApproveMax
	ms.NoError(ms.DB.Update(&corpItemAutoApprove))

	corpItemManualApprove := corpPolicy.Items[1]
	corpItemManualApprove.LoadCategory(ms.DB, true)
	corpItemManualApprove.CoverageAmount = corpItemManualApprove.Category.AutoApproveMax + 1
	ms.NoError(ms.DB.Update(&corpItemManualApprove))

	tests := []struct {
		name       string
		item       Item
		oldStatus  api.ItemCoverageStatus
		wantStatus api.ItemCoverageStatus
	}{
		{
			name:       "item without dependent gets auto approval",
			item:       itemAutoApprove,
			oldStatus:  itemAutoApprove.CoverageStatus,
			wantStatus: api.ItemCoverageStatusApproved,
		},
		{
			name:       "item requires manual approval",
			item:       itemManualApprove,
			oldStatus:  itemManualApprove.CoverageStatus,
			wantStatus: api.ItemCoverageStatusPending,
		},
		{
			name:       "item for dependent requires manual approval",
			item:       itemManualApproveDependent,
			oldStatus:  itemManualApproveDependent.CoverageStatus,
			wantStatus: api.ItemCoverageStatusPending,
		},
		{
			name:       "item for dependent gets auto approval",
			item:       itemAutoApproveDependent,
			oldStatus:  itemAutoApproveDependent.CoverageStatus,
			wantStatus: api.ItemCoverageStatusApproved,
		},
		{
			name:       "item coverage amount exceeds max",
			item:       itemExceedsMax,
			oldStatus:  itemExceedsMax.CoverageStatus,
			wantStatus: api.ItemCoverageStatusPending,
		},
		{
			name:       "item missing fields but stationary",
			item:       itemStationaryMissingFields,
			oldStatus:  itemStationaryMissingFields.CoverageStatus,
			wantStatus: api.ItemCoverageStatusApproved,
		},
		{
			name:       "mobile item missing make",
			item:       itemMobileMissingMake,
			oldStatus:  itemMobileMissingMake.CoverageStatus,
			wantStatus: api.ItemCoverageStatusPending,
		},
		{
			name:       "mobile item missing model",
			item:       itemMobileMissingModel,
			oldStatus:  itemMobileMissingModel.CoverageStatus,
			wantStatus: api.ItemCoverageStatusPending,
		},
		{
			name:       "team policy, auto approval",
			item:       corpItemAutoApprove,
			oldStatus:  corpItemAutoApprove.CoverageStatus,
			wantStatus: api.ItemCoverageStatusApproved,
		},
		{
			name:       "team policy, manual approval",
			item:       corpItemManualApprove,
			oldStatus:  corpItemManualApprove.CoverageStatus,
			wantStatus: api.ItemCoverageStatusPending,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.item.LoadPolicy(ms.DB, false)
			tt.item.Policy.LoadMembers(ms.DB, false)
			ctxUser := policy.Members[0]
			ctx := CreateTestContext(ctxUser)

			got := tt.item.SubmitForApproval(ctx)
			ms.NoError(got)

			ms.Equal(tt.wantStatus, tt.item.CoverageStatus, "incorrect status")

			var gotH PolicyHistory
			ms.NoError(ms.DB.Last(&gotH), "error fetching PolicyHistory from db")
			wantH := PolicyHistory{
				ID:        gotH.ID, // Not concerned about testing auto-generated fields
				PolicyID:  tt.item.PolicyID,
				UserID:    ctxUser.ID,
				ItemID:    nulls.NewUUID(tt.item.ID),
				Action:    api.HistoryActionUpdate,
				FieldName: FieldItemCoverageStatus,
				OldValue:  string(tt.oldStatus),
				NewValue:  string(tt.wantStatus),
				CreatedAt: gotH.CreatedAt,
				UpdatedAt: gotH.UpdatedAt,
			}

			ms.Equal(wantH, gotH, "incorrect policy history")
		})
	}
}

func (ms *ModelSuite) TestItem_SafeDeleteOrInactivate() {
	t := ms.T()

	fixConfig := FixturesConfig{
		NumberOfPolicies: 1,
		ItemsPerPolicy:   9,
	}

	fixtures := CreateItemFixtures(ms.DB, fixConfig)
	policy := fixtures.Policies[0]
	policy.LoadItems(ms.DB, false)
	user := policy.Members[0]
	items := policy.Items

	ctx := CreateTestContext(user)

	now := time.Now().UTC()
	oldTime := now.Add(time.Hour * time.Duration(domain.ItemDeleteCutOffHours+1) * -1)

	items[0].CreatedAt = oldTime
	oldDraftItem := UpdateItemStatus(ms.DB, items[0], api.ItemCoverageStatusDraft, "")

	items[1].CreatedAt = oldTime
	oldPendingItem := UpdateItemStatus(ms.DB, items[1], api.ItemCoverageStatusDraft, "")

	items[2].CreatedAt = oldTime
	oldRevisionItem := UpdateItemStatus(ms.DB, items[2], api.ItemCoverageStatusDraft, "")

	newDraftItem := UpdateItemStatus(ms.DB, items[3], api.ItemCoverageStatusDraft, "")

	newApprovedItem := items[4]
	ms.NoError(newApprovedItem.Approve(ctx, now))
	ms.Greater(newApprovedItem.PaidThroughDate, domain.ZeroDate(),
		"Approved item didn't get a paid_through_date value")

	newPendingItem := UpdateItemStatus(ms.DB, items[5], api.ItemCoverageStatusPending, "")

	newRevisionItem := UpdateItemStatus(ms.DB, items[6], api.ItemCoverageStatusRevision, "Just do it")
	newInactiveItem := UpdateItemStatus(ms.DB, items[7], api.ItemCoverageStatusInactive, "")
	newDeniedItem := UpdateItemStatus(ms.DB, items[8], api.ItemCoverageStatusDenied, "")

	tests := []struct {
		name        string
		item        Item
		wantDeleted bool
		wantStatus  api.ItemCoverageStatus
	}{
		{
			name:        "old draft item inactivate",
			item:        oldDraftItem,
			wantDeleted: false,
			wantStatus:  api.ItemCoverageStatusInactive,
		},
		{
			name:        "old pending item inactivate",
			item:        oldPendingItem,
			wantDeleted: false,
			wantStatus:  api.ItemCoverageStatusInactive,
		},
		{
			name:        "old revision item inactivate",
			item:        oldRevisionItem,
			wantDeleted: false,
			wantStatus:  api.ItemCoverageStatusInactive,
		},
		{
			name:        "new draft item",
			item:        newDraftItem,
			wantDeleted: true,
		},
		{
			name:       "new approved item",
			item:       newApprovedItem,
			wantStatus: api.ItemCoverageStatusApproved,
		},
		{
			name:        "new pending item",
			item:        newPendingItem,
			wantDeleted: true,
		},
		{
			name:        "new revision item",
			item:        newRevisionItem,
			wantDeleted: true,
		},
		{
			name:        "new inactive item",
			item:        newInactiveItem,
			wantDeleted: false,
			wantStatus:  api.ItemCoverageStatusInactive,
		},
		{
			name:        "new denied item",
			item:        newDeniedItem,
			wantDeleted: false,
			wantStatus:  api.ItemCoverageStatusDenied,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.item.SafeDeleteOrInactivate(ctx, now)
			ms.NoError(got)

			dbItem := Item{}
			err := ms.DB.Find(&dbItem, tt.item.ID)

			if tt.wantDeleted {
				ms.Error(err, `expected a No Rows error`)
				ms.False(domain.IsOtherThanNoRows(err), `expected a No Rows error`)
				return
			}
			ms.NoError(err, "error finding the item in the database")
			ms.Equal(tt.wantStatus, dbItem.CoverageStatus, "incorrect status")
		})
	}
}

func (ms *ModelSuite) TestItem_Approve() {
	f := CreateItemFixtures(ms.DB, FixturesConfig{ItemsPerPolicy: 3})

	annual := f.Items[0]
	f.ItemCategories[0].BillingPeriod = domain.BillingPeriodAnnual
	monthlyEarly := f.Items[1]
	f.ItemCategories[1].BillingPeriod = domain.BillingPeriodMonthly
	monthlyLate := f.Items[2]
	f.ItemCategories[2].BillingPeriod = domain.BillingPeriodMonthly
	Must(ms.DB.Update(&f.ItemCategories))

	sep1 := time.Date(2023, 9, 1, 0, 0, 0, 0, time.UTC)
	sep30 := time.Date(2023, 9, 30, 0, 0, 0, 0, time.UTC)

	testContext := CreateTestContext(f.Users[0])

	tests := []struct {
		name          string
		item          Item
		now           time.Time
		wantAmount    api.Currency
		wantStartDate time.Time
		wantEndDate   time.Time
	}{
		{
			name:          "annual item",
			item:          annual,
			now:           sep30,
			wantAmount:    annual.CalculateProratedPremium(ms.DB, sep30),
			wantStartDate: sep30,
			wantEndDate:   domain.EndOfYear(sep30.Year()),
		},
		{
			name:          "monthlyEarly item",
			item:          monthlyEarly,
			now:           sep1,
			wantAmount:    monthlyEarly.CalculateMonthlyPremium(ms.DB),
			wantStartDate: sep1,
			wantEndDate:   sep30,
		},
		{
			name:          "monthlyLate item",
			item:          monthlyLate,
			now:           sep30,
			wantAmount:    0,
			wantStartDate: sep30,
			wantEndDate:   sep30,
		},
	}

	for _, tt := range tests {
		ms.T().Run(tt.name, func(t *testing.T) {
			eventDetected := false
			deleteFn, err := RegisterEventDetector(domain.EventApiItemApproved, &eventDetected)
			ms.NoError(err)
			defer deleteFn()

			ms.NoError(tt.item.Approve(testContext, tt.now))

			var dbItem Item
			ms.NoError(dbItem.FindByID(ms.DB, tt.item.ID))
			ms.Equal(api.ItemCoverageStatusApproved, dbItem.CoverageStatus)
			ms.Contains(dbItem.StatusChange, ItemStatusChangeApproved)
			ms.Equal(tt.wantStartDate, dbItem.CoverageStartDate)
			ms.Equal(tt.wantEndDate, dbItem.PaidThroughDate)

			var le LedgerEntry
			ms.NoError(ms.DB.Where("item_id = ?", tt.item.ID).Order("created_at DESC").First(&le))
			ms.Equal(tt.wantAmount, -le.Amount)

			time.Sleep(time.Millisecond * 10)
			ms.True(eventDetected)
		})
	}
}

func (ms *ModelSuite) TestItem_getInitialCoverage() {
	f := CreateItemFixtures(ms.DB, FixturesConfig{ItemsPerPolicy: 3})

	annual := f.Items[0]
	f.ItemCategories[0].BillingPeriod = domain.BillingPeriodAnnual
	monthlyEarly := f.Items[1]
	f.ItemCategories[1].BillingPeriod = domain.BillingPeriodMonthly
	monthlyLate := f.Items[2]
	f.ItemCategories[2].BillingPeriod = domain.BillingPeriodMonthly
	Must(ms.DB.Update(&f.ItemCategories))

	sep19 := time.Date(2023, 9, 19, 0, 0, 0, 0, time.UTC)
	sep20 := time.Date(2023, 9, 20, 0, 0, 0, 0, time.UTC)
	sep30 := time.Date(2023, 9, 30, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name string
		item Item
		now  time.Time
		want CoveragePeriod
	}{
		{
			name: "annual item",
			item: annual,
			now:  sep30,
			want: CoveragePeriod{
				Premium:   annual.CalculateProratedPremium(ms.DB, sep30),
				StartDate: sep30,
				EndDate:   domain.EndOfYear(sep30.Year()),
			},
		},
		{
			name: "monthlyEarly item",
			item: monthlyEarly,
			now:  sep19,
			want: CoveragePeriod{
				Premium:   monthlyEarly.CalculateMonthlyPremium(ms.DB),
				StartDate: sep19,
				EndDate:   sep30,
			},
		},
		{
			name: "monthlyLate item",
			item: monthlyLate,
			now:  sep20,
			want: CoveragePeriod{
				Premium:   0,
				StartDate: sep20,
				EndDate:   sep30,
			},
		},
	}

	for _, tt := range tests {
		ms.T().Run(tt.name, func(t *testing.T) {
			got := tt.item.getInitialCoverage(ms.DB, tt.now)
			ms.Equal(tt.want, got)
		})
	}
}

func (ms *ModelSuite) TestItem_CreateCancellationCredit() {
	f := CreateItemFixtures(ms.DB, FixturesConfig{ItemsPerPolicy: 3})

	now := time.Now().UTC()

	noRefund := f.Items[0]
	noRefund.PaidThroughDate = domain.ZeroDate()
	Must(ms.DB.Update(&noRefund))

	refund := f.Items[1]
	refund.PaidThroughDate = domain.EndOfYear(now.Year())
	Must(ms.DB.Update(&refund))

	monthlyItem := f.Items[2]
	monthlyItem.PaidThroughDate = domain.EndOfMonth(now)
	Must(ms.DB.Update(&monthlyItem))
	f.ItemCategories[2].BillingPeriod = domain.BillingPeriodMonthly
	f.ItemCategories[2].RiskCategoryID = riskCategoryVehicleID
	Must(ms.DB.Update(&f.ItemCategories[2]))

	tests := []struct {
		name              string
		item              Item
		wantLedgerEntries int
	}{
		{
			name:              "annual item, no refund",
			item:              noRefund,
			wantLedgerEntries: 0,
		},
		{
			name:              "annual item, create refund",
			item:              refund,
			wantLedgerEntries: 1,
		},
		{
			name:              "monthly item",
			item:              monthlyItem,
			wantLedgerEntries: 0,
		},
	}

	for _, tt := range tests {
		ms.T().Run(tt.name, func(t *testing.T) {
			err := tt.item.CreateCancellationCredit(ms.DB, now)
			ms.NoError(err)

			var le LedgerEntries
			Must(ms.DB.Where("item_id = ?", tt.item.ID).All(&le))
			ms.Equal(tt.wantLedgerEntries, len(le))
		})
	}
}

func (ms *ModelSuite) TestItem_InactivateApprovedButEnded() {
	fixConfig := FixturesConfig{
		NumberOfPolicies: 1,
		ItemsPerPolicy:   9,
	}

	fixtures := CreateItemFixtures(ms.DB, fixConfig)
	policy := fixtures.Policies[0]
	policy.LoadItems(ms.DB, false)
	items := policy.Items

	now := time.Now().UTC()
	oldTime := now.Add(time.Hour * -25)
	futureTime := now.Add(time.Hour)

	items[0].CoverageEndDate = nulls.NewTime(oldTime)
	pastDue := UpdateItemStatus(ms.DB, items[0], api.ItemCoverageStatusApproved, "")

	items[1].CoverageEndDate = nulls.NewTime(oldTime)
	pastButInactive := UpdateItemStatus(ms.DB, items[1], api.ItemCoverageStatusInactive, "")

	items[2].CoverageEndDate = nulls.NewTime(futureTime)
	notDue := UpdateItemStatus(ms.DB, items[2], api.ItemCoverageStatusApproved, "")

	newDraftItem := UpdateItemStatus(ms.DB, items[3], api.ItemCoverageStatusDraft, "")

	var i Items
	ctx := CreateTestContext(fixtures.Users[0])
	ms.NoError(i.InactivateApprovedButEnded(ctx))

	ms.NoError(pastDue.FindByID(ms.DB, pastDue.ID), "error fetching pastDue item")
	ms.Equal(pastDue.CoverageStatus, api.ItemCoverageStatusInactive, "incorrect status for past Due")

	ms.NoError(pastButInactive.FindByID(ms.DB, pastButInactive.ID), "error fetching pastButInactive item")
	ms.Equal(pastButInactive.CoverageStatus, api.ItemCoverageStatusInactive, "incorrect status for pastButInactive")

	ms.NoError(notDue.FindByID(ms.DB, notDue.ID), "error fetching notDue item")
	ms.Equal(notDue.CoverageStatus, api.ItemCoverageStatusApproved, "incorrect status for notDue")

	ms.NoError(newDraftItem.FindByID(ms.DB, newDraftItem.ID), "error fetching newDraftItem item")
	ms.Equal(newDraftItem.CoverageStatus, api.ItemCoverageStatusDraft, "incorrect status for newDraftItem")
}

func (ms *ModelSuite) TestItem_LoadPolicyMembers() {
	fixConfig := FixturesConfig{
		NumberOfPolicies: 2,
		UsersPerPolicy:   2,
		ItemsPerPolicy:   1,
	}

	fixtures := CreateItemFixtures(ms.DB, fixConfig)
	policy := fixtures.Policies[0]
	members := policy.Members

	item := Item{ID: policy.Items[0].ID, PolicyID: policy.ID}

	item.LoadPolicyMembers(ms.DB, true)
	ms.NotEqual(uuid.Nil, item.Policy.ID, "didn't load item policy")
	ms.Len(item.Policy.Members, 2, "didn't load correct number of policy members")
	ms.Equal(members[0].ID, item.Policy.Members[0].ID, "incorrect member0 ID")
	ms.Equal(members[1].ID, item.Policy.Members[1].ID, "incorrect member1 ID")
}

func (ms *ModelSuite) TestItem_setAccountablePerson() {
	config := FixturesConfig{
		NumberOfPolicies:    2,
		DependentsPerPolicy: 1,
	}
	fixtures := CreateItemFixtures(ms.DB, config)
	policyUser := fixtures.Policies[0].Members[0]
	policyDependent := fixtures.Policies[0].Dependents[0]
	otherUser := fixtures.Policies[1].Members[0]
	otherDependent := fixtures.Policies[1].Dependents[0]

	tests := []struct {
		name     string
		item     Item
		id       uuid.UUID
		appError *api.AppError
	}{
		{
			name: "policy user",
			item: fixtures.Items[0],
			id:   policyUser.ID,
		},
		{
			name:     "other user",
			item:     fixtures.Items[0],
			id:       otherUser.ID,
			appError: &api.AppError{Key: api.ErrorNoRows, Category: api.CategoryUser},
		},
		{
			name: "policy dependent",
			item: fixtures.Items[0],
			id:   policyDependent.ID,
		},
		{
			name:     "other dependent",
			item:     fixtures.Items[0],
			id:       otherDependent.ID,
			appError: &api.AppError{Key: api.ErrorNoRows, Category: api.CategoryUser},
		},
	}

	for _, tt := range tests {
		ms.T().Run(tt.name, func(t *testing.T) {
			err := tt.item.SetAccountablePerson(ms.DB, tt.id)
			if tt.appError != nil {
				ms.Error(err, "test should have produced an error")
				ms.EqualAppError(*tt.appError, err)
				return
			}
			ms.NoError(err)

			if tt.item.PolicyUserID.Valid {
				ms.Equal(tt.id, tt.item.PolicyUserID.UUID)
			} else if tt.item.PolicyDependentID.Valid {
				ms.Equal(tt.id, tt.item.PolicyDependentID.UUID)
			} else {
				ms.Fail("neither PolicyUserID nor PolicyDependentID are valid")
			}
		})
	}
}

func (ms *ModelSuite) TestItem_calculateAnnualPremium() {
	domain.Env.PremiumFactor = 0.02

	tests := []struct {
		name     string
		coverage int
		want     int
	}{
		{
			name:     "even amount",
			coverage: 200000,
			want:     4000,
		},
		{
			name:     "round up",
			coverage: 199999,
			want:     4000,
		},
	}
	for _, tt := range tests {
		ms.T().Run(tt.name, func(t *testing.T) {
			item := Item{CoverageAmount: tt.coverage}
			got := item.CalculateAnnualPremium(ms.DB)
			ms.Equal(api.Currency(tt.want), got)
		})
	}
}

func (ms *ModelSuite) TestItem_calculateProratedPremium() {
	domain.Env.PremiumFactor = 0.02

	now := time.Date(1999, 3, 15, 0, 0, 0, 0, time.UTC)

	item := Item{Category: ItemCategory{ID: domain.GetUUID(), MinimumPremium: 3000}}

	tests := []struct {
		name     string
		coverage int
		want     int
	}{
		{
			name:     "even amount",
			coverage: 200000,
			want:     3200,
		},
		{
			name:     "round up",
			coverage: 199999,
			want:     3200,
		},
		{
			name:     "lower than minimum",
			coverage: 187450,
			want:     3000,
		},
	}
	for _, tt := range tests {
		ms.T().Run(tt.name, func(t *testing.T) {
			item.CoverageAmount = tt.coverage
			got := item.CalculateProratedPremium(ms.DB, now)
			ms.Equal(api.Currency(tt.want), got)
		})
	}
}

func (ms *ModelSuite) TestItem_CalculateMonthlyPremium() {
	f := CreateItemFixtures(ms.DB, FixturesConfig{ItemsPerPolicy: 2})

	defaultPremium := f.Items[0]
	defaultPremium.CoverageAmount = 30000
	Must(ms.DB.Update(&defaultPremium))
	f.ItemCategories[0].PremiumFactor = nulls.NewFloat64(0.02)
	Must(ms.DB.Update(&f.ItemCategories[0]))

	categoryPremium := f.Items[1]
	categoryPremium.CoverageAmount = 30000
	Must(ms.DB.Update(&categoryPremium))
	f.ItemCategories[1].PremiumFactor = nulls.NewFloat64(0.03)
	Must(ms.DB.Update(&f.ItemCategories[1]))

	tests := []struct {
		name string
		item Item
		want api.Currency
	}{
		{
			name: "default premium",
			item: defaultPremium,
			want: 50,
		},
		{
			name: "category premium",
			item: categoryPremium,
			want: 75,
		},
	}
	for _, tt := range tests {
		ms.T().Run(tt.name, func(t *testing.T) {
			got := tt.item.CalculateMonthlyPremium(ms.DB)
			ms.Equal(tt.want, got)
		})
	}
}

func (ms *ModelSuite) TestItem_calculateCancellationCredit() {
	domain.Env.PremiumFactor = 0.02
	earlyJanuary := time.Date(2019, 1, 1, 1, 1, 1, 1, time.UTC)
	midJanuary := time.Date(2019, 1, 11, 1, 1, 1, 1, time.UTC)

	tests := []struct {
		name              string
		coverage          int
		coverageStartDate time.Time
		testTime          time.Time
		want              int
	}{
		{
			name:              "Now January and created last year",
			coverage:          600000, //  600,000 * 0.02 == 12,000 ,
			coverageStartDate: time.Date(2018, 12, 1, 1, 1, 1, 1, time.UTC),
			testTime:          midJanuary,
			want:              -12000,
		},
		{
			name: "Now January and created same year",
			//  Premium = 600,000 * 0.02 = 12,000
			//  Monthly premium = 12,000 / 12 = 1,000
			//  11 months credit = 11,000
			coverage:          600000,
			coverageStartDate: time.Date(2019, 1, 5, 1, 1, 1, 1, time.UTC),
			testTime:          midJanuary,
			want:              -11000,
		},
		{
			name: "Now February same year",
			//  Premium = 600,000 * 0.02 = 12,000
			//  Monthly premium = 12,000 / 12 = 1,000
			//  10 months credit = 10,000
			coverage:          600000,
			coverageStartDate: midJanuary,
			testTime:          time.Date(2019, 2, 1, 1, 1, 1, 1, time.UTC),
			want:              -10000,
		},
		{
			name:              "November Round Up", // one month's credit
			coverage:          599990,              // slightly lower than previous test case (then one month's credit)
			coverageStartDate: midJanuary,
			testTime:          time.Date(2019, 11, 1, 1, 1, 1, 1, time.UTC),
			want:              -1000,
		},
		{
			name:              "Now December same year",
			coverage:          6000,
			coverageStartDate: earlyJanuary,
			testTime:          time.Date(2019, 12, 1, 1, 1, 1, 1, time.UTC),
			want:              0,
		},
	}
	for _, tt := range tests {
		ms.T().Run(tt.name, func(t *testing.T) {
			item := Item{
				CoverageAmount:    tt.coverage,
				CoverageStartDate: tt.coverageStartDate,
			}
			got := item.calculateCancellationCredit(ms.DB, tt.testTime)
			ms.Equal(api.Currency(tt.want), got)
		})
	}
}

func (ms *ModelSuite) TestItem_calculatePremiumChange() {
	// 10 days remaining in the year
	now := time.Date(1999, 12, 22, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name       string
		newPremium api.Currency
		oldPremium api.Currency
		want       int
	}{
		{
			name:       "no change",
			newPremium: 200000,
			oldPremium: 200000,
			want:       0,
		},
		{
			name:       "increased",
			newPremium: 730000,
			oldPremium: 365000, // difference of 1,000 per day
			want:       10000,  // times 10 days
		},
		{
			name:       "decreased",
			newPremium: 365000,
			oldPremium: 730000,
			want:       -10000,
		},
	}
	for _, tt := range tests {
		ms.T().Run(tt.name, func(t *testing.T) {
			item := Item{}
			got := item.calculatePremiumChange(now, tt.oldPremium, tt.newPremium)
			ms.Equal(api.Currency(tt.want), got)
		})
	}
}

func (ms *ModelSuite) TestItem_CreateLedgerEntry() {
	// create a separate item for each test, to make it easy to find the right entry
	f := CreateItemFixtures(ms.DB, FixturesConfig{ItemsPerPolicy: 3})

	user := f.Users[0]
	ctx := CreateTestContext(user)

	for _, item := range f.Items {
		ms.NoError(item.SetAccountablePerson(ms.DB, f.Users[0].ID))
		ms.NoError(item.Update(ctx))
	}

	now := time.Now().UTC()

	tests := []struct {
		name       string
		item       Item
		entryType  LedgerEntryType
		amount     api.Currency
		wantAmount api.Currency
	}{
		{
			name:       "new coverage, over $1",
			item:       f.Items[0],
			entryType:  LedgerEntryTypeNewCoverage,
			amount:     500,
			wantAmount: -500,
		},
		{
			name:       "refund, under $1",
			item:       f.Items[1],
			entryType:  LedgerEntryTypeCoverageRefund,
			amount:     -50,
			wantAmount: 0,
		},
		{
			name:       "renewal, over $1",
			item:       f.Items[2],
			entryType:  LedgerEntryTypeCoverageRenewal,
			amount:     300,
			wantAmount: -300,
		},
	}
	for _, tt := range tests {
		ms.T().Run(tt.name, func(t *testing.T) {
			err := tt.item.CreateLedgerEntry(ms.DB, tt.entryType, tt.amount, now)
			ms.NoError(err)

			var le LedgerEntry
			ms.NoError(ms.DB.Where("item_id = ?", tt.item.ID).First(&le))

			ms.Equal(tt.entryType, le.Type, "Type is incorrect")
			ms.Equal(tt.item.PolicyID, le.PolicyID, "PolicyID is incorrect")
			ms.Equal(tt.item.ID, le.ItemID.UUID, "ItemID is incorrect")
			ms.Equal(tt.wantAmount, le.Amount, "Amount is incorrect")
			ms.Equal(tt.item.GetAccountablePersonName(ms.DB).String(), le.Name, "Name is incorrect")

			ms.Equal(time.Now().UTC().Truncate(domain.DurationDay), le.DateSubmitted,
				"ledger entry submitted date should be the current time")
		})
	}
}

func (ms *ModelSuite) TestItem_SetPaidThroughDate() {
	f := CreateItemFixtures(ms.DB, FixturesConfig{ItemsPerPolicy: 3})
	yesterday := time.Now().UTC().AddDate(0, 0, -1).Truncate(24 * time.Hour)
	now := time.Now().UTC().Truncate(24 * time.Hour)

	zeroDateItem := f.Items[0]
	zeroDateItem.PaidThroughDate = domain.ZeroDate()
	Must(ms.DB.Update(&zeroDateItem))

	yesterdayItem := f.Items[1]
	yesterdayItem.PaidThroughDate = yesterday
	Must(ms.DB.Update(&yesterdayItem))

	tests := []struct {
		name     string
		item     Item
		date     time.Time
		wantDate time.Time
	}{
		{
			name:     "starting as ZeroDate, no change",
			item:     zeroDateItem,
			date:     domain.ZeroDate(),
			wantDate: domain.ZeroDate(),
		},
		{
			name:     "starting as yesterday, no change",
			item:     yesterdayItem,
			date:     yesterday,
			wantDate: yesterday,
		},
		{
			name:     "starting as ZeroDate, changing to now",
			item:     zeroDateItem,
			date:     now,
			wantDate: now,
		},
		{
			name:     "starting as yesterday, changing to now",
			item:     yesterdayItem,
			date:     now,
			wantDate: now,
		},
	}
	for _, tt := range tests {
		ms.T().Run(tt.name, func(t *testing.T) {
			err := tt.item.SetPaidThroughDate(ms.DB, tt.date)
			ms.NoError(err)

			ms.Equal(tt.wantDate, tt.item.PaidThroughDate)

			var updatedItem Item
			ms.NoError(ms.DB.Find(&updatedItem, tt.item.ID))
			ms.Equal(tt.wantDate, updatedItem.PaidThroughDate)
		})
	}
}

func (ms *ModelSuite) TestItem_GetAccountablePersonName() {
	f := CreateItemFixtures(ms.DB, FixturesConfig{ItemsPerPolicy: 2, DependentsPerPolicy: 1})
	item0 := f.Items[0]
	ms.NoError(item0.SetAccountablePerson(ms.DB, f.Users[0].ID))
	name := item0.GetAccountablePersonName(ms.DB)
	ms.Equal(f.Users[0].FirstName, name.First, "first name is not correct")
	ms.Equal(f.Users[0].LastName, name.Last, "last name is not correct")

	item1 := f.Items[1]
	ms.NoError(item1.SetAccountablePerson(ms.DB, f.PolicyDependents[0].ID))
	name = item1.GetAccountablePersonName(ms.DB)
	ms.Contains(f.PolicyDependents[0].Name, name.First, "first name is not correct")
	ms.Contains(f.PolicyDependents[0].Name, name.Last, "last name is not correct")
}

func (ms *ModelSuite) TestItem_GetAccountableMember() {
	f := CreateItemFixtures(ms.DB, FixturesConfig{UsersPerPolicy: 2, ItemsPerPolicy: 2, DependentsPerPolicy: 1})
	item0 := f.Policies[0].Items[0]
	item1 := f.Policies[0].Items[1]
	member0 := f.Policies[0].Members[0]
	member1 := f.Policies[0].Members[1]

	// item0 has the second member as the accountable person
	ms.NoError(item0.SetAccountablePerson(ms.DB, member1.ID))

	person := item0.GetAccountableMember(ms.DB)
	ms.Equal(member1.FirstName, person.GetName().First, "first name is not correct")
	ms.Equal(member1.LastName, person.GetName().Last, "last name is not correct")

	// item1 has no accountable person
	person = item1.GetAccountableMember(ms.DB)
	ms.Equal(member0.FirstName, person.GetName().First, "first name is not correct")
	ms.Equal(member0.LastName, person.GetName().Last, "last name is not correct")
}

func (ms *ModelSuite) Test_ItemsWithRecentStatusChanges() {
	fixtures := CreatePolicyHistoryFixtures_RecentItemStatusChanges(ms.DB)
	phFixes := fixtures.PolicyHistories

	gotRaw, gotErr := ItemsWithRecentStatusChanges(ms.DB)
	ms.NoError(gotErr)

	const tmFmt = "Jan _2 15:04:05.00"

	got := make([][2]string, len(gotRaw))
	for i, g := range gotRaw {
		got[i] = [2]string{g.Item.ID.String(), g.StatusUpdatedAt.Format(tmFmt)}
	}

	want := [][2]string{
		{phFixes[3].ItemID.UUID.String(), phFixes[3].UpdatedAt.Format(tmFmt)},
		{phFixes[7].ItemID.UUID.String(), phFixes[7].UpdatedAt.Format(tmFmt)},
	}

	ms.ElementsMatch(want, got, "incorrect results")
}

func (ms *ModelSuite) TestItem_Compare() {
	fixtures := CreateItemFixtures(ms.DB, FixturesConfig{})
	newItem := fixtures.Items[0]

	oldItem := Item{
		Name:              "OldName",
		CategoryID:        domain.GetUUID(),
		RiskCategoryID:    domain.GetUUID(),
		InStorage:         true,
		Country:           "CH",
		Description:       "OldDescription",
		PolicyDependentID: nulls.NewUUID(domain.GetUUID()),
		PolicyUserID:      nulls.NewUUID(domain.GetUUID()),
		Make:              "OldMake",
		Model:             "OldModel",
		SerialNumber:      "OldSerialNumber",
		CoverageAmount:    777,
		CoverageStatus:    api.ItemCoverageStatusRevision,
		CoverageStartDate: time.Date(1992, 2, 2, 0, 0, 0, 0, time.UTC),
		StatusReason:      "oldStatusReason",
	}

	tests := []struct {
		name string
		new  Item
		old  Item
		want []FieldUpdate
	}{
		{
			name: "single test case",
			new:  newItem,
			old:  oldItem,
			want: []FieldUpdate{
				{
					FieldName: FieldItemName,
					OldValue:  oldItem.Name,
					NewValue:  newItem.Name,
				},
				{
					FieldName: FieldItemCategoryID,
					OldValue:  oldItem.CategoryID.String(),
					NewValue:  newItem.CategoryID.String(),
				},
				{
					FieldName: FieldItemRiskCategoryID,
					OldValue:  oldItem.RiskCategoryID.String(),
					NewValue:  newItem.RiskCategoryID.String(),
				},
				{
					FieldName: FieldItemInStorage,
					OldValue:  fmt.Sprintf(`%t`, oldItem.InStorage),
					NewValue:  fmt.Sprintf(`%t`, newItem.InStorage),
				},
				{
					FieldName: FieldItemCountry,
					OldValue:  oldItem.Country,
					NewValue:  newItem.Country,
				},
				{
					FieldName: FieldItemDescription,
					OldValue:  oldItem.Description,
					NewValue:  newItem.Description,
				},
				{
					FieldName: FieldItemPolicyDependentID,
					OldValue:  oldItem.PolicyDependentID.UUID.String(),
					NewValue:  "NULL",
				},
				{
					FieldName: FieldItemPolicyUserID,
					OldValue:  oldItem.PolicyUserID.UUID.String(),
					NewValue:  "NULL",
				},
				{
					FieldName: FieldItemMake,
					OldValue:  oldItem.Make,
					NewValue:  newItem.Make,
				},
				{
					FieldName: FieldItemModel,
					OldValue:  oldItem.Model,
					NewValue:  newItem.Model,
				},
				{
					FieldName: FieldItemSerialNumber,
					OldValue:  oldItem.SerialNumber,
					NewValue:  newItem.SerialNumber,
				},
				{
					FieldName: FieldItemCoverageAmount,
					OldValue:  api.Currency(oldItem.CoverageAmount).String(),
					NewValue:  api.Currency(newItem.CoverageAmount).String(),
				},
				{
					FieldName: FieldItemCoverageStatus,
					OldValue:  string(oldItem.CoverageStatus),
					NewValue:  string(newItem.CoverageStatus),
				},
				{
					FieldName: FieldItemCoverageStartDate,
					OldValue:  oldItem.CoverageStartDate.Format(domain.DateFormat),
					NewValue:  newItem.CoverageStartDate.Format(domain.DateFormat),
				},
				{
					FieldName: FieldItemStatusReason,
					OldValue:  oldItem.StatusReason,
					NewValue:  newItem.StatusReason,
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

func (ms *ModelSuite) TestItem_canBeDeleted() {
	f := CreateItemFixtures(ms.DB, FixturesConfig{
		NumberOfPolicies: 3,
		ItemsPerPolicy:   1,
	})
	yes := f.Policies[0].Items[0]

	hasClaim := f.Policies[1].Items[0]
	createClaimFixture(ms.DB, f.Policies[1], FixturesConfig{ClaimItemsPerClaim: 1})

	hasLedgerEntry := f.Policies[2].Items[0]
	ms.NoError(hasLedgerEntry.Approve(CreateTestContext(f.Policies[2].Members[0]), time.Now().UTC()))

	tests := []struct {
		name string
		item Item
		want bool
	}{
		{
			name: "yes",
			item: yes,
			want: true,
		},
		{
			name: "has Claim",
			item: hasClaim,
			want: false,
		},
		{
			name: "has LedgerEntry",
			item: hasLedgerEntry,
			want: false,
		},
	}
	for _, tt := range tests {
		ms.T().Run(tt.name, func(t *testing.T) {
			got := tt.item.canBeDeleted(ms.DB)
			ms.Equal(tt.want, got, "item %s gave the wrong result", tt.item.ID)
		})
	}
}

func (ms *ModelSuite) TestItem_ConvertToAPI() {
	fixtures := CreateItemFixtures(ms.DB, FixturesConfig{DependentsPerPolicy: 1})
	item := fixtures.Items[0]
	item.CoverageEndDate = nulls.NewTime(time.Now().Add(domain.DurationDay * 365))
	ms.NoError(item.SetAccountablePerson(ms.DB, fixtures.PolicyDependents[0].ID))

	got := item.ConvertToAPI(ms.DB)

	ms.Equal(item.ID, got.ID, "ID is not correct")
	ms.Equal(item.Name, got.Name, "Name is not correct")
	ms.Equal(item.InStorage, got.InStorage, "InStorage is not correct")
	ms.Equal(item.Country, got.Country, "Country is not correct")
	ms.Equal(item.Description, got.Description, "Description is not correct")
	ms.Equal(item.PolicyID, got.PolicyID, "PolicyID is not correct")
	ms.Equal(item.Make, got.Make, "Make is not correct")
	ms.Equal(item.SerialNumber, got.SerialNumber, "SerialNumber is not correct")
	ms.Equal(item.Year, PointerToNullsInt(got.Year), "Year is not correct")
	ms.Equal(item.CoverageAmount, got.CoverageAmount, "CoverageAmount is not correct")
	ms.Equal(item.CoverageStatus, got.CoverageStatus, "CoverageStatus is not correct")
	ms.Equal(item.StatusChange, got.StatusChange, "StatusChange is not correct")
	ms.Equal(item.StatusReason, got.StatusReason, "StatusReason is not correct")
	ms.Equal(item.CoverageStartDate.Format(domain.DateFormat), got.CoverageStartDate,
		"CoverageStartDate is not correct")
	ms.Equal(item.CoverageEndDate.Time.Format(domain.DateFormat), *got.CoverageEndDate,
		"CoverageEndDate is not correct")
	ms.Equal(item.CreatedAt, got.CreatedAt, "CreatedAt is not correct")
	ms.Equal(item.UpdatedAt, got.UpdatedAt, "UpdatedAt is not correct")
	ms.Equal(item.Category.ConvertToAPI(ms.DB), got.Category, "Category is not correct")
	ms.Equal(item.RiskCategory.ConvertToAPI(), got.RiskCategory, "RiskCategory is not correct")
	ms.Equal(item.Category.BillingPeriod, got.BillingPeriod, "BillingPeriod is not correct")
	ms.Equal(item.CalculateMonthlyPremium(ms.DB), got.MonthlyPremium, "MonthlyPremium is not correct")
	ms.Equal(item.CalculateAnnualPremium(ms.DB), got.AnnualPremium, "AnnualPremium is not correct")
	ms.Equal(item.CalculateProratedPremium(ms.DB, time.Now().UTC()), got.ProratedAnnualPremium,
		"ProratedAnnualPremium is not correct")
	ms.Equal(item.PolicyDependentID.UUID, got.AccountablePerson.ID, "AccountablePerson ID is not correct")
	ms.Equal(fixtures.PolicyDependents[0].GetName().String(), got.AccountablePerson.Name,
		"AccountablePerson Name is not correct")
	ms.Equal(fixtures.PolicyDependents[0].GetLocation().Country, got.AccountablePerson.Country,
		"AccountablePerson Country is not correct")
	ms.True(got.CanBeUpdated, "CanBeUpdated is not correct")
	ms.True(got.CanBeDeleted, "CanBeDeleted is not correct")
}

func (ms *ModelSuite) TestItem_hasOpenClaim() {
	fixtures := CreateItemFixtures(ms.DB, FixturesConfig{ClaimsPerPolicy: 2, ClaimItemsPerClaim: 1})

	// both claims are on Policies[0].Items[0] since ClaimItemsPerClaim = 1
	safeItem := fixtures.Policies[0].Items[1]

	unsafeClaim := UpdateClaimStatus(ms.DB, fixtures.Claims[1], api.ClaimStatusReview1, "some reason")
	unsafeItem := unsafeClaim.ClaimItems[0].Item

	tests := []struct {
		name string
		item Item
		want bool
	}{
		{
			name: "true",
			item: unsafeItem,
			want: true,
		},
		{
			name: "false",
			item: safeItem,
			want: false,
		},
	}
	for _, tt := range tests {
		ms.T().Run(tt.name, func(t *testing.T) {
			got := tt.item.hasOpenClaim(ms.DB)

			ms.Equal(tt.want, got)
		})
	}
}

func (ms *ModelSuite) TestItem_Update() {
	// TODO: improve test coverage

	f := CreateItemFixtures(ms.DB, FixturesConfig{ItemsPerPolicy: 3})

	itemToDecreaseCoverage := UpdateItemStatus(ms.DB, f.Items[0], api.ItemCoverageStatusApproved, "test")
	itemToDecreaseCoverage.CoverageAmount -= 1
	itemToIncreaseCoverage := UpdateItemStatus(ms.DB, f.Items[1], api.ItemCoverageStatusPending, "test")
	itemToIncreaseCoverage.CoverageAmount += 1
	itemCannotIncreaseCoverage := UpdateItemStatus(ms.DB, f.Items[2], api.ItemCoverageStatusApproved, "test")
	itemCannotIncreaseCoverage.CoverageAmount += 1

	tests := []struct {
		name     string
		actor    User
		item     Item
		appError *api.AppError
	}{
		{
			name:     "decrease coverage",
			actor:    f.Users[0],
			item:     itemToDecreaseCoverage,
			appError: nil,
		},
		{
			name:     "increase coverage",
			actor:    f.Users[0],
			item:     itemToIncreaseCoverage,
			appError: nil,
		},
		{
			name:     "cannot increase coverage",
			actor:    f.Users[0],
			item:     itemCannotIncreaseCoverage,
			appError: &api.AppError{Key: api.ErrorItemCoverageAmountCannotIncrease, Category: api.CategoryUser},
		},
	}
	for _, tt := range tests {
		ms.T().Run(tt.name, func(t *testing.T) {
			err := tt.item.Update(CreateTestContext(tt.actor))
			if tt.appError != nil {
				ms.Error(err, "test should have produced an error")
				ms.EqualAppError(*tt.appError, err)
				return
			}
			ms.NoError(err)

			var dbItem Item
			ms.NoError(ms.DB.Find(&dbItem, tt.item.ID))
			ms.Equal(tt.item.CoverageAmount, dbItem.CoverageAmount, "CoverageAmount did not get updated")
		})
	}
}

func (ms *ModelSuite) TestItem_cancelCoverageAfterClaim() {
	f := CreateItemFixtures(ms.DB, FixturesConfig{ItemsPerPolicy: 3})

	itemApproved := UpdateItemStatus(ms.DB, f.Items[0], api.ItemCoverageStatusApproved, "test")
	itemPending := UpdateItemStatus(ms.DB, f.Items[1], api.ItemCoverageStatusPending, "test")

	now := time.Now().UTC()
	reason := "test claim approved"

	tests := []struct {
		name            string
		item            Item
		wantErrContains string
	}{
		{
			name:            "fail pending item",
			item:            itemPending,
			wantErrContains: "cannot cancel coverage on an item which is not approved",
		},
		{
			name:            "good approved item",
			item:            itemApproved,
			wantErrContains: "",
		},
	}
	for _, tt := range tests {
		ms.T().Run(tt.name, func(t *testing.T) {
			got := tt.item.cancelCoverageAfterClaim(ms.DB, reason)
			if tt.wantErrContains != "" {
				ms.Error(got, "expected an error but didn't get one")
				ms.Contains(got.Error(), tt.wantErrContains, "incorrect error")
				return
			}
			ms.NoError(got)

			var dbItem Item
			ms.NoError(ms.DB.Find(&dbItem, tt.item.ID))
			ms.Equal(tt.item.CoverageStatus, api.ItemCoverageStatusInactive, "incorrect CoverageStatus")

			var le LedgerEntry
			ms.NoError(ms.DB.Where("item_id = ?", tt.item.ID).First(&le))

			ms.Equal(LedgerEntryTypeCoverageRefund, le.Type, "LedgerEntry Type is incorrect")
			ms.Equal(tt.item.PolicyID, le.PolicyID, "LedgerEntry PolicyID is incorrect")
			ms.Equal(tt.item.ID, le.ItemID.UUID, "LedgerEntry ItemID is incorrect")

			var history PolicyHistory
			ms.NoError(ms.DB.Where("item_id = ?", tt.item.ID).First(&history))
			ms.Equal(ServiceUserID, history.UserID.String(), "History UserID is incorrect")
			ms.Equal(FieldItemCoverageStatus, history.FieldName, "History FieldName is incorrect")
			ms.Equal(api.HistoryActionUpdate, history.Action, "History Action is incorrect")

			ms.NoError(tt.item.FindByID(ms.DB, tt.item.ID), "failed retrieving item from db")
			ms.Equal(api.ItemCoverageStatusInactive, tt.item.CoverageStatus, "Item CoverageStatus is incorrect")
			ms.WithinDuration(now, tt.item.CoverageEndDate.Time, time.Hour*24, "Item CoverageEndDate is incorrect")
			ms.Equal(ItemStatusChangeInactivated+"having a claim approved", tt.item.StatusChange,
				"Item StatusChange is incorrect")
			ms.Equal(reason, tt.item.StatusReason, "Item StatusReason is incorrect")
		})
	}
}

func (ms *ModelSuite) TestItem_FindItemsIncorrectlyRenewed() {
	now := time.Now().UTC()
	thisYear := now.Year()
	lastYear := thisYear - 1
	aDateLastYear := time.Date(thisYear-1, 12, 31, 0, 0, 0, 0, time.UTC)
	aDateTwoYearsAgo := time.Date(thisYear-2, 12, 31, 0, 0, 0, 0, time.UTC)

	f := CreateItemFixtures(ms.DB, FixturesConfig{ItemsPerPolicy: 3})
	incorrectItem := f.Items[0]
	incorrectItem.PaidThroughDate = domain.EndOfYear(thisYear)
	incorrectItem.CoverageEndDate = nulls.NewTime(aDateLastYear)
	Must(ms.DB.Update(&incorrectItem))

	correctItem := f.Items[1]
	correctItem.PaidThroughDate = domain.EndOfYear(lastYear)
	correctItem.CoverageEndDate = nulls.NewTime(aDateLastYear)
	Must(ms.DB.Update(&correctItem))

	incorrectLastYear := f.Items[2]
	incorrectLastYear.PaidThroughDate = domain.EndOfYear(lastYear)
	incorrectLastYear.CoverageEndDate = nulls.NewTime(aDateTwoYearsAgo)
	Must(ms.DB.Update(&incorrectLastYear))

	tests := []struct {
		name        string
		date        time.Time
		wantItemIDs []uuid.UUID
	}{
		{
			name:        "this year has one incorrect item",
			date:        now,
			wantItemIDs: []uuid.UUID{incorrectItem.ID},
		},
		{
			name:        "last year has one incorrect item",
			date:        aDateLastYear,
			wantItemIDs: []uuid.UUID{incorrectLastYear.ID},
		},
		{
			name:        "no incorrect items",
			date:        aDateTwoYearsAgo,
			wantItemIDs: []uuid.UUID{},
		},
	}
	for _, tt := range tests {
		ms.T().Run(tt.name, func(t *testing.T) {
			var items Items
			err := items.FindItemsIncorrectlyRenewed(ms.DB, tt.date)
			ms.NoError(err)

			gotIDs := make([]uuid.UUID, len(items))
			for i := range items {
				gotIDs[i] = items[i].ID
			}
			ms.ElementsMatch(tt.wantItemIDs, gotIDs)
		})
	}
}

func (ms *ModelSuite) TestItem_RepairItemsIncorrectlyRenewed() {
	now := time.Now().UTC()
	thisYear := now.Year()
	lastYear := thisYear - 1
	aDateLastYear := time.Date(thisYear-1, 12, 31, 0, 0, 0, 0, time.UTC)
	aDateTwoYearsAgo := time.Date(thisYear-2, 12, 31, 0, 0, 0, 0, time.UTC)

	f := CreateItemFixtures(ms.DB, FixturesConfig{ItemsPerPolicy: 3})
	incorrectItem := f.Items[0]
	incorrectItem.PaidThroughDate = domain.EndOfYear(thisYear)
	incorrectItem.CoverageEndDate = nulls.NewTime(aDateLastYear)
	incorrectItem.CoverageAmount = 10000
	Must(ms.DB.Update(&incorrectItem))

	correctItem := f.Items[1]
	correctItem.PaidThroughDate = domain.EndOfYear(lastYear)
	correctItem.CoverageEndDate = nulls.NewTime(aDateLastYear)
	Must(ms.DB.Update(&correctItem))

	incorrectLastYear := f.Items[2]
	incorrectLastYear.PaidThroughDate = domain.EndOfYear(lastYear)
	incorrectLastYear.CoverageEndDate = nulls.NewTime(aDateTwoYearsAgo)
	incorrectLastYear.CoverageAmount = 20000
	Must(ms.DB.Update(&incorrectLastYear))

	actor := createAdminUserWithRole(ms.DB, AppRoleSteward)

	tests := []struct {
		name                string
		date                time.Time
		wantItemIDs         []uuid.UUID
		wantPaidThroughDate time.Time
		wantRefundAmount    api.Currency
	}{
		{
			name:                "this year has one incorrect item",
			date:                now,
			wantItemIDs:         []uuid.UUID{incorrectItem.ID},
			wantPaidThroughDate: domain.EndOfYear(lastYear),
			wantRefundAmount:    incorrectItem.CalculateAnnualPremium(ms.DB),
		},
		{
			name:                "last year has one incorrect item",
			date:                aDateLastYear,
			wantItemIDs:         []uuid.UUID{incorrectLastYear.ID},
			wantPaidThroughDate: domain.EndOfYear(lastYear - 1),
			wantRefundAmount:    incorrectLastYear.CalculateAnnualPremium(ms.DB),
		},
		{
			name:        "no incorrect items",
			date:        aDateTwoYearsAgo,
			wantItemIDs: []uuid.UUID{},
		},
	}
	for _, tt := range tests {
		ms.T().Run(tt.name, func(t *testing.T) {
			var items Items
			err := items.RepairItemsIncorrectlyRenewed(CreateTestContext(actor), tt.date)
			ms.NoError(err)

			gotIDs := make([]uuid.UUID, len(items))
			for i := range items {
				gotIDs[i] = items[i].ID

				ms.Equal(tt.wantPaidThroughDate, items[i].PaidThroughDate,
					"PaidThroughDate is incorrect on items[%d]", i)

				var fromDB Item
				Must(ms.DB.Find(&fromDB, items[i].ID))
				ms.Equal(items[i].PaidThroughDate, fromDB.PaidThroughDate, "item wasn't saved correctly")
			}
			ms.ElementsMatch(tt.wantItemIDs, gotIDs)

			if len(tt.wantItemIDs) > 0 {
				var le LedgerEntries
				Must(ms.DB.Where("item_id = ?", tt.wantItemIDs[0]).All(&le))
				ms.Len(le, 1, "expected only one ledger entry for item")
				ms.Equal(tt.wantRefundAmount, le[0].Amount, "refund amount is incorrect")
			}
		})
	}
}

func (ms *ModelSuite) Test_CountItemsToRenew() {
	now := time.Now().UTC()
	year := now.Year()
	lastMonth := now.AddDate(0, -1, 0)
	endOfLastMonth := domain.EndOfMonth(lastMonth)

	f1 := CreateItemFixtures(ms.DB, FixturesConfig{ItemsPerPolicy: 5})
	for i := range f1.Items {
		f1.Items[i].PaidThroughDate = domain.EndOfYear(year - 1)
		UpdateItemStatus(ms.DB, f1.Items[i], api.ItemCoverageStatusApproved, "")
	}

	f2 := CreateItemFixtures(ms.DB, FixturesConfig{ItemsPerPolicy: 3})
	for i := range f2.Items {
		f2.Items[i].PaidThroughDate = endOfLastMonth
		UpdateItemStatus(ms.DB, f2.Items[i], api.ItemCoverageStatusApproved, "")
		f2.ItemCategories[i].BillingPeriod = domain.BillingPeriodMonthly
		Must(ms.DB.Update(&f2.ItemCategories[i]))
	}

	tests := []struct {
		name          string
		date          time.Time
		billingPeriod int
		want          int
	}{
		{
			name:          "annual, last year, 0 items",
			date:          domain.EndOfYear(year - 1),
			want:          0,
			billingPeriod: domain.BillingPeriodAnnual,
		},
		{
			name:          "annual, this year, 5 items",
			date:          domain.EndOfYear(year),
			want:          5,
			billingPeriod: domain.BillingPeriodAnnual,
		},
		{
			name:          "monthly, last month, 0 items",
			date:          lastMonth,
			want:          0,
			billingPeriod: domain.BillingPeriodMonthly,
		},
		{
			name:          "monthly, this month, 3 items",
			date:          now,
			want:          3,
			billingPeriod: domain.BillingPeriodMonthly,
		},
	}
	for _, tt := range tests {
		ms.T().Run(tt.name, func(t *testing.T) {
			got, err := CountItemsToRenew(ms.DB, tt.date, tt.billingPeriod)
			ms.NoError(err)
			ms.Equal(tt.want, got)
		})
	}
}

func (ms *ModelSuite) TestItem_createPremiumAdjustment() {
	f := CreateItemFixtures(ms.DB, FixturesConfig{ItemsPerPolicy: 1})

	// March 15 is 20% into the year
	mar15 := time.Date(2023, 3, 15, 0, 0, 0, 0, time.UTC)

	f.ItemCategories[0].PremiumFactor = nulls.NewFloat64(0.02) // 2% premium factor
	f.ItemCategories[0].BillingPeriod = 12
	Must(ms.DB.Update(&f.ItemCategories[0]))

	oldItem := f.Items[0]
	oldItem.CoverageAmount = 1000 * domain.CurrencyFactor // $1000 before
	Must(ms.DB.Update(&oldItem))

	newItem := oldItem
	newItem.CoverageAmount = oldItem.CoverageAmount / 2 // $500 after
	Must(ms.DB.Update(&newItem))

	err := newItem.createPremiumAdjustment(ms.DB, mar15, oldItem)
	ms.NoError(err)

	var le LedgerEntries
	Must(ms.DB.Where("item_id = ?", newItem.ID).All(&le))
	ms.Len(le, 1)

	wantAmount := api.Currency(8 * domain.CurrencyFactor) // ($1000 - $500) * 2% * (1 - 20%) = $8
	ms.Equal(wantAmount, le[0].Amount)
}

func (ms *ModelSuite) TestItem_ScheduleInactivation() {
	june11 := time.Date(2023, 6, 11, 0, 0, 0, 0, time.UTC)
	jan3 := time.Date(2023, 1, 3, 0, 0, 0, 0, time.UTC)

	f := CreateItemFixtures(ms.DB, FixturesConfig{ItemsPerPolicy: 2})

	annual := f.Items[0]

	monthly := f.Items[1]
	f.ItemCategories[1].BillingPeriod = domain.BillingPeriodMonthly
	f.ItemCategories[1].RiskCategoryID = riskCategoryVehicleID
	Must(ms.DB.Update(&f.ItemCategories[1]))

	ctx := CreateTestContext(f.Users[0])

	tests := []struct {
		name                string
		item                Item
		now                 time.Time
		wantCoverageEndDate nulls.Time
	}{
		{
			name:                "annual item, mid-year",
			item:                annual,
			now:                 june11,
			wantCoverageEndDate: nulls.NewTime(domain.EndOfMonth(june11)),
		},
		{
			name:                "annual item, early in the year",
			item:                annual,
			now:                 jan3,
			wantCoverageEndDate: nulls.NewTime(jan3),
		},
		{
			name:                "monthly item",
			item:                monthly,
			now:                 jan3,
			wantCoverageEndDate: nulls.NewTime(domain.EndOfMonth(jan3)),
		},
	}
	for _, tt := range tests {
		ms.T().Run(tt.name, func(t *testing.T) {
			err := tt.item.ScheduleInactivation(ctx, tt.now)
			ms.NoError(err)

			var dbItem Item
			Must(ms.DB.Find(&dbItem, tt.item.ID))

			ms.Contains(dbItem.StatusChange, ItemStatusChangeInactivated)
			ms.Equal(tt.wantCoverageEndDate, dbItem.CoverageEndDate)
		})
	}
}
