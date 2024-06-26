//go:build development
// +build development

// This build tag ensures that this file will not be included unless
//  the `development` tag is explicitly requested (which should be never)

package models

import (
	"errors"
	"fmt"
	"log"
	"math/rand"
	"testing"
	"time"

	"github.com/gobuffalo/events"
	"github.com/gobuffalo/pop/v6"
	"github.com/stretchr/testify/require"

	"github.com/silinternational/cover-api/storage"

	"github.com/gobuffalo/buffalo"
	"github.com/gobuffalo/nulls"
	"github.com/gofrs/uuid"

	"github.com/silinternational/cover-api/api"
	"github.com/silinternational/cover-api/domain"
)

type FixturesConfig struct {
	NumberOfEntityCodes int
	NumberOfPolicies    int
	ItemsPerPolicy      int
	ClaimsPerPolicy     int
	InvitesPerPolicy    int
	ClaimItemsPerClaim  int
	ClaimFilesPerClaim  int
	UsersPerPolicy      int
	DependentsPerPolicy int
}

// Fixtures hold slices of model objects created for test fixtures
type Fixtures struct {
	Claims
	ClaimHistories
	EntityCodes
	Files
	Items
	ItemCategories
	LedgerEntries
	Policies
	PolicyDependents
	PolicyHistories
	PolicyUsers
	PolicyUserInvites
	UserAccessTokens
	Users
}

// TestBuffaloContext is a buffalo context user in tests
type TestBuffaloContext struct {
	buffalo.DefaultContext
	params map[any]any
}

// Value returns the value associated with the given key in the test context
func (b *TestBuffaloContext) Value(key any) any {
	return b.params[key]
}

// Set sets the value to be associated with the given key in the test context
func (b *TestBuffaloContext) Set(key string, val any) {
	b.params[key] = val
}

// CreateTestContext sets the domain.ContextKeyCurrentUser to the user param in the TestBuffaloContext
func CreateTestContext(user User) buffalo.Context {
	ctx := &TestBuffaloContext{
		params: map[any]any{},
	}
	ctx.Set(domain.ContextKeyCurrentUser, user)
	return ctx
}

// CreateFileFixtures generates any number of file records for testing
// all owned by the same user.
func CreateFileFixtures(tx *pop.Connection, n int, createdByID uuid.UUID) Fixtures {
	_ = storage.CreateS3Bucket()
	files := make(Files, n)
	for i := range files {
		f := File{
			Content:     []byte("GIF87a"),
			Name:        fmt.Sprintf("file_%d.gif", i),
			CreatedByID: createdByID,
		}
		if err := f.Store(tx); err != nil {
			panic(fmt.Sprintf("failed to create file fixture, %s", err))
		}
		files[i] = f
	}

	return Fixtures{
		Files: files,
	}
}

// CreateItemFixtures generates any number of item records for testing
// Uses FixturesConfig fields: NumberOfPolices, DependentsPerPolicy, UsersPerPolicy, ItemsPerPolicy, ClaimsPerPolicy,
// ClaimItemsPerClaim, ClaimFilesPerClaim
func CreateItemFixtures(tx *pop.Connection, config FixturesConfig) Fixtures {
	config.NumberOfPolicies = domain.Max(1, config.NumberOfPolicies)
	config.ItemsPerPolicy = domain.Max(1, config.ItemsPerPolicy)
	numberOfClaimItems := config.ClaimsPerPolicy * config.ClaimItemsPerClaim
	config.ItemsPerPolicy = domain.Max(config.ItemsPerPolicy, numberOfClaimItems)

	fixtures := CreatePolicyFixtures(tx, config)
	policies := fixtures.Policies
	items := make(Items, config.ItemsPerPolicy*config.NumberOfPolicies)
	claims := make(Claims, config.ClaimsPerPolicy*config.NumberOfPolicies)

	categories := CreateCategoryFixtures(tx, len(items)).ItemCategories
	for i := range policies {
		for j := 0; j < config.ItemsPerPolicy; j++ {
			idx := i*config.ItemsPerPolicy + j
			items[idx] = createItemFixture(tx, policies[i].ID, categories[idx].ID)
		}
		policies[i].LoadItems(tx, false)

		for k := 0; k < config.ClaimsPerPolicy; k++ {
			idx := i*config.ClaimsPerPolicy + k
			claims[idx] = createClaimFixture(tx, policies[i], config)
		}
		policies[i].LoadClaims(tx, false)
	}

	fixtures.Items = items
	fixtures.ItemCategories = categories
	fixtures.Claims = claims

	return fixtures
}

