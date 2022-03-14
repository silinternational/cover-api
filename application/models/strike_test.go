package models

import (
	"testing"
)

func (ms *ModelSuite) TestStrikes_RecentForClaim() {
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

	strikes := Strikes{
		{Description: "For Policy with one strike", PolicyID: policyOneStrike.ID},
		{Description: "For Policy with two strikes - A", PolicyID: policyTwoStrikes.ID},
		{Description: "For Policy with two strikes - B", PolicyID: policyTwoStrikes.ID},
		{Description: "For Policy has old strike - A", PolicyID: policyHasOldStrike.ID},
		{Description: "For Policy has old strike - B", PolicyID: policyHasOldStrike.ID},
	}
	ms.NoError(ms.DB.Create(&strikes), "error creating strikes fixtures")

	oldStrike := strikes[3]

	// Merely calling the db.Update function doesn't overwrite the created_at value
	q := ms.DB.RawQuery("Update strikes SET created_at = ? WHERE id = ?", oldDate, oldStrike.ID)
	ms.NoError(q.Exec(), "error updating old strike fixture")

	tests := []struct {
		name    string
		claim   Claim
		wantIDs []string
	}{
		{
			name:    "no strikes",
			claim:   policyNoStrikes.Claims[0],
			wantIDs: []string{},
		},
		{
			name:    "has one strike",
			claim:   policyOneStrike.Claims[0],
			wantIDs: []string{strikes[0].ID.String()},
		},
		{
			name:    "has two strikes",
			claim:   policyTwoStrikes.Claims[0],
			wantIDs: []string{strikes[1].ID.String(), strikes[2].ID.String()},
		},
		{
			name:    "has old strike",
			claim:   policyHasOldStrike.Claims[0],
			wantIDs: []string{strikes[4].ID.String()},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got Strikes

			err := got.RecentForClaim(ms.DB, &tt.claim)
			ms.NoError(err)

			ms.Len(got, len(tt.wantIDs), "incorrect number of strikes")
		})
	}
}
