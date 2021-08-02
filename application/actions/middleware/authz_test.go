package middleware

import (
	"testing"

	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"
)

func Test_getResourceIDSubresource(t *testing.T) {
	tests := []struct {
		name         string
		path         string
		wantResource string
		wantID       uuid.UUID
		wantSub      string
	}{
		{
			name:         "users",
			path:         "/users",
			wantResource: "users",
			wantID:       uuid.Nil,
			wantSub:      "",
		},
		{
			name:         "users",
			path:         "users",
			wantResource: "users",
			wantID:       uuid.Nil,
			wantSub:      "",
		},
		{
			name:         "users/b25c0e49-ffdb-4589-a07e-9e27c036ff3c",
			path:         "/users/b25c0e49-ffdb-4589-a07e-9e27c036ff3c",
			wantResource: "users",
			wantID:       uuid.FromStringOrNil("b25c0e49-ffdb-4589-a07e-9e27c036ff3c"),
			wantSub:      "",
		},
		{
			name:         "users/b25c0e49-ffdb-4589-a07e-9e27c036ff3c/status",
			path:         "/users/b25c0e49-ffdb-4589-a07e-9e27c036ff3c/status",
			wantResource: "users",
			wantID:       uuid.FromStringOrNil("b25c0e49-ffdb-4589-a07e-9e27c036ff3c"),
			wantSub:      "status",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotResource, gotID, gotSub := getResourceIDSubresource(tt.path)
			assert.Equal(t, tt.wantResource, gotResource)
			assert.Equal(t, tt.wantID, gotID)
			assert.Equal(t, tt.wantSub, gotSub)
		})
	}
}
