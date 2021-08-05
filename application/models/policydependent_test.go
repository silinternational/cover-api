package models

import (
	"testing"
	"time"
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
				Relationship: PolicyDependentRelationshipSpouse,
				Location:     "USA",
			},
			wantErr: false,
		},
		{
			name: "minimum Child",
			policyDependent: PolicyDependent{
				Name:           "John Doe",
				Relationship:   PolicyDependentRelationshipChild,
				Location:       "USA",
				ChildBirthYear: time.Now().UTC().AddDate(-18, 0, 0).Year(),
			},
			wantErr: false,
		},
		{
			name: "missing Name",
			policyDependent: PolicyDependent{
				Relationship:   PolicyDependentRelationshipChild,
				Location:       "USA",
				ChildBirthYear: time.Now().UTC().AddDate(-18, 0, 0).Year(),
			},
			wantErr:  true,
			errField: "PolicyDependent.Name",
		},
		{
			name: "missing Relationship",
			policyDependent: PolicyDependent{
				Name:           "Jane Smith",
				Location:       "USA",
				ChildBirthYear: time.Now().UTC().AddDate(-18, 0, 0).Year(),
			},
			wantErr:  true,
			errField: "PolicyDependent.Relationship",
		},
		{
			name: "missing Location",
			policyDependent: PolicyDependent{
				Name:           "Jane Smith",
				Relationship:   PolicyDependentRelationshipChild,
				ChildBirthYear: time.Now().UTC().AddDate(-18, 0, 0).Year(),
			},
			wantErr:  true,
			errField: "PolicyDependent.Location",
		},
		{
			name: "missing ChildBirthYear",
			policyDependent: PolicyDependent{
				Name:         "Jane Smith",
				Relationship: PolicyDependentRelationshipChild,
			},
			wantErr:  true,
			errField: "PolicyDependent.ChildBirthYear",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vErr, _ := tt.policyDependent.Validate(DB)
			if tt.wantErr {
				ms.Equal(1, vErr.Count(), "Expected an error, but did not get one")
				ms.Lenf(vErr.Get(tt.errField), 1, "Expected an error on field %v, but got none (errors: %+v)", tt.errField, vErr.Errors)
			} else {
				ms.Falsef(vErr.HasAny(), "Unexpected error: %+v", vErr)
			}
		})
	}
}
