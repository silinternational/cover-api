package grifts

import (
	"fmt"
	"regexp"
	"time"

	"github.com/gobuffalo/nulls"

	"github.com/silinternational/cover-api/api"

	"github.com/gofrs/uuid"
	"github.com/markbates/grift/grift"

	"github.com/silinternational/cover-api/domain"
	"github.com/silinternational/cover-api/models"
)

var _ = grift.Namespace("db", func() {
	grift.Desc("seed", "Seeds a database")
	_ = grift.Add("seed", func(c *grift.Context) error {
		countUsers := models.Users{}
		count, err := models.DB.Count(countUsers)
		if err != nil {
			return err
		}

		if count > 0 {
			fmt.Printf("\nINFO: It appears that the grifts have already been run, "+
				"since there are already %v users.\n", count)
			return nil
		}

		fixUsers, err := createUserFixtures()
		if err != nil {
			return err
		}

		fixPolicies, err := createPolicyFixtures(fixUsers)
		if err != nil {
			return err
		}

		fixCats, err := createCategories()
		if err != nil {
			return err
		}

		_, err = createItemFixtures(fixPolicies, fixCats)
		if err != nil {
			return err
		}

		_, err = createClaimFixtures(fixPolicies)
		if err != nil {
			return err
		}

		return nil
	})
})

func createUserFixtures() ([]*models.User, error) {
	userUUIDs := []string{
		"e5447366-26b2-4256-b2ab-58c92c3d54cc",
		"3d79902f-c204-4922-b479-57f0ec41eabe",
		"babcf980-e1f0-42d3-b2b0-2e4704159f4f",
		"44dc63fa-1227-4bea-b34a-416a26c3e077",
		"2a96a5a6-971a-403d-8276-c41657bc57ce",
	}

	fixUsers := []*models.User{
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
		fixUsers[i].ID = uuid.FromStringOrNil(uu)
		err := models.DB.Create(fixUsers[i])
		if err != nil {
			err = fmt.Errorf("error creating user fixture ... %+v\n %v",
				fixUsers[i], err.Error())
			return fixUsers, err
		}
	}

	oneYearFromNow := time.Now().UTC().Add(time.Second * 60 * 60 * 24 * 365)
	fixUserTokens := make(models.UserAccessTokens, len(fixUsers))
	for i := range fixUserTokens {
		fixUserTokens[i].ID = domain.GetUUID()
		fixUserTokens[i].UserID = fixUsers[i].ID
		fixUserTokens[i].TokenHash = models.HashClientIdAccessToken(fixUsers[i].Email)
		fixUserTokens[i].ExpiresAt = oneYearFromNow

		err := models.DB.Create(&fixUserTokens[i])
		if err != nil {
			err = fmt.Errorf("error creating user token fixture ... %+v\n %v", fixUsers[i], err.Error())
			return fixUsers, err
		}
	}

	return fixUsers, nil
}

func createPolicyFixtures(fixUsers []*models.User) ([]*models.Policy, error) {
	policyUUIDs := []string{
		"31447366-26b2-4256-b2ab-58c92c3d54cc",
		"3279902f-c204-4922-b479-57f0ec41eabe",
		"33bcf980-e1f0-42d3-b2b0-2e4704159f4f",
		"34dc63fa-1227-4bea-b34a-416a26c3e077",
		"3596a5a6-971a-403d-8276-c41657bc57ce",
	}

	if len(policyUUIDs) != len(fixUsers) {
		err := fmt.Errorf("mismatching count of fixtures in createPolicyFixtures. "+
			"Expected the number of user fixtures to be %d, but got %d",
			len(policyUUIDs), len(fixUsers))
		return []*models.Policy{}, err
	}

	fixPolicies := make([]*models.Policy, len(fixUsers))

	for i, uu := range policyUUIDs {
		user := fixUsers[i]
		fixPolicies[i] = &models.Policy{
			ID:          uuid.FromStringOrNil(uu),
			Type:        api.PolicyTypeHousehold,
			HouseholdID: nulls.NewString(fmt.Sprintf("HID-%s-%s", user.FirstName, user.LastName)),
		}

		err := models.DB.Create(fixPolicies[i])
		if err != nil {
			err = fmt.Errorf("error creating policy fixture ... %+v\n %v",
				fixPolicies[i], err.Error())
			return []*models.Policy{}, err
		}
	}

	fixPolicyUsers := make([]*models.PolicyUser, len(fixUsers))

	for i, u := range fixUsers {
		fixPolicyUsers[i] = &models.PolicyUser{
			ID:       domain.GetUUID(),
			PolicyID: fixPolicies[i].ID,
			UserID:   u.ID,
		}

		err := models.DB.Create(fixPolicyUsers[i])
		if err != nil {
			err = fmt.Errorf("error creating policy users fixture ... %+v\n %v",
				fixPolicyUsers[i], err.Error())
			return []*models.Policy{}, err
		}
	}

	return fixPolicies, nil
}

