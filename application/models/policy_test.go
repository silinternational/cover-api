package models

import (
	"testing"

	"github.com/gobuffalo/nulls"

	"github.com/silinternational/cover-api/api"
	"github.com/silinternational/cover-api/domain"
)

func (ms *ModelSuite) TestPolicy_Validate() {
	t := ms.T()
	tests := []struct {
		name     string
		Policy   Policy
		wantErr  bool
		errField string
	}{
		{
			name: "invalid",
			Policy: Policy{
				Type: "invalid",
			},
			wantErr:  true,
			errField: "Policy.Type",
		},
		{
			name:     "missing type",
			Policy:   Policy{},
			wantErr:  true,
			errField: "Policy.Type",
		},
		{
			name: "valid type, missing household id",
			Policy: Policy{
				Type: api.PolicyTypeHousehold,
			},
			wantErr:  true,
			errField: "Policy.HouseholdID",
		},
		{
			name: "household type, should not have cost center",
			Policy: Policy{
				Type:        api.PolicyTypeHousehold,
				HouseholdID: nulls.NewString("abc123"),
				CostCenter:  "forbidden",
			},
			wantErr:  true,
			errField: "Policy.CostCenter",
		},
		{
			name: "household type, should not have account",
			Policy: Policy{
				Type:        api.PolicyTypeHousehold,
				HouseholdID: nulls.NewString("abc123"),
				Account:     "forbidden",
			},
			wantErr:  true,
			errField: "Policy.Account",
		},
		{
			name: "household type, should not have entity code id",
			Policy: Policy{
				Type:         api.PolicyTypeHousehold,
				HouseholdID:  nulls.NewString("abc123"),
				EntityCodeID: nulls.NewUUID(domain.GetUUID()),
			},
			wantErr:  true,
			errField: "Policy.EntityCodeID",
		},
		{
			name: "corporate type, should not have household id",
			Policy: Policy{
				Type:         api.PolicyTypeCorporate,
				HouseholdID:  nulls.NewString("abc123"),
				CostCenter:   "abc123",
				Account:      "123456",
				EntityCodeID: nulls.NewUUID(domain.GetUUID()),
			},
			wantErr:  true,
			errField: "Policy.HouseholdID",
		},
		{
			name: "corporate type, should have cost center",
			Policy: Policy{
				Type:         api.PolicyTypeCorporate,
				Account:      "123456",
				EntityCodeID: nulls.NewUUID(domain.GetUUID()),
			},
			wantErr:  true,
			errField: "Policy.CostCenter",
		},
		{
			name: "corporate type, should have account",
			Policy: Policy{
				Type:         api.PolicyTypeCorporate,
				HouseholdID:  nulls.NewString("abc123"),
				CostCenter:   "abc123",
				EntityCodeID: nulls.NewUUID(domain.GetUUID()),
			},
			wantErr:  true,
			errField: "Policy.Account",
		},
		{
			name: "corporate type, should have entity code id",
			Policy: Policy{
				Type:        api.PolicyTypeCorporate,
				HouseholdID: nulls.NewString("abc123"),
				CostCenter:  "abc123",
				Account:     "123456",
			},
			wantErr:  true,
			errField: "Policy.HouseholdID",
		},
		{
			name: "valid household type",
			Policy: Policy{
				Type:        api.PolicyTypeHousehold,
				HouseholdID: nulls.NewString("123456"),
			},
			wantErr:  false,
			errField: "",
		},
		{
			name: "valid corporate type",
			Policy: Policy{
				Type:         api.PolicyTypeCorporate,
				CostCenter:   "abc123",
				Account:      "123456",
				EntityCodeID: nulls.NewUUID(domain.GetUUID()),
			},
			wantErr:  false,
			errField: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vErr, _ := tt.Policy.Validate(DB)
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

func (ms *ModelSuite) TestPolicy_LoadMembers() {
	rando := randStr(6)
	policy := Policy{
		Type:        api.PolicyTypeHousehold,
		HouseholdID: nulls.NewString(rando),
	}
	MustCreate(ms.DB, &policy)

	user := User{
		Email:     rando + "@testerson.com",
		FirstName: "Test",
		LastName:  "Testerson",
		IsBlocked: false,
		StaffID:   rando,
		AppRole:   AppRoleUser,
	}
	MustCreate(ms.DB, &user)

	pu := PolicyUser{
		PolicyID: policy.ID,
		UserID:   user.ID,
	}
	MustCreate(ms.DB, &pu)

	policy.LoadMembers(ms.DB, false)
	ms.Len(policy.Members, 1)
}

func (ms *ModelSuite) TestPolicy_LoadDependents() {
	rando := randStr(6)
	policy := Policy{
		Type:        api.PolicyTypeHousehold,
		HouseholdID: nulls.NewString(rando),
	}
	MustCreate(ms.DB, &policy)

	user := User{
		Email:     rando + "@testerson.com",
		FirstName: "Test",
		LastName:  "Testerson",
		IsBlocked: false,
		StaffID:   rando,
		AppRole:   AppRoleUser,
	}
	MustCreate(ms.DB, &user)

	pu := PolicyDependent{
		PolicyID:       policy.ID,
		Name:           rando + "-kiddo",
		Relationship:   api.PolicyDependentRelationshipChild,
		Location:       "Bahamas",
		ChildBirthYear: 2000,
	}
	MustCreate(ms.DB, &pu)

	policy.LoadDependents(ms.DB, false)
	ms.Len(policy.Dependents, 1)
}

func (ms *ModelSuite) TestPolicy_itemCoverageTotals() {
	fixConfig := FixturesConfig{
		NumberOfPolicies:    2,
		UsersPerPolicy:      2,
		DependentsPerPolicy: 2,
		ItemsPerPolicy:      5,
	}

	fixtures := CreateItemFixtures(ms.DB, fixConfig)
	policy := fixtures.Policies[0]
	policy.LoadItems(ms.DB, false)
	items := policy.Items

	// give two items a dependant and calculate expected values
	dependant := policy.Dependents[0]
	coverageForPolicy := 0
	coverageForDep := 0
	for i := range items {
		// Set to approved
		if i < 4 {
			items[i] = UpdateItemStatus(ms.DB, items[i], api.ItemCoverageStatusApproved)
			coverageForPolicy += items[i].CoverageAmount
		}

		if i == 2 || i == 3 {
			items[i].PolicyDependentID = nulls.NewUUID(dependant.ID)
			ms.NoError(ms.DB.Update(&items[i]), "error trying to change item DependantID")
			coverageForDep += items[i].CoverageAmount
		}
	}

	policy.Items = Items{} // ensure the LoadItems gets called

	got := policy.itemCoverageTotals(ms.DB)

	ms.Equal(coverageForPolicy, got[policy.ID], "incorrect policy coverage total")
	ms.Equal(coverageForDep, got[dependant.ID], "incorrect dependant coverage total")
	ms.Greater(coverageForPolicy, coverageForDep, "double checking exposed a problem with the test design")

	// Note this includes the dependant total twice, which is OK for testing purposes
	gotTotal := 0
	for _, v := range got {
		gotTotal += v
	}

	want := coverageForPolicy + coverageForDep
	ms.Equal(want, gotTotal, "incorrect coverage grand total")
}
