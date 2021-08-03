package models

import (
	"testing"
)

func (ms *ModelSuite) TestItemCategories_Validate() {
	t := ms.T()
	tests := []struct {
		name         string
		itemCategory ItemCategory
		wantErr      bool
		errField     string
	}{
		{
			name: "minimum",
			itemCategory: ItemCategory{
				Name:   "computers",
				Status: "enabled",
			},
			wantErr: false,
		},
		{
			name: "missing Name",
			itemCategory: ItemCategory{
				Status: "enabled",
			},
			wantErr:  true,
			errField: "ItemCategory.Name",
		},
		{
			name: "missing Status",
			itemCategory: ItemCategory{
				Name: "computers",
			},
			wantErr:  true,
			errField: "ItemCategory.Status",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vErr, _ := tt.itemCategory.Validate(DB)
			if tt.wantErr {
				ms.Equal(1, vErr.Count(), "Expected an error, but did not get one")
				ms.Lenf(vErr.Get(tt.errField), 1, "Expected an error on field %v, but got none (errors: %+v)", tt.errField, vErr.Errors)
			} else {
				ms.Falsef(vErr.HasAny(), "Unexpected error: %+v", vErr)
			}
		})
	}
}