func createCategories() ([]uuid.UUID, error) {
	const itemCategoriesSql = `
INSERT INTO "item_categories" ("id", "risk_category_id", "name", "help_text", "status", "auto_approve_max", "created_at", "updated_at", "legacy_id", "require_make_model") VALUES
('d4632d64-67b5-4795-a7de-66b95312fa7e',	'3be38915-7092-44f2-90ef-26f48214b34f',	'Computers, tablets, and phones',	'Includes printers, screens, peripherals, and extras',	'Enabled',	300000,	'2021-08-27 19:46:28',	'2021-08-27 19:46:28',	1, true),
('9c682e38-78fd-475b-9810-3a7f2e9f1fe4',	'7bed3c00-23cf-4282-b2b8-da89426cef2f',	'Clothing',	'-',	'Enabled',	300000,	'2021-08-27 19:46:28',	'2021-08-27 19:46:28',	10, false),
('4b06f087-3fb0-4345-82e8-803645962db0',	'3be38915-7092-44f2-90ef-26f48214b34f',	'Medical',	'Eyewear, insulin pumps, CPAP, prosthetics, and more',	'Enabled',	300000,	'2021-08-27 19:46:28',	'2021-08-27 19:46:28',	11, true),
('61081c4d-b6e3-47c5-aca7-373fa7d30896',	'3be38915-7092-44f2-90ef-26f48214b34f',	'Photography and recording',	'Includes video, audio, peripherals, and extras',	'Enabled',	300000,	'2021-08-27 19:46:28',	'2021-08-27 19:46:28',	2, true),
('863a3306-78f9-4aca-add5-0abda3a1ef02',	'3be38915-7092-44f2-90ef-26f48214b34f',	'Other',	'-',	'Enabled',	300000,	'2021-08-27 19:46:28',	'2021-08-27 19:46:28',	3, true),
('faa39da0-981e-4fcf-92fc-2c047fd21f15',	'3be38915-7092-44f2-90ef-26f48214b34f',	'Musical instruments',	'Includes peripherals and extras',	'Enabled',	300000,	'2021-08-27 19:46:28',	'2021-08-27 19:46:28',	4, true),
('660629ef-ff63-4ace-8263-993897de7d6b',	'7bed3c00-23cf-4282-b2b8-da89426cef2f',	'Appliances and home electronics',	'Washing machines, ovens, theater equipment, and more',	'Enabled',	300000,	'2021-08-27 19:46:28',	'2021-08-27 19:46:28',	5, false),
('aa304ce5-be3d-45eb-929e-b4575973c0d3',	'7bed3c00-23cf-4282-b2b8-da89426cef2f',	'Home goods',	'Furniture, kitchenware, decorations, linens, and more',	'Enabled',	300000,	'2021-08-27 19:46:28',	'2021-08-27 19:46:28',	6, false),
('722c03e5-7852-44b9-b86a-af5d63b39d0e',	'7bed3c00-23cf-4282-b2b8-da89426cef2f',	'Field site electronics',	'Solar panels, power systems, antennae, and more',	'Enabled',	300000,	'2021-08-27 19:46:28',	'2021-08-27 19:46:28',	7, false),
('0f7aa101-bfdb-4a19-a182-c5ff1d16f6b2',	'7bed3c00-23cf-4282-b2b8-da89426cef2f',	'Books and media',	'Books, CDs, DVDs, and more',	'Enabled',	300000,	'2021-08-27 19:46:28',	'2021-08-27 19:46:28',	8, false),
('036e5315-18ca-4404-8435-72a695f2c9a7',	'3be38915-7092-44f2-90ef-26f48214b34f',	'Travel and recreation',	'Includes suitcases, travel bags, cycling, skating, sports. No motorized vehicles.',	'Enabled',	300000,	'2021-08-27 19:46:28',	'2021-08-27 19:46:28',	9, true);
`
	if err := models.DB.RawQuery(itemCategoriesSql).Exec(); err != nil {
		panic("error loading item categories, " + err.Error())
	}

	r, err := regexp.Compile(`\('([0-9a-f-]*)`)
	if err != nil {
		panic("invalid regular expression, " + err.Error())
	}
	matches := r.FindAllStringSubmatch(itemCategoriesSql, -1)
	if len(matches) == 0 {
		panic("found no category UUIDs in SQL query")
	}

	categoryUUIDs := make([]uuid.UUID, len(matches))
	for i, match := range matches {
		categoryUUIDs[i], err = uuid.FromString(match[1])
		if err != nil {
			panic(fmt.Sprintf("invalid UUID %s, %s", match, err))
		}
	}

	return categoryUUIDs, nil
}

