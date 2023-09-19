package grifts

import (
	"errors"
	"fmt"
	"log"
	"regexp"
	"time"

	"github.com/gobuffalo/grift/grift"
	"github.com/gobuffalo/nulls"
	"github.com/gobuffalo/pop/v6"

	"github.com/silinternational/cover-api/api"

	"github.com/gofrs/uuid"

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

		if count > 1 {
			fmt.Printf("\nINFO: It appears that the grifts have already been run, "+
				"since there are already %v users.\n", count)
			return nil
		}

		return models.DB.Transaction(func(tx *pop.Connection) error {
			assignRiskCategoryCostCenters(tx)

			entityCodes, err := createEntityCodes(tx)
			if err != nil {
				return err
			}

			fixUsers, err := createUserFixtures(tx)
			if err != nil {
				return err
			}

			fixPolicies, err := createPolicyFixtures(tx, fixUsers, entityCodes)
			if err != nil {
				return err
			}

			if _, err := createCategories(tx); err != nil {
				return err
			}

			fixItems, err := createItemFixtures(tx, fixPolicies, fixUsers)
			if err != nil {
				return err
			}

			fixClaims, err := createClaimFixtures(tx, fixPolicies, fixItems)
			if err != nil {
				return err
			}

			err = createLedgerEntryFixtures(tx, fixItems, fixClaims)
			if err != nil {
				return err
			}

			return nil
		})
	})
})

func createUserFixtures(tx *pop.Connection) ([]*models.User, error) {
	userUUIDs := []string{
		"11147366-26b2-4256-b2ab-58c92c3d54c1",
		"11247366-26b2-4256-b2ab-58c92c3d54c2",
		"11347366-26b2-4256-b2ab-58c92c3d54c3",
		"1249902f-c204-4922-b479-57f0ec41eab4",
		"125cf980-e1f0-42d3-b2b0-2e4704159f45",
		"126c63fa-1227-4bea-b34a-416a26c3e076",
		"1276a5a6-971a-403d-8276-c41657bc57c7",
	}

	fixUsers := []*models.User{
		{
			Email:        "clark.kent@example.org",
			FirstName:    "Clark",
			LastName:     "Kent",
			LastLoginUTC: time.Now().UTC().Add(time.Hour * -48),
			StaffID:      nulls.NewString("111111"),
			AppRole:      models.AppRoleSteward,
		},
		{
			Email:         "bruce.wayne@example.org",
			EmailOverride: "batman@example.org",
			FirstName:     "Bruce",
			LastName:      "Wayne",
			LastLoginUTC:  time.Now().UTC().Add(time.Hour * -47),
			StaffID:       nulls.NewString("111222"),
			AppRole:       models.AppRoleSignator,
		},
		{
			Email:         "Jason.Todd@example.org",
			EmailOverride: "robin@example.org",
			FirstName:     "Jason",
			LastName:      "Todd",
			LastLoginUTC:  time.Now().UTC().Add(time.Hour * -46),
			StaffID:       nulls.NewString("111333"),
			AppRole:       models.AppRoleSteward,
		},
		{
			Email:        "jane.eyre@example.org",
			FirstName:    "Jane",
			LastName:     "Eyre",
			LastLoginUTC: time.Now().UTC().Add(time.Hour * -36),
			StaffID:      nulls.NewString("222222"),
		},
		{
			Email:        "carol.danvers@example.org",
			FirstName:    "Carol",
			LastName:     "Danvers",
			IsBlocked:    true,
			LastLoginUTC: time.Now().UTC().Add(time.Hour * -24),
			StaffID:      nulls.NewString("333333"),
		},
		{
			Email:        "denethor.ben.ecthelion@example.org",
			FirstName:    "Denethor",
			LastName:     "Ben Ecthelion",
			LastLoginUTC: time.Now().UTC().Add(time.Hour * -18),
			StaffID:      nulls.NewString("444444"),
		},
		{
			Email:        "john.smith@example.org",
			FirstName:    "John",
			LastName:     "Smith",
			LastLoginUTC: time.Now().UTC().Add(time.Hour * -12),
			StaffID:      nulls.NewString("555555"),
		},
	}

	for i, uu := range userUUIDs {
		fixUsers[i].ID = uuid.FromStringOrNil(uu)
		err := fixUsers[i].Create(tx)
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

		err := tx.Create(&fixUserTokens[i])
		if err != nil {
			err = fmt.Errorf("error creating user token fixture ... %+v\n %v", fixUsers[i], err.Error())
			return fixUsers, err
		}
	}

	return fixUsers, nil
}