func createItemFixture(tx *pop.Connection, policyID uuid.UUID, categoryID uuid.UUID) Item {
	item := Item{
		Name:              randStr(10),
		CategoryID:        categoryID,
		RiskCategoryID:    RiskCategoryStationaryID(),
		Country:           randStr(10),
		Description:       randStr(40),
		PolicyID:          policyID,
		Make:              randStr(10),
		Model:             randStr(10),
		SerialNumber:      randStr(10),
		CoverageAmount:    (int(rand.Int31n(100)) + 100) * domain.CurrencyFactor,
		CoverageStartDate: time.Date(2010, 4, 1, 0, 0, 0, 0, time.UTC),
		// By default, CoverageStatus gets set to Draft by the Item.Create function
	}
	MustCreate(tx, &item)
	return item
}

func UpdateItemStatus(tx *pop.Connection, item Item, status api.ItemCoverageStatus, reason string) Item {
	item.CoverageStatus = status
	item.StatusReason = reason
	if err := tx.Update(&item); err != nil {
		panic("error trying to update item status for test: " + err.Error())
	}
	return item
}

func UpdateClaimStatus(tx *pop.Connection, claim Claim, status api.ClaimStatus, reason string) Claim {
	claim.Status = status
	claim.StatusReason = reason
	if err := tx.Update(&claim); err != nil {
		panic("error trying to update claim status for test: " + err.Error())
	}
	return claim
}

type UpdateClaimItemsParams struct {
	PayoutOption    api.PayoutOption
	FMV             api.Currency
	IsRepairable    bool
	RepairEstimate  api.Currency
	ReplaceEstimate api.Currency
	RepairActual    api.Currency
	ReplaceActual   api.Currency
}

// UpdateClaimItems sets the claim items to a state ready for submission.
func UpdateClaimItems(tx *pop.Connection, claim Claim, params UpdateClaimItemsParams) Claim {
	claim.LoadClaimItems(tx, false)
	for i := range claim.ClaimItems {
		claim.ClaimItems[i].PayoutOption = params.PayoutOption
		claim.ClaimItems[i].IsRepairable = nulls.NewBool(params.IsRepairable)
		claim.ClaimItems[i].RepairEstimate = params.RepairEstimate
		claim.ClaimItems[i].ReplaceEstimate = params.ReplaceEstimate
		claim.ClaimItems[i].RepairActual = params.RepairActual
		claim.ClaimItems[i].ReplaceActual = params.ReplaceActual
		claim.ClaimItems[i].FMV = params.FMV
		if err := tx.Update(&claim.ClaimItems[0]); err != nil {
			panic("error trying to update claim items: " + err.Error())
		}
	}
	return claim
}

