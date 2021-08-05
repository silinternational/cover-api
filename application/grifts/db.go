package grifts

import (
	"fmt"
	"time"

	"github.com/silinternational/riskman-api/api"

	"github.com/gofrs/uuid"
	"github.com/markbates/grift/grift"
	"github.com/silinternational/riskman-api/domain"

	"github.com/silinternational/riskman-api/models"
)

func stringToUUID(input string) uuid.UUID {
	id, _ := uuid.FromString(input)
	return id
}

var _ = grift.Namespace("db", func() {
	grift.Desc("seed", "Seeds a database")
	_ = grift.Add("seed", func(c *grift.Context) error {
		// USERS Table
		userUUIDs := []string{
			"e5447366-26b2-4256-b2ab-58c92c3d54cc",
			"3d79902f-c204-4922-b479-57f0ec41eabe",
			"babcf980-e1f0-42d3-b2b0-2e4704159f4f",
			"44dc63fa-1227-4bea-b34a-416a26c3e077",
			"2a96a5a6-971a-403d-8276-c41657bc57ce",
		}

		fixtureUsers := []*models.User{
			{
				Email:        "clark.kent@example.org",
				FirstName:    "Clark",
				LastName:     "Kent",
				LastLoginUTC: time.Now().UTC().Add(time.Hour * -48),
				StaffID:      "111111",
				AppRole:      models.AppRoleAdmin,
			},
			{
				Email:        "jane.eyre@example.org",
				FirstName:    "Jane",
				LastName:     "Eyre",
				LastLoginUTC: time.Now().UTC().Add(time.Hour * -36),
				StaffID:      "222222",
			},
			{
				Email:        "carol.danvers@example.org",
				FirstName:    "Carol",
				LastName:     "Danvers",
				IsBlocked:    true,
				LastLoginUTC: time.Now().UTC().Add(time.Hour * -24),
				StaffID:      "333333",
			},
			{
				Email:        "denethor.ben.ecthelion@example.org",
				FirstName:    "Denethor",
				LastName:     "Ben Ecthelion",
				LastLoginUTC: time.Now().UTC().Add(time.Hour * -18),
				StaffID:      "444444",
			},
			{
				Email:        "john.smith@example.org",
				FirstName:    "John",
				LastName:     "Smith",
				LastLoginUTC: time.Now().UTC().Add(time.Hour * -12),
				StaffID:      "555555",
			},
		}

		for i, uu := range userUUIDs {
			fixtureUsers[i].ID = stringToUUID(uu)
			err := models.DB.Create(fixtureUsers[i])
			if err != nil {
				err = fmt.Errorf("error creating user fixture ... %+v\n %v",
					fixtureUsers[i], err.Error())
				return err
			}
		}

		oneYearFromNow := time.Now().UTC().Add(time.Second * 60 * 60 * 24 * 365)
		fixtureUserTokens := make(models.UserAccessTokens, len(fixtureUsers))
		for i := range fixtureUserTokens {
			fixtureUserTokens[i].ID = domain.GetUUID()
			fixtureUserTokens[i].UserID = fixtureUsers[i].ID
			fixtureUserTokens[i].TokenHash = models.HashClientIdAccessToken(fixtureUsers[i].Email)
			fixtureUserTokens[i].ExpiresAt = oneYearFromNow

			err := models.DB.Create(&fixtureUserTokens[i])
			if err != nil {
				err = fmt.Errorf("error creating user token fixture ... %+v\n %v", fixtureUsers[i], err.Error())
				return err
			}
		}

		policyUUIDs := []string{
			"31447366-26b2-4256-b2ab-58c92c3d54cc",
			"3279902f-c204-4922-b479-57f0ec41eabe",
			"33bcf980-e1f0-42d3-b2b0-2e4704159f4f",
			"34dc63fa-1227-4bea-b34a-416a26c3e077",
			"3596a5a6-971a-403d-8276-c41657bc57ce",
		}

		fixturePolicies := make([]*models.Policy, len(fixtureUsers))

		for i, uu := range policyUUIDs {
			user := fixtureUsers[i]
			fixturePolicies[i] = &models.Policy{
				ID:          stringToUUID(uu),
				Type:        api.PolicyTypeHousehold,
				HouseholdID: fmt.Sprintf("HID-%s-%s", user.FirstName, user.LastName),
			}

			err := models.DB.Create(fixturePolicies[i])
			if err != nil {
				err = fmt.Errorf("error creating policy fixture ... %+v\n %v",
					fixturePolicies[i], err.Error())
				return err
			}
		}

		return nil

	})
})
