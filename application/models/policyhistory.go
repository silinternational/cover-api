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

// RecentItemStatusChanges hydrates the PolicyHistories with those that
//  have been created in the last week and that also have
//  a field_name of CoverageStatus and
//  an action of Update
func (p *PolicyHistories) RecentItemStatusChanges(tx *pop.Connection) error {
	now := time.Now().UTC()
	cutoffDate := now.Add(-1 * domain.DurationWeek)
	err := tx.Where(QueryRecentStatusChanges, cutoffDate, FieldItemCoverageStatus, api.HistoryActionUpdate).All(p)

	if domain.IsOtherThanNoRows(err) {
		return appErrorFromDB(err, api.ErrorQueryFailure)
	}
	return nil
}

func (p *PolicyHistories) getUniqueIDTimes() map[string]time.Time {
	uniqueIDTimes := map[string]time.Time{}

	for _, h := range *p {
		if !h.ItemID.Valid {
			continue
		}
		id := h.ItemID.UUID.String()
		previousTime, ok := uniqueIDTimes[id]
		if !ok {
			uniqueIDTimes[id] = h.CreatedAt
			continue
		}
		if h.CreatedAt.After(previousTime) {
			uniqueIDTimes[id] = h.CreatedAt
		}
	}
	return uniqueIDTimes
}
