package models

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"net/http"
	"reflect"

	"github.com/go-playground/validator/v10"
	"github.com/gobuffalo/events"
	"github.com/gobuffalo/nulls"
	"github.com/gobuffalo/pop/v5"
	"github.com/gofrs/uuid"
	"github.com/jackc/pgconn"
	"github.com/jackc/pgerrcode"

	"github.com/silinternational/cover-api/api"
	"github.com/silinternational/cover-api/domain"
)

// DB is a connection to the database to be used throughout the application.
var DB *pop.Connection

const tokenBytes = 32

type (
	Permission  int
	SubResource string
)

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
	IsActorAllowedTo(*pop.Connection, User, Permission, SubResource, *http.Request) bool
}

type Createable interface {
	Create(tx *pop.Connection) error
}

type FieldUpdate struct {
	FieldName string
	OldValue  string
	NewValue  string
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

	// register custom validators for custom types
	for tag, vFunc := range fieldValidators {
		if err = mValidate.RegisterValidation(tag, vFunc, false); err != nil {
			log.Fatal(fmt.Errorf("failed to register validation for %s: %s", tag, err))
		}
	}

	// register struct-level validators
	mValidate.RegisterStructValidation(claimStructLevelValidation, Claim{})
	mValidate.RegisterStructValidation(claimItemStructLevelValidation, ClaimItem{})
	mValidate.RegisterStructValidation(policyStructLevelValidation, Policy{})
	mValidate.RegisterStructValidation(itemStructLevelValidation, Item{})
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
func CurrentUser(ctx context.Context) User {
	user, _ := ctx.Value(domain.ContextKeyCurrentUser).(User)
	domain.NewExtra(ctx, "user_id", user.ID)
	return user
}

// Tx retrieves the database transaction from the context
func Tx(ctx context.Context) *pop.Connection {
	tx, ok := ctx.Value("tx").(*pop.Connection)
	if !ok {
		return DB
	}
	return tx
}

func fieldByName(i interface{}, name ...string) reflect.Value {
	if len(name) < 1 {
		return reflect.Value{}
	}
	f := reflect.ValueOf(i).Elem().FieldByName(name[0])
	if !f.IsValid() {
		return fieldByName(i, name[1:]...)
	}
	return f
}

func create(tx *pop.Connection, m interface{}) error {
	uuidField := fieldByName(m, "ID")
	if uuidField.IsValid() && uuidField.Interface().(uuid.UUID).Version() == 0 {
		uuidField.Set(reflect.ValueOf(domain.GetUUID()))
	}

	valErrs, err := tx.ValidateAndCreate(m)
	if err != nil {
		return appErrorFromDB(err, api.ErrorCreateFailure)
	}

	if valErrs.HasAny() {
		return api.NewAppError(
			errors.New(flattenPopErrors(valErrs)),
			api.ErrorValidation,
			api.CategoryUser,
		)
	}
	return nil
}

func appErrorFromDB(err error, defaultKey api.ErrorKey) error {
	if err == nil {
		return nil
	}

	appErr := api.NewAppError(err, defaultKey, api.CategoryInternal)

	if !domain.IsOtherThanNoRows(err) {
		appErr.Category = api.CategoryUser
		appErr.Key = api.ErrorNoRows
		return appErr
	}

	var pgError *pgconn.PgError
	if errors.As(err, &pgError) {
		appErr.Err = fmt.Errorf("%w Detail: %s", pgError, pgError.Detail)

		switch pgError.Code {
		case pgerrcode.ForeignKeyViolation:
			appErr.Key = api.ErrorForeignKeyViolation
			appErr.Category = api.CategoryUser
		case pgerrcode.UniqueViolation:
			appErr.Key = api.ErrorUniqueKeyViolation
			appErr.Category = api.CategoryUser
		}
	}

	return appErr
}

func save(tx *pop.Connection, m interface{}) error {
	uuidField := fieldByName(m, "ID")
	if uuidField.IsValid() && uuidField.Interface().(uuid.UUID).Version() == 0 {
		uuidField.Set(reflect.ValueOf(domain.GetUUID()))
	}

	valErrs, err := tx.ValidateAndSave(m)
	if err != nil {
		return api.NewAppError(err, api.ErrorSaveFailure, api.CategoryInternal)
	}

	if valErrs != nil && valErrs.HasAny() {
		return api.NewAppError(
			errors.New(flattenPopErrors(valErrs)),
			api.ErrorValidation,
			api.CategoryUser,
		)
	}

	return nil
}

func update(tx *pop.Connection, m interface{}) error {
	valErrs, err := tx.ValidateAndUpdate(m)
	if err != nil {
		return appErrorFromDB(err, api.ErrorUpdateFailure)
	}

	if valErrs.HasAny() {
		return api.NewAppError(
			errors.New(flattenPopErrors(valErrs)),
			api.ErrorValidation,
			api.CategoryUser,
		)
	}
	return nil
}

// This can include an event payload, which is a map[string]interface{}
func emitEvent(e events.Event) {
	if err := events.Emit(e); err != nil {
		domain.ErrLogger.Printf("error emitting event %s ... %v", e.Kind, err)
	}
}

func addFile(tx *pop.Connection, m interface{}, fileID uuid.UUID) error {
	var f File

	if err := f.Find(tx, fileID); err != nil {
		return err
	}

	fileField := fieldByName(m, "FileID", "PhotoFileID")
	if !fileField.IsValid() {
		return errors.New("error identifying File ID field")
	}

	oldID := fileField.Interface().(nulls.UUID)
	fileField.Set(reflect.ValueOf(nulls.NewUUID(f.ID)))
	idField := fieldByName(m, "ID")
	if !idField.IsValid() {
		return errors.New("error identifying ID field")
	}

	if err := tx.Update(m); err != nil {
		return fmt.Errorf("failed to update the file ID column, %s", err)
	}

	if err := f.SetLinked(tx); err != nil {
		return fmt.Errorf("error marking file %s as linked, %s", f.ID, err)
	}

	if !oldID.Valid {
		return nil
	}

	oldFile := File{ID: oldID.UUID}
	if err := oldFile.ClearLinked(tx); err != nil {
		domain.ErrLogger.Printf("error marking old file %s as unlinked, %s", oldFile.ID, err)
	}

	return nil
}
