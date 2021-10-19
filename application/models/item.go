package models

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/gobuffalo/buffalo"
	"github.com/gobuffalo/events"
	"github.com/gobuffalo/nulls"
	"github.com/gobuffalo/pop/v5"
	"github.com/gobuffalo/validate/v3"
	"github.com/gofrs/uuid"

	"github.com/silinternational/cover-api/api"
	"github.com/silinternational/cover-api/domain"
)

var ValidItemCoverageStatuses = map[api.ItemCoverageStatus]struct{}{
	api.ItemCoverageStatusDraft:    {},
	api.ItemCoverageStatusPending:  {},
	api.ItemCoverageStatusRevision: {},
	api.ItemCoverageStatusApproved: {},
	api.ItemCoverageStatusDenied:   {},
	api.ItemCoverageStatusInactive: {},
}

// Items is a slice of Item objects
type Items []Item

// Item model
type Item struct {
	ID                uuid.UUID              `db:"id"`
	Name              string                 `db:"name" validate:"required"`
	CategoryID        uuid.UUID              `db:"category_id" validate:"required"`
	RiskCategoryID    uuid.UUID              `db:"risk_category_id" validate:"required"`
	InStorage         bool                   `db:"in_storage"`
	Country           string                 `db:"country"`
	Description       string                 `db:"description"`
	PolicyID          uuid.UUID              `db:"policy_id" validate:"required"`
	PolicyDependentID nulls.UUID             `db:"policy_dependent_id"`
	PolicyUserID      nulls.UUID             `db:"policy_user_id"`
	Make              string                 `db:"make"`
	Model             string                 `db:"model"`
	SerialNumber      string                 `db:"serial_number"`
	CoverageAmount    int                    `db:"coverage_amount" validate:"min=0"`
	PurchaseDate      time.Time              `db:"purchase_date"`
	CoverageStatus    api.ItemCoverageStatus `db:"coverage_status" validate:"itemCoverageStatus"`
	StatusChange      string                 `db:"status_change"`
	CoverageStartDate time.Time              `db:"coverage_start_date"`
	StatusReason      string                 `db:"status_reason" validate:"required_if=CoverageStatus Revision,required_if=CoverageStatus Denied"`
	LegacyID          nulls.Int              `db:"legacy_id"`
	CreatedAt         time.Time              `db:"created_at"`
	UpdatedAt         time.Time              `db:"updated_at"`

	Category     ItemCategory `belongs_to:"item_categories" validate:"-"`
	RiskCategory RiskCategory `belongs_to:"risk_categories" validate:"-"`
	Policy       Policy       `belongs_to:"policies" validate:"-"`
}

// Validate gets run every time you call pop.ValidateAndSave, pop.ValidateAndCreate, or pop.ValidateAndUpdate
func (i *Item) Validate(tx *pop.Connection) (*validate.Errors, error) {
	return validateModel(i), nil
}

func (i *Item) Create(tx *pop.Connection) error {
	if _, ok := ValidItemCoverageStatuses[i.CoverageStatus]; !ok {
		i.CoverageStatus = api.ItemCoverageStatusDraft
	}

	return create(tx, i)
}

func (i *Item) Update(ctx context.Context) error {
	tx := Tx(ctx)
	var oldItem Item
	if err := oldItem.FindByID(tx, i.ID); err != nil {
		return appErrorFromDB(err, api.ErrorQueryFailure)
	}
	if validTrans, err := isItemTransitionValid(oldItem.CoverageStatus, i.CoverageStatus); err != nil {
		panic(err)
	} else {
		if !validTrans {
			err := fmt.Errorf("invalid item coverage status transition from %s to %s",
				oldItem.CoverageStatus, i.CoverageStatus)
			appErr := api.NewAppError(err, api.ErrorValidation, api.CategoryUser)
			return appErr
		}
	}

	i.LoadPolicy(tx, false)

	updates := i.Compare(oldItem)
	for ii := range updates {
		history := i.Policy.NewHistory(ctx, api.HistoryActionUpdate, updates[ii])
		history.ItemID = nulls.NewUUID(i.ID)
		if err := history.Create(tx); err != nil {
			return err
		}
	}

	if oldItem.CoverageAmount != i.CoverageAmount {
		amount := i.calculatePremiumChange(time.Now().UTC(), oldItem.CoverageAmount)
		if err := i.CreateLedgerEntry(Tx(ctx), LedgerEntryTypeCoverageChange, amount); err != nil {
			return err
		}
	}

	return update(tx, i)
}

