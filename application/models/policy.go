package models

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gobuffalo/nulls"
	"github.com/gobuffalo/pop/v5"
	"github.com/gobuffalo/validate/v3"
	"github.com/gofrs/uuid"
	"github.com/pkg/errors"

	"github.com/silinternational/cover-api/api"
	"github.com/silinternational/cover-api/domain"
)

type Policies []Policy

var ValidPolicyTypes = map[api.PolicyType]struct{}{
	api.PolicyTypeHousehold: {},
	api.PolicyTypeCorporate: {},
}

type Policy struct {
	ID            uuid.UUID      `db:"id"`
	Type          api.PolicyType `db:"type" validate:"policyType"`
	HouseholdID   nulls.String   `db:"household_id"` // validation is checked at the struct level
	CostCenter    string         `db:"cost_center" validate:"required_if=Type Corporate"`
	AccountDetail string         `db:"account_detail"`
	Account       string         `db:"account" validate:"required_if=Type Corporate"`
	EntityCodeID  nulls.UUID     `db:"entity_code_id"` // validation is checked at the struct level
	Notes         string         `db:"notes"`
	LegacyID      nulls.Int      `db:"legacy_id"`
	Email         string         `db:"email"`
	IdentCode     string         `db:"ident_code"`
	CreatedAt     time.Time      `db:"created_at"`
	UpdatedAt     time.Time      `db:"updated_at"`

	Claims     Claims           `has_many:"claims" validate:"-"`
	Dependents PolicyDependents `has_many:"policy_dependents" validate:"-"`
	Items      Items            `has_many:"items" validate:"-"`
	Members    Users            `many_to_many:"policy_users" validate:"-"`
	EntityCode EntityCode       `belongs_to:"entity_codes" validate:"-"`
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
func (p *Policy) Update(ctx context.Context) error {
	tx := Tx(ctx)
	var oldPolicy Policy
	if err := oldPolicy.FindByID(tx, p.ID); err != nil {
		return appErrorFromDB(err, api.ErrorQueryFailure)
	}

	updates := p.Compare(oldPolicy)
	for i := range updates {
		history := p.NewHistory(ctx, api.HistoryActionUpdate, updates[i])
		if err := history.Create(tx); err != nil {
			return err
		}
	}

	return update(tx, p)
}

// CreateCorporateType creates a new Corporate type policy for the user.
//   The EntityCodeID, CostCenter and Account must have non-blank values
func (p *Policy) CreateCorporateType(tx *pop.Connection, actor User) error {
	p.Type = api.PolicyTypeCorporate
	p.Email = actor.EmailOfChoice()

	if err := p.Create(tx); err != nil {
		return err
	}

	polUser := PolicyUser{
		PolicyID: p.ID,
		UserID:   actor.ID,
	}

	return polUser.Create(tx)
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

	if perm == PermissionCreate && sub == "" {
		return true
	}

	return p.isMember(tx, actor.ID)
}

func (p *Policy) isMember(tx *pop.Connection, id uuid.UUID) bool {
	p.LoadMembers(tx, false)
	for _, m := range p.Members {
		if m.ID == id {
			return true
		}
	}
	return false
}

func (p *Policy) MemberHasEmail(tx *pop.Connection, emailAddress string) bool {
	p.LoadMembers(tx, false)
	for _, m := range p.Members {
		if m.Email == emailAddress {
			return true
		}
	}
	return false
}

func (p *Policy) isDependent(tx *pop.Connection, id uuid.UUID) bool {
	p.LoadDependents(tx, false)
	for _, d := range p.Dependents {
		if d.ID == id {
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

// LoadEntityCode - a simple wrapper method for loading the entity code on the struct
func (p *Policy) LoadEntityCode(tx *pop.Connection, reload bool) {
	if p.EntityCode.ID == uuid.Nil || reload {
		if err := tx.Load(p, "EntityCode"); err != nil {
			panic("database error loading Policy.EntityCode, " + err.Error())
		}
	}
}

func (p *Policy) ConvertToAPI(tx *pop.Connection, hydrate bool) api.Policy {
	p.LoadEntityCode(tx, true)
	p.LoadMembers(tx, true)

	apiPolicy := api.Policy{
		ID:            p.ID,
		Type:          p.Type,
		HouseholdID:   p.HouseholdID.String,
		CostCenter:    p.CostCenter,
		Account:       p.Account,
		AccountDetail: p.AccountDetail,
		EntityCode:    p.EntityCode.ConvertToAPI(tx),
		Members:       p.Members.ConvertToPolicyMembers(),
		CreatedAt:     p.CreatedAt,
		UpdatedAt:     p.UpdatedAt,
	}

	if hydrate {
		p.LoadClaims(tx, true)
		p.LoadDependents(tx, true)
		apiPolicy.Claims = p.Claims.ConvertToAPI(tx)
		apiPolicy.Dependents = p.Dependents.ConvertToAPI()
	}

	return apiPolicy
}

func (p *Policies) ConvertToAPI(tx *pop.Connection) api.Policies {
	policies := make(api.Policies, len(*p))
	for i, pp := range *p {
		policies[i] = pp.ConvertToAPI(tx, false)
	}

	return policies
}

func (p *Policies) All(tx *pop.Connection) error {
	return appErrorFromDB(tx.All(p), api.ErrorQueryFailure)
}

func (p *Policies) Query(tx *pop.Connection, query api.Query) error {
	q := tx.Order("updated_at DESC")

	if query.Limit > 0 {
		q = q.Limit(query.Limit)
	}

	for k, v := range query.Search {
		switch k {
		case "name":
			q = p.SearchByname(q, v)
		}
	}

	return appErrorFromDB(q.All(p), api.ErrorQueryFailure)
}

func (p *Policies) SearchByname(q *pop.Query, name string) *pop.Query {
	return q.Join("policy_users", "policies.id = policy_users.policy_id").
		Join("users", "users.id = policy_users.user_id").
		Where(`users.first_name ILIKE ? OR users.last_name ILIKE ?`, name, name)
}

func (p *Policy) AddDependent(tx *pop.Connection, input api.PolicyDependentInput) error {
	if p == nil {
		return errors.New("policy is nil in AddDependent")
	}

	dependent := PolicyDependent{
		PolicyID:       p.ID,
		Name:           input.Name,
		Relationship:   input.Relationship,
		Country:        input.Country,
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

// Compare returns a list of fields that are different between two objects
func (p *Policy) Compare(old Policy) []FieldUpdate {
	var updates []FieldUpdate

	if p.EntityCodeID != old.EntityCodeID {
		updates = append(updates, FieldUpdate{
			OldValue:  old.EntityCodeID.UUID.String(),
			NewValue:  p.EntityCodeID.UUID.String(),
			FieldName: "EntityCodeID",
		})
	}

	if p.Type != old.Type {
		updates = append(updates, FieldUpdate{
			OldValue:  string(old.Type),
			NewValue:  string(p.Type),
			FieldName: "Type",
		})
	}

	if p.CostCenter != old.CostCenter {
		updates = append(updates, FieldUpdate{
			OldValue:  old.CostCenter,
			NewValue:  p.CostCenter,
			FieldName: "CostCenter",
		})
	}

	if p.Account != old.Account {
		updates = append(updates, FieldUpdate{
			OldValue:  old.Account,
			NewValue:  p.Account,
			FieldName: "Account",
		})
	}

	if p.HouseholdID != old.HouseholdID {
		updates = append(updates, FieldUpdate{
			OldValue:  old.HouseholdID.String,
			NewValue:  p.HouseholdID.String,
			FieldName: "HouseholdID",
		})
	}

	if p.Notes != old.Notes {
		updates = append(updates, FieldUpdate{
			OldValue:  old.Notes,
			NewValue:  p.Notes,
			FieldName: "Notes",
		})
	}

	return updates
}

func (p *Policy) NewHistory(ctx context.Context, action string, fieldUpdate FieldUpdate) PolicyHistory {
	return PolicyHistory{
		Action:    action,
		PolicyID:  p.ID,
		UserID:    CurrentUser(ctx).ID,
		FieldName: fieldUpdate.FieldName,
		OldValue:  fmt.Sprintf("%s", fieldUpdate.OldValue),
		NewValue:  fmt.Sprintf("%s", fieldUpdate.NewValue),
	}
}

func (p *Policy) calculateAnnualPremium(tx *pop.Connection) api.Currency {
	p.LoadItems(tx, false)
	var premium api.Currency
	for _, item := range p.Items {
		premium += item.CalculateAnnualPremium()
	}
	if int(premium) < domain.Env.PremiumMinimum {
		return api.Currency(domain.Env.PremiumMinimum)
	}
	return premium
}
