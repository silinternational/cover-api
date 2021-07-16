package grifts

import (
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/markbates/grift/grift"

	"github.com/silinternational/riskman-api/models"
)

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
				Email: "clark.kent@example.org",
			},
			{
				Email: "jane.eyre@example.org",
			},
			{
				Email: "jane.doe@example.org",
			},
			{
				Email: "denethor.ben.ecthelion@example.org",
			},
			{
				Email: "john.smith@example.org",
			},
		}

		for i, user := range fixtureUsers {
			fixtureUsers[i].UUID, _ = uuid.FromString(userUUIDs[i])
			err := models.DB.Create(fixtureUsers[i])
			if err != nil {
				err = fmt.Errorf("error loading user fixture ... %+v\n %v", user, err.Error())
				return err
			}
		}
		return nil
	})
})
