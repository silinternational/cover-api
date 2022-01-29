package models

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gobuffalo/nulls"
	"github.com/gobuffalo/pop/v5"
	"github.com/gofrs/uuid"

	"github.com/silinternational/cover-api/api"
	"github.com/silinternational/cover-api/domain"
	"github.com/silinternational/cover-api/fin"
)

const accountSeparator = " / "

type LedgerEntryType string

func (t LedgerEntryType) IsClaim() bool {
	if t == LedgerEntryTypeClaim || t == LedgerEntryTypeClaimAdjustment {
		return true
	}
	return false
}

const (
	LedgerEntryTypeNewCoverage      = LedgerEntryType("NewCoverage")
	LedgerEntryTypeCoverageChange   = LedgerEntryType("CoverageChange")
	LedgerEntryTypeCoverageRefund   = LedgerEntryType("CoverageRefund")
	LedgerEntryTypeCoverageRenewal  = LedgerEntryType("CoverageRenewal")
	LedgerEntryTypePolicyAdjustment = LedgerEntryType("PolicyAdjustment")
	LedgerEntryTypeClaim            = LedgerEntryType("Claim")
	LedgerEntryTypeLegacy5          = LedgerEntryType("5")
	LedgerEntryTypeClaimAdjustment  = LedgerEntryType("ClaimAdjustment")
	LedgerEntryTypeLegacy20         = LedgerEntryType("20")
)

var ValidLedgerEntryTypes = map[LedgerEntryType]struct{}{
	LedgerEntryTypeNewCoverage:      {},
	LedgerEntryTypeCoverageChange:   {},
	LedgerEntryTypeCoverageRefund:   {},
	LedgerEntryTypeCoverageRenewal:  {},
	LedgerEntryTypePolicyAdjustment: {},
	LedgerEntryTypeClaim:            {},
	LedgerEntryTypeLegacy5:          {},
	LedgerEntryTypeClaimAdjustment:  {},
	LedgerEntryTypeLegacy20:         {},
}

type LedgerEntries []LedgerEntry