// createClaimFixture generates a Claim, a number of ClaimItems, and a number of ClaimFiles
// Uses FixturesConfig fields: ClaimItemsPerClaim, ClaimFilesPerClaim
func createClaimFixture(tx *pop.Connection, policy Policy, config FixturesConfig) Claim {
	if len(policy.Items) < config.ClaimItemsPerClaim {
		panic(fmt.Sprintf("policy fixture must have at least %d items, it only has %d",
			config.ClaimItemsPerClaim, len(policy.Items)))
	}

	claim := Claim{
		PolicyID:            policy.ID,
		IncidentDate:        time.Date(2020, 5, 1, 12, 0, 0, 0, time.UTC),
		IncidentType:        api.ClaimIncidentTypePhysicalDamage,
		IncidentDescription: randStr(25),
		// Status is set to Draft by default
	}
	MustCreate(tx, &claim)

	totalPayout := 0

	claim.ClaimItems = make(ClaimItems, config.ClaimItemsPerClaim)
	for i := range claim.ClaimItems {
		item := policy.Items[i]
		nextPayout := 85 * domain.CurrencyFactor
		claim.ClaimItems[i] = ClaimItem{
			ID:              uuid.UUID{},
			ClaimID:         claim.ID,
			ItemID:          item.ID,
			IsRepairable:    nulls.NewBool(false),
			RepairEstimate:  60 * domain.CurrencyFactor,
			RepairActual:    70 * domain.CurrencyFactor,
			ReplaceEstimate: 100 * domain.CurrencyFactor,
			ReplaceActual:   85 * domain.CurrencyFactor,
			PayoutOption:    api.PayoutOptionRepair,
			PayoutAmount:    api.Currency(nextPayout),
			FMV:             130 * domain.CurrencyFactor,
			City:            randStr(10),
			State:           randStr(2),
			Country:         randStr(10),
		}
		MustCreate(tx, &claim.ClaimItems[i])
		totalPayout += nextPayout
	}

	claim.TotalPayout = api.Currency(totalPayout)
	Must(tx.Update(&claim))

	policyCopy := policy
	policyCopy.LoadMembers(tx, false)
	files := CreateFileFixtures(tx, config.ClaimFilesPerClaim, policyCopy.Members[0].ID).Files
	for _, file := range files {
		if _, err := claim.AttachFile(tx, api.ClaimFileAttachInput{FileID: file.ID}); err != nil {
			panic("failed to attach claim file, " + err.Error())
		}
	}

	claim.LoadClaimItems(tx, true)
	claim.LoadClaimFiles(tx, true)

	return claim
}

// CreateCategoryFixtures generates any number of category records for testing
// even indexed categories are Stationary and odd indexed ones are Mobile
func CreateCategoryFixtures(tx *pop.Connection, n int) Fixtures {
	CreateRiskCategories(tx)

	categories := make(ItemCategories, n)

	for i := range categories {
		if i%2 == 0 {
			categories[i].RiskCategoryID = RiskCategoryStationaryID()
			categories[i].RequireMakeModel = false
		} else {
			categories[i].RiskCategoryID = RiskCategoryMobileID()
			categories[i].RequireMakeModel = true
		}

		categories[i].Key = randStr(10)
		categories[i].Name = randStr(10)
		categories[i].HelpText = randStr(40)
		categories[i].Status = api.ItemCategoryStatusEnabled
		categories[i].AutoApproveMax = 3000 * domain.CurrencyFactor //  $3,000
		categories[i].PremiumFactor = nulls.NewFloat64(0.02)
		categories[i].BillingPeriod = domain.BillingPeriodAnnual
		categories[i].MinimumDeductible = 1
		MustCreate(tx, &categories[i])
	}

	return Fixtures{
		ItemCategories: categories,
	}
}

func createAdminUserWithRole(tx *pop.Connection, role UserAppRole) User {
	user := CreateUserFixtures(tx, 1).Users[0]
	user.AppRole = role
	if err := user.Update(tx); err != nil {
		panic("failed to update user as an admin " + err.Error())
	}
	return user
}

func CreateAdminUsers(tx *pop.Connection) map[UserAppRole]User {
	return map[UserAppRole]User{
		AppRoleSteward:  createAdminUserWithRole(tx, AppRoleSteward),
		AppRoleSignator: createAdminUserWithRole(tx, AppRoleSignator),
	}
}

