package models

func (ms *ModelSuite) TestPolicyHistories_RecentItemStatusChanges() {
	fixtures := CreatePolicyHistoryFixtures_RecentItemStatusChanges(ms.DB)
	phFixes := fixtures.PolicyHistories

	var gotPHs PolicyHistories

	ms.NoError(gotPHs.RecentItemStatusChanges(ms.DB), "error calling function")
	got := make([]string, len(gotPHs))
	for i, g := range gotPHs {
		got[i] = g.ItemID.UUID.String()
	}

	want := []string{
		phFixes[7].ItemID.UUID.String(),
		phFixes[3].ItemID.UUID.String(),
	}

	ms.Equal(want, got, "incorrect results")
}