type LedgerEntry struct {
	ID uuid.UUID `db:"id"`

	PolicyID         uuid.UUID       `db:"policy_id"`
	ItemID           nulls.UUID      `db:"item_id"`
	ClaimID          nulls.UUID      `db:"claim_id"`
	EntityCode       string          `db:"entity_code"`
	RiskCategoryName string          `db:"risk_category_name"`
	RiskCategoryCC   string          `db:"risk_category_cc"` // Risk Category Cost Center
	Type             LedgerEntryType `db:"type" validate:"ledgerEntryType"`
	PolicyType       api.PolicyType  `db:"policy_type" validate:"policyType"`
	HouseholdID      string          `db:"household_id"`
	CostCenter       string          `db:"cost_center"`
	AccountNumber    string          `db:"account_number"`
	IncomeAccount    string          `db:"income_account"`
	Name             string          `db:"name"`
	Amount           api.Currency    `db:"amount"`
	DateSubmitted    time.Time       `db:"date_submitted"`
	DateEntered      nulls.Time      `db:"date_entered"`
	LegacyID         nulls.Int       `db:"legacy_id"`

	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`

	Claim *Claim `belongs_to:"claims" validate:"-"`
}

func (le *LedgerEntry) GetID() uuid.UUID {
	return le.ID
}

func (le *LedgerEntry) FindByID(tx *pop.Connection, id uuid.UUID) error {
	return tx.Find(le, id)
}

func (le *LedgerEntry) IsActorAllowedTo(tx *pop.Connection, user User, perm Permission, sub SubResource, req *http.Request) bool {
	if user.IsAdmin() {
		return true
	}
	return false
}

func (le *LedgerEntry) Create(tx *pop.Connection) error {
	return create(tx, le)
}

func (le *LedgerEntry) Update(tx *pop.Connection) error {
	return update(tx, le)
}

// AllNotEntered returns all the non-entered entries (date_entered is null) up to the given cutoff time.
func (le *LedgerEntries) AllNotEntered(tx *pop.Connection, cutoff time.Time) error {
	err := tx.Where("date_submitted < ? ", cutoff).
		Where("date_entered IS NULL").All(le)

	return appErrorFromDB(err, api.ErrorQueryFailure)
}

type TransactionBlocks map[string]LedgerEntries // keyed by account

func (le *LedgerEntries) ToCsv(date time.Time) []byte {
	sage := fin.NewBatch(fin.ProviderTypeSage, date)

	blocks := le.MakeBlocks()
	for account, ledgerEntries := range blocks {
		if len(ledgerEntries) == 0 {
			continue
		}
		var balance int
		for _, l := range ledgerEntries {
			sage.AppendToBatch(fin.Transaction{
				Account:     domain.Env.ExpenseAccount,
				Amount:      int(l.Amount),
				Description: l.transactionDescription(),
				Reference:   l.transactionReference(),
				Date:        l.DateSubmitted,
			})

			balance -= int(l.Amount)
		}
		sage.AppendToBatch(fin.Transaction{
			Account:     account,
			Amount:      balance,
			Description: ledgerEntries[0].balanceDescription(),
			Reference:   "",
			Date:        date,
		})
	}

	return sage.BatchToCSV()
}

func (le *LedgerEntries) MakeBlocks() TransactionBlocks {
	blocks := TransactionBlocks{}
	for _, e := range *le {
		key := e.IncomeAccount + e.RiskCategoryCC
		blocks[key] = append(blocks[key], e)
	}
	return blocks
}

// Reconcile marks each LedgerEntry as "entered" into the accounting system, and makes any
// necessary updates to the referenced objects, such as setting Claim status to Paid.
func (le *LedgerEntries) Reconcile(ctx context.Context) error {
	now := time.Now().UTC()
	for _, e := range *le {
		if err := e.Reconcile(ctx, now); err != nil {
			return err
		}
	}
	return nil
}

// Reconcile marks the LedgerEntry as "entered" into the accounting system, and makes any
// necessary updates to the referenced objects, such as setting Claim status to Paid.
func (le *LedgerEntry) Reconcile(ctx context.Context, now time.Time) error {
	tx := Tx(ctx)

	le.DateEntered = nulls.NewTime(now)
	if err := le.Update(tx); err != nil {
		return err
	}

	le.LoadClaim(tx)
	if le.Claim != nil {
		le.Claim.Status = api.ClaimStatusPaid
		// Use Update instead of UpdateStatus so the ClaimItem(s) get updated as well
		if err := le.Claim.Update(ctx); err != nil {
			return err
		}
	}
	return nil
}

func (le *LedgerEntry) transactionDescription() string {
	description := fmt.Sprintf("%s %s", le.RiskCategoryName, le.Type)
	if le.PolicyType == api.PolicyTypeHousehold {
		description = le.Name + " " + description
	}

	return description
}

// Merge HouseholdID and CostCenter into one column
func (le *LedgerEntry) transactionReference() string {
	if le.PolicyType == api.PolicyTypeHousehold {
		return fmt.Sprintf("MC %s", le.HouseholdID)
	}
	if le.LegacyID.Valid {
		// In the legacy data, the cost center column was prepended with the entity code and account number
		return le.CostCenter
	}
	return fmt.Sprintf("%s %s%s", le.EntityCode, le.AccountNumber, le.CostCenter)
}

func (le *LedgerEntry) balanceDescription() string {
	entity := le.EntityCode
	e := EntityCode{Code: le.EntityCode}

	// Don't need to use a transaction since entity codes shouldn't be changing during this operation.
	if err := e.FindByCode(DB); err == nil && e.ParentEntity != "" {
		entity = e.ParentEntity
	}

	premiumsOrClaims := "Premiums"
	if le.Type.IsClaim() {
		premiumsOrClaims = "Claims"

		// Claims transactions use the same account for all entities
		entity = "all"
	}

	return fmt.Sprintf("Total %s %s %s", entity, le.RiskCategoryName, premiumsOrClaims)
}

// NewLedgerEntry creates a basic LedgerEntry with common fields completed.
// Requires pre-hydration of policy.EntityCode. If item is not nil, item.RiskCategory must be hydrated.
func NewLedgerEntry(policy Policy, item *Item, claim *Claim) LedgerEntry {
	costCenter := ""
	if policy.Type == api.PolicyTypeTeam {
		costCenter = policy.CostCenter + accountSeparator + policy.AccountDetail
	}
	le := LedgerEntry{
		PolicyID:      policy.ID,
		PolicyType:    policy.Type,
		EntityCode:    policy.EntityCode.Code,
		DateSubmitted: time.Now().UTC(),
		AccountNumber: policy.Account,
		IncomeAccount: policy.EntityCode.IncomeAccount,
		CostCenter:    costCenter,
		HouseholdID:   policy.HouseholdID.String,
	}
	if item != nil {
		le.ItemID = nulls.NewUUID(item.ID)
		le.RiskCategoryName = item.RiskCategory.Name
		le.RiskCategoryCC = item.RiskCategory.CostCenter
	}
	if claim != nil {
		le.ClaimID = nulls.NewUUID(claim.ID)
	}
	return le
}

// LoadClaim - a simple wrapper method for loading the claim
func (le *LedgerEntry) LoadClaim(tx *pop.Connection) {
	if le.ClaimID.Valid {
		if err := tx.Load(le, "Claim"); err != nil {
			panic("error loading ledger entry claim: " + err.Error())
		}
	}
}

// FindRenewals finds the coverage renewal ledger entries for the given year
func (le *LedgerEntries) FindRenewals(tx *pop.Connection, year int) error {
	if err := tx.Where("type = ?", LedgerEntryTypeCoverageRenewal).
		Where("EXTRACT(YEAR FROM date_submitted) = ?", year).
		All(le); err != nil {
		return appErrorFromDB(err, api.ErrorQueryFailure)
	}
	return nil
}

func (le *LedgerEntries) ConvertToAPI(tx *pop.Connection) api.LedgerEntries {
	ledgerEntries := make(api.LedgerEntries, len(*le))
	for i, l := range *le {
		ledgerEntries[i] = l.ConvertToAPI(tx)
	}
	return ledgerEntries
}

func (le *LedgerEntry) ConvertToAPI(tx *pop.Connection) api.LedgerEntry {
	return api.LedgerEntry{
		ID:               le.ID,
		PolicyID:         le.PolicyID,
		ItemID:           le.ItemID,
		ClaimID:          le.ClaimID,
		EntityCode:       le.EntityCode,
		RiskCategoryName: le.RiskCategoryName,
		RiskCategoryCC:   le.RiskCategoryCC,
		Type:             api.LedgerEntryType(le.Type),
		PolicyType:       le.PolicyType,
		HouseholdID:      le.HouseholdID,
		CostCenter:       le.CostCenter,
		AccountNumber:    le.AccountNumber,
		IncomeAccount:    le.IncomeAccount,
		Name:             le.Name,
		Amount:           le.Amount,
		DateSubmitted:    le.DateSubmitted,
		DateEntered:      convertTimeToAPI(le.DateEntered),
		CreatedAt:        le.CreatedAt,
		UpdatedAt:        le.UpdatedAt,
	}
}
