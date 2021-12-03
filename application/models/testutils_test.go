package models

import (
	"testing"
)

func (ms *ModelSuite) TestCreateItemFixtures() {
	tests := []struct {
		name                 string
		config               FixturesConfig
		wantCategories       int
		wantItems            int
		wantPolicies         int
		wantPolicyDependents int
		wantPolicyUsers      int
		wantUsers            int
	}{
		{
			name: "single policy, single item",
			config: FixturesConfig{
				ItemsPerPolicy:      1,
				NumberOfPolicies:    1,
				UsersPerPolicy:      1,
				DependentsPerPolicy: 0,
			},
			wantItems:            1,
			wantPolicies:         1,
			wantPolicyDependents: 0,
			wantPolicyUsers:      1,
			wantUsers:            1,
		},
		{
			name: "multiple policies, multiple items each",
			config: FixturesConfig{
				ItemsPerPolicy:      4,
				NumberOfPolicies:    3,
				UsersPerPolicy:      2,
				DependentsPerPolicy: 1,
			},
			wantItems:            12,
			wantPolicies:         3,
			wantPolicyDependents: 3,
			wantPolicyUsers:      6,
			wantUsers:            6,
		},
	}
	for _, tt := range tests {
		ms.T().Run(tt.name, func(t *testing.T) {
			got := CreateItemFixtures(ms.DB, tt.config)
			ms.Equal(tt.wantItems, len(got.Items), "incorrect number of Items")
			ms.Equal(tt.wantPolicies, len(got.Policies), "incorrect number of Policies")
			ms.Equal(tt.wantPolicyDependents, len(got.PolicyDependents), "incorrect number of PolicyDependents")
			ms.Equal(tt.wantPolicyUsers, len(got.PolicyUsers), "incorrect number of PolicyUsers")
			ms.Equal(tt.wantUsers, len(got.Users), "incorrect number of Users")

			ms.Equal(tt.config.UsersPerPolicy, len(got.Policies[0].Members),
				"Policy.Members is not hydrated")
			ms.Equal(tt.config.DependentsPerPolicy, len(got.Policies[0].Dependents),
				"Policy.Dependents is not hydrated")
		})
	}
}

func (ms *ModelSuite) TestCreatePolicyFixtures() {
	tests := []struct {
		name                 string
		config               FixturesConfig
		wantPolicies         int
		wantPolicyDependents int
		wantPolicyUsers      int
		wantUsers            int
	}{
		{
			name: "single policy",
			config: FixturesConfig{
				NumberOfPolicies:    1,
				UsersPerPolicy:      1,
				DependentsPerPolicy: 0,
			},
			wantPolicies:         1,
			wantPolicyDependents: 0,
			wantPolicyUsers:      1,
			wantUsers:            1,
		},
		{
			name: "multiple policies",
			config: FixturesConfig{
				NumberOfPolicies:    2,
				UsersPerPolicy:      2,
				DependentsPerPolicy: 1,
			},
			wantPolicies:         2,
			wantPolicyDependents: 2,
			wantPolicyUsers:      4,
			wantUsers:            4,
		},
	}
	for _, tt := range tests {
		ms.T().Run(tt.name, func(t *testing.T) {
			got := CreatePolicyFixtures(ms.DB, tt.config)
			ms.Equal(tt.wantPolicies, len(got.Policies), "incorrect number of Policies")
			ms.Equal(tt.wantPolicyDependents, len(got.PolicyDependents), "incorrect number of PolicyDependents")
			ms.Equal(tt.wantPolicyUsers, len(got.PolicyUsers), "incorrect number of PolicyUsers")
			ms.Equal(tt.wantUsers, len(got.Users), "incorrect number of Users")

			ms.Equal(tt.config.UsersPerPolicy, len(got.Policies[0].Members),
				"Policy.Members is not hydrated")
			ms.Equal(tt.config.DependentsPerPolicy, len(got.Policies[0].Dependents),
				"Policy.Dependents is not hydrated")
		})
	}
}