// CreateUserFixtures generates any number of user records for testing. The access token for
// each user is the same as the user's Email.
func CreateUserFixtures(tx *pop.Connection, n int) Fixtures {
	unique := domain.GetUUID().String()

	users := make(Users, n)
	accessTokenFixtures := make(UserAccessTokens, n)
	for i := range users {
		users[i].Email = fmt.Sprintf("user%d_%s@example.com", i, unique)
		randSuffix := randStr(5)
		users[i].FirstName = "first" + randSuffix
		users[i].LastName = "last" + randSuffix
		users[i].LastLoginUTC = time.Now()
		users[i].StaffID = nulls.NewString(randStr(10))
		users[i].AppRole = AppRoleCustomer
		users[i].City = randStr(10)
		users[i].State = randStr(2)
		users[i].Country = randStr(10)
		MustCreate(tx, &users[i])

		accessTokenFixtures[i].UserID = users[i].ID
		accessTokenFixtures[i].TokenHash = HashAccessToken(users[i].Email)
		accessTokenFixtures[i].ExpiresAt = time.Now().UTC().Add(time.Minute * 60)
		accessTokenFixtures[i].LastUsedAt = nulls.NewTime(time.Now())
		MustCreate(tx, &accessTokenFixtures[i])
	}

	return Fixtures{
		Users:            users,
		UserAccessTokens: accessTokenFixtures,
	}
}

// CreatePolicyFixtures generates any number of policy records and associated policy users
// Uses FixturesConfig fields: NumberOfPolicies, DependentsPerPolicy, UsersPerPolicy
func CreatePolicyFixtures(tx *pop.Connection, config FixturesConfig) Fixtures {
	config.NumberOfPolicies = domain.Max(1, config.NumberOfPolicies)
	config.UsersPerPolicy = domain.Max(1, config.UsersPerPolicy)

	createHouseholdEntity(tx)
	entCodes := make(EntityCodes, config.NumberOfEntityCodes)
	for i := range entCodes {
		entCodes[i] = CreateEntityFixture(tx)
	}

	var policyUsers PolicyUsers
	var policyDependents PolicyDependents
	var policyInvites PolicyUserInvites
	var users Users

	policies := make(Policies, config.NumberOfPolicies)
	for i := range policies {
		policies[i].Name = randStr(20)
		policies[i].Type = api.PolicyTypeHousehold
		policies[i].EntityCodeID = HouseholdEntityID()
		policies[i].HouseholdID = nulls.NewString(randStr(10))
		policies[i].Notes = randStr(20)
		MustCreate(tx, &policies[i])

		f := CreatePolicyUserFixtures(tx, policies[i], config.UsersPerPolicy)
		users = append(users, f.Users...)
		policyUsers = append(policyUsers, f.PolicyUsers...)

		policies[i].LoadMembers(tx, false)

		f = CreatePolicyDependentFixtures(tx, policies[i], config.DependentsPerPolicy)
		policyDependents = append(policyDependents, f.PolicyDependents...)

		policies[i].LoadDependents(tx, false)
	}

	if config.InvitesPerPolicy > 0 {
		f := CreatePolicyUserInviteFixtures(tx, policies, config.InvitesPerPolicy)
		policyInvites = f.PolicyUserInvites
	}

	return Fixtures{
		EntityCodes:       entCodes,
		Policies:          policies,
		PolicyDependents:  policyDependents,
		PolicyUserInvites: policyInvites,
		PolicyUsers:       policyUsers,
		Users:             users,
	}
}

// CreateTeamPolicyFixtures generates any number of policy records (of type Team) and associated policy users
// Uses FixturesConfig fields: NumberOfPolicies, DependentsPerPolicy, UsersPerPolicy
func CreateTeamPolicyFixtures(tx *pop.Connection, config FixturesConfig) Fixtures {
	config.NumberOfPolicies = domain.Max(1, config.NumberOfPolicies)
	config.UsersPerPolicy = domain.Max(1, config.UsersPerPolicy)
	config.NumberOfEntityCodes = domain.Max(1, config.NumberOfEntityCodes)

	entCodes := make(EntityCodes, config.NumberOfEntityCodes)
	for i := range entCodes {
		entCodes[i] = CreateEntityFixture(tx)
	}

	var policyUsers PolicyUsers
	var policyDependents PolicyDependents
	var users Users

	policies := make(Policies, config.NumberOfPolicies)
	for i := range policies {
		policies[i].Name = randStr(20)
		policies[i].Type = api.PolicyTypeTeam

		policies[i].EntityCodeID = entCodes[i%config.NumberOfEntityCodes].ID
		policies[i].Notes = randStr(20)
		policies[i].CostCenter = randStr(6)
		policies[i].Account = randStr(10)
		MustCreate(tx, &policies[i])

		f := CreatePolicyUserFixtures(tx, policies[i], config.UsersPerPolicy)
		users = append(users, f.Users...)
		policyUsers = append(policyUsers, f.PolicyUsers...)

		policies[i].LoadMembers(tx, false)

		f = CreatePolicyDependentFixtures(tx, policies[i], config.DependentsPerPolicy)
		policyDependents = append(policyDependents, f.PolicyDependents...)

		policies[i].LoadDependents(tx, false)
	}
	return Fixtures{
		EntityCodes:      entCodes,
		Policies:         policies,
		PolicyDependents: policyDependents,
		PolicyUsers:      policyUsers,
		Users:            users,
	}
}

