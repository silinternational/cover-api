package models

import (
	"bufio"
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gobuffalo/nulls"
	"github.com/gobuffalo/pop/v6"
	"github.com/gobuffalo/validate/v3"
	"github.com/gofrs/uuid"
	"github.com/pkg/errors"

	"github.com/silinternational/cover-api/api"
	"github.com/silinternational/cover-api/domain"
	"github.com/silinternational/cover-api/log"
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

	Claims     Claims            `has_many:"claims" validate:"-" order_by:"incident_date desc"`
	Dependents PolicyDependents  `has_many:"policy_dependents" validate:"-" order_by:"name"`
	Invites    PolicyUserInvites `has_many:"policy_user_invites" validate:"-" order_by:"invitee_name"`
	Items      Items             `has_many:"items" validate:"-" order_by:"coverage_status asc,updated_at desc"`
	Members    Users             `many_to_many:"policy_users" validate:"-"`
	EntityCode EntityCode        `belongs_to:"entity_codes" validate:"-"`
}

// Validate gets run every time you call a "pop.Validate*" (pop.ValidateAndSave, pop.ValidateAndCreate, pop.ValidateAndUpdate) method.
func (p *Policy) Validate(tx *pop.Connection) (*validate.Errors, error) {
	return validateModel(p), nil
}