// Compare returns a list of fields that are different between two objects
func (i *Item) Compare(old Item) []FieldUpdate {
	var updates []FieldUpdate

	if i.Name != old.Name {
		updates = append(updates, FieldUpdate{
			OldValue:  old.Name,
			NewValue:  i.Name,
			FieldName: FieldItemName,
		})
	}

	if i.CategoryID != old.CategoryID {
		updates = append(updates, FieldUpdate{
			OldValue:  old.CategoryID.String(),
			NewValue:  i.CategoryID.String(),
			FieldName: FieldItemCategoryID,
		})
	}

	if i.RiskCategoryID != old.RiskCategoryID {
		updates = append(updates, FieldUpdate{
			OldValue:  old.RiskCategoryID.String(),
			NewValue:  i.RiskCategoryID.String(),
			FieldName: FieldItemRiskCategoryID,
		})
	}

	if i.InStorage != old.InStorage {
		updates = append(updates, FieldUpdate{
			OldValue:  fmt.Sprintf(`%t`, old.InStorage),
			NewValue:  fmt.Sprintf(`%t`, i.InStorage),
			FieldName: FieldItemInStorage,
		})
	}

	if i.Country != old.Country {
		updates = append(updates, FieldUpdate{
			OldValue:  old.Country,
			NewValue:  i.Country,
			FieldName: FieldItemCountry,
		})
	}

	if i.Description != old.Description {
		updates = append(updates, FieldUpdate{
			OldValue:  old.Description,
			NewValue:  i.Description,
			FieldName: FieldItemDescription,
		})
	}

	if i.PolicyDependentID != old.PolicyDependentID {
		updates = append(updates, FieldUpdate{
			OldValue:  old.PolicyDependentID.UUID.String(),
			NewValue:  i.PolicyDependentID.UUID.String(),
			FieldName: FieldItemPolicyDependentID,
		})
	}

	if i.PolicyUserID != old.PolicyUserID {
		updates = append(updates, FieldUpdate{
			OldValue:  old.PolicyUserID.UUID.String(),
			NewValue:  i.PolicyUserID.UUID.String(),
			FieldName: FieldItemPolicyUserID,
		})
	}

	if i.Make != old.Make {
		updates = append(updates, FieldUpdate{
			OldValue:  old.Make,
			NewValue:  i.Make,
			FieldName: FieldItemMake,
		})
	}

	if i.Model != old.Model {
		updates = append(updates, FieldUpdate{
			OldValue:  old.Model,
			NewValue:  i.Model,
			FieldName: FieldItemModel,
		})
	}

	if i.SerialNumber != old.SerialNumber {
		updates = append(updates, FieldUpdate{
			OldValue:  old.SerialNumber,
			NewValue:  i.SerialNumber,
			FieldName: FieldItemSerialNumber,
		})
	}

	if i.CoverageAmount != old.CoverageAmount {
		updates = append(updates, FieldUpdate{
			OldValue:  api.Currency(old.CoverageAmount).String(),
			NewValue:  api.Currency(i.CoverageAmount).String(),
			FieldName: FieldItemCoverageAmount,
		})
	}

	if i.PurchaseDate != old.PurchaseDate {
		updates = append(updates, FieldUpdate{
			OldValue:  old.PurchaseDate.Format(domain.DateFormat),
			NewValue:  i.PurchaseDate.Format(domain.DateFormat),
			FieldName: FieldItemPurchaseDate,
		})
	}

	if i.CoverageStatus != old.CoverageStatus {
		updates = append(updates, FieldUpdate{
			OldValue:  string(old.CoverageStatus),
			NewValue:  string(i.CoverageStatus),
			FieldName: FieldItemCoverageStatus,
		})
	}

	if i.CoverageStartDate != old.CoverageStartDate {
		updates = append(updates, FieldUpdate{
			OldValue:  old.CoverageStartDate.Format(domain.DateFormat),
			NewValue:  i.CoverageStartDate.Format(domain.DateFormat),
			FieldName: FieldItemCoverageStartDate,
		})
	}

	if i.StatusReason != old.StatusReason {
		updates = append(updates, FieldUpdate{
			OldValue:  old.StatusReason,
			NewValue:  i.StatusReason,
			FieldName: FieldItemStatusReason,
		})
	}

	return updates
}

