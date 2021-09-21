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
	EntityID      nulls.UUID `db:"entity_id"`
	Amount        int        `db:"amount"`
	DateSubmitted string     `db:"date_submitted"`

	// The following fields are primarily for legacy data and may not be needed long-term
	// However, some may be useful as a permanent record in case policies change...TBD.
	DateEntered string    `db:"date_entered"`
	JeRecNum    nulls.Int `db:"je_rec_num"`
	JeRecType   nulls.Int `db:"je_rec_type"`
	PolicyType  nulls.Int `db:"policy_type"`
	AccNum      string    `db:"acc_num"`
	AccCostCtr1 string    `db:"acc_cost_ctr1"`
	AccCostCtr2 string    `db:"acc_cost_ctr2"`
	Entity      string    `db:"entity"`
	FirstName   string    `db:"first_name"`
	LastName    string    `db:"last_name"`

	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

func (ec *LedgerEntry) Create(tx *pop.Connection) error {
	return create(tx, ec)
}
