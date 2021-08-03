package models

import (
	"net/http"
	"time"

	"github.com/silinternational/riskman-api/domain"

	"github.com/gobuffalo/pop/v5"
	"github.com/gobuffalo/validate/v3"
	"github.com/gofrs/uuid"
)

type ItemCoverageStatus string

const (
	ItemCoverageStatusDraft    = ItemCoverageStatus("Draft")
	ItemCoverageStatusPending  = ItemCoverageStatus("Pending")
	ItemCoverageStatusApproved = ItemCoverageStatus("Approved")
	ItemCoverageStatusDenied   = ItemCoverageStatus("Denied")
)

var ValidItemCoverageStatuses = map[ItemCoverageStatus]struct{}{
	ItemCoverageStatusDraft:    {},
	ItemCoverageStatusPending:  {},
	ItemCoverageStatusApproved: {},
	ItemCoverageStatusDenied:   {},
}

// Items is a slice of Item objects
type Items []Item

// Item model
type Item struct {
	ID                uuid.UUID          `db:"id"`
	Name              string             `db:"name" validate:"required"`
	CategoryID        uuid.UUID          `db:"category_id" validate:"required"`
	InStorage         bool               `db:"in_storage"`
	Country           string             `db:"country"`
	Description       string             `db:"description"`
	PolicyID          uuid.UUID          `db:"policy_id" validate:"required"`
	PolicyDependentID uuid.UUID          `db:"policy_dependent_id"`
	Make              string             `db:"make"`
	Model             string             `db:"model"`
	SerialNumber      string             `db:"serial_number"`
	CoverageAmount    int                `db:"coverage_amount"`
	PurchaseDate      time.Time          `db:"purchase_date"`
	CoverageStatus    ItemCoverageStatus `db:"coverage_status" validate:"itemCategoryStatus"`
	CoverageStartDate time.Time          `db:"coverage_start_date"`
	CreatedAt         time.Time          `db:"created_at"`
	UpdatedAt         time.Time          `db:"updated_at"`

	Policy Policy `belongs_to:"policies"`
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
