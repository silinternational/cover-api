package models

import (
	"testing"
)

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
				Policies:            1,
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
				Policies:            2,
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
		})
	}
}
