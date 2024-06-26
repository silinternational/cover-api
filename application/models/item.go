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
	"github.com/gobuffalo/pop/v6"
	"github.com/gobuffalo/validate/v3"
	"github.com/gofrs/uuid"

	"github.com/silinternational/cover-api/api"
	"github.com/silinternational/cover-api/domain"
	"github.com/silinternational/cover-api/log"
)

// MonthlyCutoffDay is the day of the month before which monthly-billed coverage can be added for the current month.
// On this day or later, coverage billing begins on the first of the next month.
const MonthlyCutoffDay = 20

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
	Description       string                 `db:"description"`
	PolicyID          uuid.UUID              `db:"policy_id" validate:"required"`
	PolicyDependentID nulls.UUID             `db:"policy_dependent_id"`
	PolicyUserID      nulls.UUID             `db:"policy_user_id"`
	Make              string                 `db:"make"`
	Model             string                 `db:"model"`
	Year              nulls.Int              `db:"year"`
	SerialNumber      string                 `db:"serial_number"`
	CoverageAmount    int                    `db:"coverage_amount" validate:"min=0"`
	CoverageStatus    api.ItemCoverageStatus `db:"coverage_status" validate:"itemCoverageStatus"`
	PaidThroughDate   time.Time              `db:"paid_through_date"`
	StatusChange      string                 `db:"status_change"`
	CoverageStartDate time.Time              `db:"coverage_start_date"`
	CoverageEndDate   nulls.Time             `db:"coverage_end_date"`
	StatusReason      string                 `db:"status_reason" validate:"required_if=CoverageStatus Revision,required_if=CoverageStatus Denied"`
	City              string                 `db:"city"`
	State             string                 `db:"state"`
	Country           string                 `db:"country"`
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

// CreateWithHistory validates and stores the data as a new record in the database, assigning a new ID if needed.
// Also creates a PolicyHistory record.
func (i *Item) CreateWithHistory(ctx context.Context) error {
	tx := Tx(ctx)

	if err := i.Create(tx); err != nil {
		return err
	}

	history := i.NewHistory(ctx, api.HistoryActionCreate, FieldUpdate{})
	if err := history.Create(tx); err != nil {
		return err
	}
	return nil
}

// Create an Item but not a history record. Use CreateWithHistory if history is needed.
func (i *Item) Create(tx *pop.Connection) error {
	if _, ok := ValidItemCoverageStatuses[i.CoverageStatus]; !ok {
		i.CoverageStatus = api.ItemCoverageStatusDraft
	}
	i.LoadPolicy(tx, false)
	if i.Policy.Type == api.PolicyTypeHousehold && !i.Policy.HouseholdID.Valid {
		err := errors.New("policy does not have a household ID")
		return api.NewAppError(err, api.ErrorPolicyHasNoHouseholdID, api.CategoryUser)
	}
	return create(tx, i)
}

func (i *Item) Update(ctx context.Context) error {
	tx := Tx(ctx)
	var oldItem Item
	if err := oldItem.FindByID(tx, i.ID); err != nil {
		return appErrorFromDB(err, api.ErrorQueryFailure)
	}
	if !isItemTransitionValid(oldItem.CoverageStatus, i.CoverageStatus) {
		err := fmt.Errorf("invalid item coverage status transition from %s to %s",
			oldItem.CoverageStatus, i.CoverageStatus)
		appErr := api.NewAppError(err, api.ErrorValidation, api.CategoryUser)
		return appErr
	}

	if i.hasOpenClaim(tx) {
		err := errors.New("item cannot be updated because it has an active claim")
		return api.NewAppError(err, api.ErrorItemHasActiveClaim, api.CategoryUser)
	}

	updates := i.Compare(oldItem)
	for ii := range updates {
		history := i.NewHistory(ctx, api.HistoryActionUpdate, updates[ii])
		history.ItemID = nulls.NewUUID(i.ID)
		if err := history.Create(tx); err != nil {
			return err
		}
	}

	if i.CoverageStatus == api.ItemCoverageStatusApproved {
		if oldItem.CoverageAmount < i.CoverageAmount {
			err := errors.New("item coverage amount cannot be increased")
			return api.NewAppError(err, api.ErrorItemCoverageAmountCannotIncrease, api.CategoryUser)
		}

		i.LoadCategory(tx, false)
		if i.Category.GetBillingPeriod() == domain.BillingPeriodAnnual {
			if err := i.createPremiumAdjustment(tx, time.Now().UTC(), oldItem); err != nil {
				return err
			}
		}
	}

	return update(tx, i)
}

