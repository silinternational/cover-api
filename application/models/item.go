package models

import (
	"time"

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
	ID   uuid.UUID `db:"id"`
	Name string    `db:"name" validate:"required"`

	/*
		category_id
		in_storage
		country
		description
		policy_id
		policy_dependent_id
		make
		model
		serial_number
		coverage_amount
		purchase_date
		coverage_status
		coverage_start_date
	*/
	Status    ItemCoverageStatus `db:"status" validate:"itemCoverageStatus"`
	CreatedAt time.Time          `db:"created_at"`
	UpdatedAt time.Time          `db:"updated_at"`
}

// Validate gets run every time you call pop.ValidateAndSave, pop.ValidateAndCreate, or pop.ValidateAndUpdate
func (r *Item) Validate(tx *pop.Connection) (*validate.Errors, error) {
	return validateModel(r), nil
}

func (r *Item) GetID() uuid.UUID {
	return r.ID
}

func (r *Item) FindByID(tx *pop.Connection, id uuid.UUID) error {
	return tx.Find(r, id)
}
