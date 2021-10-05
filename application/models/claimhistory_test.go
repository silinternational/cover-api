package models

import (
	"time"

	"github.com/gobuffalo/pop/v5"
	"github.com/gofrs/uuid"

	"github.com/silinternational/cover-api/api"
	"github.com/silinternational/cover-api/domain"
)

// CreateClaimHistoryFixtures generates a Policy with three Claims each
//   with four ClaimHistory entries as follows
//	 Status/Create  [not included as "recent" because not update]
//	 ReferenceNumber/Update [not included as "recent" because not on Status field]
//	 Status/Update [could be included, if date is recent]
//	 Status/Update [could be included, if date is recent]
func CreateClaimHistoryFixtures_RecentClaimStatusChanges(tx *pop.Connection) Fixtures {

	config := FixturesConfig{
		NumberOfPolicies:   1,
		ItemsPerPolicy:     3,
		ClaimsPerPolicy:    3,
		ClaimItemsPerClaim: 1,
	}

	fixtures := CreateItemFixtures(tx, config)
	policy := fixtures.Policies[0]
	user := policy.Members[0]
	claims := fixtures.Claims

	allNewClaim := claims[0]
	mixedNewClaim := claims[1]
	noneNewClaim := claims[2]

	cHistories := make(ClaimHistories, len(claims)*4)

	// Hydrate a set of claimHistories as follows
	//  index n:   Status/Create
	//  index n+1: ReferenceNumber/Update
	//  index n+2: Status/Update
	//  index n+3: Status/Update
	hydrateCHsForClaim := func(startIndex int, claimID uuid.UUID) {
		cHistories[startIndex] = ClaimHistory{
			ClaimID:   claimID,
			Action:    api.HistoryActionCreate,
			FieldName: FieldClaimStatus,
		}
		cHistories[startIndex+1] = ClaimHistory{
			ClaimID:   claimID,
			Action:    api.HistoryActionUpdate,
			FieldName: "ReferenceNumber",
		}
		cHistories[startIndex+2] = ClaimHistory{
			ClaimID:   claimID,
			Action:    api.HistoryActionUpdate,
			FieldName: FieldClaimStatus,
		}
		cHistories[startIndex+3] = ClaimHistory{
			ClaimID:   claimID,
			Action:    api.HistoryActionUpdate,
			FieldName: FieldClaimStatus,
		}
	}

	hydrateCHsForClaim(0, allNewClaim.ID)
	hydrateCHsForClaim(4, mixedNewClaim.ID)
	hydrateCHsForClaim(8, noneNewClaim.ID)

	for i, _ := range cHistories {
		cHistories[i].UserID = user.ID
		MustCreate(tx, &cHistories[i])
	}

	changeCHTime := func(index int, chTime time.Time) {
		q := "UPDATE claim_histories SET created_at = ?, updated_at = ? WHERE id = ?"
		if err := tx.RawQuery(q, chTime, chTime, cHistories[index].ID).Exec(); err != nil {
			panic("error updating updated_at fields: " + err.Error())
		}

		cHistories[index].CreatedAt = chTime
		cHistories[index].UpdatedAt = chTime
	}

	// Give the histories distinguishable times
	now := time.Now().UTC()
	recentTime1 := now.Add(-2 * time.Minute)
	recentTime2 := now.Add(-1 * time.Minute)
	oldTime := now.Add(-2 * domain.DurationWeek)

	for _, i := range []int{0, 1, 2} {
		changeCHTime(i, recentTime1)
	}
	changeCHTime(3, recentTime2)

	for _, i := range []int{4, 5, 6, 8, 9, 10, 11} {
		changeCHTime(i, oldTime)
	}

	fixtures.ClaimHistories = cHistories
	return fixtures
}

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