func (i *Item) Destroy(tx *pop.Connection) error {
	return destroy(tx, i)
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
			OldValue:  NullsUUIDToString(old.PolicyDependentID),
			NewValue:  NullsUUIDToString(i.PolicyDependentID),
			FieldName: FieldItemPolicyDependentID,
		})
	}

	if i.PolicyUserID != old.PolicyUserID {
		updates = append(updates, FieldUpdate{
			OldValue:  NullsUUIDToString(old.PolicyUserID),
			NewValue:  NullsUUIDToString(i.PolicyUserID),
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

	if i.PaidThroughDate != old.PaidThroughDate {
		updates = append(updates, FieldUpdate{
			OldValue:  old.PaidThroughDate.Format(domain.DateFormat),
			NewValue:  i.PaidThroughDate.Format(domain.DateFormat),
			FieldName: FieldItemPaidThroughDate,
		})
	}

	if i.StatusReason != old.StatusReason {
		updates = append(updates, FieldUpdate{
			OldValue:  old.StatusReason,
			NewValue:  i.StatusReason,
			FieldName: FieldItemStatusReason,
		})
	}

	if i.Year != old.Year {
		oldYearBytes, _ := old.Year.MarshalJSON()
		newYearBytes, _ := i.Year.MarshalJSON()
		updates = append(updates, FieldUpdate{
			OldValue:  string(oldYearBytes),
			NewValue:  string(newYearBytes),
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
// and if it has a Draft, Revision or Pending status.
// If the item's status is Denied or Inactive, it does nothing.
// Otherwise, it changes its status to Inactive.
func (i *Item) SafeDeleteOrInactivate(ctx context.Context, now time.Time) error {
	tx := Tx(ctx)
	switch i.CoverageStatus {
	case api.ItemCoverageStatusInactive, api.ItemCoverageStatusDenied:
		return nil
	case api.ItemCoverageStatusApproved:
		return i.ScheduleInactivation(ctx, now)
	case api.ItemCoverageStatusDraft, api.ItemCoverageStatusRevision, api.ItemCoverageStatusPending:
		if i.isNewEnough() && i.canBeDeleted(tx) {
			return i.Destroy(tx)
		}
		return i.Inactivate(ctx)
	default:
		panic(`invalid item status in SafeDeleteOrInactivate`)
	}
}

// CreateCancellationCredit creates a credit for the refund of annual coverage if the
// premium has been charged for the year.
func (i *Item) CreateCancellationCredit(tx *pop.Connection, now time.Time) error {
	if i.PaidThroughDate.Before(now) {
		return nil
	}

	i.LoadCategory(tx, false)
	if i.Category.GetBillingPeriod() == domain.BillingPeriodMonthly {
		return nil
	}

	creditAmount := i.calculateCancellationCredit(tx, now)

	if err := i.CreateLedgerEntry(tx, LedgerEntryTypeCoverageRefund, creditAmount, now); err != nil {
		return err
	}

	return i.SetPaidThroughDate(tx, domain.ZeroDate())
}

// canBeDeleted checks for child records with restricted constraints and returns false if any are found
func (i *Item) canBeDeleted(tx *pop.Connection) bool {
	var claimItems ClaimItems
	if n, err := tx.Where("item_id = ?", i.ID).Count(&claimItems); err != nil {
		panic(fmt.Sprintf("error counting claim_items with item_id %s, %s", i.ID, err))
	} else if n > 0 {
		return false
	}
	var ledgerEntries LedgerEntries
	if n, err := tx.Where("item_id = ?", i.ID).Count(&ledgerEntries); err != nil {
		panic(fmt.Sprintf("error counting ledger_entries with item_id %s, %s", i.ID, err))
	} else if n > 0 {
		return false
	}
	return true
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

// ScheduleInactivation sets the item's StatusChange and CoverageEndDate
func (i *Item) ScheduleInactivation(ctx context.Context, t time.Time) error {
	user := CurrentUser(ctx)
	i.StatusChange = ItemStatusChangeInactivated + user.Name()

	tx := Tx(ctx)
	i.LoadCategory(tx, false)

	// If the item was created before this year, and it qualifies for a full-year refund, set
	// its CoverageEndDate to the current day. Otherwise, set it to the end of the current month.
	if i.Category.GetBillingPeriod() == domain.BillingPeriodAnnual && i.shouldGiveFullYearRefund(t) {
		i.CoverageEndDate = nulls.NewTime(t)
	} else {
		i.CoverageEndDate = nulls.NewTime(domain.EndOfMonth(t))
	}
	return i.Update(ctx)
}

// cancelCoverageAfterClaim sets the item's CoverageEndDate to the current time and creates a corresponding
// credit ledger entry
func (i *Item) cancelCoverageAfterClaim(tx *pop.Connection, reason string) error {
	if i.CoverageStatus != api.ItemCoverageStatusApproved {
		return errors.New("cannot cancel coverage on an item which is not approved")
	}

	history := PolicyHistory{
		Action:    api.HistoryActionUpdate,
		PolicyID:  i.PolicyID,
		ItemID:    nulls.NewUUID(i.ID),
		UserID:    uuid.FromStringOrNil(ServiceUserID),
		FieldName: FieldItemCoverageStatus,
		OldValue:  string(api.ItemCoverageStatusApproved),
		NewValue:  string(api.ItemCoverageStatusInactive),
	}

	if err := history.Create(tx); err != nil {
		return err
	}

	now := time.Now().UTC()

	i.LoadCategory(tx, false)
	if i.Category.GetBillingPeriod() == domain.BillingPeriodAnnual {
		amount := i.calculatePremiumChange(now, i.CalculateAnnualPremium(tx), 0)
		if err := i.CreateLedgerEntry(tx, LedgerEntryTypeCoverageRefund, amount, now); err != nil {
			return err
		}
		if err := i.SetPaidThroughDate(tx, domain.ZeroDate()); err != nil {
			return err
		}
	}

	if reason == "" {
		reason = "coverage cancelled"
	}

	i.CoverageEndDate = nulls.NewTime(now)
	i.CoverageStatus = api.ItemCoverageStatusInactive
	i.StatusChange = ItemStatusChangeInactivated + "having a claim approved"
	i.StatusReason = reason

	return tx.Update(i)
}

func (i *Item) Inactivate(ctx context.Context) error {
	i.CoverageStatus = api.ItemCoverageStatusInactive
	return i.Update(ctx)
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

func isItemTransitionValid(status1, status2 api.ItemCoverageStatus) bool {
	if status1 == status2 {
		return true
	}
	targets, ok := itemStatusTransitions()[status1]
	if !ok {
		panic("unexpected initial item coverage status - " + string(status1))
	}

	for _, target := range targets {
		if status2 == target {
			return true
		}
	}

	return false
}

// isItemActionAllowed does not check whether the actor is the owner of the item.
// Otherwise, it checks whether the item can be acted on using a certain action based on its
// current coverage status and "sub-resource" (e.g. submit, approve, ...)
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
	// A non-admin can delete/inactivate it or update it
	case api.ItemCoverageStatusPending:
		if perm == PermissionUpdate {
			return true
		}

		if !actorIsAdmin {
			return perm == PermissionDelete && sub == ""
		}
		return perm == PermissionCreate && (sub == api.ResourceApprove || sub == api.ResourceRevision || sub == api.ResourceDeny)

	// An item with approved status can only be deleted/inactivated or updated
	case api.ItemCoverageStatusApproved:
		return sub == "" && (perm == PermissionDelete || perm == PermissionUpdate)
	}

	return false
}

// SubmitForApproval takes the item from Draft or Revision status to Pending or Approved status.
// It assumes that the item's current status has already been validated.
func (i *Item) SubmitForApproval(ctx context.Context) error {
	tx := Tx(ctx)

	minimumCoverageAmount := i.getMinimumCoverage(tx)

	if i.CoverageAmount < minimumCoverageAmount {
		err := fmt.Errorf("coverage_amount must be at least %s", api.Currency(minimumCoverageAmount).String())
		return api.NewAppError(err, api.ErrorItemCoverageAmountTooLow, api.CategoryUser)
	}

	i.CoverageStatus = api.ItemCoverageStatusPending

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

func (i *Item) getMinimumCoverage(tx *pop.Connection) int {
	i.LoadCategory(tx, false)
	return i.Category.MinimumCoverage
}

// Checks whether the item has a category that expects the make and model fields
// to be hydrated. Returns true if they are hydrated or if the item has
// a lenient category.
func (i *Item) areFieldsValidForAutoApproval(tx *pop.Connection) bool {
	i.LoadCategory(tx, false)

	if !i.Category.RequireMakeModel {
		return true
	}
	return i.Make != `` && i.Model != ``
}

func (i *Item) canAutoApprove(tx *pop.Connection) bool {
	if !i.areFieldsValidForAutoApproval(tx) {
		return false
	}

	i.LoadCategory(tx, false)
	if i.CoverageAmount > i.Category.AutoApproveMax {
		return false
	}

	i.LoadPolicy(tx, false)
	if i.Policy.Type == api.PolicyTypeTeam {
		return true
	}

	totals := i.Policy.itemCoverageTotals(tx)

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
	return i.Approve(ctx, time.Now().UTC())
}

// Approve takes the item from Pending coverage status to Approved.
// It assumes that the item's current status has already been validated.
func (i *Item) Approve(ctx context.Context, now time.Time) error {
	i.CoverageStatus = api.ItemCoverageStatusApproved

	if i.StatusChange != ItemStatusChangeAutoApproved {
		user := CurrentUser(ctx)
		i.StatusChange = ItemStatusChangeApproved + user.Name()
	}

	if err := i.Update(ctx); err != nil {
		return err
	}

	e := events.Event{
		Kind:    domain.EventApiItemApproved,
		Message: fmt.Sprintf("Item Approved: %s  ID: %s", i.Name, i.ID.String()),
		Payload: events.Payload{domain.EventPayloadID: i.ID},
	}
	emitEvent(e)

	tx := Tx(ctx)
	coverage := i.getInitialCoverage(tx, now)

	if err := i.CreateLedgerEntry(tx, LedgerEntryTypeNewCoverage, coverage.Premium, now); err != nil {
		return err
	}

	i.CoverageStartDate = coverage.StartDate
	if err := tx.UpdateColumns(i, "coverage_start_date", "updated_at"); err != nil {
		return appErrorFromDB(err, api.ErrorUpdateFailure)
	}

	return i.SetPaidThroughDate(tx, coverage.EndDate)
}

func (i *Item) getInitialCoverage(tx *pop.Connection, now time.Time) CoveragePeriod {
	var coverage CoveragePeriod

	i.LoadCategory(tx, false)
	if i.Category.GetBillingPeriod() == domain.BillingPeriodMonthly {
		// After the cutoff day, no premiums are billed until the next month. See CVR-730.
		if now.Day() < MonthlyCutoffDay {
			coverage.Premium = i.CalculateMonthlyPremium(tx)
		}
		coverage.EndDate = domain.EndOfMonth(now)
	} else {
		coverage.Premium = i.CalculateProratedPremium(tx, now)
		coverage.EndDate = domain.EndOfYear(now.Year())
	}

	// Coverage start date is the same regardless of billing, effectively giving free coverage
	// in some cases. See CVR-729.
	coverage.StartDate = now
	return coverage
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

// LoadCategory - a simple wrapper method for loading the item category
func (i *Item) LoadCategory(tx *pop.Connection, reload bool) {
	if i.Category.ID == uuid.Nil || reload {
		if err := tx.Load(i, "Category"); err != nil {
			panic("error loading item category: " + err.Error())
		}
	}
}

func (i *Item) ConvertToAPI(tx *pop.Connection) api.Item {
	i.LoadCategory(tx, false)
	i.LoadRiskCategory(tx, false)

	var coverageEndDate *string
	if i.CoverageEndDate.Valid {
		s := i.CoverageEndDate.Time.Format(domain.DateFormat)
		coverageEndDate = &s
	}

	apiItem := api.Item{
		ID:                    i.ID,
		Name:                  i.Name,
		Category:              i.Category.ConvertToAPI(tx),
		RiskCategory:          i.RiskCategory.ConvertToAPI(),
		InStorage:             i.InStorage,
		Country:               i.Country,
		Description:           i.Description,
		PolicyID:              i.PolicyID,
		Make:                  i.Make,
		Model:                 i.Model,
		SerialNumber:          i.SerialNumber,
		Year:                  NullsIntToPointer(i.Year),
		CoverageAmount:        i.CoverageAmount,
		CoverageStatus:        i.CoverageStatus,
		StatusChange:          i.StatusChange,
		StatusReason:          i.StatusReason,
		CoverageStartDate:     i.CoverageStartDate.Format(domain.DateFormat),
		CoverageEndDate:       coverageEndDate,
		BillingPeriod:         i.Category.GetBillingPeriod(),
		AnnualPremium:         i.CalculateAnnualPremium(tx),
		MonthlyPremium:        i.CalculateMonthlyPremium(tx),
		ProratedAnnualPremium: i.CalculateProratedPremium(tx, time.Now().UTC()),
		CanBeDeleted:          i.canBeDeleted(tx),
		CanBeUpdated:          !i.hasOpenClaim(tx),
		CreatedAt:             i.CreatedAt,
		UpdatedAt:             i.UpdatedAt,
	}
	person := i.GetAccountablePerson(tx)
	if person != nil {
		apiItem.AccountablePerson = api.AccountablePerson{
			ID:      person.GetID(),
			Name:    person.GetName().String(),
			Country: person.GetLocation().Country,
		}
	}
	return apiItem
}

// This function is only intended to be used for items that have been active
// but are now scheduled to become inactive.
// As such, any credit ledger entries should have already been created.
func (i *Item) inactivateEnded(ctx context.Context) error {
	tx := Tx(ctx)

	if i.hasOpenClaim(tx) {
		err := errors.New("item cannot be made inactive because it has an active claim")
		return api.NewAppError(err, api.ErrorItemHasActiveClaim, api.CategoryUser)
	}

	history := i.NewHistory(ctx,
		api.HistoryActionUpdate,
		FieldUpdate{
			OldValue:  string(i.CoverageStatus),
			NewValue:  string(api.ItemCoverageStatusInactive),
			FieldName: FieldItemCoverageStatus,
		})
	history.ItemID = nulls.NewUUID(i.ID)
	if err := history.Create(tx); err != nil {
		return err
	}

	log.WithFields(map[string]any{"item_id": i.ID}).Infof("marking item as %s", api.ItemCoverageStatusInactive)

	i.CoverageStatus = api.ItemCoverageStatusInactive

	return update(tx, i)
}

// InactivateApprovedButEnded fetches all the items that have coverage_status=Approved
// and coverage_end_date before today and then
// saves them with coverage_status=Inactive
func (i *Items) InactivateApprovedButEnded(ctx context.Context) error {
	tx := Tx(ctx)

	user := CurrentUser(ctx)
	if user.ID == uuid.Nil {
		return errors.New("InactivateApprovedButEnded must be given a context with a valid user")
	}

	endDate := time.Now().UTC().Format(domain.DateFormat)
	if err := tx.Where(`coverage_status = ? AND coverage_end_date < ?`,
		api.ItemCoverageStatusApproved, endDate).All(i); domain.IsOtherThanNoRows(err) {
		return fmt.Errorf("error fetching items that are approved but have "+
			"a coverage end date before %s: %s", endDate, err)
	}

	errCount := 0
	var lastErr error
	for _, ii := range *i {
		if err := ii.inactivateEnded(ctx); err != nil {
			errCount++
			log.Error("InactivateApprovedButEnded error,", err)
			lastErr = err
		}
	}
	if lastErr != nil {
		return fmt.Errorf("InactivateApprovedButEnded had %d errors. Last error: %s", errCount, lastErr)
	}

	return nil
}

func (i *Items) ConvertToAPI(tx *pop.Connection) api.Items {
	apiItems := make(api.Items, len(*i))
	for j, ii := range *i {
		apiItems[j] = ii.ConvertToAPI(tx)
	}

	return apiItems
}

// CalculateAnnualPremium returns the premium amount for the category's billing period:
func (i *Item) CalculateBillingPremium(tx *pop.Connection) api.Currency {
	i.LoadCategory(tx, false)
	billingPeriod := i.Category.GetBillingPeriod()

	switch billingPeriod {
	case domain.BillingPeriodMonthly:
		return i.CalculateMonthlyPremium(tx)
	case domain.BillingPeriodAnnual:
		return i.CalculateAnnualPremium(tx)
	}

	log.Fatalf("invalid billing period found in item category %s", i.Name)
	return 0
}

// CalculateAnnualPremium returns the rounded product of the item's CoverageAmount and the category's
// PremiumFactor, with the category's minimum premium applied
func (i *Item) CalculateAnnualPremium(tx *pop.Connection) api.Currency {
	i.LoadCategory(tx, false)
	factor := domain.Env.PremiumFactor
	if i.Category.PremiumFactor.Valid {
		factor = i.Category.PremiumFactor.Float64
	}
	premium := int(math.Round(float64(i.CoverageAmount) * factor))

	return api.Currency(domain.Max(premium, i.Category.MinimumPremium))
}

func (i *Item) CalculateProratedPremium(tx *pop.Connection, t time.Time) api.Currency {
	p := domain.CalculatePartialYearValue(int(i.CalculateAnnualPremium(tx)), t)

	i.LoadCategory(tx, false)
	return api.Currency(domain.Max(p, i.Category.MinimumPremium))
}

// CalculateMonthlyPremium returns the rounded product of the item's CoverageAmount and the category's
// PremiumFactor divided by BillingPeriodAnnual, with the category's minimum premium applied
func (i *Item) CalculateMonthlyPremium(tx *pop.Connection) api.Currency {
	i.LoadCategory(tx, false)
	factor := domain.Env.PremiumFactor
	if i.Category.PremiumFactor.Valid {
		factor = i.Category.PremiumFactor.Float64
	}
	premium := int(math.Round(float64(i.CoverageAmount) * factor))

	premium = domain.Max(premium, i.Category.MinimumPremium)
	return api.Currency(premium / 12)
}

// True if coverage on the item started in a previous year and the current
// month is January.
func (i *Item) shouldGiveFullYearRefund(t time.Time) bool {
	return i.CoverageStartDate.Year() < t.Year() && t.Month() == 1
}

func (i *Item) calculateCancellationCredit(tx *pop.Connection, t time.Time) api.Currency {
	// If we're in December already, then no credit
	if t.Month() == 12 {
		return 0
	}

	premium := int(i.CalculateAnnualPremium(tx))

	// If the coverage was from a previous year and today is still in January,
	//   give a full year's refund.
	if i.shouldGiveFullYearRefund(t) {
		return api.Currency(-1 * premium)
	}

	// Otherwise, give credit for the following calendar months
	credit := domain.CalculateMonthlyRefundValue(premium, t)
	return api.Currency(-1 * credit)
}

func (i *Item) calculatePremiumChange(t time.Time, oldPremium, newPremium api.Currency) api.Currency {
	// These will be positive numbers
	credit := domain.CalculatePartialYearValue(int(oldPremium), t)
	charge := domain.CalculatePartialYearValue(int(newPremium), t)

	return api.Currency(charge - credit)
}

// NewItemFromApiInput creates a new `Item` from a `ItemCreate`.
func NewItemFromApiInput(c buffalo.Context, input api.ItemCreate, policyID uuid.UUID) (Item, error) {
	item := Item{}
	tx := Tx(c)

	var itemCat ItemCategory
	if err := itemCat.FindByID(tx, input.CategoryID); err != nil {
		return item, err
	}

	user := CurrentUser(c)
	riskCatID := itemCat.RiskCategoryID
	if input.RiskCategoryID != nil && user.IsAdmin() {
		riskCatID = *input.RiskCategoryID
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
	item.Year = PointerToNullsInt(input.Year)
	item.CoverageAmount = input.CoverageAmount
	item.CoverageStatus = input.CoverageStatus

	if err := item.SetAccountablePerson(tx, input.AccountablePersonID); err != nil {
		return item, err
	}

	return item, nil
}

// SetAccountablePerson sets the appropriate field to the given ID, but does not update the database
func (i *Item) SetAccountablePerson(tx *pop.Connection, id uuid.UUID) error {
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

// CreateLedgerEntry creates a ledger entry for the given type and amount. It makes any needed adjustments and negates
// the amount before saving to the database.
func (i *Item) CreateLedgerEntry(tx *pop.Connection, entryType LedgerEntryType, amount api.Currency, date time.Time) error {
	adjustedAmount, err := adjustLedgerAmount(amount, entryType)
	if err != nil {
		return err
	}

	i.LoadPolicy(tx, false)
	i.LoadRiskCategory(tx, false)
	i.Policy.LoadEntityCode(tx, false)
	name := i.GetAccountablePersonName(tx).String()

	le := NewLedgerEntry(name, i.Policy, i, nil, date)
	le.Type = entryType
	le.Amount = -adjustedAmount

	if err := le.Create(tx); err != nil {
		return err
	}

	return nil
}

func (i *Item) SetPaidThroughDate(tx *pop.Connection, date time.Time) error {
	date = date.Truncate(24 * time.Hour)
	if date == i.PaidThroughDate {
		return nil
	}

	i.PaidThroughDate = date
	if err := tx.UpdateColumns(i, "paid_through_date", "updated_at"); err != nil {
		return appErrorFromDB(err, api.ErrorUpdateFailure)
	}
	return nil
}

// GetAccountablePersonName gets the name of the accountable person. In case of error, empty strings
// are returned.
func (i *Item) GetAccountablePersonName(tx *pop.Connection) Name {
	person := i.GetAccountablePerson(tx)
	if person == nil {
		return Name{}
	}
	return person.GetName()
}

// GetAccountablePersonLocation gets the location of the accountable person
func (i *Item) GetAccountablePersonLocation(tx *pop.Connection) Location {
	person := i.GetAccountablePerson(tx)
	if person == nil {
		return Location{}
	}
	return person.GetLocation()
}

// GetAccountablePerson gets the accountable person as a Person interface
func (i *Item) GetAccountablePerson(tx *pop.Connection) Person {
	var person Person

	if i.PolicyUserID.Valid {
		var user User
		if err := user.FindByID(tx, i.PolicyUserID.UUID); err != nil {
			panic("item PolicyUserID contains invalid user ID")
		}
		person = &user
	}
	if i.PolicyDependentID.Valid {
		var dep PolicyDependent
		if err := dep.FindByID(tx, i.PolicyDependentID.UUID); err != nil {
			panic("item PolicyDependentID contains invalid user ID")
		}
		person = &dep
	}
	return person
}

// GetAccountableMember gets either the accountable person if they are a User or
// the first member of the item's policy
func (i *Item) GetAccountableMember(tx *pop.Connection) Person {
	var person Person
	if i.PolicyUserID.Valid {
		var user User
		if err := user.FindByID(tx, i.PolicyUserID.UUID); err != nil {
			panic("item PolicyUserID contains invalid user ID")
		}
		person = &user
		return person
	}

	i.LoadPolicy(tx, false)
	i.Policy.LoadMembers(tx, false)
	if len(i.Policy.Members) < 1 {
		panic("item policy has no members")
	}
	person = &i.Policy.Members[0]
	return person
}

func (i *Item) GetMakeModel() string {
	return strings.TrimSpace(i.Make + " " + i.Model)
}

// hasOpenClaim returns a value of true when the item has an open Claim
func (i *Item) hasOpenClaim(tx *pop.Connection) bool {
	// closed states are those in which related items can be edited, e.g. can change CoverageAmount
	closedClaimStatuses := []api.ClaimStatus{
		api.ClaimStatusDraft,
		api.ClaimStatusPaid,
		api.ClaimStatusDenied,
	}

	var claims Claims
	n, err := tx.Where("claim_items.item_id = ?", i.ID).
		Where("claims.status NOT IN (?)", closedClaimStatuses).
		Join("claim_items", "claims.id = claim_items.claim_id").
		Count(&claims)
	if err != nil {
		panic(err.Error())
	}
	return n > 0
}

// ItemsWithRecentStatusChanges returns the RecentItems associated with
// items that have had their CoverageStatus changed recently.
// The slice is sorted by updated time with most recent first.
func ItemsWithRecentStatusChanges(tx *pop.Connection) (api.RecentItems, error) {
	var pHistories PolicyHistories

	if err := pHistories.RecentItemStatusChanges(tx); err != nil {
		return api.RecentItems{}, err
	}

	// Fetch the actual items from the database and convert them to api types
	items := make(api.RecentItems, len(pHistories))
	for i, next := range pHistories {
		var item Item
		if !next.ItemID.Valid {
			continue
		}
		if err := item.FindByID(tx, next.ItemID.UUID); err != nil {
			panic("error finding item by ID: " + err.Error())
		}

		apiItem := item.ConvertToAPI(tx)
		items[i] = api.RecentItem{Item: apiItem, StatusUpdatedAt: next.CreatedAt}
	}

	return items, nil
}

// NewHistory returns a new PolicyHistory template object, not yet added to the database.
func (i *Item) NewHistory(ctx context.Context, action string, fieldUpdate FieldUpdate) PolicyHistory {
	return PolicyHistory{
		Action:    action,
		PolicyID:  i.PolicyID,
		ItemID:    nulls.NewUUID(i.ID),
		UserID:    CurrentUser(ctx).ID,
		FieldName: fieldUpdate.FieldName,
		OldValue:  fmt.Sprintf("%s", fieldUpdate.OldValue),
		NewValue:  fmt.Sprintf("%s", fieldUpdate.NewValue),
	}
}

// FindItemsIncorrectlyRenewed locates any items that were incorrectly renewed for another year of coverage. These are
// identified as items that are marked as paid through the year but have an earlier coverage_end_date.
func (i *Items) FindItemsIncorrectlyRenewed(tx *pop.Connection, date time.Time) error {
	year := date.Year()
	firstDayOfYear := time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC)

	err := tx.Where("paid_through_date >= ?", domain.EndOfYear(year)).
		Where("coverage_end_date < ?", firstDayOfYear).All(i)
	if err != nil {
		return appErrorFromDB(err, api.ErrorQueryFailure)
	}
	return nil
}

// RepairItemsIncorrectlyRenewed repairs items that were incorrectly renewed for another year of coverage. These are
// identified as items that are marked as paid through the year but have an earlier coverage_end_date.
func (i *Items) RepairItemsIncorrectlyRenewed(c buffalo.Context, date time.Time) error {
	tx := Tx(c)
	if err := i.FindItemsIncorrectlyRenewed(tx, date); err != nil {
		return err
	}

	for idx, item := range *i {
		if !item.CoverageEndDate.Valid {
			err := errors.New("item coverage_end_date is not set, can't proceed with repair")
			return api.NewAppError(err, api.ErrorItemNeedsCoverageEndDate, api.CategoryInternal)
		}

		correctDate := item.CoverageEndDate.Time
		incorrectDate := item.PaidThroughDate
		annualPremium := item.CalculateAnnualPremium(tx)                                          // TODO: get the amount from the ledger entry, in case the coverage amount has changed
		refund := annualPremium * api.Currency(incorrectDate.Sub(correctDate)/(time.Hour*24*365)) // TODO: use CalculatePartialYearValue?

		now := time.Now().UTC()
		if err := item.CreateLedgerEntry(tx, LedgerEntryTypeCoverageRefund, -refund, now); err != nil {
			return err
		}

		if err := (*i)[idx].SetPaidThroughDate(tx, correctDate); err != nil {
			return err
		}
	}
	return nil
}

func CountItemsToRenew(tx *pop.Connection, date time.Time, billingPeriod int) (int, error) {
	var items Items
	count, err := tx.Where("coverage_status = ?", api.ItemCoverageStatusApproved).
		Where("paid_through_date < ?", date).
		Join("item_categories ic", "ic.id = items.category_id").
		Where("ic.billing_period = ?", billingPeriod).
		Count(&items)
	if err != nil {
		return 0, appErrorFromDB(err, api.ErrorQueryFailure)
	}
	return count, nil
}

func (i *Item) createPremiumAdjustment(tx *pop.Connection, date time.Time, oldItem Item) error {
	oldPremium := oldItem.CalculateBillingPremium(tx)
	newPremium := i.CalculateBillingPremium(tx)

	if oldPremium != newPremium && !i.CoverageEndDate.Valid {
		amount := i.calculatePremiumChange(date, oldPremium, newPremium)
		if err := i.CreateLedgerEntry(tx, LedgerEntryTypeCoverageChange, amount, date); err != nil {
			return err
		}
	}
	return nil
}