func createHouseholdEntity(tx *pop.Connection) {
	var e EntityCode
	if err := tx.Find(&e, HouseholdEntityID()); err != nil {
		if domain.IsOtherThanNoRows(err) {
			panic("database error finding household entity")
		}
		e.ID = HouseholdEntityID()
		e.Code = "MMB"
		e.Name = "Household"
		e.Active = true
		e.IncomeAccount = "40200"
		if err := tx.Create(&e); err != nil {
			panic("failed to create household entity")
		}
	}
}

// CreatePolicyUserFixtures generates any number of user and policy user records
func CreatePolicyUserFixtures(tx *pop.Connection, policy Policy, n int) Fixtures {
	users := CreateUserFixtures(tx, n).Users

	policyUsers := make(PolicyUsers, n)
	for i := range policyUsers {
		policyUsers[i].PolicyID = policy.ID
		policyUsers[i].UserID = users[i].ID
		MustCreate(tx, &policyUsers[i])
	}

	return Fixtures{
		PolicyUsers: policyUsers,
		Users:       users,
	}
}

// CreatePolicyDependentFixtures generates any number of policy dependent records
func CreatePolicyDependentFixtures(tx *pop.Connection, policy Policy, n int) Fixtures {
	policyDependents := make(PolicyDependents, n)
	for i := range policyDependents {
		policyDependents[i].PolicyID = policy.ID
		policyDependents[i].Name = randStr(10) + " " + randStr(10)
		policyDependents[i].Relationship = api.PolicyDependentRelationshipChild
		policyDependents[i].City = randStr(10)
		policyDependents[i].State = randStr(2)
		policyDependents[i].Country = randStr(10)
		policyDependents[i].ChildBirthYear = time.Now().Year() - 18

		if policy.Type == api.PolicyTypeTeam {
			policyDependents[i].Relationship = api.PolicyDependentRelationshipNone
			policyDependents[i].ChildBirthYear = 0
		}

		MustCreate(tx, &policyDependents[i])
	}

	return Fixtures{
		PolicyDependents: policyDependents,
	}
}

func CreateRiskCategories(tx *pop.Connection) {
	if n, err := tx.Count(&RiskCategory{}); err != nil {
		panic("failed to count the risk categories in the database")
	} else if n > 0 {
		return
	}

	riskCategoryMobile := RiskCategory{
		ID:         RiskCategoryMobileID(),
		Name:       "mobile",
		CostCenter: "MOBILE",
	}
	MustCreate(tx, &riskCategoryMobile)

	riskCategoryStationary := RiskCategory{
		ID:         RiskCategoryStationaryID(),
		Name:       "stationary",
		CostCenter: "STATIONARY",
	}
	MustCreate(tx, &riskCategoryStationary)

	riskCategoryVehicle := RiskCategory{
		ID:         RiskCategoryVehicleID(),
		Name:       "vehicle",
		CostCenter: "VEHICLE",
	}
	MustCreate(tx, &riskCategoryVehicle)
}

