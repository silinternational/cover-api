package models

import (
	"testing"
	"time"

	"github.com/silinternational/cover-api/api"
	"github.com/silinternational/cover-api/domain"
)

func (ms *ModelSuite) TestPolicyDependent_Validate() {
	t := ms.T()
	tests := []struct {
		name            string
		policyDependent PolicyDependent
		wantErr         bool
		errField        string
	}{
		{
			name: "minimum Spouse",
			policyDependent: PolicyDependent{
				Name:         "Jane Smith",
				Relationship: api.PolicyDependentRelationshipSpouse,
				Country:      "USA",
				CountryCode:  "USA",
			},
			wantErr: false,
		},
		{
			name: "minimum Child",
			policyDependent: PolicyDependent{
				Name:           "John Doe",
				Relationship:   api.PolicyDependentRelationshipChild,
				Country:        "USA",
				CountryCode:    "USA",
				ChildBirthYear: time.Now().UTC().Year() - 18,
			},
			wantErr: false,
		},
		{
			name: "missing Name",
			policyDependent: PolicyDependent{
				Relationship:   api.PolicyDependentRelationshipChild,
				Country:        "USA",
				CountryCode:    "USA",
				ChildBirthYear: time.Now().UTC().Year() - 18,
			},
			wantErr:  true,
			errField: "PolicyDependent.Name",
		},
		{
			name: "missing Relationship",
			policyDependent: PolicyDependent{
				Name:           "Jane Smith",
				Country:        "USA",
				CountryCode:    "USA",
				ChildBirthYear: time.Now().UTC().Year() - 18,
			},
			wantErr:  true,
			errField: "PolicyDependent.Relationship",
		},
		{
			name: "missing Country",
			policyDependent: PolicyDependent{
				Name:           "Jane Smith",
				Relationship:   api.PolicyDependentRelationshipChild,
				ChildBirthYear: time.Now().UTC().Year() - 18,
			},
			wantErr:  true,
			errField: "PolicyDependent.Country",
		},
		{
			name: "missing ChildBirthYear",
			policyDependent: PolicyDependent{
				Name:         "Jane Smith",
				CountryCode:  "USA",
				Relationship: api.PolicyDependentRelationshipChild,
			},
			wantErr:  true,
			errField: "PolicyDependent.ChildBirthYear",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vErr, _ := tt.policyDependent.Validate(DB)
			if tt.wantErr {
				ms.GreaterOrEqualf(vErr.Count(), 1, "Expected an error, but did not get one")
				ms.Lenf(vErr.Get(tt.errField), 1, "Expected an error on field %v, but got none (errors: %+v)", tt.errField, vErr.Errors)
			} else {
				ms.Falsef(vErr.HasAny(), "Unexpected error: %+v", vErr)
			}
		})
	}
}

func (ms *ModelSuite) TestPolicyDependent_ConvertToAPI() {
	dependent := PolicyDependent{
		ID:             domain.GetUUID(),
		Name:           randStr(10),
		Relationship:   api.PolicyDependentRelationshipChild,
		Country:        randStr(10),
		ChildBirthYear: domain.RandomInsecureIntInRange(2000, 2020),
	}

	got := dependent.ConvertToAPI(ms.DB)

	ms.Equal(dependent.ID, got.ID, "ID is not correct")
	ms.Equal(dependent.Name, got.Name, "Name is not correct")
	ms.Equal(dependent.Relationship, got.Relationship, "Relationship is not correct")
	ms.Equal(dependent.Country, got.Country, "Country is not correct")
	ms.Equal(dependent.ChildBirthYear, got.ChildBirthYear, "ChildBirthYear is not correct")
}
