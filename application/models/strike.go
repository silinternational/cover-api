package models

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/gobuffalo/nulls"
	"github.com/gobuffalo/pop/v5"
	"github.com/gobuffalo/validate/v3"
	"github.com/gofrs/uuid"

	"github.com/silinternational/cover-api/api"
	"github.com/silinternational/cover-api/domain"
)

// Items is a slice of Item objects
type Strikes []Item

// Strike model
type Strike struct {
	ID          uuid.UUID `db:"id"`
	Description string    `db:"description"`
	PolicyID    uuid.UUID `db:"policy_id" validate:"required"`
	CreatedAt   time.Time `db:"created_at"`
	UpdatedAt   time.Time `db:"updated_at"`
}

// Validate gets run every time you call pop.ValidateAndSave, pop.ValidateAndCreate, or pop.ValidateAndUpdate
func (s *Strike) Validate(tx *pop.Connection) (*validate.Errors, error) {
	return validateModel(s), nil
}

func (s *Strike) Create(tx *pop.Connection) error {
	return create(tx, s)
}

func (s *Strike) Update(ctx context.Context) error {
	tx := Tx(ctx)
	//var oldStrike Strike
	//if err := oldStrike.FindByID(tx, s.ID); err != nil {
	//	return appErrorFromDB(err, api.ErrorQueryFailure)
	//}

	return update(tx, s)
}

func (s *Strike) GetID() uuid.UUID {
	return s.ID
}

func (s *Strike) FindByID(tx *pop.Connection, id uuid.UUID) error {
	return tx.Find(s, id)
}

// isNewEnough checks whether the item was created in the last X hours
func (s *Strike) isNewEnough() bool {
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

// IsActorAllowedTo ensure the actor is either an admin, or a member of this policy to perform any permission
func (s *Strike) IsActorAllowedTo(tx *pop.Connection, actor User, perm Permission, sub SubResource, req *http.Request) bool {
	return actor.IsAdmin()
}

func (s *Strike) ConvertToAPI(tx *pop.Connection) api.Item {
	apiStrike := api.Strike{
		ID:          s.ID,
		Description: s.Description,
		PolicyID:    s.PolicyID,
		CreatedAt:   s.CreatedAt,
		UpdatedAt:   s.UpdatedAt,
	}

	return apiStrike
}

// This function is only intended to be used for items that have been active
// but are now scheduled to become inactive.
// As such, any credit ledger entries should have already been created.
func (s *Strike) inactivateEnded(ctx context.Context) error {
	tx := Tx(ctx)

	if !i.canBeUpdated(tx) {
		err := errors.New("item cannot be updated because it has an active claim")
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

	i.CoverageStatus = api.ItemCoverageStatusInactive

	return update(tx, i)
}

func (s *Strikes) ConvertToAPI(tx *pop.Connection) api.Items {
	apiStrikes := make(api.Strikes, len(*s))
	for j, ss := range *s {
		apiStrikes[j] = ss.ConvertToAPI(tx)
	}

	return apiStrikes
}
