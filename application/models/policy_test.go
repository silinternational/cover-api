package models

import (
	"testing"

	"github.com/silinternational/riskman-api/api"
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
			name: "valid type",
			Policy: Policy{
				Type:        api.PolicyTypeHousehold,
				HouseholdID: "123456",
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
		HouseholdID: rando,
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

	err := policy.LoadMembers(ms.DB, false)
	ms.Nil(err)
	ms.Len(policy.Members, 1)
}

func (ms *ModelSuite) TestPolicy_LoadDependents() {
	rando := randStr(6)
	policy := Policy{
		Type:        api.PolicyTypeHousehold,
		HouseholdID: rando,
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
		Relationship:   PolicyDependentRelationshipChild,
		Location:       "Bahamas",
		ChildBirthYear: 2000,
	}
	MustCreate(ms.DB, &pu)

	err := policy.LoadDependents(ms.DB, false)
	ms.NoError(err)
	ms.Len(policy.Dependents, 1)
}
