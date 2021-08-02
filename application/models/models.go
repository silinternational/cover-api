package models

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log"
	"net/http"

	"github.com/go-playground/validator/v10"
	"github.com/gobuffalo/buffalo"
	"github.com/gobuffalo/pop/v5"
	"github.com/gofrs/uuid"

	"github.com/gobuffalo/validate/v3"

	"github.com/silinternational/riskman-api/domain"
)

// DB is a connection to the database to be used throughout the application.
var DB *pop.Connection

// Model validation tool
var mValidate *validator.Validate

const tokenBytes = 32

type Permission int

const (
	PermissionView Permission = iota
	PermissionList
	PermissionCreate
	PermissionUpdate
	PermissionDelete
	PermissionDenied
)

type Authable interface {
	GetID() uuid.UUID
	FindByID(*pop.Connection, uuid.UUID) error
	IsActorAllowedTo(User, Permission, string, *http.Request) bool
}

func init() {
	var err error
	env := domain.Env.GoEnv
	DB, err = pop.Connect(env)
	if err != nil {
		domain.ErrLogger.Printf("error connecting to database ... %v", err)
		log.Fatal(err)
	}
	pop.Debug = env == "development"

	// Just make sure we can use the crypto/rand library on our system
	if _, err = getRandomToken(); err != nil {
		log.Fatal(fmt.Errorf("error using crypto/rand ... %v", err))
	}

	// initialize model validation library
	mValidate = validator.New()
}

func getRandomToken() (string, error) {
	rb := make([]byte, tokenBytes)

	_, err := rand.Read(rb)
	if err != nil {
		return "", err
	}

	return base64.URLEncoding.EncodeToString(rb), nil
}

// CurrentUser retrieves the current user from the context.
func CurrentUser(c buffalo.Context) User {
	user, _ := c.Value(domain.ContextKeyCurrentUser).(User)
	domain.NewExtra(c, "user_id", user.ID)
	return user
}

func validateModel(m interface{}) *validate.Errors {
	verrs := validate.NewErrors()

	if err := mValidate.Struct(m); err != nil {
		for _, err := range err.(validator.ValidationErrors) {
			verrs.Add(err.StructNamespace(), err.Error())
		}
	}
	return verrs
}

// Tx retrieves the database transaction from the context
func Tx(ctx context.Context) *pop.Connection {
	tx, ok := ctx.Value("tx").(*pop.Connection)
	if !ok {
		return DB
	}
	return tx
}
