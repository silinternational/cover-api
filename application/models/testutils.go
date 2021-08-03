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

// UserFixtures hold slices of model objects created with new user fixtures
type UserFixtures struct {
	Users
	UserAccessTokens
}

// CreateUserFixtures generates any number of user records for testing. The access token for
// each user is the same as the user's Email.
func CreateUserFixtures(tx *pop.Connection, n int) UserFixtures {
	unique := domain.GetUUID().String()

	users := make(Users, n)
	accessTokenFixtures := make(UserAccessTokens, n)
	for i := range users {
		users[i].Email = fmt.Sprintf("user%d_%s@example.com", i, unique)
		iStr := strconv.Itoa(i)
		users[i].FirstName = "first" + iStr
		users[i].LastName = "last" + iStr
		users[i].LastLoginUTC = time.Now()
		users[i].StaffID = strconv.Itoa(rand.Int())
		MustCreate(tx, &users[i])

		accessTokenFixtures[i].UserID = users[i].ID
		accessTokenFixtures[i].AccessToken = HashClientIdAccessToken(users[i].Email)
		accessTokenFixtures[i].ExpiresAt = time.Now().Add(time.Minute * 60)
		accessTokenFixtures[i].LastUsedAt = nulls.NewTime(time.Now())
		MustCreate(tx, &accessTokenFixtures[i])
	}

	return UserFixtures{
		Users:            users,
		UserAccessTokens: accessTokenFixtures,
	}
}

// MustCreate saves a record to the database with validation. Panics if any error occurs.
func MustCreate(tx *pop.Connection, f interface{}) {
	// Use `create` instead of `tx.Create` to check validation rules
	err := create(tx, f)
	if err != nil {
		panic(fmt.Sprintf("error creating %T fixture, %s", f, err))
	}
}
