// +build development

// This build tag ensures that this file will not be included unless
//  the `development` tag is explicitly requested (which should be never)

package models

import (
	"fmt"
	"math/rand"
	"strconv"
	"time"

	"github.com/gobuffalo/buffalo"
	"github.com/gobuffalo/nulls"
	"github.com/gobuffalo/pop/v5"
	"github.com/silinternational/riskman-api/domain"
)

type FixturesConfig struct {
	ItemsPerPolicy      int
	NumberOfPolicies    int
	UsersPerPolicy      int
	DependentsPerPolicy int
}

// Fixtures hold slices of model objects created for test fixtures
type Fixtures struct {
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

// CreateItemFixtures generates any number of item records for testing
// Uses FixturesConfig fields: Polices, DependentsPerPolicy, UsersPerPolicy, ItemsPerPolicy
func CreateItemFixtures(tx *pop.Connection, config FixturesConfig) Fixtures {
	fixtures := CreatePolicyFixtures(tx, config)
	policies := fixtures.Policies
	items := make(Items, config.ItemsPerPolicy*config.NumberOfPolicies)

	categories := CreateCategoryFixtures(tx, len(items)).ItemCategories
	for i := range policies {
		for j := 0; j < config.ItemsPerPolicy; j++ {
			idx := i*config.ItemsPerPolicy + j
			items[idx].Name = randStr(10)
			items[idx].CategoryID = categories[idx].ID
			items[idx].Country = randStr(10)
			items[idx].Description = randStr(40)
			items[idx].PolicyID = policies[i].ID
			items[idx].Make = randStr(10)
			items[idx].Model = randStr(10)
			items[idx].SerialNumber = randStr(10)
			items[idx].CoverageAmount = int(rand.Int31n(100)) + 100
			items[idx].PurchaseDate = time.Date(2010, 4, 1, 12, 0, 0, 0, time.UTC)
			items[idx].CoverageStartDate = items[idx].PurchaseDate
			items[idx].CoverageStatus = ItemCoverageStatusApproved
			MustCreate(tx, &items[idx])
		}
	}

	fixtures.Items = items

	return fixtures
}

// CreateCategoryFixtures generates any number of category records for testing
func CreateCategoryFixtures(tx *pop.Connection, n int) Fixtures {
	CreateRiskCategories(tx)

	categories := make(ItemCategories, n)
	for i := range categories {
		categories[i].RiskCategoryID = RiskCategoryMobileID()
		categories[i].Name = randStr(10)
		categories[i].HelpText = randStr(40)
		categories[i].Status = ItemCategoryStatusEnabled
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
		MustCreate(tx, &users[i])

		accessTokenFixtures[i].UserID = users[i].ID
		accessTokenFixtures[i].TokenHash = HashClientIdAccessToken(users[i].Email)
		accessTokenFixtures[i].ExpiresAt = time.Now().Add(time.Minute * 60)
		accessTokenFixtures[i].LastUsedAt = nulls.NewTime(time.Now())
		MustCreate(tx, &accessTokenFixtures[i])
	}

	return Fixtures{
		Users:            users,
		UserAccessTokens: accessTokenFixtures,
	}
}

// CreatePolicyFixtures generates any number of policy records and associated policy users
// Uses FixturesConfig fields: Polices, DependentsPerPolicy, UsersPerPolicy
func CreatePolicyFixtures(tx *pop.Connection, config FixturesConfig) Fixtures {
	var policyUsers PolicyUsers
	var policyDependents PolicyDependents
	var users Users

	policies := make(Policies, config.NumberOfPolicies)
	for i := range policies {
		policies[i].Type = PolicyTypeHousehold
		policies[i].Account = randStr(10)
		policies[i].EntityCode = randStr(10)
		policies[i].CostCenter = randStr(10)
		policies[i].HouseholdID = randStr(10)
		MustCreate(tx, &policies[i])

		f := CreatePolicyUserFixtures(tx, policies[i], config.UsersPerPolicy)
		users = append(users, f.Users...)
		policyUsers = append(policyUsers, f.PolicyUsers...)

		if err := policies[i].LoadMembers(tx, false); err != nil {
			panic("failed to load members on policy " + policies[i].ID.String())
		}

		f = CreatePolicyDependentFixtures(tx, policies[i], config.DependentsPerPolicy)
		policyDependents = append(policyDependents, f.PolicyDependents...)

		if err := policies[i].LoadDependents(tx, false); err != nil {
			panic("failed to load dependents on policy " + policies[i].ID.String())
		}
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
		policyDependents[i].Relationship = PolicyDependentRelationshipChild
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
