package models

import (
	"net/http"
	"time"

	"github.com/silinternational/riskman-api/api"

	"github.com/gobuffalo/nulls"
	"github.com/gobuffalo/pop/v5"
	"github.com/gobuffalo/validate/v3"
	"github.com/gofrs/uuid"

	"github.com/silinternational/riskman-api/domain"
)

var ValidItemCoverageStatuses = map[api.ItemCoverageStatus]struct{}{
	api.ItemCoverageStatusDraft:    {},
	api.ItemCoverageStatusPending:  {},
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
	CreatedAt         time.Time              `db:"created_at"`
	UpdatedAt         time.Time              `db:"updated_at"`

	Category ItemCategory `belongs_to:"item_categories" validate:"-"`
	Policy   Policy       `belongs_to:"policies" validate:"-"`
}

// Validate gets run every time you call pop.ValidateAndSave, pop.ValidateAndCreate, or pop.ValidateAndUpdate
func (i *Item) Validate(tx *pop.Connection) (*validate.Errors, error) {
	return validateModel(i), nil
}

func (i *Item) Create(tx *pop.Connection) error {
	return create(tx, i)
}

func (i *Item) Update(tx *pop.Connection) error {
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
	i.CoverageStatus = api.ItemCoverageStatusInactive
	return i.Update(tx)
}

// IsActorAllowedTo ensure the actor is either an admin, or a member of this policy to perform any permission
func (i *Item) IsActorAllowedTo(tx *pop.Connection, actor User, perm Permission, sub SubResource, req *http.Request) bool {
	if actor.IsAdmin() {
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

// LoadPolicy - a simple wrapper method for loading the policy
func (i *Item) LoadPolicy(tx *pop.Connection, reload bool) {
	if i.Policy.ID == uuid.Nil || reload {
		if err := tx.Load(i, "Policy"); err != nil {
			panic("error loading item policy: " + err.Error())
		}
	}
}

// LoadCategory - a simple wrapper method for loading an item category on the struct
func (i *Item) LoadCategory(tx *pop.Connection, reload bool) {
	if i.Category.ID == uuid.Nil || reload {
		if err := tx.Load(i, "Category"); err != nil {
			msg := "error loading item category: " + err.Error()
			panic(msg)
		}
	}
}

func ConvertItem(tx *pop.Connection, item Item) api.Item {
	item.LoadCategory(tx, false)

	iCat := ConvertItemCategory(tx, item.Category)

	return api.Item{
		ID:                item.ID,
		Name:              item.Name,
		CategoryID:        item.CategoryID,
		Category:          iCat,
		InStorage:         item.InStorage,
		Country:           item.Country,
		Description:       item.Description,
		PolicyID:          item.PolicyID,
		Make:              item.Make,
		Model:             item.Model,
		SerialNumber:      item.SerialNumber,
		CoverageAmount:    item.CoverageAmount,
		PurchaseDate:      item.PurchaseDate.Format(domain.DateFormat),
		CoverageStatus:    item.CoverageStatus,
		CoverageStartDate: item.CoverageStartDate.Format(domain.DateFormat),
		CreatedAt:         item.CreatedAt,
		UpdatedAt:         item.UpdatedAt,
	}
}

func ConvertItems(tx *pop.Connection, items Items) api.Items {
	apiItems := make(api.Items, len(items))
	for i, p := range items {
		apiItems[i] = ConvertItem(tx, p)
	}

	return apiItems
}