func (i *Item) GetID() uuid.UUID {
	return i.ID
}

func (i *Item) FindByID(tx *pop.Connection, id uuid.UUID) error {
	return tx.Find(i, id)
}

// SafeDeleteOrInactivate deletes the item if it is newish (less than 72 hours old)
//  and if it has a Draft, Revision or Pending status.
//  If the item's status is Denied or Inactive, it does nothing.
//  Otherwise, it changes its status to Inactive.
func (i *Item) SafeDeleteOrInactivate(ctx context.Context, actor User) error {
	switch i.CoverageStatus {
	case api.ItemCoverageStatusInactive, api.ItemCoverageStatusDenied:
		return nil
	case api.ItemCoverageStatusApproved:
		return i.Inactivate(ctx)
	case api.ItemCoverageStatusDraft, api.ItemCoverageStatusRevision, api.ItemCoverageStatusPending:
		// TODO: figure out when a destroy will work and when it won't, based on searching for child records
		// that may have been created for an item that traverses states and then back to one of these
		// "safe-to-delete" states.
		//if i.isNewEnough() {
		//	return Tx(ctx).Destroy(i)
		//}
		return i.Inactivate(ctx)
	default:
		panic(`invalid item status in SafeDeleteOrInactivate`)
	}
	return nil
}

// isNewEnough checks whether the item was created in the last X hours
func (i *Item) isNewEnough() bool {
	oldTime, err := time.Parse(time.RFC3339, "1970-01-01T00:07:41+00:00")
	if err != nil {
		panic("error parsing old time format: " + err.Error())
	}

	if !i.CreatedAt.After(oldTime) {
		panic("item doesn't have a valid CreatedAt date")
	}

	cutOffDate := time.Now().UTC().Add(time.Hour * -domain.ItemDeleteCutOffHours)
	return !i.CreatedAt.Before(cutOffDate)
}

// Inactivate sets the item's CoverageStatus to Inactive
//  TODO deal with coverage payment changes
func (i *Item) Inactivate(ctx context.Context) error {
	i.CoverageStatus = api.ItemCoverageStatusInactive

	user := CurrentUser(ctx)
	i.StatusChange = ItemStatusChangeInactivated + user.Name()
	if err := i.Update(ctx); err != nil {
		return err
	}

	return i.CreateLedgerEntry(Tx(ctx), LedgerEntryTypeCoverageChange, i.calculateCancellationCredit(time.Now().UTC()))
}

// IsActorAllowedTo ensure the actor is either an admin, or a member of this policy to perform any permission
func (i *Item) IsActorAllowedTo(tx *pop.Connection, actor User, perm Permission, sub SubResource, req *http.Request) bool {
	isAdmin := actor.IsAdmin()
	if !isItemActionAllowed(isAdmin, i.CoverageStatus, perm, sub) {
		return false
	}

	if isAdmin {
		return true
	}

	i.LoadPolicyMembers(tx, false)

	for _, m := range i.Policy.Members {
		if m.ID == actor.ID {
			return true
		}
	}

	return false
}

func itemStatusTransitions() map[api.ItemCoverageStatus][]api.ItemCoverageStatus {
	return map[api.ItemCoverageStatus][]api.ItemCoverageStatus{
		api.ItemCoverageStatusDraft: {
			api.ItemCoverageStatusPending,
			api.ItemCoverageStatusApproved,
			api.ItemCoverageStatusInactive,
		},
		api.ItemCoverageStatusPending: {
			api.ItemCoverageStatusRevision,
			api.ItemCoverageStatusApproved,
			api.ItemCoverageStatusDenied,
			api.ItemCoverageStatusInactive,
		},
		api.ItemCoverageStatusRevision: {
			api.ItemCoverageStatusPending,
			api.ItemCoverageStatusApproved,
			api.ItemCoverageStatusDenied,
			api.ItemCoverageStatusInactive,
		},
		api.ItemCoverageStatusApproved: {
			api.ItemCoverageStatusPending,
			api.ItemCoverageStatusInactive,
		},
		api.ItemCoverageStatusDenied:   {},
		api.ItemCoverageStatusInactive: {},
	}
}

