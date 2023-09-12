package models

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gobuffalo/pop/v6"
	"github.com/gobuffalo/validate/v3"
	"github.com/gofrs/uuid"

	"github.com/silinternational/cover-api/api"
	"github.com/silinternational/cover-api/log"
)

var ValidPolicyDependentRelationships = map[api.PolicyDependentRelationship]struct{}{
	api.PolicyDependentRelationshipNone:   {},
	api.PolicyDependentRelationshipSpouse: {},
	api.PolicyDependentRelationshipChild:  {},
}

type PolicyDependents []PolicyDependent

type PolicyDependent struct {
	ID             uuid.UUID                       `db:"id"`
	PolicyID       uuid.UUID                       `db:"policy_id"`
	Name           string                          `db:"name" validate:"required"`
	Relationship   api.PolicyDependentRelationship `db:"relationship" validate:"policyDependentRelationship"`
	City           string                          `db:"city"`
	State          string                          `db:"state"`
	Country        string                          `db:"country" validate:"required"`
	CountryCode    string                          `db:"country_code" validate:"required"`
	ChildBirthYear int                             `db:"child_birth_year" validate:"policyDependentChildBirthYear,required_if=Relationship Child"`

	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

// Validate gets run every time you call a "pop.Validate*" (pop.ValidateAndSave, pop.ValidateAndCreate, pop.ValidateAndUpdate) method.
func (p *PolicyDependent) Validate(tx *pop.Connection) (*validate.Errors, error) {
	return validateModel(p), nil
}

func (p *PolicyDependent) GetID() uuid.UUID {
	return p.ID
}

func (p *PolicyDependent) FindByID(tx *pop.Connection, id uuid.UUID) error {
	return find(tx, p, id)
}

func (p *PolicyDependent) Create(tx *pop.Connection) error {
	var country Country
	if err := country.FindByName(tx, p.Country); err != nil {
		err := fmt.Errorf("invalid country name %s", p.Country)
		return api.NewAppError(err, api.ErrorInvalidDependentCountryName, api.CategoryUser)
	}
	p.CountryCode = country.ID

	return create(tx, p)
}

func (p *PolicyDependent) Update(tx *pop.Connection) error {
	var policy Policy
	if err := policy.FindByID(tx, p.PolicyID); err != nil {
		panic("error finding dependent's policy: " + err.Error())
	}

	var country Country
	if err := country.FindByName(tx, p.Country); err != nil {
		err := fmt.Errorf("invalid country name %s", p.Country)
		return api.NewAppError(err, api.ErrorInvalidDependentCountryName, api.CategoryUser)
	}
	p.CountryCode = country.ID

	p.FixTeamRelationship(policy)

	return update(tx, p)
}

func (p *PolicyDependent) Destroy(tx *pop.Connection) error {
	return destroy(tx, p)
}

// RelatedItemNames returns a slice of the names of Items that are related to this dependent
func (p *PolicyDependent) RelatedItemNames(tx *pop.Connection) []string {
	names := []string{}
	for _, item := range p.RelatedItems(tx) {
		names = append(names, item.Name)
	}
	return names
}

// RelatedItems returns a slice of the Items that are related to this dependent
func (p *PolicyDependent) RelatedItems(tx *pop.Connection) Items {
	var items Items
	if err := tx.Where("policy_dependent_id = ?", p.ID).All(&items); err != nil {
		panic(fmt.Sprintf("error fetching items with policy_dependent_id %s, %s", p.ID, err))
	}

	return items
}

// IsActorAllowedTo ensure the actor is either an admin, or a member of this policy to perform any permission
func (p *PolicyDependent) IsActorAllowedTo(tx *pop.Connection, actor User, perm Permission, sub SubResource, r *http.Request) bool {
	if actor.IsAdmin() {
		return true
	}

	var policy Policy
	if err := policy.FindByID(tx, p.PolicyID); err != nil {
		log.Error("failed to load policy for dependent:", err)
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

func (p *PolicyDependent) ConvertToAPI(tx *pop.Connection) api.PolicyDependent {
	p.OverrideCountryName(tx)
	return api.PolicyDependent{
		ID:             p.ID,
		Name:           p.Name,
		Relationship:   p.Relationship,
		Country:        p.GetLocation().Country,
		CountryCode:    p.GetLocation().CountryCode,
		ChildBirthYear: p.ChildBirthYear,
	}
}

func (p *PolicyDependents) ConvertToAPI(tx *pop.Connection) api.PolicyDependents {
	deps := make(api.PolicyDependents, len(*p))
	for i, pp := range *p {
		deps[i] = pp.ConvertToAPI(tx)
	}
	return deps
}

func (p *PolicyDependent) GetLocation() Location {
	return Location{
		City:        p.City,
		State:       p.State,
		Country:     p.Country,
		CountryCode: p.CountryCode,
	}
}

func (p *PolicyDependent) GetName() Name {
	return Name{
		First: p.Name,
	}
}

func (p *PolicyDependent) FixTeamRelationship(policy Policy) {
	if policy.Type == api.PolicyTypeTeam {
		p.Relationship = api.PolicyDependentRelationshipNone
		p.ChildBirthYear = 0
	}
}

func (p *PolicyDependent) OverrideCountryName(tx *pop.Connection) {
	if p.CountryCode == "" {
		return
	}

	var country Country
	if err := country.FindByCode(tx, p.CountryCode); err != nil {
		log.Errorf("found invalid country code %q on dependent %s", p.CountryCode, p.ID)
	} else {
		p.Country = country.Name
	}
}
