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

func (i *Item) GetID() uuid.UUID {
	return i.ID
}

func (i *Item) FindByID(tx *pop.Connection, id uuid.UUID) error {
	return tx.Find(i, id)
}

// IsActorAllowedTo ensure the actor is either an admin, or a member of this policy to perform any permission
func (i *Item) IsActorAllowedTo(tx *pop.Connection, user User, perm Permission, sub SubResource, req *http.Request) bool {
	if user.IsAdmin() {
		return true
	}

	if err := i.LoadPolicy(tx, false); err != nil {
		domain.ErrLogger.Printf("failed to load policy for item: %s", err)
		return false
	}

	if err := i.Policy.LoadMembers(tx, false); err != nil {
		domain.ErrLogger.Printf("failed to load members on policy: %s", err)
		return false
	}

	for _, m := range i.Policy.Members {
		if m.ID == user.ID {
			return true
		}
	}

	return false
}

// LoadPolicy - a simple wrapper method for loading the policy
func (i *Item) LoadPolicy(tx *pop.Connection, reload bool) error {
	if i.Policy.ID == uuid.Nil || reload {
		if err := tx.Load(i, "Policy"); err != nil {
			return err
		}
	}

	return nil
}

// LoadCategory - a simple wrapper method for loading an item category on the struct
func (i *Item) LoadCategory(tx *pop.Connection, reload bool) error {
	if i.Category.ID == uuid.Nil || reload {
		if err := tx.Load(i, "Category"); err != nil {
			return err
		}
	}

	return nil
}

func ConvertItem(tx *pop.Connection, item Item) (api.Item, error) {
	if err := item.LoadCategory(tx, false); err != nil {
		return api.Item{}, err
	}

	iCat, err := ConvertItemCategory(tx, item.Category)
	if err != nil {
		return api.Item{}, err
	}

	return api.Item{
		ID:                item.ID,
		Name:              item.Name,
		CategoryID:        item.CategoryID,
		Category:          iCat,
		InStorage:         item.InStorage,
		Country:           item.Country,
		Description:       item.Description,
		Make:              item.Make,
		Model:             item.Model,
		SerialNumber:      item.SerialNumber,
		CoverageAmount:    item.CoverageAmount,
		PurchaseDate:      item.PurchaseDate,
		CoverageStatus:    item.CoverageStatus,
		CoverageStartDate: item.CoverageStartDate,
		CreatedAt:         item.CreatedAt,
		UpdatedAt:         item.UpdatedAt,
	}, nil
}

func ConvertItems(tx *pop.Connection, items Items) (api.Items, error) {
	apiItems := make(api.Items, len(items))
	for i, p := range items {
		var err error
		apiItems[i], err = ConvertItem(tx, p)
		if err != nil {
			return nil, err
		}
	}

	return apiItems, nil
}