func isItemTransitionValid(status1, status2 api.ItemCoverageStatus) (bool, error) {
	if status1 == status2 {
		return true, nil
	}
	targets, ok := itemStatusTransitions()[status1]
	if !ok {
		return false, errors.New("unexpected initial item coverage status - " + string(status1))
	}

	for _, target := range targets {
		if status2 == target {
			return true, nil
		}
	}

	return false, nil
}

// isItemActionAllowed does not check whether the actor is the owner of the item.
//  Otherwise, it checks whether the item can be acted on using a certain action based on its
//    current coverage status and "sub-resource" (e.g. submit, approve, ...)
func isItemActionAllowed(actorIsAdmin bool, oldStatus api.ItemCoverageStatus, perm Permission, sub SubResource) bool {
	switch oldStatus {

	// An item with Draft or Revision coverage status can have an update done on it itself or a create done on its "submit"
	// It can also be deleted/inactivated
	case api.ItemCoverageStatusDraft, api.ItemCoverageStatusRevision:
		if sub == "" && (perm == PermissionUpdate || perm == PermissionDelete) {
			return true
		}

		return sub == api.ResourceSubmit && perm == PermissionCreate

	// An item with Pending status can have a create done on it by an admin for revision, approve, deny
	// A non-admin can delete/inactivate it
	case api.ItemCoverageStatusPending:
		if !actorIsAdmin {
			return perm == PermissionDelete && sub == ""
		}
		return perm == PermissionCreate && (sub == api.ResourceApprove || sub == api.ResourceRevision || sub == api.ResourceDeny)

	// An item with approved status can only be deleted/inactivated
	case api.ItemCoverageStatusApproved:
		return sub == "" && perm == PermissionDelete
	}

	return false
}

// SubmitForApproval takes the item from Draft or Revision status to Pending or Approved status.
// It assumes that the item's current status has already been validated.
func (i *Item) SubmitForApproval(ctx context.Context) error {
	tx := Tx(ctx)

	i.CoverageStatus = api.ItemCoverageStatusPending

	i.Load(tx)

	if i.canAutoApprove(tx) {
		return i.AutoApprove(ctx)
	}

	i.StatusChange = ItemStatusChangeSubmitted
	if err := i.Update(ctx); err != nil {
		return err
	}

	e := events.Event{
		Kind:    domain.EventApiItemSubmitted,
		Message: fmt.Sprintf("Item Submitted: %s  ID: %s", i.Name, i.ID.String()),
		Payload: events.Payload{domain.EventPayloadID: i.ID},
	}
	emitEvent(e)

	return nil
}

// Checks whether the item has a category that expects the make and model fields
//   to be hydrated. Returns true if they are hydrated of if the item has
//   a lenient category.
// Assumes the item already has its Category loaded
func (i *Item) areFieldsValidForAutoApproval(tx *pop.Connection) bool {
	if !i.Category.RequireMakeModel {
		return true
	}
	return i.Make != `` && i.Model != ``
}

// Assumes the item already has its Category loaded
func (i *Item) canAutoApprove(tx *pop.Connection) bool {
	if i.CoverageAmount > i.Category.AutoApproveMax {
		return false
	}
	if !i.areFieldsValidForAutoApproval(tx) {
		return false
	}

	policy := Policy{ID: i.PolicyID}
	totals := policy.itemCoverageTotals(tx)

	policyTotal := totals[i.PolicyID]

	if policyTotal+i.CoverageAmount > domain.Env.PolicyMaxCoverage {
		return false
	}

	if !i.PolicyDependentID.Valid {
		return true
	}

	// Dependents have different rules based on the total amounts of all their items
	depTotal := totals[i.PolicyDependentID.UUID]
	return depTotal+i.CoverageAmount <= domain.Env.DependentAutoApproveMax
}

