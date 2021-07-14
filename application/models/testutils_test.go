package models

import (
	"fmt"
	"reflect"
	"strconv"

	"github.com/gobuffalo/pop/v5"

	"github.com/silinternational/riskman-api/domain"
)

// mustCreate saves a record to the database. Panics if any error occurs.
func mustCreate(tx *pop.Connection, f interface{}) {
	value := reflect.ValueOf(f)

	if value.Type().Kind() != reflect.Ptr {
		panic("mustCreate requires a pointer")
	}

	uuidField := value.Elem().FieldByName("UUID")
	if uuidField.IsValid() {
		uuidField.Set(reflect.ValueOf(domain.GetUUID()))
	}

	err := tx.Create(f)
	if err != nil {
		panic(fmt.Sprintf("error creating %T fixture, %s", f, err))
	}
}

type UserFixtures struct {
	Users
}

// createUserFixtures generates any number of user records for testing.
func createUserFixtures(tx *pop.Connection, n int) UserFixtures {

	unique := domain.GetUUID().String()

	users := make(Users, n)
	for i := range users {
		users[i].Email = unique + "_user" + strconv.Itoa(i) + "@example.com"
		mustCreate(tx, &users[i])
	}

	return UserFixtures{
		Users: users,
	}
}
