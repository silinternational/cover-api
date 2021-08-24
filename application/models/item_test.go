package models

import (
	"testing"

	"github.com/silinternational/cover-api/api"
)

//isItemActionAllowed

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
