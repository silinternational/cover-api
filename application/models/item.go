package models

import (
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

func (i *Item) Update(tx *pop.Connection, oldStatus api.ItemCoverageStatus) error {
	validTrans, err := isItemTransitionValid(oldStatus, i.CoverageStatus)
	if err != nil {
		panic(err)
	}
	if !validTrans {
		err := fmt.Errorf("invalid item coverage status transition from %s to %s",
			oldStatus, i.CoverageStatus)
		appErr := api.NewAppError(err, api.ErrorValidation, api.CategoryUser)
		return appErr
	}
	return update(tx, i)
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
func (i *Item) SafeDeleteOrInactivate(tx *pop.Connection, actor User) error {
	switch i.CoverageStatus {
	case api.ItemCoverageStatusInactive, api.ItemCoverageStatusDenied:
		return nil
	case api.ItemCoverageStatusApproved:
		return i.Inactivate(tx)
	case api.ItemCoverageStatusDraft, api.ItemCoverageStatusRevision, api.ItemCoverageStatusPending:
		if i.isNewEnough() {
			return tx.Destroy(i)
		}
		return i.Inactivate(tx)
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
func (i *Item) Inactivate(tx *pop.Connection) error {
	oldStatus := i.CoverageStatus
	i.CoverageStatus = api.ItemCoverageStatusInactive
	return i.Update(tx, oldStatus)
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
func (i *Item) SubmitForApproval(tx *pop.Connection) error {
	oldStatus := i.CoverageStatus
	i.CoverageStatus = api.ItemCoverageStatusPending

	i.Load(tx)

	if i.canAutoApprove(tx) {
		return i.AutoApprove(tx)
	}

	if err := i.Update(tx, oldStatus); err != nil {
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
func (i *Item) Revision(tx *pop.Connection, reason string) error {
	oldStatus := i.CoverageStatus
	i.CoverageStatus = api.ItemCoverageStatusRevision
	i.StatusReason = reason
	if err := i.Update(tx, oldStatus); err != nil {
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
func (i *Item) AutoApprove(tx *pop.Connection) error {
	e := events.Event{
		Kind:    domain.EventApiItemAutoApproved,
		Message: fmt.Sprintf("Item AutoApproved: %s  ID: %s", i.Name, i.ID.String()),
		Payload: events.Payload{domain.EventPayloadID: i.ID},
	}
	emitEvent(e)

	return i.Approve(tx)
}

// Approve takes the item from Pending coverage status to Approved.
// It assumes that the item's current status has already been validated.
func (i *Item) Approve(tx *pop.Connection) error {
	oldStatus := i.CoverageStatus
	i.CoverageStatus = api.ItemCoverageStatusApproved
	if err := i.Update(tx, oldStatus); err != nil {
		return err
	}

	e := events.Event{
		Kind:    domain.EventApiItemApproved,
		Message: fmt.Sprintf("Item Approved: %s  ID: %s", i.Name, i.ID.String()),
		Payload: events.Payload{domain.EventPayloadID: i.ID},
	}
	emitEvent(e)

	return i.CreateLedgerEntry(tx)
}

// Deny takes the item from Pending coverage status to Denied.
// It assumes that the item's current status has already been validated.
func (i *Item) Deny(tx *pop.Connection, reason string) error {
	oldStatus := i.CoverageStatus
	i.CoverageStatus = api.ItemCoverageStatusDenied
	i.StatusReason = reason
	if err := i.Update(tx, oldStatus); err != nil {
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
		CoverageStartDate:      i.CoverageStartDate.Format(domain.DateFormat),
		AccountableUserID:      i.PolicyUserID,
		AccountableDependentID: i.PolicyDependentID,
		AnnualPremium:          i.GetAnnualPremium(),
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

func (i *Item) GetAnnualPremium() int {
	p := int(math.Round(float64(i.CoverageAmount) * domain.Env.PremiumFactor))
	if p < domain.Env.PremiumMinimum {
		return domain.Env.PremiumMinimum
	}
	return p
}

func (i *Item) GetProratedPremium(t time.Time) int {
	p := domain.CalculatePartialYearValue(i.GetAnnualPremium(), t)
	if p < domain.Env.PremiumMinimum {
		return domain.Env.PremiumMinimum
	}
	return p
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

func (i *Item) setAccountablePerson(tx *pop.Connection, id uuid.UUID) error {
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

func (i *Item) CreateLedgerEntry(tx *pop.Connection) error {
	i.LoadPolicy(tx, false)
	i.Policy.LoadEntityCode(tx, false)

	firstName, lastName := i.GetAccountablePersonName(tx)
	le := LedgerEntry{
		PolicyID:           i.PolicyID,
		ItemID:             nulls.NewUUID(i.ID),
		EntityCode:         i.Policy.EntityCode.Code,
		Amount:             i.GetProratedPremium(time.Now().UTC()),
		DateSubmitted:      time.Now().UTC(),
		AccountNumber:      i.Policy.Account,
		AccountCostCenter1: i.Policy.CostCenter,
		IncomeAccount:      i.getIncomeAccount(tx),
		FirstName:          firstName,
		LastName:           lastName,
	}
	return le.Create(tx)
}

// getIncomeAccount maps the item data to the income account for billing
//
// WARNING: requires Policy and Policy.EntityCode to be pre-loaded
func (i *Item) getIncomeAccount(tx *pop.Connection) string {
	// TODO: move hard-coded account numbers to the database or to environment variables
	accountMap := map[string]string{
		"MMB/STM": "40200",
		"SIL":     "43250",
		"WBT":     "44250",
	}

	billingEntity := "MMB/STM"
	if i.Policy.EntityCodeID.Valid {
		switch i.Policy.EntityCode.Code {
		case "SIL":
			billingEntity = "SIL"
		default:
			billingEntity = "WBT"
		}
	}
	i.LoadRiskCategory(tx, false)
	incomeAccount := accountMap[billingEntity] + i.RiskCategory.CostCenter

	return incomeAccount
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