// CreatePolicyUserInviteFixtures generates any number of policies with one
// primary member and one policy user invite records
func CreatePolicyUserInviteFixtures(tx *pop.Connection, policies Policies, n int) Fixtures {
	if len(policies) == 0 {
		config := FixturesConfig{
			NumberOfPolicies: n,
		}
		fixtures := CreatePolicyFixtures(tx, config)
		policies = fixtures.Policies
	}

	count := len(policies) * n

	invites := make(PolicyUserInvites, count)
	for i := range policies {
		for j := 0; j < n; j++ {
			invNum := i*n + j
			member := policies[i].Members[0]
			invites[invNum].PolicyID = policies[i].ID
			invites[invNum].InviteeName = fmt.Sprintf("Invitee Name%d", invNum)
			invites[invNum].InviterName = member.Name()
			invites[invNum].InviterEmail = member.EmailOfChoice()
			invites[invNum].InviterMessage = fmt.Sprintf("message_%d", invNum)
			invites[invNum].Email = fmt.Sprintf("invitee_%d@example.org", invNum)
			MustCreate(tx, &invites[invNum])
		}
	}

	return Fixtures{
		Policies:          policies,
		PolicyUserInvites: invites,
	}
}

func CreateLedgerFixtures(tx *pop.Connection, config FixturesConfig) Fixtures {
	f := CreateItemFixtures(tx, config)

	user := f.Users[0]
	ctx := CreateTestContext(user)
	f.LedgerEntries = make(LedgerEntries, len(f.Items))
	for i := range f.Items {
		Must(f.Items[i].Approve(ctx, time.Now().UTC()))
		Must(tx.Where("item_id = ?", f.Items[i].ID).First(&f.LedgerEntries[i]))
	}
	return f
}

func CreateLedgerReportFixtures(tx *pop.Connection, reports *LedgerReports) {
	for i := range *reports {
		MustCreate(tx, &(*reports)[i])
	}
}

// MustCreate saves a record to the database with validation. Panics if any error occurs.
func MustCreate(tx *pop.Connection, f Creatable) {
	// Use `create` instead of `tx.Create` to check validation rules
	err := f.Create(tx)
	if err != nil {
		panic(fmt.Sprintf("error creating %T fixture, %s", f, err))
	}
}

func randStr(n int) string {
	const chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = chars[rand.Int63()%int64(len(chars))]
	}
	return string(b)
}

func DestroyAll() {
	// delete all LedgerReports
	var ledgerReports LedgerReports
	destroyTable(&ledgerReports)

	// delete all Files and ClaimFiles
	var files Files
	destroyTable(&files)

	// delete all Ledger Entries
	var ledgerEntries LedgerEntries
	destroyTable(&ledgerEntries)

	// delete all ClaimItems
	var claimItems ClaimItems
	destroyTable(&claimItems)

	// delete all Claims
	var claims Claims
	destroyTable(&claims)

	// delete all Users and UserAccessTokens
	var users Users
	destroyTable(&users)

	// delete all Policies, PolicyUsers, PolicyDependents, PolicyHistory records, and Items
	var policies Policies
	destroyTable(&policies)

	// delete all ItemCategories
	var categories ItemCategories
	destroyTable(&categories)

	// delete all RiskCategories
	var rCats RiskCategories
	destroyTable(&rCats)

	// delete all Notifications
	var ns Notifications
	destroyTable(&ns)

	// delete all Invites
	var invites PolicyUserInvites
	destroyTable(&invites)

	// delete all EntityCodes
	var entityCodes EntityCodes
	destroyTable(&entityCodes)
}

func destroyTable(i any) {
	if err := DB.All(i); err != nil {
		panic(err.Error())
	}
	if err := DB.Destroy(i); err != nil {
		panic(err.Error())
	}
}

func InsertTestData() {
	insertServiceUser()
}

func insertServiceUser() {
	serviceUser := User{
		ID:        uuid.FromStringOrNil(ServiceUserID),
		Email:     "service_user@example.com",
		FirstName: "Service",
		LastName:  "User",
	}
	if err := DB.Create(&serviceUser); err != nil {
		panic("failed to insert service user: " + err.Error())
	}
}