func createItemFixtures(fixPolicies []*models.Policy, fixICats []uuid.UUID) ([]*models.Item, error) {
	itemUUIDs := []string{
		"71117366-26b2-4256-b2ab-58c92c3d54cc",
		"7212902f-c204-4922-b479-57f0ec41eabe",
		"7321f980-e1f0-42d3-b2b0-2e4704159f4f",
		"742263fa-1227-4bea-b34a-416a26c3e077",
		"7531a5a6-971a-403d-8276-c41657bc57ce",
		"76327366-26b2-4256-b2ab-58c92c3d54cc",
		"7741902f-c204-4922-b479-57f0ec41eabe",
		"7842f980-e1f0-42d3-b2b0-2e4704159f4f",
		"795163fa-1227-4bea-b34a-416a26c3e077",
		"7052a5a6-971a-403d-8276-c41657bc57ce",
	}

	if len(itemUUIDs)/2 != len(fixPolicies) {
		err := fmt.Errorf("mismatching count of fixtures in createItemFixtures. "+
			"Expected the number of policy fixtures to be %d, but got %d",
			len(itemUUIDs)/2, len(fixPolicies))
		return []*models.Item{}, err
	}

	fixItems := make([]*models.Item, len(itemUUIDs))
	countICats := len(fixICats)

	for i, uu := range itemUUIDs {
		fixItems[i] = &models.Item{
			ID:                uuid.FromStringOrNil(uu),
			Name:              fmt.Sprintf("IName-%d", i),
			CategoryID:        fixICats[i%countICats], // cycle through item categories
			InStorage:         false,
			Country:           fmt.Sprintf("ICountry%d", i),
			Description:       fmt.Sprintf("This is the description for item %d.", i),
			PolicyID:          fixPolicies[i/2].ID,
			Make:              fmt.Sprintf("IMake-%d", i),
			Model:             fmt.Sprintf("IModel-%d", i),
			SerialNumber:      fmt.Sprintf("ISN-%d", i),
			CoverageAmount:    50 * (i + 1) * domain.CurrencyFactor, // increments of $50 starting at $50
			CoverageStatus:    api.ItemCoverageStatusApproved,
			CoverageStartDate: time.Now().UTC().Add(time.Hour * time.Duration((i+1)*-40)),
			PurchaseDate:      time.Now().UTC().Add(time.Hour * time.Duration((i+1)*-48)),
		}

		err := models.DB.Create(fixItems[i])
		if err != nil {
			err = fmt.Errorf("error creating item fixture ... %+v\n %v",
				fixItems[i], err.Error())
			return []*models.Item{}, err
		}
	}

	return fixItems, nil
}

func createClaimFixtures(fixPolicies []*models.Policy) ([]models.Claim, error) {
	claimUUIDs := []string{
		"023b599d-dd17-4eb9-9895-da462f52526a",
		"1eba86ef-e801-4a9c-a500-fe507040d004",
		"2e1caab9-6ba4-45f5-bb0a-40e9a406e3a0",
		"37a5b5e4-8e52-4276-be3c-ee3d320ad0dc",
		"41176ee9-b6cc-4064-9295-8fbab81d8a99",
	}

	if len(claimUUIDs) > len(fixPolicies) {
		err := fmt.Errorf("mismatching count of fixtures in createPolicyFixtures. "+
			"Expected the number of user fixtures to be %d, but got %d",
			len(claimUUIDs), len(fixPolicies))
		return nil, err
	}

	fixClaims := make([]models.Claim, len(fixPolicies))

	for i, uu := range claimUUIDs {
		fixClaims[i] = models.Claim{
			ID:        uuid.FromStringOrNil(uu),
			PolicyID:  fixPolicies[i].ID,
			Status:    api.ClaimStatusDraft,
			EventType: api.ClaimEventTypeOther,
		}

		err := fixClaims[i].Create(models.DB)
		if err != nil {
			err = fmt.Errorf("error creating claim fixture ... %+v\n %v",
				fixClaims[i], err.Error())
			return nil, err
		}
	}

	return fixClaims, nil
}
