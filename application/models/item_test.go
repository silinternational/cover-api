package models

import (
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
			name:         "approved with update and no sub resource - NO",
			actorIsAdmin: true,
			startStatus:  api.ItemCoverageStatusApproved,
			permission:   PermissionUpdate,
			subRes:       "",
			want:         false,
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
		PurchaseDate:      time.Now().UTC().Add(time.Hour * -48),
		CoverageStartDate: time.Now().UTC().Add(time.Hour * 48),
	}
	itemExceedsPolicy := goodItem
	itemExceedsPolicy.CoverageAmount = domain.Env.PolicyMaxCoverage - coverageForPolicy + 1

	tests := []struct {
		name            string
		item            Item
		wantErrContains string
	}{
		{
			name: "good item",
			item: goodItem,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.item.Create(ms.DB)

			if tt.wantErrContains != "" {
				ms.Error(got)
				ms.Contains(got.Error(), tt.wantErrContains)
				return
			}
			ms.NoError(got)

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
	items := policy.Items
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
	itemAutoApprove.Load(ms.DB)
	itemAutoApprove.CoverageAmount = itemAutoApprove.Category.AutoApproveMax - 1
	ms.NoError(ms.DB.Update(&itemAutoApprove), "error updating item fixture for test")

	itemManualApprove.Load(ms.DB)
	itemManualApprove.CoverageAmount = itemManualApprove.Category.AutoApproveMax + 1
	ms.NoError(ms.DB.Update(&itemManualApprove), "error updating item fixture for test")

	itemManualApproveDependent.CoverageAmount = domain.Env.DependantAutoApproveMax - itemDependent.CoverageAmount + 1
	itemManualApproveDependent.PolicyDependentID = nulls.NewUUID(dependent.ID)
	ms.NoError(ms.DB.Update(&itemManualApproveDependent), "error updating item fixture for test")

	itemAutoApproveDependent.CoverageAmount = domain.Env.DependantAutoApproveMax - itemDependent.CoverageAmount - 1
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

	tests := []struct {
		name       string
		item       Item
		wantStatus api.ItemCoverageStatus
	}{
		{
			name:       "item without dependent gets auto approval",
			item:       itemAutoApprove,
			wantStatus: api.ItemCoverageStatusApproved,
		},
		{
			name:       "item requires manual approval",
			item:       itemManualApprove,
			wantStatus: api.ItemCoverageStatusPending,
		},
		{
			name:       "item for dependent requires manual approval",
			item:       itemManualApproveDependent,
			wantStatus: api.ItemCoverageStatusPending,
		},
		{
			name:       "item for dependent gets auto approval",
			item:       itemAutoApproveDependent,
			wantStatus: api.ItemCoverageStatusApproved,
		},
		{
			name:       "item coverage amount exceeds max",
			item:       itemExceedsMax,
			wantStatus: api.ItemCoverageStatusPending,
		},
		{
			name:       "item missing fields but stationary",
			item:       itemStationaryMissingFields,
			wantStatus: api.ItemCoverageStatusApproved,
		},
		{
			name:       "mobile item missing make",
			item:       itemMobileMissingMake,
			wantStatus: api.ItemCoverageStatusPending,
		},
		{
			name:       "mobile item missing model",
			item:       itemMobileMissingModel,
			wantStatus: api.ItemCoverageStatusPending,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.item.SubmitForApproval(ms.DB)
			ms.NoError(got)

			ms.Equal(tt.wantStatus, tt.item.CoverageStatus, "incorrect status")
		})
	}
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
			err := tt.item.setAccountablePerson(ms.DB, tt.id)
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
