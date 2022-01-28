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
	api.PolicyTypeTeam:      {},
}

type Policy struct {
	ID            uuid.UUID      `db:"id"`
	Name          string         `db:"name" validate:"required"`
	Type          api.PolicyType `db:"type" validate:"policyType"`
	HouseholdID   nulls.String   `db:"household_id"` // validation is checked at the struct level
	CostCenter    string         `db:"cost_center"`  // validation is checked at the struct level
	AccountDetail string         `db:"account_detail"`
	Account       string         `db:"account"`        // validation is checked at the struct level
	EntityCodeID  uuid.UUID      `db:"entity_code_id"` // validation is checked at the struct level
	Notes         string         `db:"notes"`
	LegacyID      nulls.Int      `db:"legacy_id"`
	Email         string         `db:"email"`
	CreatedAt     time.Time      `db:"created_at"`
	UpdatedAt     time.Time      `db:"updated_at"`

	Claims     Claims           `has_many:"claims" validate:"-" order_by:"incident_date desc"`
	Dependents PolicyDependents `has_many:"policy_dependents" validate:"-" order_by:"name"`
	Items      Items            `has_many:"items" validate:"-" order_by:"coverage_status asc,updated_at desc"`
	Members    Users            `many_to_many:"policy_users" validate:"-"`
	EntityCode EntityCode       `belongs_to:"entity_codes" validate:"-"`
}

// Validate gets run every time you call a "pop.Validate*" (pop.ValidateAndSave, pop.ValidateAndCreate, pop.ValidateAndUpdate) method.
func (p *Policy) Validate(tx *pop.Connection) (*validate.Errors, error) {
	return validateModel(p), nil
}

// CreateWithContext stores the Policy data as a new record in the database.
func (p *Policy) CreateWithContext(ctx context.Context) error {
	tx := Tx(ctx)

	if err := p.Create(tx); err != nil {
		return err
	}

	history := p.NewHistory(ctx, api.HistoryActionCreate, FieldUpdate{})
	if err := history.Create(tx); err != nil {
		return err
	}
	return nil
}

// Create a Policy but not a history record. Use CreateWithContext if history is needed.
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