// Revision takes the item from Pending coverage status to Revision.
// It assumes that the item's current status has already been validated.
func (i *Item) Revision(ctx context.Context, reason string) error {
	i.CoverageStatus = api.ItemCoverageStatusRevision
	i.StatusReason = reason

	user := CurrentUser(ctx)
	i.StatusChange = ItemStatusChangeRevisions + user.Name()

	if err := i.Update(ctx); err != nil {
		return err
	}

	e := events.Event{
		Kind:    domain.EventApiItemRevision,
		Message: fmt.Sprintf("Item to Revision: %s  ID: %s", i.Name, i.ID.String()),
		Payload: events.Payload{domain.EventPayloadID: i.ID},
	}
	emitEvent(e)

	return nil
}

// AutoApprove fires an event and marks the item as `Approved`
// It assumes that the item's current status has already been validated.
func (i *Item) AutoApprove(ctx context.Context) error {
	e := events.Event{
		Kind:    domain.EventApiItemAutoApproved,
		Message: fmt.Sprintf("Item AutoApproved: %s  ID: %s", i.Name, i.ID.String()),
		Payload: events.Payload{domain.EventPayloadID: i.ID},
	}
	emitEvent(e)

	i.StatusChange = ItemStatusChangeAutoApproved
	return i.Approve(ctx, true)
}

// Approve takes the item from Pending coverage status to Approved.
// It assumes that the item's current status has already been validated.
// Only emits an event for an email notification if requested.
// (No need to emit it following an auto-approval which has already emitted and event.)
func (i *Item) Approve(ctx context.Context, doEmitEvent bool) error {
	i.CoverageStatus = api.ItemCoverageStatusApproved

	if i.StatusChange != ItemStatusChangeAutoApproved {
		user := CurrentUser(ctx)
		i.StatusChange = ItemStatusChangeApproved + user.Name()
	}

	if err := i.Update(ctx); err != nil {
		return err
	}

	if doEmitEvent {
		e := events.Event{
			Kind:    domain.EventApiItemApproved,
			Message: fmt.Sprintf("Item Approved: %s  ID: %s", i.Name, i.ID.String()),
			Payload: events.Payload{domain.EventPayloadID: i.ID},
		}
		emitEvent(e)
	}

	amount := i.calculateProratedPremium(time.Now().UTC())
	return i.CreateLedgerEntry(Tx(ctx), LedgerEntryTypeNewCoverage, amount)
}

// Deny takes the item from Pending coverage status to Denied.
// It assumes that the item's current status has already been validated.
func (i *Item) Deny(ctx context.Context, reason string) error {
	i.CoverageStatus = api.ItemCoverageStatusDenied
	i.StatusReason = reason

	i.LoadPolicy(Tx(ctx), false)

	user := CurrentUser(ctx)
	i.StatusChange = ItemStatusChangeDenied + user.Name()
	if err := i.Update(ctx); err != nil {
		return err
	}

	e := events.Event{
		Kind:    domain.EventApiItemDenied,
		Message: fmt.Sprintf("Item Denied: %s  ID: %s", i.Name, i.ID.String()),
		Payload: events.Payload{domain.EventPayloadID: i.ID},
	}
	emitEvent(e)

	return nil
}

// LoadPolicy - a simple wrapper method for loading the policy
func (i *Item) LoadPolicy(tx *pop.Connection, reload bool) {
	if i.Policy.ID == uuid.Nil || reload {
		if err := tx.Load(i, "Policy"); err != nil {
			panic("error loading item policy: " + err.Error())
		}
	}
}

// LoadRiskCategory - a simple wrapper method for loading the risk category
func (i *Item) LoadRiskCategory(tx *pop.Connection, reload bool) {
	if i.RiskCategory.ID == uuid.Nil || reload {
		if err := tx.Load(i, "RiskCategory"); err != nil {
			panic("error loading item risk category: " + err.Error())
		}
	}
}

// LoadPolicyMembers - a simple wrapper method for loading the policy and its members
func (i *Item) LoadPolicyMembers(tx *pop.Connection, reload bool) {
	i.LoadPolicy(tx, reload)

	i.Policy.LoadMembers(tx, reload)
}

