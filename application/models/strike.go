package models

import (
	"context"
	"net/http"
	"time"

	"github.com/gobuffalo/pop/v5"
	"github.com/gobuffalo/validate/v3"
	"github.com/gofrs/uuid"

	"github.com/silinternational/cover-api/api"
	"github.com/silinternational/cover-api/domain"
)

// Items is a slice of Item objects
type Strikes []Strike

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
	return update(Tx(ctx), s)
}

func (s *Strike) Destroy(tx *pop.Connection) error {
	return destroy(tx, s)
}

func (s *Strike) GetID() uuid.UUID {
	return s.ID
}

func (s *Strike) FindByID(tx *pop.Connection, id uuid.UUID) error {
	return tx.Find(s, id)
}

// IsActorAllowedTo ensures the actor is an admin
func (s *Strike) IsActorAllowedTo(tx *pop.Connection, actor User, perm Permission, sub SubResource, req *http.Request) bool {
	return actor.IsAdmin()
}

func (s *Strike) ConvertToAPI() api.Strike {
	apiStrike := api.Strike{
		ID:          s.ID,
		Description: s.Description,
		PolicyID:    s.PolicyID,
		CreatedAt:   s.CreatedAt,
		UpdatedAt:   s.UpdatedAt,
	}

	return apiStrike
}

func (s *Strikes) ConvertToAPI(tx *pop.Connection) api.Strikes {
	apiStrikes := make(api.Strikes, len(*s))
	for j, ss := range *s {
		apiStrikes[j] = ss.ConvertToAPI()
	}

	return apiStrikes
}

// RecentForPolicy gets the strikes that have a matching policy_id and are newer than a year
// before the cutOff date and no more recent than the cutOff date
func (s *Strikes) RecentForPolicy(tx *pop.Connection, policyID uuid.UUID, cutOff time.Time) error {
	yearBefore := cutOff.AddDate(0, -domain.Env.StrikeLifetimeMonths, 0)

	if err := tx.Where("policy_id = ? AND created_at > ? AND created_at < ?",
		policyID, yearBefore, cutOff).All(s); err != nil {
	}

	return nil
}