// CreateTeam creates a new Team type policy for the user.
//   The EntityCodeID, CostCenter and Account must have non-blank values
func (p *Policy) CreateTeam(ctx context.Context) error {
	tx := Tx(ctx)
	actor := CurrentUser(ctx)

	p.Type = api.PolicyTypeTeam
	p.Email = actor.EmailOfChoice()

	if err := p.CreateWithContext(ctx); err != nil {
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
		Name:          p.Name,
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

func (p *Policies) AllActive(tx *pop.Connection) error {
	return appErrorFromDB(tx.Q().Scope(scopeFilterPoliciesByActive("true")).All(p), api.ErrorQueryFailure)
}

func (p *Policies) Query(tx *pop.Connection, params api.QueryParams) (*pop.Paginator, error) {
	q := tx.Order("updated_at DESC")

	q.Paginate(params.Page(), params.Limit())

	if v := params.Search(); v != "" {
		q.Scope(scopeSearchPolicies(v))
	}

	if v := params.Filter("active"); v != "" {
		q.Scope(scopeFilterPoliciesByActive(v))
	}

	return q.Paginator, appErrorFromDB(q.All(p), api.ErrorQueryFailure)
}

func scopeSearchPolicies(searchText string) pop.ScopeFunc {
	searchText = "%" + searchText + "%"

	// Include policies that have a related user whose
	// CONCAT(users.first_name, ' ', users.last_name) contains the search string
	//  --or--
	// whose own cost_center, household_id or name contain the search string
	return func(q *pop.Query) *pop.Query {
		return q.Where("policies.id IN ("+
			"SELECT policies.id FROM policies "+
			"    JOIN policy_users pu ON policies.id = pu.policy_id "+
			"    JOIN users on users.id = pu.user_id "+
			"    AND ("+
			"        CONCAT(users.first_name, ' ', users.last_name) ILIKE ? "+
			"        OR policies.cost_center ILIKE ? OR policies.household_id ILIKE ? "+
			"        OR policies.name ILIKE ?"+
			"    ) "+
			")", searchText, searchText, searchText, searchText)
	}
}

func scopeFilterPoliciesByActive(active string) pop.ScopeFunc {
	return func(q *pop.Query) *pop.Query {
		if active == "true" {
			return q.Where("policies.id IN (SELECT policies.id FROM policies,items " +
				"WHERE policies.id=items.policy_id AND items.coverage_status='Approved')")
		}
		if active == "false" {
			return q.Where("policies.id NOT IN (SELECT policies.id FROM policies,items " +
				"WHERE policies.id=items.policy_id AND items.coverage_status='Approved')")
		}
		return q
	}
}

func (p *Policy) AddDependent(tx *pop.Connection, input api.PolicyDependentInput) (PolicyDependent, error) {
	if p == nil {
		return PolicyDependent{}, errors.New("policy is nil in AddDependent")
	}

	dependent := PolicyDependent{
		PolicyID:       p.ID,
		Name:           input.Name,
		Relationship:   input.Relationship,
		Country:        input.Country,
		ChildBirthYear: input.ChildBirthYear,
	}

	dependent.FixTeamRelationship(*p)

	if err := dependent.Create(tx); err != nil {
		return dependent, err
	}

	return dependent, nil
}

func (p *Policy) AddClaim(ctx context.Context, input api.ClaimCreateInput) (Claim, error) {
	if p == nil {
		return Claim{}, errors.New("policy is nil in AddClaim")
	}

	claim := ConvertClaimCreateInput(input)
	claim.PolicyID = p.ID

	if err := claim.CreateWithContext(ctx); err != nil {
		return Claim{}, err
	}

	return claim, nil
}

// Compare returns a list of fields that are different between two objects
func (p *Policy) Compare(old Policy) []FieldUpdate {
	var updates []FieldUpdate

	if p.Name != old.Name {
		updates = append(updates, FieldUpdate{
			OldValue:  old.Name,
			NewValue:  p.Name,
			FieldName: "Name",
		})
	}

	if p.EntityCodeID != old.EntityCodeID {
		updates = append(updates, FieldUpdate{
			OldValue:  old.EntityCodeID.String(),
			NewValue:  p.EntityCodeID.String(),
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

func (p *Policy) NewHouseholdInvite(tx *pop.Connection, invite api.PolicyUserInviteCreate, cUser User) error {
	var user User
	if err := user.FindByEmail(tx, invite.Email); domain.IsOtherThanNoRows(err) {
		return err
	}

	// if user doesn't yet exist, create an invite
	if user.ID == uuid.Nil {
		return p.createInvite(tx, invite, cUser)
	}

	// if user already exists, make sure they don't already have a household policy
	user.LoadPolicies(tx, false)
	for _, p := range user.Policies {
		if p.HouseholdID.Valid {
			err := errors.New("Invited User already has a household policy")
			return api.NewAppError(err, api.ErrorPolicyInviteAlreadyHasHousehold, api.CategoryUser)
		}
	}

	pUser := PolicyUser{
		PolicyID: p.ID,
		UserID:   user.ID,
	}

	if err := pUser.Create(tx); err != nil {
		return appErrorFromDB(err, api.ErrorCreateFailure)
	}
	return nil
}

func (p *Policy) NewTeamInvite(tx *pop.Connection, invite api.PolicyUserInviteCreate, cUser User) error {
	var user User
	if err := user.FindByEmail(tx, invite.Email); domain.IsOtherThanNoRows(err) {
		return err
	}

	// if user doesn't yet exist, create an invite
	if user.ID == uuid.Nil {
		return p.createInvite(tx, invite, cUser)
	}

	// if user already exists, just associate them with the Policy
	pUser := PolicyUser{
		PolicyID: p.ID,
		UserID:   user.ID,
	}

	if err := pUser.Create(tx); err != nil {
		return appErrorFromDB(err, api.ErrorCreateFailure)
	}
	return nil
}

func (p *Policy) createInvite(tx *pop.Connection, invite api.PolicyUserInviteCreate, cUser User) error {
	// create invite
	puInvite := PolicyUserInvite{
		PolicyID:       p.ID,
		Email:          invite.Email,
		InviteeName:    invite.Name,
		InviterName:    cUser.Name(),
		InviterEmail:   cUser.Email,
		InviterMessage: invite.InviterMessage,
	}

	if err := puInvite.Create(tx); err != nil {
		return appErrorFromDB(err, api.ErrorCreateFailure)
	}
	return nil
}

// ProcessAnnualCoverage creates coverage renewal ledger entries for all items covered for the given year.
// Does not create new records for items already processed.
func (p *Policies) ProcessAnnualCoverage(tx *pop.Connection, year int) error {
	for _, pp := range *p {
		if err := pp.ProcessAnnualCoverage(tx, year); err != nil {
			return fmt.Errorf("error processing annual coverage for policy %s: %w", pp.ID, err)
		}
	}
	return nil
}

func (p *Policy) ProcessAnnualCoverage(tx *pop.Connection, year int) error {
	var items Items
	if err := tx.Where("coverage_status = ?", api.ItemCoverageStatusApproved).
		Where("paid_through_year < ?", year).
		Where("policy_id  = ?", p.ID).
		All(&items); err != nil {
		return api.NewAppError(err, api.ErrorQueryFailure, api.CategoryInternal)
	}

	totalAnnualPremium := map[uuid.UUID]api.Currency{}
	for i := range items {
		items[i].PaidThroughYear = year
		if err := tx.UpdateColumns(&items[i], "paid_through_year", "updated_at"); err != nil {
			return fmt.Errorf("failed to update paid_through_year for item %s: %w", items[i].ID, err)
		}
		totalAnnualPremium[items[i].RiskCategoryID] += items[i].CalculateAnnualPremium()
	}

	for id, amount := range totalAnnualPremium {
		err := p.CreateRenewalLedgerEntry(tx, id, amount)
		if err != nil {
			return err
		}
	}
	return nil
}

func (p *Policy) CreateRenewalLedgerEntry(tx *pop.Connection, riskCategoryID uuid.UUID, amount api.Currency) error {
	p.LoadEntityCode(tx, false)

	var rc RiskCategory
	if err := rc.FindByID(tx, riskCategoryID); err != nil {
		return fmt.Errorf("failed to find risk category %s: %w", riskCategoryID, err)
	}

	le := NewLedgerEntry(*p, nil, nil)
	le.Type = LedgerEntryTypeCoverageRenewal
	le.Amount = amount
	le.Name = p.Name
	le.EntityCode = p.EntityCode.Code
	le.RiskCategoryName = rc.Name
	le.RiskCategoryCC = rc.CostCenter

	if err := le.Create(tx); err != nil {
		return fmt.Errorf("failed to create ledger entry for policy %s: %w", p.ID, err)
	}

	return nil
}
