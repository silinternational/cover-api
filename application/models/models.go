package models

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"reflect"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/gobuffalo/events"
	"github.com/gobuffalo/nulls"
	"github.com/gobuffalo/pop/v6"
	"github.com/gofrs/uuid"
	"github.com/jackc/pgconn"
	"github.com/jackc/pgerrcode"

	"github.com/silinternational/cover-api/api"
	"github.com/silinternational/cover-api/domain"
)

// DB is a connection to the database to be used throughout the application.
var DB *pop.Connection

const tokenBytes = 32

const uuidNamespaceString = "89cbb2e8-5832-11ec-af6a-95df0dd7b2c5"

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

	ClaimStatusChangeReturnedToDraft = "Returned to draft by "
	ClaimStatusChangeReview1         = "Submitted for first review"
	ClaimStatusChangeRevisions       = "Revisions requested by "
	ClaimStatusChangeReceipt         = "Receipt requested by "
	ClaimStatusChangeReview2         = "Submitted for second review by "
	ClaimStatusChangeReview3         = "Submitted for payout approval by "
	ClaimStatusChangeApproved        = "Approved by "
	ClaimStatusChangeDenied          = "Denied by "

	ItemStatusChangeSubmitted    = "Submitted for approval"
	ItemStatusChangeAutoApproved = "Auto approved"
	ItemStatusChangeApproved     = "Approved by "
	ItemStatusChangeRevisions    = "Revisions requested by "
	ItemStatusChangeDenied       = "Denied by "
	ItemStatusChangeInactivated  = "Deactivated by "

	FieldClaimPolicyID            = "PolicyID"
	FieldClaimReferenceNumber     = "ReferenceNumber"
	FieldClaimIncidentDate        = "IncidentDate"
	FieldClaimIncidentType        = "IncidentType"
	FieldClaimIncidentDescription = "IncidentDescription"
	FieldClaimStatus              = "Status"
	FieldClaimReviewDate          = "ReviewDate"
	FieldClaimReviewerID          = "ReviewerID"
	FieldClaimPaymentDate         = "PaymentDate"
	FieldClaimTotalPayout         = "TotalPayout"
	FieldClaimStatusReason        = "StatusReason"
	FieldClaimCity                = "City"
	FieldClaimState               = "State"
	FieldClaimCountry             = "Country"

	FieldClaimItemItemID          = "ItemID"
	FieldClaimItemIsRepairable    = "IsRepairable"
	FieldClaimItemRepairEstimate  = "RepairEstimate"
	FieldClaimItemRepairActual    = "RepairActual"
	FieldClaimItemReplaceEstimate = "ReplaceEstimate"
	FieldClaimItemReplaceActual   = "ReplaceActual"
	FieldClaimItemPayoutOption    = "PayoutOption"
	FieldClaimItemPayoutAmount    = "PayoutAmount"
	FieldClaimItemFMV             = "FMV"
	FieldClaimItemReviewDate      = "ReviewDate"
	FieldClaimItemReviewerID      = "ReviewerID"
	FieldClaimItemLocation        = "Location"

	FieldItemName              = "Name"
	FieldItemCategoryID        = "CategoryID"
	FieldItemRiskCategoryID    = "RiskCategoryID"
	FieldItemInStorage         = "InStorage"
	FieldItemCountry           = "Country"
	FieldItemDescription       = "Description"
	FieldItemPolicyDependentID = "PolicyDependentID"
	FieldItemPolicyUserID      = "PolicyUserID"
	FieldItemMake              = "Make"
	FieldItemModel             = "Model"
	FieldItemSerialNumber      = "SerialNumber"
	FieldItemCoverageAmount    = "CoverageAmount"
	FieldItemCoverageStatus    = "CoverageStatus"
	FieldItemCoverageStartDate = "CoverageStartDate"
	FieldItemPaidThroughYear   = "PaidThroughYear"
	FieldItemStatusReason      = "CoverageStatusReason"
)

var uuidNamespace uuid.UUID

type Authable interface {
	GetID() uuid.UUID
	FindByID(*pop.Connection, uuid.UUID) error
	IsActorAllowedTo(*pop.Connection, User, Permission, SubResource, *http.Request) bool
}

type Creatable interface {
	Create(*pop.Connection) error
}

type Updatable interface {
	Update(*pop.Connection) error
}

type Person interface {
	GetID() uuid.UUID
	GetLocation() Location
	GetName() Name
}

