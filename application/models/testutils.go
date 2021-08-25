// +build development

// This build tag ensures that this file will not be included unless
//  the `development` tag is explicitly requested (which should be never)

package models

import (
	"fmt"
	"math/rand"
	"strconv"
	"time"

	"github.com/silinternational/cover-api/storage"

	"github.com/gobuffalo/buffalo"
	"github.com/gobuffalo/nulls"
	"github.com/gobuffalo/pop/v5"
	"github.com/gofrs/uuid"

	"github.com/silinternational/cover-api/api"
	"github.com/silinternational/cover-api/domain"
)

type FixturesConfig struct {
	NumberOfPolicies    int
	ItemsPerPolicy      int
	ClaimsPerPolicy     int
	ClaimItemsPerClaim  int
	UsersPerPolicy      int
	DependentsPerPolicy int
}

// Fixtures hold slices of model objects created for test fixtures
type Fixtures struct {
	Files
	Items
	ItemCategories
	Policies
	PolicyDependents
	PolicyUsers
	UserAccessTokens
	Users
}

// TestBuffaloContext is a buffalo context user in tests
type TestBuffaloContext struct {
	buffalo.DefaultContext
	params map[interface{}]interface{}
}

// Value returns the value associated with the given key in the test context
func (b *TestBuffaloContext) Value(key interface{}) interface{} {
	return b.params[key]
}

// Set sets the value to be associated with the given key in the test context
func (b *TestBuffaloContext) Set(key string, val interface{}) {
	b.params[key] = val
}

// CreateTestContext sets the domain.ContextKeyCurrentUser to the user param in the TestBuffaloContext
func CreateTestContext(user User) buffalo.Context {
	ctx := &TestBuffaloContext{
		params: map[interface{}]interface{}{},
	}
	ctx.Set(domain.ContextKeyCurrentUser, user)
	return ctx
}

// CreateFileFixtures generates any number of file records for testing
//  all owned by the same user.
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
// Uses FixturesConfig fields: NumberOfPolices, DependentsPerPolicy, UsersPerPolicy, ItemsPerPolicy
func CreateItemFixtures(tx *pop.Connection, config FixturesConfig) Fixtures {
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
			claims[idx] = createClaimFixture(tx, policies[i].ID, config)
		}
		policies[i].LoadClaims(tx, false)
	}

	fixtures.Items = items
	fixtures.ItemCategories = categories

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
		CoverageAmount:    int(rand.Int31n(100)) + 100,
		PurchaseDate:      time.Date(2010, 4, 1, 12, 0, 0, 0, time.UTC),
		CoverageStartDate: time.Date(2010, 4, 1, 13, 0, 0, 0, time.UTC),
		CoverageStatus:    api.ItemCoverageStatusApproved,
	}
	MustCreate(tx, &item)
	return item
}

func createClaimFixture(tx *pop.Connection, policyID uuid.UUID, config FixturesConfig) Claim {
	claim := Claim{
		PolicyID:         policyID,
		EventDate:        time.Date(2020, 5, 1, 12, 0, 0, 0, time.UTC),
		EventType:        api.ClaimEventTypeImpact,
		EventDescription: randStr(25),
		Status:           api.ClaimStatusPending,
	}
	MustCreate(tx, &claim)

	icFixtures := CreateCategoryFixtures(tx, config.ClaimItemsPerClaim)

	claim.ClaimItems = make(ClaimItems, config.ClaimItemsPerClaim)
	for i := range claim.ClaimItems {
		item := createItemFixture(tx, policyID, icFixtures.ItemCategories[i].ID)
		claim.ClaimItems[i] = ClaimItem{
			ID:              uuid.UUID{},
			ClaimID:         claim.ID,
			ItemID:          item.ID,
			Status:          api.ClaimItemStatusPending,
			IsRepairable:    false,
			RepairEstimate:  0,
			RepairActual:    0,
			ReplaceEstimate: 10000,
			ReplaceActual:   8500,
			PayoutOption:    "",
			PayoutAmount:    8500,
			FMV:             13000,
			ReviewDate:      nulls.Time{},
			ReviewerID:      nulls.UUID{},
		}
		MustCreate(tx, &claim.ClaimItems[i])
	}

	return claim
}

// CreateCategoryFixtures generates any number of category records for testing
func CreateCategoryFixtures(tx *pop.Connection, n int) Fixtures {
	CreateRiskCategories(tx)

	categories := make(ItemCategories, n)
	for i := range categories {
		categories[i].RiskCategoryID = nulls.NewUUID(RiskCategoryMobileID())
		categories[i].Name = randStr(10)
		categories[i].HelpText = randStr(40)
		categories[i].Status = api.ItemCategoryStatusEnabled
		categories[i].AutoApproveMax = 500
		MustCreate(tx, &categories[i])
	}

	return Fixtures{
		ItemCategories: categories,
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
		iStr := strconv.Itoa(i)
		users[i].FirstName = "first" + iStr
		users[i].LastName = "last" + iStr
		users[i].LastLoginUTC = time.Now()
		users[i].StaffID = randStr(10)
		users[i].AppRole = AppRoleUser
		MustCreate(tx, &users[i])

		accessTokenFixtures[i].UserID = users[i].ID
		accessTokenFixtures[i].TokenHash = HashClientIdAccessToken(users[i].Email)
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
	var policyUsers PolicyUsers
	var policyDependents PolicyDependents
	var users Users

	policies := make(Policies, config.NumberOfPolicies)
	for i := range policies {
		policies[i].Type = api.PolicyTypeHousehold
		policies[i].Account = randStr(10)
		policies[i].EntityCode = randStr(10)
		policies[i].CostCenter = randStr(10)
		policies[i].HouseholdID = randStr(10)
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
		Policies:         policies,
		PolicyDependents: policyDependents,
		PolicyUsers:      policyUsers,
		Users:            users,
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
		policyDependents[i].Name = randStr(10)
		policyDependents[i].Relationship = api.PolicyDependentRelationshipChild
		policyDependents[i].Location = randStr(10)
		policyDependents[i].ChildBirthYear = time.Now().Year() - 18
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
		ID:        RiskCategoryMobileID(),
		Name:      "mobile",
		PolicyMax: 25000,
	}
	MustCreate(tx, &riskCategoryMobile)

	riskCategoryStationary := RiskCategory{
		ID:        RiskCategoryStationaryID(),
		Name:      "stationary",
		PolicyMax: 25000,
	}
	MustCreate(tx, &riskCategoryStationary)
}

// MustCreate saves a record to the database with validation. Panics if any error occurs.
func MustCreate(tx *pop.Connection, f interface{}) {
	// Use `create` instead of `tx.Create` to check validation rules
	err := create(tx, f)
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
	// delete all Users and UserAccessTokens
	var users Users
	destroyTable(&users)

	// delete all Claims and ClaimItems
	var claimItems ClaimItems
	destroyTable(&claimItems)
	var claims Claims
	destroyTable(&claims)

	// delete all Policies, PolicyUsers, PolicyDependents, PolicyHistory records, and Items
	var policies Policies
	destroyTable(&policies)

	// delete all ItemCategories
	var categories ItemCategories
	destroyTable(&categories)

	// delete all RiskCategories
	var rCats RiskCategories
	destroyTable(&rCats)
}

func destroyTable(i interface{}) {
	if err := DB.All(i); err != nil {
		panic(err.Error())
	}
	if err := DB.Destroy(i); err != nil {
		panic(err.Error())
	}
}
