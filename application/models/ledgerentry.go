package models

import (
	"time"

	"github.com/gobuffalo/nulls"
	"github.com/gobuffalo/pop/v5"
	"github.com/gofrs/uuid"
)

type LedgerEntries []LedgerEntry

type LedgerEntry struct {
	ID uuid.UUID `db:"id"`

	PolicyID      uuid.UUID  `db:"policy_id"`
	ItemID        nulls.UUID `db:"item_id"`
	EntityCodeID  nulls.UUID `db:"entity_code_id"`
	Amount        int        `db:"amount"`
	DateSubmitted time.Time  `db:"date_submitted"`
	DateEntered   nulls.Time `db:"date_entered"`

	// The following fields are primarily for legacy data and may not be needed long-term
	// However, some may be useful as a permanent record in case policies change...TBD.
	LegacyID           nulls.Int `db:"legacy_id"`
	RecordType         nulls.Int `db:"record_type"`
	PolicyType         nulls.Int `db:"policy_type"`
	AccountNumber      string    `db:"account_number"`
	AccountCostCenter1 string    `db:"account_cost_center1"`
	AccountCostCenter2 string    `db:"account_cost_center2"`
	EntityCode         string    `db:"entity_code"`
	FirstName          string    `db:"first_name"`
	LastName           string    `db:"last_name"`

	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

func (ec *LedgerEntry) Create(tx *pop.Connection) error {
	return create(tx, ec)
}