type Location struct {
	City    string
	State   string
	Country string
}

func (l Location) String() string {
	s := l.City + ", "
	if l.State != "" {
		s += l.State
	}
	if l.Country != "" {
		s += " " + l.Country
	}
	return strings.Trim(s, " ,")
}

type Name struct {
	First string
	Last  string
}

func (n Name) String() string {
	return strings.TrimSpace(strings.TrimSpace(n.First) + " " + strings.TrimSpace(n.Last))
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
	mValidate.RegisterStructValidation(notificationStructLevelValidation, Notification{})

	// get fixed IDs
	riskCategoryStationaryID = uuid.FromStringOrNil(RiskCategoryStationaryIDString)
	riskCategoryMobileID = uuid.FromStringOrNil(RiskCategoryMobileIDString)
	householdEntityID = uuid.FromStringOrNil(HouseholdEntityIDString)
	uuidNamespace = uuid.FromStringOrNil(uuidNamespaceString)
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
	return user
}

// Tx retrieves the database transaction from the context
func Tx(ctx context.Context) *pop.Connection {
	tx, ok := ctx.Value(domain.ContextKeyTx).(*pop.Connection)
	if !ok {
		domain.Logger.Print("no transaction found in context, called from: " + domain.GetFunctionName(2))
		return DB
	}
	return tx
}

func fieldByName(i any, name ...string) reflect.Value {
	if len(name) < 1 {
		return reflect.Value{}
	}
	f := reflect.ValueOf(i).Elem().FieldByName(name[0])
	if !f.IsValid() {
		return fieldByName(i, name[1:]...)
	}
	return f
}

func create(tx *pop.Connection, m any) error {
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
		appErr.Err = fmt.Errorf("%w Detail: %s", err, pgError.Detail)

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

func find(tx *pop.Connection, m any, id uuid.UUID) error {
	err := tx.Find(m, id)
	return appErrorFromDB(err, api.ErrorQueryFailure)
}

func save(tx *pop.Connection, m any) error {
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

func update(tx *pop.Connection, m any) error {
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

func destroy(tx *pop.Connection, m any) error {
	err := tx.Destroy(m)
	return appErrorFromDB(err, api.ErrorDestroyFailure)
}

// This can include an event payload, which is a map[string]any
func emitEvent(e events.Event) {
	if err := events.Emit(e); err != nil {
		domain.ErrLogger.Printf("error emitting event %s ... %v", e.Kind, err)
	}
}

func addFile(tx *pop.Connection, m Updatable, f File) error {
	if f.URL == "" {
		if err := tx.Find(&f, f.ID); err != nil {
			return appErrorFromDB(
				fmt.Errorf("error finding file %w", err),
				api.ErrorResourceNotFound,
			)
		}
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

	if err := m.Update(tx); err != nil {
		return appErrorFromDB(err, api.ErrorQueryFailure)
	}

	if err := f.SetLinked(tx); err != nil {
		return err
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

func convertUUIDToAPI(id nulls.UUID) *uuid.UUID {
	if id.Valid {
		return &id.UUID
	}
	return nil
}

func convertTimeToAPI(t nulls.Time) *time.Time {
	if t.Valid {
		return &t.Time
	}
	return nil
}

func GetV5UUID(seed string) uuid.UUID {
	return uuid.NewV5(uuidNamespace, seed)
}

func GetHHID(staffID string) string {
	if domain.Env.HouseholdIDLookupURL == "" {
		return ""
	}

	req, err := http.NewRequest(http.MethodGet, domain.Env.HouseholdIDLookupURL+staffID, nil)
	if err != nil {
		domain.ErrLogger.Printf("HHID API error, %s", err)
		return ""
	}
	req.SetBasicAuth(domain.Env.HouseholdIDLookupUsername, domain.Env.HouseholdIDLookupPassword)

	client := &http.Client{Timeout: time.Second * 30}
	response, err := client.Do(req)
	if err != nil {
		domain.ErrLogger.Printf("HHID API error, %s", err)
		return ""
	}
	defer response.Body.Close()

	dec := json.NewDecoder(response.Body)
	var v struct {
		ID string `json:"householdIdOut"`
	}
	if err = dec.Decode(&v); err != nil {
		domain.ErrLogger.Printf("HHID API error decoding response, %s", err)
		return ""
	}
	return v.ID
}