func createEntityCodes(tx *pop.Connection) ([]models.EntityCode, error) {
	ec1 := models.EntityCode{
		Active:        true,
		Code:          "XYZ",
		Name:          "XYZ entity code",
		IncomeAccount: "12345",
		ParentEntity:  "ABC",
	}
	if err := ec1.Create(tx); err != nil {
		return []models.EntityCode{}, err
	}
	ec2 := models.EntityCode{
		Active:        true,
		Code:          "ABC",
		Name:          "ABC entity code",
		IncomeAccount: "7890a",
		ParentEntity:  "",
	}
	if err := ec2.Create(tx); err != nil {
		return []models.EntityCode{}, err
	}
	ec3 := models.EntityCode{
		Active:        false,
		Code:          "OLD",
		Name:          "old entity code",
		IncomeAccount: "00000",
		ParentEntity:  "",
	}
	if err := ec3.Create(tx); err != nil {
		return []models.EntityCode{}, err
	}

	return []models.EntityCode{ec1, ec2, ec3}, nil
}

func createPolicyFixtures(tx *pop.Connection, fixUsers []*models.User, entityCodes models.EntityCodes) ([]*models.Policy, error) {
	policyUUIDs := []string{
		"31147366-26b2-4256-b2ab-58c92c3d54cc",
		"31247366-26b2-4256-b2ab-58c92c3d54cc",
		"31347366-26b2-4256-b2ab-58c92c3d54cc",
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
			ID:   uuid.FromStringOrNil(uu),
			Type: api.PolicyTypeHousehold,
		}
		if i < len(entityCodes) {
			fixPolicies[i].Name = fmt.Sprintf("Policy %d", i)
			fixPolicies[i].EntityCodeID = entityCodes[i].ID
			fixPolicies[i].Account = domain.RandomString(6, "0123456789")
			fixPolicies[i].AccountDetail = domain.RandomString(10, "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ")
			fixPolicies[i].CostCenter = domain.RandomString(8, "0123456789")
			fixPolicies[i].Type = api.PolicyTypeTeam
		} else {
			fixPolicies[i].Name = user.LastName + " household"
			fixPolicies[i].EntityCodeID = models.HouseholdEntityID()
			fixPolicies[i].HouseholdID = nulls.NewString(fmt.Sprintf("HID-%s-%s", user.FirstName, user.LastName))
		}

		err := tx.Create(fixPolicies[i])
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

		err := tx.Create(fixPolicyUsers[i])
		if err != nil {
			err = fmt.Errorf("error creating policy users fixture ... %+v\n %v",
				fixPolicyUsers[i], err.Error())
			return []*models.Policy{}, err
		}
	}

	return fixPolicies, nil
}

