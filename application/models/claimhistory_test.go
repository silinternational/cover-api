package models

func (ms *ModelSuite) TestClaimHistories_RecentClaimStatusChanges() {
	fixtures := CreateClaimHistoryFixtures_RecentClaimStatusChanges(ms.DB)
	chFixes := fixtures.ClaimHistories

	var gotCHs ClaimHistories

	ms.NoError(gotCHs.RecentClaimStatusChanges(ms.DB), "error calling function")
	got := make([]string, len(gotCHs))
	for i, g := range gotCHs {
		got[i] = g.ClaimID.String()
	}

	want := []string{
		chFixes[7].ClaimID.String(),
		chFixes[3].ClaimID.String(),
	}

	ms.Equal(want, got, "incorrect results")
}
