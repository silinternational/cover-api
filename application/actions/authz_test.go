package actions

import (
	"testing"

	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"
)

func (as *ActionSuite) Test_getResourceIDSubresource() {
	tests := []struct {
		name           string
		path           string
		wantResource   string
		wantID         uuid.UUID
		wantSub        string
		wantPartsCount int
	}{
		{
			name:           "users",
			path:           "/users",
			wantResource:   "users",
			wantID:         uuid.Nil,
			wantSub:        "",
			wantPartsCount: 1,
		},
		{
			name:           "users",
			path:           "users",
			wantResource:   "users",
			wantID:         uuid.Nil,
			wantSub:        "",
			wantPartsCount: 1,
		},
		{
			name:           "users/b25c0e49-ffdb-4589-a07e-9e27c036ff3c",
			path:           "/users/b25c0e49-ffdb-4589-a07e-9e27c036ff3c",
			wantResource:   "users",
			wantID:         uuid.FromStringOrNil("b25c0e49-ffdb-4589-a07e-9e27c036ff3c"),
			wantSub:        "",
			wantPartsCount: 2,
		},
		{
			name:           "users/b25c0e49-ffdb-4589-a07e-9e27c036ff3c/status",
			path:           "/users/b25c0e49-ffdb-4589-a07e-9e27c036ff3c/status",
			wantResource:   "users",
			wantID:         uuid.FromStringOrNil("b25c0e49-ffdb-4589-a07e-9e27c036ff3c"),
			wantSub:        "status",
			wantPartsCount: 3,
		},
		{
			name:           "users/abc123/status",
			path:           "/users/abc123/status",
			wantResource:   "users",
			wantID:         uuid.Nil,
			wantSub:        "",
			wantPartsCount: 3,
		},
		{
			name:           "users/abc123",
			path:           "/users/abc123",
			wantResource:   "users",
			wantID:         uuid.Nil,
			wantSub:        "",
			wantPartsCount: 2,
		},
		{
			name:           "users/abc123/",
			path:           "/users/abc123/",
			wantResource:   "users",
			wantID:         uuid.Nil,
			wantSub:        "",
			wantPartsCount: 2,
		},
	}
	for _, tt := range tests {
		as.T().Run(tt.name, func(t *testing.T) {
			gotResource, gotID, gotSub, partsCount := getResourceIDSubresource(tt.path)
			assert.Equal(t, tt.wantResource, gotResource)
			assert.Equal(t, tt.wantID, gotID)
			assert.Equal(t, tt.wantSub, gotSub)
			assert.Equal(t, tt.wantPartsCount, partsCount)
		})
	}
}
