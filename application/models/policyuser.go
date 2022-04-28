package models

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gobuffalo/nulls"
	"github.com/gobuffalo/pop/v5"
	"github.com/gobuffalo/validate/v3"
	"github.com/gofrs/uuid"

	"github.com/silinternational/cover-api/api"
	"github.com/silinternational/cover-api/domain"
)

type PolicyUsers []PolicyUser

type PolicyUser struct {
	ID       uuid.UUID `db:"id"`
	PolicyID uuid.UUID `db:"policy_id"`
	UserID   uuid.UUID `db:"user_id"`

	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

// Validate gets run every time you call a "pop.Validate*" (pop.ValidateAndSave, pop.ValidateAndCreate, pop.ValidateAndUpdate) method.
func (p *PolicyUser) Validate(tx *pop.Connection) (*validate.Errors, error) {
	return validateModel(p), nil
}

// Create stores the data as a new record in the database.
func (p *PolicyUser) Create(tx *pop.Connection) error {
	return create(tx, p)
}

func (p *PolicyUser) GetID() uuid.UUID {
	return p.ID
}

func (p *PolicyUser) FindByID(tx *pop.Connection, id uuid.UUID) error {
	return tx.Find(p, id)
}

// IsActorAllowedTo ensure the actor is either an admin, or a member of this policy to perform any permission
func (p *PolicyUser) IsActorAllowedTo(tx *pop.Connection, actor User, perm Permission, sub SubResource, r *http.Request) bool {
	if actor.IsAdmin() {
		return true
	}

	var policy Policy
	if err := policy.FindByID(tx, p.PolicyID); err != nil {
		domain.ErrLogger.Printf("failed to load policy for dependent: %s", err)
		return false
	}

	policy.LoadMembers(tx, false)

	for _, m := range policy.Members {
		if m.ID == actor.ID {
			return true
		}
	}

	return false
}

func (p *PolicyUser) FindByPolicyAndUserIDs(tx *pop.Connection, policyID, userID uuid.UUID) error {
	return tx.Where(`policy_id = ? AND user_id = ?`, policyID, userID).First(p)
}

// Delete removes a policy member if there is an additional PolicyUser for the related policy and
//  nulls out the PolicyUserID on all related items
func (p *PolicyUser) Delete(ctx context.Context) error {
	tx := Tx(ctx)

	var pUsers PolicyUsers
	if err := tx.Where("policy_id = ?", p.PolicyID).All(&pUsers); domain.IsOtherThanNoRows(err) {
		panic(fmt.Sprintf("error fetching policy users with policy_id %s, %s", p.PolicyID.String(), err))
	}

	// Ensure there is at least one backup policy user
	if len(pUsers) < 2 {
		err := api.NewAppError(errors.New("may not delete the last of a policy's users"),
			api.ErrorPolicyUserIsTheLast, api.CategoryForbidden)
		return err
	}

	// update all related items with a null PolicyUserID
	items := p.RelatedItems(tx)
	for _, i := range items {
		i.PolicyUserID = nulls.UUID{}
		if err := i.Update(ctx); err != nil {
			panic("error updating item with no policy user: " + err.Error())
		}
	}

	// Destroy the PolicyUser
	if err := tx.Destroy(p); err != nil {
		err := api.NewAppError(fmt.Errorf("error deleting policy user with id: %s, %s", p.ID.String(), err),
			api.ErrorPolicyUserDelete, api.CategoryDatabase)
		return err
	}

	return nil
}

// RelatedItems returns a slice of the Items that are related to this policy_user
func (p *PolicyUser) RelatedItems(tx *pop.Connection) Items {
	var items Items
	if err := tx.Where("policy_user_id = ?", p.ID).All(&items); err != nil {
		panic(fmt.Sprintf("error fetching items with policy_user_id %s, %s", p.ID, err))
	}

	return items
}