// Load - a simple wrapper method for loading child objects
func (i *Item) Load(tx *pop.Connection) {
	if i.Category.ID == uuid.Nil {
		if err := tx.Load(i, "Category", "RiskCategory"); err != nil {
			panic("error loading item child objects: " + err.Error())
		}
	}
}

func (i *Item) ConvertToAPI(tx *pop.Connection) api.Item {
	i.Load(tx)
	return api.Item{
		ID:                     i.ID,
		Name:                   i.Name,
		Category:               i.Category.ConvertToAPI(tx),
		RiskCategory:           i.RiskCategory.ConvertToAPI(),
		InStorage:              i.InStorage,
		Country:                i.Country,
		Description:            i.Description,
		PolicyID:               i.PolicyID,
		Make:                   i.Make,
		Model:                  i.Model,
		SerialNumber:           i.SerialNumber,
		CoverageAmount:         i.CoverageAmount,
		PurchaseDate:           i.PurchaseDate.Format(domain.DateFormat),
		CoverageStatus:         i.CoverageStatus,
		StatusChange:           i.StatusChange,
		CoverageStartDate:      i.CoverageStartDate.Format(domain.DateFormat),
		AccountableUserID:      i.PolicyUserID,
		AccountableDependentID: i.PolicyDependentID,
		AnnualPremium:          i.CalculateAnnualPremium(),
		CreatedAt:              i.CreatedAt,
		UpdatedAt:              i.UpdatedAt,
	}
}

func (i *Items) ConvertToAPI(tx *pop.Connection) api.Items {
	apiItems := make(api.Items, len(*i))
	for j, ii := range *i {
		apiItems[j] = ii.ConvertToAPI(tx)
	}

	return apiItems
}

// CalculateAnnualPremium returns the rounded product of the item's CoverageAmount and the PremiumFactor
func (i *Item) CalculateAnnualPremium() api.Currency {
	p := int(math.Round(float64(i.CoverageAmount) * domain.Env.PremiumFactor))
	return api.Currency(p)
}

func (i *Item) calculateProratedPremium(t time.Time) api.Currency {
	p := domain.CalculatePartialYearValue(int(i.CalculateAnnualPremium()), t)
	return api.Currency(p)
}

func (i *Item) calculateCancellationCredit(t time.Time) api.Currency {
	p := domain.CalculatePartialYearValue(int(i.CalculateAnnualPremium()), t)
	return api.Currency(-1 * p)
}

func (i *Item) calculatePremiumChange(t time.Time, oldCoverageAmount int) api.Currency {
	oldItem := Item{CoverageAmount: oldCoverageAmount}

	oldPremium := oldItem.CalculateAnnualPremium()
	credit := domain.CalculatePartialYearValue(int(oldPremium), t)

	newPremium := i.CalculateAnnualPremium()
	charge := domain.CalculatePartialYearValue(int(newPremium), t)

	return api.Currency(charge - credit)
}

// NewItemFromApiInput creates a new `Item` from a `ItemInput`.
func NewItemFromApiInput(c buffalo.Context, input api.ItemInput, policyID uuid.UUID) (Item, error) {
	item := Item{}
	if err := parseItemDates(input, &item); err != nil {
		return item, err
	}

	tx := Tx(c)

	var itemCat ItemCategory
	if err := itemCat.FindByID(tx, input.CategoryID); err != nil {
		return item, err
	}

	user := CurrentUser(c)
	riskCatID := itemCat.RiskCategoryID
	if input.RiskCategoryID.Valid && user.IsAdmin() {
		riskCatID = input.RiskCategoryID.UUID
	}

	item.Name = input.Name
	item.CategoryID = input.CategoryID
	item.RiskCategoryID = riskCatID
	item.InStorage = input.InStorage
	item.Country = input.Country
	item.Description = input.Description
	item.PolicyID = policyID
	item.Make = input.Make
	item.Model = input.Model
	item.SerialNumber = input.SerialNumber
	item.CoverageAmount = input.CoverageAmount
	item.CoverageStatus = input.CoverageStatus

	if err := item.setAccountablePerson(tx, input.AccountablePersonID); err != nil {
		return item, err
	}

	return item, nil
}

