package models

import (
	"time"

	"github.com/gobuffalo/nulls"
	"github.com/gobuffalo/pop/v5"
	"github.com/gofrs/uuid"

	"github.com/silinternational/cover-api/api"
	"github.com/silinternational/cover-api/domain"
)

// CreatePolicyHistoryFixtures generates a Policy with three Items each with
//   four PolicyHistory entries as follows
//	 CoverageStatus/Create  [not included because not update]
//	 Name/Update [not included because not on CoverageStatus field]
//	 CoverageStatus/Update [could be included, if date is recent]
//	 CoverageStatus/Update [could be included, if date is recent]
func CreatePolicyHistoryFixtures_RecentItemStatusChanges(tx *pop.Connection) Fixtures {
	config := FixturesConfig{
		NumberOfPolicies: 1,
		ItemsPerPolicy:   3,
	}

	fixtures := CreateItemFixtures(tx, config)
	policy := fixtures.Policies[0]
	user := policy.Members[0]
	items := fixtures.Items

	allNewItem := items[0]
	mixedNewItem := items[1]
	noneNewItem := items[2]

	pHistories := make(PolicyHistories, len(items)*4+1)

	// Hydrate a set of policyHistories as follows
	//  index n:   CoverageStatus/Create
	//  index n+1: Name/Update
	//  index n+2: CoverageStatus/Update
	//  index n+3: CoverageStatus/Update
	hydratePHsForItem := func(startIndex int, itemID uuid.UUID) {
		pHistories[startIndex] = PolicyHistory{
			ItemID:    nulls.NewUUID(itemID),
			Action:    api.HistoryActionCreate,
			FieldName: FieldItemCoverageStatus,
		}
		pHistories[startIndex+1] = PolicyHistory{
			ItemID:    nulls.NewUUID(itemID),
			Action:    api.HistoryActionUpdate,
			FieldName: "Name",
		}
		pHistories[startIndex+2] = PolicyHistory{
			ItemID:    nulls.NewUUID(itemID),
			Action:    api.HistoryActionUpdate,
			FieldName: FieldItemCoverageStatus,
		}
		pHistories[startIndex+3] = PolicyHistory{
			ItemID:    nulls.NewUUID(itemID),
			Action:    api.HistoryActionUpdate,
			FieldName: FieldItemCoverageStatus,
		}
	}

	hydratePHsForItem(0, allNewItem.ID)
	hydratePHsForItem(4, mixedNewItem.ID)
	hydratePHsForItem(8, noneNewItem.ID)

	// Make sure a null item_id doesn't slip through
	pHistories[12] = PolicyHistory{
		Action:    api.HistoryActionUpdate,
		FieldName: FieldItemCoverageStatus,
	}

	for i := range pHistories {
		pHistories[i].PolicyID = policy.ID
		pHistories[i].UserID = user.ID
		MustCreate(tx, &pHistories[i])
	}

	changePHTime := func(index int, chTime time.Time) {
		q := "UPDATE policy_histories SET created_at = ?, updated_at = ? WHERE id = ?"
		if err := tx.RawQuery(q, chTime, chTime, pHistories[index].ID).Exec(); err != nil {
			panic("error updating updated_at fields: " + err.Error())
		}

		pHistories[index].CreatedAt = chTime
		pHistories[index].UpdatedAt = chTime
	}

	// Give the histories distinguishable times
	now := time.Now().UTC()
	recentTime1 := now.Add(-2 * time.Minute)
	recentTime2 := now.Add(-1 * time.Minute)
	oldTime := now.Add(-2 * domain.DurationWeek)

	for _, i := range []int{0, 1, 2} {
		changePHTime(i, recentTime1)
	}
	changePHTime(3, recentTime2)

	for _, i := range []int{4, 5, 6, 8, 9, 10, 11} {
		changePHTime(i, oldTime)
	}

	fixtures.PolicyHistories = pHistories
	return fixtures
}

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
