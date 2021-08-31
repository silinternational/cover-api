package models

import (
	"errors"
	"fmt"
	"net/http"
	"time"

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
	Make              string                 `db:"make"`
	Model             string                 `db:"model"`
	SerialNumber      string                 `db:"serial_number"`
	CoverageAmount    int                    `db:"coverage_amount"`
	PurchaseDate      time.Time              `db:"purchase_date"`
	CoverageStatus    api.ItemCoverageStatus `db:"coverage_status" validate:"itemCoverageStatus"`
	CoverageStartDate time.Time              `db:"coverage_start_date"`
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
//  and if there are no ClaimItems associated with it.
//  Otherwise, it changes its status to Inactive.
func (i *Item) SafeDeleteOrInactivate(tx *pop.Connection, actor User) error {
	// TODO Add a check related to whether the item already got included in the billing process.

	if !i.isNewEnough() {
		return i.Inactivate(tx)
	}

	clItems := ClaimItems{}
	clICount, err := tx.Where("item_id = ?", i.ID).Count(&clItems)
	if err != nil {
		return api.NewAppError(
			err, api.ErrorQueryFailure, api.CategoryDatabase,
		)
	}

	if clICount > 0 {
		return i.Inactivate(tx)
	}

	return tx.Destroy(i)
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

	i.LoadPolicy(tx, false)

	i.Policy.LoadMembers(tx, false)

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
// TODO emit an event for the correct status transition
func (i *Item) SubmitForApproval(tx *pop.Connection) error {
	oldStatus := i.CoverageStatus
	i.CoverageStatus = api.ItemCoverageStatusPending

	i.Load(tx)

	if i.canAutoApprove(tx) {
		i.CoverageStatus = api.ItemCoverageStatusApproved
	}

	return i.Update(tx, oldStatus)
}

// Assumes the item already has its Category loaded
func (i *Item) canAutoApprove(tx *pop.Connection) bool {
	if i.CoverageAmount > i.Category.AutoApproveMax {
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
	return depTotal+i.CoverageAmount <= domain.Env.DependantAutoApproveMax
}

// Revision takes the item from Pending coverage status to Revision.
// It assumes that the item's current status has already been validated.
// TODO emit an event for the the status transition
func (i *Item) Revision(tx *pop.Connection) error {
	oldStatus := i.CoverageStatus
	i.CoverageStatus = api.ItemCoverageStatusRevision
	return i.Update(tx, oldStatus)
}

// Approve takes the item from Pending coverage status to Approved.
// It assumes that the item's current status has already been validated.
// TODO emit an event for the the status transition
func (i *Item) Approve(tx *pop.Connection) error {
	oldStatus := i.CoverageStatus
	i.CoverageStatus = api.ItemCoverageStatusApproved
	return i.Update(tx, oldStatus)
}

// Deny takes the item from Pending coverage status to Denied.
// It assumes that the item's current status has already been validated.
// TODO emit an event for the the status transition
func (i *Item) Deny(tx *pop.Connection) error {
	oldStatus := i.CoverageStatus
	i.CoverageStatus = api.ItemCoverageStatusDenied
	return i.Update(tx, oldStatus)
}

// LoadPolicy - a simple wrapper method for loading the policy
func (i *Item) LoadPolicy(tx *pop.Connection, reload bool) {
	if i.Policy.ID == uuid.Nil || reload {
		if err := tx.Load(i, "Policy"); err != nil {
			panic("error loading item policy: " + err.Error())
		}
	}
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
		ID:                i.ID,
		Name:              i.Name,
		Category:          i.Category.ConvertToAPI(tx),
		RiskCategory:      i.RiskCategory.ConvertToAPI(),
		InStorage:         i.InStorage,
		Country:           i.Country,
		Description:       i.Description,
		PolicyID:          i.PolicyID,
		Make:              i.Make,
		Model:             i.Model,
		SerialNumber:      i.SerialNumber,
		CoverageAmount:    i.CoverageAmount,
		PurchaseDate:      i.PurchaseDate.Format(domain.DateFormat),
		CoverageStatus:    i.CoverageStatus,
		CoverageStartDate: i.CoverageStartDate.Format(domain.DateFormat),
		CreatedAt:         i.CreatedAt,
		UpdatedAt:         i.UpdatedAt,
	}
}

func (i *Items) ConvertToAPI(tx *pop.Connection) api.Items {
	apiItems := make(api.Items, len(*i))
	for j, ii := range *i {
		apiItems[j] = ii.ConvertToAPI(tx)
	}

	return apiItems
}
