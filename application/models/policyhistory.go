package models

import (
	"time"

	"github.com/gobuffalo/nulls"
	"github.com/gobuffalo/pop/v5"
	"github.com/gobuffalo/validate/v3"
	"github.com/gofrs/uuid"

	"github.com/silinternational/cover-api/api"
	"github.com/silinternational/cover-api/domain"
)

type PolicyHistories []PolicyHistory

type PolicyHistory struct {
	ID        uuid.UUID  `db:"id"`
	PolicyID  uuid.UUID  `db:"policy_id"`
	UserID    uuid.UUID  `db:"user_id"`
	Action    string     `db:"action"`
	FieldName string     `db:"field_name"`
	ItemID    nulls.UUID `db:"item_id"`
	OldValue  string     `db:"old_value"`
	NewValue  string     `db:"new_value"`
	CreatedAt time.Time  `db:"created_at"`
	UpdatedAt time.Time  `db:"updated_at"`
}

// Validate gets run every time you call a "pop.Validate*" (pop.ValidateAndSave, pop.ValidateAndCreate, pop.ValidateAndUpdate) method.
func (p *PolicyHistory) Validate(tx *pop.Connection) (*validate.Errors, error) {
	return validateModel(p), nil
}

func (p *PolicyHistory) Create(tx *pop.Connection) error {
	return create(tx, p)
}

func (p *PolicyHistory) GetID() uuid.UUID {
	return p.ID
}

func (p *PolicyHistory) FindByID(tx *pop.Connection, id uuid.UUID) error {
	return tx.Find(p, id)
}

// RecentItemStatusChanges finds the PolicyHistories that
//  have been created in the last week and that also have
//  a field_name of CoverageStatus and
//  an action of Update
// NOTE: only `ItemID` and `CreatedAt` are hydrated by this function
func (p *PolicyHistories) RecentItemStatusChanges(tx *pop.Connection) error {
	now := time.Now().UTC()
	cutoffDate := now.Add(-1 * domain.DurationWeek)
	err := tx.RawQuery(`
SELECT item_id,max(created_at) AS created_at
FROM policy_histories
WHERE created_at > ? AND field_name = ? AND action = ? AND item_id IS NOT NULL
GROUP BY item_id
ORDER BY created_at DESC
`, cutoffDate, FieldItemCoverageStatus, api.HistoryActionUpdate).All(p)

	if domain.IsOtherThanNoRows(err) {
		return appErrorFromDB(err, api.ErrorQueryFailure)
	}
	return nil
}