// CreateWithHistory stores the Policy data as a new record in the database. Also creates a PolicyHistory record.
func (p *Policy) CreateWithHistory(ctx context.Context) error {
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

// Create a Policy but not a history record. Use CreateWithHistory if history is needed.
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
// The EntityCodeID, CostCenter and Account must have non-blank values
func (p *Policy) CreateTeam(ctx context.Context) error {
	tx := Tx(ctx)
	actor := CurrentUser(ctx)

	p.Type = api.PolicyTypeTeam
	p.Email = actor.EmailOfChoice()

	if err := p.CreateWithHistory(ctx); err != nil {
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

func (p *Policy) FindByHouseholdID(tx *pop.Connection, householdID string) error {
	err := tx.Where("household_id = ?", householdID).First(p)
	return appErrorFromDB(err, api.ErrorQueryFailure)
}

func (p *Policy) FindByTeamDetails(tx *pop.Connection, entityCode, account, costCenter, accountDetail string) error {
	err := tx.
		Where("entity_code_id = ?", EntityCodeID(entityCode)).
		Where("account = ?", account).
		Where("cost_center = ?", costCenter).
		Where("account_detail = ?", accountDetail).
		First(p)
	return appErrorFromDB(err, api.ErrorQueryFailure)
}

// IsActorAllowedTo ensure the actor is either an admin, or a member of this policy to perform any permission
func (p *Policy) IsActorAllowedTo(tx *pop.Connection, actor User, perm Permission, sub SubResource, r *http.Request) bool {
	if actor.IsAdmin() {
		return true
	}

	if sub == api.ResourceStrikes {
		return false
	}

	switch perm {
	case PermissionList:
		return true
	case PermissionCreate:
		if sub == "" {
			return true
		}
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
// the policy ID with the total of all the items' coverage amounts as well as
// an entry for each dependant with the total of each of their items' coverage amounts
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

// LoadInvites - a simple wrapper method for loading policy user invites on the struct
func (p *Policy) LoadInvites(tx *pop.Connection, reload bool) {
	if len(p.Invites) == 0 || reload {
		if err := tx.Load(p, "Invites"); err != nil {
			panic("database error loading Policy.Invites, " + err.Error())
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

// GetPolicyUserIDs loads the members of the Policy and also returns a list of the corresponding
// PolicyUser IDs
func (p *Policy) GetPolicyUserIDs(tx *pop.Connection, reload bool) []uuid.UUID {
	p.LoadMembers(tx, reload)
	puIDS := make([]uuid.UUID, len(p.Members))
	for i, m := range p.Members {
		var polUser PolicyUser
		if err := tx.Where("policy_id = ? AND user_id = ?", p.ID, m.ID).First(&polUser); err != nil {
			panic("database error finding policy user for policy, " + err.Error())
		}
		puIDS[i] = polUser.ID
	}
	return puIDS
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
	polUserIDs := p.GetPolicyUserIDs(tx, true)

	apiPolicy := api.Policy{
		ID:            p.ID,
		Name:          p.Name,
		Type:          p.Type,
		HouseholdID:   p.HouseholdID.String,
		CostCenter:    p.CostCenter,
		Account:       p.Account,
		AccountDetail: p.AccountDetail,
		EntityCode:    p.EntityCode.ConvertToAPI(tx, false),
		Members:       p.Members.ConvertToPolicyMembers(polUserIDs),
		CreatedAt:     p.CreatedAt,
		UpdatedAt:     p.UpdatedAt,
	}

	if hydrate {
		p.hydrateApiPolicy(tx, &apiPolicy)
	}

	return apiPolicy
}

func (p *Policy) hydrateApiPolicy(tx *pop.Connection, apiPolicy *api.Policy) {
	p.LoadClaims(tx, true)
	p.LoadDependents(tx, true)
	p.LoadInvites(tx, true)
	apiPolicy.Claims = p.Claims.ConvertToAPI(tx)
	apiPolicy.Dependents = p.Dependents.ConvertToAPI()
	apiPolicy.Invites = p.Invites.ConvertToAPI()

	var reports LedgerReports
	if err := reports.AllForPolicy(tx, p.ID); domain.IsOtherThanNoRows(err) {
		log.Errorf("error retrieving ledger reports for policy %s: %s", p.ID.String(), err)
		return
	}
	if len(reports) > 0 {
		apiPolicy.LedgerReports = reports.ConvertToAPI(tx)
	}

	var strikes Strikes
	if err := strikes.RecentForPolicy(tx, p.ID, time.Now().UTC()); domain.IsOtherThanNoRows(err) {
		log.Errorf("error retrieving recent strikes for policy %s: %s", p.ID.String(), err)
		return
	}

	apiPolicy.Strikes = strikes.ConvertToAPI(tx)
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

	// If a matching record exists, use that one and don't create a duplicate
	if err := dependent.FindByName(tx, p.ID, input.Name); err != nil {
		if domain.IsOtherThanNoRows(err) {
			return PolicyDependent{}, err
		}
	} else {
		if (dependent.Relationship == input.Relationship ||
			dependent.Relationship == api.PolicyDependentRelationshipNone && input.Relationship == "") &&
			dependent.Country == input.Country &&
			dependent.ChildBirthYear == input.ChildBirthYear {
			return dependent, nil
		}
		err = errors.New("cannot create a new PolicyDependent with same Name as existing record")
		return PolicyDependent{}, api.NewAppError(err, api.ErrorPolicyDependentDuplicateName, api.CategoryUser)
	}

	if err := dependent.Create(tx); err != nil {
		return PolicyDependent{}, err
	}

	return dependent, nil
}

func (p *Policy) AddClaim(ctx context.Context, input api.ClaimCreateInput) (Claim, error) {
	if p == nil {
		return Claim{}, errors.New("policy is nil in AddClaim")
	}

	claim := ConvertClaimCreateInput(input)
	claim.PolicyID = p.ID

	if err := claim.CreateWithHistory(ctx); err != nil {
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
		premium += item.CalculateAnnualPremium(tx)
	}
	return api.Currency(domain.Max(int(premium), domain.Env.PremiumMinimum))
}

func (p *Policy) currentCoverageTotal(tx *pop.Connection) api.Currency {
	p.LoadItems(tx, false)
	var coverage int
	for _, item := range p.Items {
		if item.CoverageStatus == api.ItemCoverageStatusApproved {
			coverage += item.CoverageAmount
		}
	}

	return api.Currency(coverage)
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

// ProcessRenewals creates coverage renewal ledger entries for all items covered for the given period.
// Does not create new records for items already processed.
func (p *Policies) ProcessRenewals(tx *pop.Connection, date time.Time, billingPeriod int) error {
	for _, pp := range *p {
		if err := pp.ProcessRenewals(tx, date, billingPeriod); err != nil {
			return fmt.Errorf("error processing annual coverage for policy %s: %w", pp.ID, err)
		}
	}
	return nil
}

// ProcessRenewals creates coverage renewal ledger entries for all items covered for the given period.
// Does not create new records for items already processed.
func (p *Policy) ProcessRenewals(tx *pop.Connection, date time.Time, billingPeriod int) error {
	var items Items
	if err := tx.Where("coverage_status = ?", api.ItemCoverageStatusApproved).
		Where("paid_through_date < ?", date).
		Where("policy_id  = ?", p.ID).
		Join("item_categories ic", "items.category_id = ic.id").
		Where("ic.billing_period = ?", billingPeriod).
		All(&items); err != nil {
		return appErrorFromDB(err, api.ErrorQueryFailure)
	}

	paidThroughDate := date
	switch billingPeriod {
	case domain.BillingPeriodAnnual:
		paidThroughDate = domain.EndOfYear(date.Year())
	case domain.BillingPeriodMonthly:
		paidThroughDate = domain.EndOfMonth(date)
	}

	totalPremiumGroupedByRiskCategory := map[uuid.UUID]api.Currency{}
	for i := range items {
		if err := items[i].SetPaidThroughDate(tx, paidThroughDate); err != nil {
			return err
		}
		totalPremiumGroupedByRiskCategory[items[i].RiskCategoryID] += items[i].CalculateAnnualPremium(tx)
	}

	for riskCategoryID, amount := range totalPremiumGroupedByRiskCategory {
		err := p.CreateRenewalLedgerEntry(tx, riskCategoryID, amount)
		if err != nil {
			return api.NewAppError(err, api.ErrorCreateRenewalEntry, api.CategoryInternal)
		}
	}
	return nil
}

// CreateRenewalLedgerEntry creates a new ledger entry for coverage renewal
func (p *Policy) CreateRenewalLedgerEntry(tx *pop.Connection, riskCategoryID uuid.UUID, amount api.Currency) error {
	p.LoadEntityCode(tx, false)

	var rc RiskCategory
	if err := rc.FindByID(tx, riskCategoryID); err != nil {
		return fmt.Errorf("failed to find risk category %s: %w", riskCategoryID, err)
	}

	now := time.Now().UTC()

	le := NewLedgerEntry("", *p, nil, nil, now)
	le.Type = LedgerEntryTypeCoverageRenewal
	le.Amount = -amount
	le.EntityCode = p.EntityCode.Code
	le.RiskCategoryName = rc.Name
	le.RiskCategoryCC = rc.CostCenter

	if err := le.Create(tx); err != nil {
		return fmt.Errorf("failed to create ledger entry for policy %s: %w", p.ID, err)
	}

	return nil
}

func ImportPolicies(tx *pop.Connection, file io.Reader) (api.PoliciesImportResponse, error) {
	var response api.PoliciesImportResponse

	r := csv.NewReader(bufio.NewReader(file))
	header, err := r.Read()
	if err == io.EOF {
		err := fmt.Errorf("empty policy CSV file: %w", err)
		return response, api.NewAppError(err, api.ErrorUnknown, api.CategoryUser)
	}

	var vehicleCategory ItemCategory
	if err := tx.Where("risk_category_id = ?", riskCategoryVehicleID).First(&vehicleCategory); err != nil {
		err := fmt.Errorf("failed to find an item category for vehicles: %w", err)
		return response, api.NewAppError(err, api.ErrorUnknown, api.CategoryInternal)
	}

	for n := 0; ; n++ {
		csvLine, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			err := fmt.Errorf("failed to read from policy CSV file on row %d: %w", n+2, err)
			return response, api.NewAppError(err, api.ErrorUnknown, api.CategoryUser)
		}

		data := map[string]string{}
		for i, value := range csvLine {
			data[header[i]] = strings.TrimSpace(value)
		}
		policiesCreated, itemsCreated, err := importPolicy(tx, data, vehicleCategory.ID, time.Now().UTC())
		if err != nil {
			err := fmt.Errorf("error importing policy on row %d: %w", n+2, err)
			return response, api.NewAppError(err, api.ErrorUnknown, api.CategoryUser)
		}
		response.PoliciesCreated += policiesCreated
		response.ItemsCreated += itemsCreated
	}

	response.LinesProcessed = n
	return response, nil
}

func importPolicy(tx *pop.Connection, data map[string]string, catID uuid.UUID, now time.Time) (int, int, error) {
	// fields common to both policy types
	const (
		Year         = "Veh_Year"
		Make         = "Veh_Make"
		Model        = "Veh_Model"
		VIN          = "Veh_VIN"
		CoveredValue = "Covered_Value"
		StartDate    = "Start_Date"
		Country      = "Country_Description"
	)

	// fields for household policies
	const (
		HouseholdID = "Account_Number"
		NameCust    = "NAMECUST"
	)

	// fields for team policies
	const (
		Person        = "Person"
		ItemName      = "Statement Name"
		PolicyName    = "Policy Name"
		Entity        = "Entity"
		CostCenter    = "Cost Center"
		Account       = "Account"
		AccountDetail = "Ledger Entry Desc"
	)

	var p Policy

	if data[HouseholdID] == "" {
		err := p.FindByTeamDetails(tx, data[Entity], data[Account], data[CostCenter], data[AccountDetail])
		if err != nil {
			if domain.IsOtherThanNoRows(err) {
				return 0, 0, err
			}
			name := data[PolicyName]
			if name == "" {
				name = data[Entity] + " " + data[Account] + data[CostCenter]
			}
			p.Name = name
			p.Type = api.PolicyTypeTeam
			p.EntityCodeID = EntityCodeID(data[Entity])
			p.Account = data[Account]
			p.CostCenter = data[CostCenter]
			p.AccountDetail = data[AccountDetail]

			if err := p.Create(tx); err != nil {
				return 0, 0, appErrorFromDB(err, api.ErrorCreateFailure)
			}
		}
	} else {
		err := p.FindByHouseholdID(tx, data[HouseholdID])
		if err != nil {
			if domain.IsOtherThanNoRows(err) {
				return 0, 0, err
			}

			p.Name = data[NameCust] + " household"
			p.Type = api.PolicyTypeHousehold
			p.HouseholdID = nulls.NewString(data[HouseholdID])
			p.EntityCodeID = householdEntityID

			if err := p.Create(tx); err != nil {
				return 0, 0, appErrorFromDB(err, api.ErrorCreateFailure)
			}
		}
	}

	var dependent PolicyDependent
	if data[Person] != "" {
		var err error
		dependent, err = p.AddDependent(tx, api.PolicyDependentInput{Name: data[Person]})
		if err != nil {
			return 0, 0, err
		}
	}

	value, err := parseCoveredValue(data[CoveredValue])
	if err != nil {
		return 0, 0, err
	}

	startDate, err := time.Parse("1/2/2006 15:04", data[StartDate])
	if err != nil {
		startDate = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	}

	itemName := data[ItemName]
	if itemName == "" {
		itemName = fmt.Sprintf("%s %s %s", data[Year], data[Make], data[Model])
	}

	i := Item{
		Name:              itemName,
		CategoryID:        catID,
		Country:           data[Country],
		PolicyID:          p.ID,
		Make:              data[Make],
		Model:             data[Model],
		SerialNumber:      data[VIN],
		CoverageAmount:    value,
		CoverageStatus:    api.ItemCoverageStatusApproved,
		CoverageStartDate: startDate,
		RiskCategoryID:    riskCategoryVehicleID,
		PaidThroughDate:   domain.EndOfMonth(now).AddDate(0, -1, 0),
	}

	if data[Year] != "" {
		year, err := parseVehicleYear(data[Year])
		if err != nil {
			return 0, 0, err
		}
		i.Year = nulls.NewInt(year)
	}

	if !dependent.ID.IsNil() {
		i.PolicyDependentID = nulls.NewUUID(dependent.ID)
	}

	if err := i.Create(tx); err != nil {
		return 0, 0, appErrorFromDB(err, api.ErrorCreateFailure)
	}
	return 1, 1, nil
}

func parseCoveredValue(s string) (int, error) {
	s = strings.Trim(s, "$")
	s = strings.ReplaceAll(s, ",", "")
	value, err := strconv.Atoi(s)
	if err != nil {
		return 0, fmt.Errorf("invalid covered value %q: %w", s, err)
	}
	if value < 1 || value > 100_000 {
		return 0, fmt.Errorf("invalid covered value %d", value)
	}
	return value * domain.CurrencyFactor, nil
}

func parseVehicleYear(s string) (int, error) {
	year, err := strconv.Atoi(s)
	if err != nil {
		return 0, fmt.Errorf("invalid vehicle year %q: %w", s, err)
	}
	if year < 0 || year > 2050 || (year >= 100 && year < 1913) {
		return 0, fmt.Errorf("invalid vehicle year %d", year)
	}
	if year >= 50 && year <= 99 {
		return year + 1900, nil
	}
	if year < 50 {
		return year + 2000, nil
	}
	return year, nil
}
