package models

import (
	"net/http"
	"time"

	"github.com/gobuffalo/nulls"
	"github.com/gobuffalo/pop/v5"
	"github.com/gobuffalo/validate/v3"
	"github.com/gofrs/uuid"
	"github.com/pkg/errors"

	"github.com/silinternational/cover-api/api"
)

type Policies []Policy

var ValidPolicyTypes = map[api.PolicyType]struct{}{
	api.PolicyTypeHousehold: {},
	api.PolicyTypeCorporate: {},
}

type Policy struct {
	ID          uuid.UUID      `db:"id"`
	Type        api.PolicyType `db:"type" validate:"policyType"`
	HouseholdID nulls.String   `db:"household_id"`
	CostCenter  string         `db:"cost_center" validate:"required_if=Type Corporate"`
	Account     string         `db:"account" validate:"required_if=Type Corporate"`
	EntityCode  string         `db:"entity_code" validate:"required_if=Type Corporate"`
	Notes       string         `db:"notes"`
	LegacyID    nulls.Int      `db:"legacy_id"`
	CreatedAt   time.Time      `db:"created_at"`
	UpdatedAt   time.Time      `db:"updated_at"`

	Claims     Claims           `has_many:"claims" validate:"-"`
	Dependents PolicyDependents `has_many:"policy_dependents" validate:"-"`
	Items      Items            `has_many:"items" validate:"-"`
	Members    Users            `many_to_many:"policy_users" validate:"-"`
}

// Validate gets run every time you call a "pop.Validate*" (pop.ValidateAndSave, pop.ValidateAndCreate, pop.ValidateAndUpdate) method.
func (p *Policy) Validate(tx *pop.Connection) (*validate.Errors, error) {
	return validateModel(p), nil
}

// Create stores the Policy data as a new record in the database.
func (p *Policy) Create(tx *pop.Connection) error {
	return create(tx, p)
}

// Update writes the Policy data to an existing database record.
func (p *Policy) Update(tx *pop.Connection) error {
	return update(tx, p)
}

func (p *Policy) GetID() uuid.UUID {
	return p.ID
}

func (p *Policy) FindByID(tx *pop.Connection, id uuid.UUID) error {
	return tx.Find(p, id)
}

// IsActorAllowedTo ensure the actor is either an admin, or a member of this policy to perform any permission
func (p *Policy) IsActorAllowedTo(tx *pop.Connection, actor User, perm Permission, sub SubResource, r *http.Request) bool {
	if actor.IsAdmin() || perm == PermissionList {
		return true
	}

	p.LoadMembers(tx, false)

	for _, m := range p.Members {
		if m.ID == actor.ID {
			return true
		}
	}

	return false
}

// itemCoverageTotals returns a map with an entry for
//  the policy ID with the total of all the items' coverage amounts as well as
//  an entry for each dependant with the total of each of their items' coverage amounts
func (p *Policy) itemCoverageTotals(tx *pop.Connection) map[uuid.UUID]int {
	p.LoadItems(tx, false)

	addToTotals := func(newKey uuid.UUID, newAmount int, totals map[uuid.UUID]int) {
		oldTotal, ok := totals[newKey]
		if !ok {
			totals[newKey] = newAmount
		} else {
			totals[newKey] = oldTotal + newAmount
		}
	}

	totals := map[uuid.UUID]int{}

	for _, item := range p.Items {
		if item.CoverageStatus != api.ItemCoverageStatusApproved {
			continue
		}
		if item.PolicyDependentID.Valid {
			addToTotals(item.PolicyDependentID.UUID, item.CoverageAmount, totals)
		}
		addToTotals(p.ID, item.CoverageAmount, totals)
	}

	return totals
}

// LoadClaims - a simple wrapper method for loading claims on the struct
func (p *Policy) LoadClaims(tx *pop.Connection, reload bool) {
	if len(p.Claims) == 0 || reload {
		if err := tx.Load(p, "Claims"); err != nil {
			panic("database error loading Policy.Claims, " + err.Error())
		}
	}
}

// LoadDependents - a simple wrapper method for loading dependents on the struct
func (p *Policy) LoadDependents(tx *pop.Connection, reload bool) {
	if p == nil {
		panic("policy is nil in Policy.LoadDependents")
	}
	if len(p.Dependents) == 0 || reload {
		if err := tx.Load(p, "Dependents"); err != nil {
			panic("database error loading Policy.Dependents, " + err.Error())
		}
	}
}

// LoadItems - a simple wrapper method for loading items on the struct
func (p *Policy) LoadItems(tx *pop.Connection, reload bool) {
	if len(p.Items) == 0 || reload {
		if err := tx.Load(p, "Items"); err != nil {
			panic("database error loading Policy.Items, " + err.Error())
		}
	}
}

// LoadMembers - a simple wrapper method for loading members on the struct
func (p *Policy) LoadMembers(tx *pop.Connection, reload bool) {
	if len(p.Members) == 0 || reload {
		if err := tx.Load(p, "Members"); err != nil {
			panic("database error loading Policy.Members, " + err.Error())
		}
	}
}

func (p *Policy) ConvertToAPI(tx *pop.Connection) api.Policy {
	p.LoadClaims(tx, true)
	p.LoadDependents(tx, true)
	p.LoadMembers(tx, true)

	claims := p.Claims.ConvertToAPI(tx)
	dependents := p.Dependents.ConvertToAPI()
	members := p.Members.ConvertToPolicyMembers()

	return api.Policy{
		ID:          p.ID,
		Type:        p.Type,
		HouseholdID: p.HouseholdID.String,
		CostCenter:  p.CostCenter,
		Account:     p.Account,
		EntityCode:  p.EntityCode,
		CreatedAt:   p.CreatedAt,
		UpdatedAt:   p.UpdatedAt,
		Claims:      claims,
		Dependents:  dependents,
		Members:     members,
	}
}

func (p *Policies) ConvertToAPI(tx *pop.Connection) api.Policies {
	policies := make(api.Policies, len(*p))
	for i, pp := range *p {
		policies[i] = pp.ConvertToAPI(tx)
	}

	return policies
}

func (p *Policy) AddDependent(tx *pop.Connection, input api.PolicyDependentInput) error {
	if p == nil {
		return errors.New("policy is nil in AddDependent")
	}

	dependent := PolicyDependent{
		PolicyID:       p.ID,
		Name:           input.Name,
		Relationship:   input.Relationship,
		Location:       input.Location,
		ChildBirthYear: input.ChildBirthYear,
	}

	if err := dependent.Create(tx); err != nil {
		return err
	}

	return nil
}

func (p *Policy) AddClaim(tx *pop.Connection, input api.ClaimCreateInput) (Claim, error) {
	if p == nil {
		return Claim{}, errors.New("policy is nil in AddClaim")
	}

	claim := ConvertClaimCreateInput(input)
	claim.PolicyID = p.ID

	if err := claim.Create(tx); err != nil {
		return Claim{}, err
	}

	return claim, nil
}
