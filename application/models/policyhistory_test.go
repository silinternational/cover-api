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

	pHistories := make(PolicyHistories, len(items)*4)

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

	for i, _ := range pHistories {
		pHistories[i].PolicyID = policy.ID
		pHistories[i].UserID = user.ID
		MustCreate(tx, &pHistories[i])
	}

	now := time.Now().UTC()
	oldTime := now.Add(-2 * domain.DurationWeek)

	makePHOld := func(index int) {
		q := "UPDATE policy_histories SET created_at = ?, updated_at = ? WHERE id = ?"
		if err := tx.RawQuery(q, oldTime, oldTime, pHistories[index].ID).Exec(); err != nil {
			panic("error updating updated_at fields: " + err.Error())
		}

		pHistories[index].CreatedAt = oldTime
		pHistories[index].UpdatedAt = oldTime
	}

	for _, i := range []int{4, 5, 6, 8, 9, 10, 11} {
		makePHOld(i)
	}

	fixtures.PolicyHistories = pHistories
	return fixtures
}

func (ms *ModelSuite) TestPolicyHistories_RecentItemStatusChanges() {
	fixtures := CreatePolicyHistoryFixtures_RecentItemStatusChanges(ms.DB)
	phFixes := fixtures.PolicyHistories

	var gotPHs PolicyHistories

	ms.NoError(gotPHs.RecentItemStatusChanges(ms.DB), "error calling function")
	got := make([][2]string, len(gotPHs))
	for i, g := range gotPHs {
		got[i] = [2]string{g.ID.String(), g.ItemID.UUID.String()}
	}

	want := [][2]string{
		{phFixes[2].ID.String(), phFixes[2].ItemID.UUID.String()},
		{phFixes[3].ID.String(), phFixes[3].ItemID.UUID.String()},
		{phFixes[7].ID.String(), phFixes[7].ItemID.UUID.String()},
	}

	ms.ElementsMatch(want, got, "incorrect results")
}

func (ms *ModelSuite) TestPolicyHistories_getUniqueIDTimes() {
	itemID0 := domain.GetUUID()
	itemID1 := domain.GetUUID()
	itemID2 := domain.GetUUID()

	time0 := time.Date(2000, 1, 1, 1, 0, 0, 0, time.UTC)
	time1 := time.Date(2001, 1, 1, 1, 0, 0, 0, time.UTC)
	time2 := time.Date(2002, 1, 1, 1, 0, 0, 0, time.UTC)
	time3 := time.Date(2003, 1, 1, 1, 0, 0, 0, time.UTC)

	pHistories := PolicyHistories{
		{ItemID: nulls.NewUUID(itemID0), CreatedAt: time1},
		{ItemID: nulls.NewUUID(itemID2), CreatedAt: time0},
		{ItemID: nulls.NewUUID(itemID1), CreatedAt: time2},
		{ItemID: nulls.NewUUID(itemID0), CreatedAt: time3},
		{ItemID: nulls.NewUUID(itemID2), CreatedAt: time0},
	}

	got := pHistories.getUniqueIDTimes()

	want := map[string]time.Time{
		itemID0.String(): time3,
		itemID1.String(): time2,
		itemID2.String(): time0,
	}

	ms.Equal("", assertMapsStringTimeEqual(want, got), "incorrect resulting map")
}