func createCategories(tx *pop.Connection) ([]uuid.UUID, error) {
	const itemCategoriesSql = `
INSERT INTO "item_categories" ("id", "risk_category_id", "name", "help_text",
	"status", "auto_approve_max", "require_make_model", "premium_factor", "premium_factor_high", "premium_threshold",
	"billing_period", "created_at", "updated_at", "legacy_id")
VALUES
('d4632d64-67b5-4795-a7de-66b95312fa7e', '3be38915-7092-44f2-90ef-26f48214b34f',
	'Computers, tablets, and phones', 'Includes printers, screens, peripherals, and extras',
	'Enabled', 300000, true, 0.02, NULL, NULL,
	12, '2021-08-27 19:46:28', '2021-08-27 19:46:28', 1),
('9c682e38-78fd-475b-9810-3a7f2e9f1fe4', '7bed3c00-23cf-4282-b2b8-da89426cef2f',
	'Clothing', '-',
	'Enabled', 300000, false, 0.02, NULL, NULL,
	12, '2021-08-27 19:46:28', '2021-08-27 19:46:28', 10),
('4b06f087-3fb0-4345-82e8-803645962db0', '3be38915-7092-44f2-90ef-26f48214b34f',
	'Medical', 'Eyewear, insulin pumps, CPAP, prosthetics, and more',
	'Enabled', 300000, true, 0.02, NULL, NULL,
	12, '2021-08-27 19:46:28', '2021-08-27 19:46:28', 11),
('61081c4d-b6e3-47c5-aca7-373fa7d30896', '3be38915-7092-44f2-90ef-26f48214b34f',
	'Photography and recording', 'Includes video, audio, peripherals, and extras',
	'Enabled', 300000, true, 0.02, NULL, NULL,
	12, '2021-08-27 19:46:28', '2021-08-27 19:46:28', 2),
('863a3306-78f9-4aca-add5-0abda3a1ef02', '3be38915-7092-44f2-90ef-26f48214b34f',
	'Other', '-',
	'Enabled', 300000, true, 0.02, NULL, NULL,
	12, '2021-08-27 19:46:28', '2021-08-27 19:46:28', 3),
('faa39da0-981e-4fcf-92fc-2c047fd21f15', '3be38915-7092-44f2-90ef-26f48214b34f',
	'Musical instruments', 'Includes peripherals and extras',
	'Enabled', 300000, true, 0.02, NULL, NULL,
	12, '2021-08-27 19:46:28', '2021-08-27 19:46:28', 4),
('660629ef-ff63-4ace-8263-993897de7d6b', '7bed3c00-23cf-4282-b2b8-da89426cef2f',
	'Appliances and home electronics', 'Washing machines, ovens, theater equipment, and more',
	'Enabled', 300000, false, 0.02, NULL, NULL,
	12, '2021-08-27 19:46:28', '2021-08-27 19:46:28', 5),
('aa304ce5-be3d-45eb-929e-b4575973c0d3', '7bed3c00-23cf-4282-b2b8-da89426cef2f',
	'Home goods', 'Furniture, kitchenware, decorations, linens, and more',
	'Enabled', 300000, false, 0.02, NULL, NULL,
	12, '2021-08-27 19:46:28', '2021-08-27 19:46:28', 6),
('722c03e5-7852-44b9-b86a-af5d63b39d0e', '7bed3c00-23cf-4282-b2b8-da89426cef2f',
	'Field site electronics', 'Solar panels, power systems, antennae, and more',
	'Enabled', 300000, false, 0.02, NULL, NULL,
	12, '2021-08-27 19:46:28', '2021-08-27 19:46:28', 7),
('0f7aa101-bfdb-4a19-a182-c5ff1d16f6b2', '7bed3c00-23cf-4282-b2b8-da89426cef2f',
	'Books and media', 'Books, CDs, DVDs, and more',
	'Enabled', 300000, false, 0.02, NULL, NULL,
	12, '2021-08-27 19:46:28', '2021-08-27 19:46:28', 8),
('036e5315-18ca-4404-8435-72a695f2c9a7', '3be38915-7092-44f2-90ef-26f48214b34f',
	'Travel and recreation', 'Includes suitcases, travel bags, cycling, skating, sports. No motorized vehicles.',
	'Enabled', 300000, true, 0.02, NULL, NULL,
	12, '2021-08-27 19:46:28', '2021-08-27 19:46:28', 9),
('0619a0ba-785e-428d-858c-96d3bd56929a', '3be38915-7092-44f2-90ef-26f48214b34f',
	'Cars and Heavy Vehicles', 'Coverage for Cars and Heavy Vehicles does not include liability and is not intended to fulfill local regulations or compete with local insurance offerings.',
	'Enabled', 8000000, true, 0.0216, 0.0252, 1000000,
	1, '2021-08-27 19:46:28', '2021-08-27 19:46:28', NULL);
`
	if err := tx.RawQuery(itemCategoriesSql).Exec(); err != nil {
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

func createItemFixtures(tx *pop.Connection, fixPolicies []*models.Policy, users []*models.User) ([]*models.Item, error) {
	itemUUIDs := []string{
		"71117366-26b2-4256-b2ab-58c92c3d54c1",
		"71127366-26b2-4256-b2ab-58c92c3d54c2",
		"72217366-26b2-4256-b2ab-58c92c3d54c3",
		"72227366-26b2-4256-b2ab-58c92c3d54c4",
		"73317366-26b2-4256-b2ab-58c92c3d54c5",
		"73327366-26b2-4256-b2ab-58c92c3d54c6",
		"7411f980-e1f0-42d3-b2b0-2e4704159f47",
		"742263fa-1227-4bea-b34a-416a26c3e078",
		"7511a5a6-971a-403d-8276-c41657bc57c9",
		"75227366-26b2-4256-b2ab-58c92c3d54ca",
		"7611902f-c204-4922-b479-57f0ec41eabb",
		"7622f980-e1f0-42d3-b2b0-2e4704159f4c",
		"771163fa-1227-4bea-b34a-416a26c3e07d",
		"7722a5a6-971a-403d-8276-c41657bc57ce",
	}

	if len(itemUUIDs)/2 != len(fixPolicies) {
		err := fmt.Errorf("mismatching count of fixtures in createItemFixtures. "+
			"Expected the number of policy fixtures to be %d, but got %d",
			len(itemUUIDs)/2, len(fixPolicies))
		return []*models.Item{}, err
	}

	itemCats := []models.ItemCategory{}
	if err := tx.All(&itemCats); err != nil {
		return []*models.Item{}, errors.New("error fetching item categories: " + err.Error())
	}

	fixItems := make([]*models.Item, len(itemUUIDs))
	countICats := len(itemCats)

	for i, uu := range itemUUIDs {
		fixItems[i] = &models.Item{
			ID:                uuid.FromStringOrNil(uu),
			Name:              fmt.Sprintf("IName-%d", i),
			CategoryID:        itemCats[i%countICats].ID, // cycle through item categories
			RiskCategoryID:    itemCats[i%countICats].RiskCategoryID,
			InStorage:         false,
			Country:           fmt.Sprintf("ICountry%d", i),
			Description:       fmt.Sprintf("This is the description for item %d.", i),
			PolicyID:          fixPolicies[i/2].ID,
			PolicyUserID:      nulls.NewUUID(users[i/2].ID),
			Make:              fmt.Sprintf("IMake-%d", i),
			Model:             fmt.Sprintf("IModel-%d", i),
			SerialNumber:      fmt.Sprintf("ISN-%d", i),
			CoverageAmount:    50 * (i + 1) * domain.CurrencyFactor, // increments of $50 starting at $50
			CoverageStatus:    api.ItemCoverageStatusApproved,
			PaidThroughDate:   domain.EndOfYear(time.Now().UTC().Year()),
			CoverageStartDate: time.Now().UTC().Add(time.Hour * time.Duration((i+1)*-40)),
		}

		err := tx.Create(fixItems[i])
		if err != nil {
			err = fmt.Errorf("error creating item fixture ... %+v\n %v",
				fixItems[i], err.Error())
			return []*models.Item{}, err
		}
	}

	return fixItems, nil
}

func createClaimFixtures(tx *pop.Connection, fixPolicies []*models.Policy, items []*models.Item) ([]*models.Claim, error) {
	claimUUIDs := []string{
		"023b599d-dd17-4eb9-9895-da462f52526a",
		"1eba86ef-e801-4a9c-a500-fe507040d004",
		"2e1caab9-6ba4-45f5-bb0a-40e9a406e3a0",
		"37a5b5e4-8e52-4276-be3c-ee3d320ad0dc",
		"41176ee9-b6cc-4064-9295-8fbab81d8a99",
	}

	claimItemUUIDs := []string{
		"055c1c87-874c-45ba-afe0-358d35c3ac9a",
		"99941712-2d5f-46a0-ab3c-930d39e65796",
		"b50b5be4-8611-4c25-83a8-0066cec17155",
		"6c1bb8ce-de1c-4d74-9131-dada7ce50a5e",
		"c376aaf6-7788-4ff6-97ae-c8570a2b8b75",
	}

	if len(claimUUIDs) > len(fixPolicies) {
		err := fmt.Errorf("mismatching count of fixtures in createClaimFixtures. "+
			"Expected the number of policy fixtures to be %d, but got %d",
			len(claimUUIDs), len(fixPolicies))
		return nil, err
	}

	if len(claimUUIDs) > len(items)*2 {
		err := fmt.Errorf("mismatching count of fixtures in createClaimFixtures. "+
			"Expected the number of item fixtures to be %d, but got %d",
			len(claimUUIDs), len(items)*2)
		return nil, err
	}

	fixClaims := make([]*models.Claim, len(fixPolicies))

	for i, uu := range claimUUIDs {
		fixClaims[i] = &models.Claim{
			ID:           uuid.FromStringOrNil(uu),
			PolicyID:     fixPolicies[i].ID,
			Status:       api.ClaimStatusDraft,
			IncidentType: api.ClaimIncidentTypeOther,
		}

		err := fixClaims[i].Create(tx)
		if err != nil {
			err = fmt.Errorf("error creating claim fixture ... %+v\n %v",
				fixClaims[i], err.Error())
			return nil, err
		}

		ci := models.ClaimItem{
			ID:           uuid.FromStringOrNil(claimItemUUIDs[i]),
			ClaimID:      fixClaims[i].ID,
			ItemID:       items[i*2].ID,
			PayoutOption: api.PayoutOptionRepair,
		}

		err = ci.Create(tx)
		if err != nil {
			err = fmt.Errorf("error creating claim item fixture ... %+v\n %v",
				ci, err.Error())
			return nil, err
		}

		fixClaims[i].LoadClaimItems(tx, false)
	}

	return fixClaims, nil
}

func createLedgerEntryFixtures(tx *pop.Connection, items []*models.Item, claims []*models.Claim) error {
	// Two entries for Team Policies
	if err := items[0].CreateLedgerEntry(tx, models.LedgerEntryTypeNewCoverage, 1021); err != nil {
		return err
	}

	if err := items[2].CreateLedgerEntry(tx, models.LedgerEntryTypeCoverageChange, 519); err != nil {
		return err
	}

	// Two entries for Household Policies
	if err := items[4].CreateLedgerEntry(tx, models.LedgerEntryTypeNewCoverage, 9876); err != nil {
		return err
	}
	if err := items[5].CreateLedgerEntry(tx, models.LedgerEntryTypeCoverageChange, 1234); err != nil {
		return err
	}

	// Team policy claim
	claims[0].TotalPayout = 2849
	if err := approveClaim(tx, claims[0]); err != nil {
		return err
	}
	if err := claims[0].CreateLedgerEntry(tx); err != nil {
		return err
	}

	// Household policy claim
	claims[4].TotalPayout = 85432
	if err := approveClaim(tx, claims[4]); err != nil {
		return err
	}
	if err := claims[4].CreateLedgerEntry(tx); err != nil {
		return err
	}

	var lEntries models.LedgerEntries
	if err := tx.All(&lEntries); err != nil {
		return err
	}

	// Make one ledger entry already dealt with, in order to
	// create ledger entry reports
	lEntries[0].DateEntered = nulls.NewTime(time.Now().UTC())
	if err := tx.Update(&lEntries[0]); err != nil {
		return err
	}

	return nil
}

func approveClaim(tx *pop.Connection, claim *models.Claim) error {
	claim.Status = api.ClaimStatusApproved
	return tx.Update(claim)
}

func assignRiskCategoryCostCenters(tx *pop.Connection) {
	mobile := models.RiskCategory{
		ID:         models.RiskCategoryMobileID(),
		CostCenter: "MCMC12",
	}
	if err := tx.UpdateColumns(&mobile, "cost_center"); err != nil {
		log.Fatalf("failed to set cost_center on risk category, %s", err)
	}

	stationary := models.RiskCategory{
		ID:         models.RiskCategoryStationaryID(),
		CostCenter: "MPRO12",
	}
	if err := tx.UpdateColumns(&stationary, "cost_center"); err != nil {
		log.Fatalf("failed to set cost_center on risk category, %s", err)
	}
}
