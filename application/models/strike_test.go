package models

import (
	"testing"
	"time"
)

func (ms *ModelSuite) TestStrikes_RecentForPolicy() {
	t := ms.T()

	fixConfig := FixturesConfig{
		NumberOfPolicies: 4,
		ClaimsPerPolicy:  1,
	}

	fixtures := CreateItemFixtures(ms.DB, fixConfig)

	policyNoStrikes := fixtures.Policies[0]
	policyOneStrike := fixtures.Policies[1]
	policyTwoStrikes := fixtures.Policies[2]
	policyHasOldStrike := fixtures.Policies[3]

	oldDate := policyHasOldStrike.Claims[0].IncidentDate.AddDate(-2, 0, 0)
	strikeDates := [][]*time.Time{
		{},                // Policy with no strikes
		[]*time.Time{nil}, // Policy with one strike
		{nil, nil},        // Policy with two strikes
		{&oldDate, nil},   // Policy with an old strike and a new strike
	}

	strikes := CreateStrikeFixtures(ms.DB, fixtures.Policies, strikeDates)

	cutOff := time.Now().UTC()
	tests := []struct {
		name    string
		policy  Policy
		wantIDs []string
	}{
		{
			name:    "no strikes",
			policy:  policyNoStrikes,
			wantIDs: []string{},
		},
		{
			name:    "has one strike",
			policy:  policyOneStrike,
			wantIDs: []string{strikes[0].ID.String()},
		},
		{
			name:    "has two strikes",
			policy:  policyTwoStrikes,
			wantIDs: []string{strikes[1].ID.String(), strikes[2].ID.String()},
		},
		{
			name:    "has old strike",
			policy:  policyHasOldStrike,
			wantIDs: []string{strikes[4].ID.String()},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got Strikes

			err := got.RecentForPolicy(ms.DB, tt.policy.ID, cutOff)
			ms.NoError(err)

			ms.Len(got, len(tt.wantIDs), "incorrect number of strikes")
		})
	}
}
