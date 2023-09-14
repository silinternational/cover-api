package models

import (
	"testing"

	"github.com/silinternational/cover-api/api"
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
				Status: api.ItemCategoryStatusEnabled,
			},
			wantErr: false,
		},
		{
			name: "missing Name",
			itemCategory: ItemCategory{
				Status: api.ItemCategoryStatusEnabled,
			},
			wantErr:  true,
			errField: "ItemCategory.Name",
		},
		{
			name: "invalid Status",
			itemCategory: ItemCategory{
				Name:   "computers",
				Status: "bogus",
			},
			wantErr:  true,
			errField: "ItemCategory.Status",
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

func (ms *ModelSuite) TestItemCategory_ConvertToAPI() {
	fixtures := CreateItemFixtures(ms.DB, FixturesConfig{DependentsPerPolicy: 1})
	cat := fixtures.ItemCategories[0]
	got := cat.ConvertToAPI(ms.DB)

	ms.Equal(cat.ID, got.ID, "ID is not correct")
	ms.Equal(cat.Name, got.Name, "Name is not correct")
	ms.Equal(cat.HelpText, got.HelpText, "HelpText is not correct")
	ms.Equal(cat.RiskCategory.ID, got.RiskCategory.ID, "RiskCategory.ID is not correct")
	ms.Equal(cat.RequireMakeModel, got.RequireMakeModel, "RequireMakeModel is not correct")
	ms.Equal(cat.BillingPeriod, got.BillingPeriod, "BillingPeriod is not correct")
	ms.Equal(cat.PremiumFactor, got.PremiumFactor, "PremiumFactor is not correct")
	ms.Equal(cat.CreatedAt, got.CreatedAt, "CreatedAt is not correct")
	ms.Equal(cat.UpdatedAt, got.UpdatedAt, "UpdatedAt is not correct")
}
