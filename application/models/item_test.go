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
			subRes:       ItemSubmit,
			want:         false,
		},
		{
			name:         "draft with create and wrong sub resource - NO",
			actorIsAdmin: false,
			startStatus:  api.ItemCoverageStatusDraft,
			permission:   PermissionCreate,
			subRes:       ItemApprove,
			want:         false,
		},
		{
			name:         "draft with create and submit sub resource - YES",
			actorIsAdmin: false,
			startStatus:  api.ItemCoverageStatusDraft,
			permission:   PermissionCreate,
			subRes:       ItemSubmit,
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
			subRes:       ItemSubmit,
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
			subRes:       ItemSubmit,
			want:         false,
		},
		{
			name:         "revision with create and wrong sub resource - NO",
			actorIsAdmin: false,
			startStatus:  api.ItemCoverageStatusRevision,
			permission:   PermissionCreate,
			subRes:       ItemApprove,
			want:         false,
		},
		{
			name:         "revision with create and submit sub resource - YES",
			actorIsAdmin: false,
			startStatus:  api.ItemCoverageStatusRevision,
			permission:   PermissionCreate,
			subRes:       ItemSubmit,
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
			subRes:       ItemSubmit,
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
			subRes:       ItemRevision,
			want:         true,
		},
		{
			name:         "pending with create and approve sub resource - YES",
			actorIsAdmin: true,
			startStatus:  api.ItemCoverageStatusPending,
			permission:   PermissionCreate,
			subRes:       ItemApprove,
			want:         true,
		},
		{
			name:         "pending with create and deny sub resource - YES",
			actorIsAdmin: true,
			startStatus:  api.ItemCoverageStatusPending,
			permission:   PermissionCreate,
			subRes:       ItemDeny,
			want:         true,
		},
		{
			name:         "pending with create and revision sub resource but non-admin - NO",
			actorIsAdmin: false,
			startStatus:  api.ItemCoverageStatusPending,
			permission:   PermissionCreate,
			subRes:       ItemRevision,
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
			subRes:       ItemSubmit,
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
			subRes:       ItemDeny,
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
			subRes:       ItemDeny,
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
			name:            "item exceeds policy max",
			item:            itemExceedsPolicy,
			wantErrContains: "pushes policy total over max allowed",
		},
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
		ItemsPerPolicy:      6,
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
	ms.NoError(ms.DB.Update(&itemDependent), "error updating item fixture for test")

	// specify other items for testing
	itemAutoApprove := fixtures.Policies[1].Items[0]
	itemManualApprove := items[2]
	itemManualApproveDependent := items[3]
	itemAutoApproveDependent := items[4]
	itemExceedsMax := items[5]

	// set them all to Draft status
	itemAutoApprove.CoverageStatus = api.ItemCoverageStatusDraft
	itemManualApprove.CoverageStatus = api.ItemCoverageStatusDraft
	itemManualApproveDependent.CoverageStatus = api.ItemCoverageStatusDraft
	itemAutoApproveDependent.CoverageStatus = api.ItemCoverageStatusDraft
	itemExceedsMax.CoverageStatus = api.ItemCoverageStatusDraft

	// set there coverage amounts to be helpful for the tests and set the dependent as needed
	itemAutoApprove.LoadCategory(ms.DB, false)
	itemAutoApprove.CoverageAmount = itemAutoApprove.Category.AutoApproveMax - 1
	ms.NoError(ms.DB.Update(&itemAutoApprove), "error updating item fixture for test")

	itemManualApprove.LoadCategory(ms.DB, false)
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

	tests := []struct {
		name            string
		item            Item
		wantStatus      api.ItemCoverageStatus
		wantErrContains string
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
			name:            "item coverage amount exceeds max",
			item:            itemExceedsMax,
			wantErrContains: "pushes policy total over max allowed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.item.SubmitForApproval(ms.DB)

			if tt.wantErrContains != "" {
				ms.Error(got)
				ms.Contains(got.Error(), tt.wantErrContains, "incorrect error")
				return
			}

			ms.NoError(got)

			ms.Equal(tt.wantStatus, tt.item.CoverageStatus, "incorrect status")
		})
	}
}