func parseItemDates(input api.ItemInput, modelItem *Item) error {
	pDate, err := time.Parse(domain.DateFormat, input.PurchaseDate)
	if err != nil {
		err = errors.New("failed to parse item purchase date, " + err.Error())
		appErr := api.NewAppError(err, api.ErrorItemInvalidPurchaseDate, api.CategoryUser)
		return appErr
	}
	modelItem.PurchaseDate = pDate

	csDate, err := time.Parse(domain.DateFormat, input.CoverageStartDate)
	if err != nil {
		err = errors.New("failed to parse item coverage start date, " + err.Error())
		appErr := api.NewAppError(err, api.ErrorItemInvalidCoverageStartDate, api.CategoryUser)
		return appErr
	}
	modelItem.CoverageStartDate = csDate

	return nil
}

// setAccountablePerson sets the appropriate field to the given ID, but does not update the database
func (i *Item) setAccountablePerson(tx *pop.Connection, id uuid.UUID) error {
	if id == uuid.Nil {
		return api.NewAppError(errors.New("accountable person ID must not be nil"), api.ErrorItemNullAccountablePerson, api.CategoryUser)
	}

	i.LoadPolicy(tx, false)

	if i.Policy.isMember(tx, id) {
		i.PolicyUserID = nulls.NewUUID(id)
		i.PolicyDependentID = nulls.UUID{}
		return nil
	}

	if i.Policy.isDependent(tx, id) {
		i.PolicyDependentID = nulls.NewUUID(id)
		i.PolicyUserID = nulls.UUID{}
		return nil
	}

	return api.NewAppError(errors.New("accountable person ID not found"), api.ErrorNoRows, api.CategoryUser)
}

func (i *Item) CreateLedgerEntry(tx *pop.Connection, entryType LedgerEntryType, amount api.Currency) error {
	i.LoadPolicy(tx, false)
	i.LoadRiskCategory(tx, false)
	i.Policy.LoadEntityCode(tx, false)

	firstName, lastName := i.GetAccountablePersonName(tx)

	le := NewLedgerEntry(i.Policy, i, nil)
	le.Type = entryType
	le.Amount = amount
	le.FirstName = firstName
	le.LastName = lastName
	le.DateSubmitted = i.CoverageStartDate

	return le.Create(tx)
}

// GetAccountablePersonName gets the name of the accountable person. In case of error, empty strings
// are returned.
func (i *Item) GetAccountablePersonName(tx *pop.Connection) (firstName, lastName string) {
	if i.PolicyUserID.Valid {
		var user User
		_ = user.FindByID(tx, i.PolicyUserID.UUID)
		return user.FirstName, user.LastName
	}
	if i.PolicyDependentID.Valid {
		var dep PolicyDependent
		_ = dep.FindByID(tx, i.PolicyDependentID.UUID)
		names := strings.SplitN(dep.Name, " ", 2)
		firstName = names[0]
		if len(names) > 1 {
			lastName = names[1]
		}
	}
	return firstName, lastName
}

func (i *Item) GetMakeModel() string {
	return strings.TrimSpace(i.Make + " " + i.Model)
}

// ItemsWithRecentStatusChanges returns the RecentItems associated with
//  items that have had their CoverageStatus changed recently.
//  The slice is sorted by updated time with most recent first.
func ItemsWithRecentStatusChanges(tx *pop.Connection) (api.RecentItems, error) {
	var pHistories PolicyHistories

	if err := pHistories.RecentItemStatusChanges(tx); err != nil {
		return api.RecentItems{}, err
	}

	// Fetch the actual items from the database and convert them to api types
	items := make(api.RecentItems, len(pHistories))
	for i, next := range pHistories {
		var item Item
		if err := item.FindByID(tx, next.ItemID.UUID); err != nil {
			panic("error finding item by ID: " + err.Error())
		}

		apiItem := item.ConvertToAPI(tx)
		items[i] = api.RecentItem{Item: apiItem, StatusUpdatedAt: next.CreatedAt}
	}

	return items, nil
}