// RegisterEventDetector is a helper method for testing if events are triggered
// call with the kind of event and a pointer to a boolean and it'll update the boolean
// to true if the event kind is detected. A 10ms delay may be required between emit and detection
func RegisterEventDetector(kind string, detected *bool) (events.DeleteFn, error) {
	return events.Listen(func(e events.Event) {
		if e.Kind == kind {
			*detected = true
		}
	})
}

// CreatePolicyHistoryFixtures generates a Policy with three Items each with
// four PolicyHistory entries as follows:
// - CoverageStatus/Create  [not included because not update]
// - Name/Update [not included because not on CoverageStatus field]
// - CoverageStatus/Update [could be included, if date is recent]
// - CoverageStatus/Update [could be included, if date is recent]
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

// CreateClaimHistoryFixtures generates a Policy with three Claims each
// with four ClaimHistory entries as follows:
// - Status/Create  [not included as "recent" because not update]
// - ReferenceNumber/Update [not included as "recent" because not on Status field]
// - Status/Update [could be included, if date is recent]
// - Status/Update [could be included, if date is recent]
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

	for i := range cHistories {
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

func CreateEntityFixture(tx *pop.Connection) EntityCode {
	code := randStr(8)
	e := EntityCode{
		Code:   code,
		Name:   code + " Name",
		Active: true,
	}
	MustCreate(tx, &e)
	return e
}

// ConvertPolicyType converts a household policy to a Team policy. Creates a new Entity
// for the policy.
func ConvertPolicyType(tx *pop.Connection, policy Policy) Policy {
	policy.Type = api.PolicyTypeTeam
	policy.CostCenter = "CC1234"
	policy.Account = "111222"
	policy.AccountDetail = "Acct Detail"
	policy.EntityCodeID = CreateEntityFixture(tx).ID

	if err := tx.Update(&policy); err != nil {
		panic("error converting policy to Team, " + err.Error())
	}

	return policy
}

// CreateStrikeFixtures generates any number of strike records per policy provided
func CreateStrikeFixtures(tx *pop.Connection, policies Policies, dates [][]*time.Time) Strikes {
	if len(dates) > len(policies) {
		log.Panicf("Not enough policies (%d) for the dates provided (%d)", len(policies), len(dates))
	}

	strikes := Strikes{}

	for i := range dates {
		for j := range dates[i] {
			strike := Strike{
				Description: fmt.Sprintf("Strike %d for Policy %d", j, i),
				PolicyID:    policies[i].ID,
			}
			MustCreate(tx, &strike)

			if dates[i][j] != nil {
				// Merely calling the db.Update function doesn't overwrite the created_at value
				q := tx.RawQuery("Update strikes SET created_at = ? WHERE id = ?", dates[i][j], strike.ID)
				Must(q.Exec())
			}
			strikes = append(strikes, strike)
		}
	}
	return strikes
}

func Must(err error) {
	if err != nil {
		panic(err.Error())
	}
}

// AssertSameAppError verifies that the actual error contains an AppError and that the key and category match expected
func AssertSameAppError(t *testing.T, expected api.AppError, actual error) {
	require.Error(t, actual, "error is nil")
	var appErr *api.AppError
	require.True(t, errors.As(actual, &appErr),
		"error does not contain an api.AppError, message: %s", actual.Error())
	require.Equal(t, expected.Key, appErr.Key,
		"error key does not match, message: %s", actual.Error())
	require.Equal(t, expected.Category, appErr.Category,
		"error category does not match, message: %s", actual.Error())
}

func CreateUniqueInvite(createdAt time.Time, id uuid.UUID) PolicyUserInvite {
	randomStr := randStr(5)
	return PolicyUserInvite{
		ID:           domain.GetUUID(),
		PolicyID:     id,
		Email:        "test_user" + randomStr + "@example.org",
		InviteeName:  "Test User" + randomStr,
		InviterName:  "Tester" + randomStr,
		InviterEmail: "test" + randomStr + "@example.org",
		CreatedAt:    createdAt,
	}
}
