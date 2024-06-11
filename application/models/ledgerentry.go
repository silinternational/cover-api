package models

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/gobuffalo/nulls"
	"github.com/gobuffalo/pop/v6"
	"github.com/gofrs/uuid"

	"github.com/silinternational/cover-api/api"
	"github.com/silinternational/cover-api/domain"
	"github.com/silinternational/cover-api/fin"
)

const (
	accountSeparator = " / "
	csvPolicyHeader  = `"Amount","Description","Reference","Date Entered"` + "\n"
)

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

func (t LedgerEntryType) Description(claimPayoutOption string, amount api.Currency) string {
	switch t {
	case LedgerEntryTypeNewCoverage:
		return "Coverage premium: Add"
	case LedgerEntryTypeCoverageRenewal:
		return "Coverage premium: Renew"
	case LedgerEntryTypeCoverageRefund:
		return "Coverage reimbursement: Remove"
	case LedgerEntryTypeCoverageChange, LedgerEntryTypePolicyAdjustment:
		if amount >= 0 { // reimbursements/reductions are positive and charges are negative
			return "Coverage reimbursement: Reduce"
		}
		return "Coverage premium: Increase"
	case LedgerEntryTypeClaim, LedgerEntryTypeClaimAdjustment:
		switch claimPayoutOption {
		case FieldClaimItemFMV:
			return "Claim payout: Fair Market Value"
		case FieldClaimItemReplaceActual:
			return "Claim payout: Replace"
		case FieldClaimItemRepairActual:
			return "Claim payout: Repair"
		default:
			return "Claim transaction"
		}
	}
	return "unknown transaction type"
}

type LedgerEntries []LedgerEntry

type LedgerEntry struct {
	ID uuid.UUID `db:"id"`

	PolicyID          uuid.UUID       `db:"policy_id"`
	ItemID            nulls.UUID      `db:"item_id"`
	ClaimID           nulls.UUID      `db:"claim_id"`
	EntityCode        string          `db:"entity_code"`
	RiskCategoryName  string          `db:"risk_category_name"`
	RiskCategoryCC    string          `db:"risk_category_cc"` // Risk Category Cost Center
	Type              LedgerEntryType `db:"type" validate:"ledgerEntryType"`
	PolicyType        api.PolicyType  `db:"policy_type" validate:"policyType"`
	HouseholdID       string          `db:"household_id"`
	CostCenter        string          `db:"cost_center"`
	AccountNumber     string          `db:"account_number"`
	IncomeAccount     string          `db:"income_account"`
	Name              string          `db:"name"` // This will normally be the name of the assigned_to person
	PolicyName        string          `db:"policy_name"`
	ClaimPayoutOption string          `db:"claim_payout_option"`
	Amount            api.Currency    `db:"amount"`         // reimbursements/reductions are positive and charges are negative
	DateSubmitted     time.Time       `db:"date_submitted"` // date added to ledger
	DateEntered       nulls.Time      `db:"date_entered"`   // date entered into accounting system
	LegacyID          nulls.Int       `db:"legacy_id"`

	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`

	Claim *Claim `belongs_to:"claims" validate:"-"`
	Item  *Item  `belongs_to:"items" validate:"-"`
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

func (le *LedgerEntries) ExportForPolicy() ([]byte, string) {
	report := fin.NewBatch(fin.ReportFormatPolicy, "", time.Now())
	for _, l := range *le {
		report.AppendToBatch("", fin.Transaction{
			EntityCode:        l.EntityCode,
			RiskCategoryName:  l.RiskCategoryName,
			RiskCategoryCC:    l.RiskCategoryCC,
			Type:              string(l.Type),
			PolicyType:        l.PolicyType,
			HouseholdID:       l.HouseholdID,
			CostCenter:        l.CostCenter,
			AccountNumber:     l.AccountNumber,
			IncomeAccount:     l.IncomeAccount,
			Name:              l.Name,
			PolicyName:        l.PolicyName,
			ClaimPayoutOption: l.ClaimPayoutOption,
			Amount:            l.Amount,
			Date:              l.DateSubmitted,
			Description:       l.getDescription(),
		})
	}

	return report.RenderBatch()
}

type TransactionBlocks map[string]LedgerEntries // keyed by account

// NewReport creates a new LedgerReport with the current LedgerEntries
func (le *LedgerEntries) NewReport(ctx context.Context, reportFormat, reportType string, date time.Time) LedgerReport {
	report := LedgerReport{
		Date: date,
		Type: reportType,
	}

	content, contentType := le.ExportReport(reportFormat, reportType, report.Date)
	ext := "csv"
	if contentType == domain.ContentZip {
		ext = "zip"
	}

	report.File = File{
		Name: fmt.Sprintf("%s_%s_%s_%s.%s",
			domain.Env.AppName, reportFormat, reportType, report.Date.Format(domain.DateFormat), ext),
		Content:     content,
		ContentType: contentType,
		CreatedByID: CurrentUser(ctx).ID,
	}
	report.LedgerEntries = *le

	return report
}

func (le *LedgerEntries) ExportReport(reportFormat, reportType string, date time.Time) ([]byte, string) {
	report := le.prepareReport(reportFormat, reportType, date)
	return report.RenderBatch()
}

func (le *LedgerEntries) prepareReport(reportFormat, reportType string, date time.Time) fin.Report {
	report := fin.NewBatch(reportFormat, reportType, date)
	ref := ""

	blocks := le.MakeBlocks()
	for account, ledgerEntries := range blocks {
		if len(ledgerEntries) == 0 {
			continue
		}
		var balance int
		le := ledgerEntries[0]
		if le.Type.IsClaim() {
			account = account[len(le.PolicyType):]
		}
		desc := le.balanceDescription()
		blockName := strings.TrimPrefix(desc, "Total ")
		blockName = strings.ReplaceAll(blockName, " ", "_")

		for _, l := range ledgerEntries {
			report.AppendToBatch(blockName, fin.Transaction{
				EntityCode:        l.EntityCode,
				RiskCategoryName:  l.RiskCategoryName,
				RiskCategoryCC:    l.RiskCategoryCC,
				Type:              string(l.Type),
				PolicyType:        l.PolicyType,
				HouseholdID:       l.HouseholdID,
				CostCenter:        l.CostCenter,
				AccountNumber:     l.AccountNumber,
				IncomeAccount:     l.IncomeAccount,
				Name:              l.Name,
				PolicyName:        l.PolicyName,
				ClaimPayoutOption: l.ClaimPayoutOption,
				Amount:            l.Amount,
				Date:              l.DateSubmitted,
				Description:       l.getDescription(),
			})

			balance -= int(l.Amount)
		}

		report.AppendToBatch(blockName, fin.Transaction{
			Account:     account,
			Amount:      api.Currency(balance),
			Description: desc,
			Reference:   &ref,
			Date:        date,
		})
	}
	return report
}

func (le *LedgerEntries) MakeBlocks() TransactionBlocks {
	blocks := TransactionBlocks{}
	for _, e := range *le {
		key := e.IncomeAccount + e.RiskCategoryCC

		// Split claims up by Policy Type
		if e.Type.IsClaim() {
			key = string(e.PolicyType) + key
		}

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

// getDescription returns text that is base on other fields of the LedgerEntry
// For household-type entries this returns `<entry.Type.Description> / <Policy.Name>`.
// For other entries this returns `<entry.Type.Description> / <Policy.Name> (<accountable person name>)`,
// not including `<` and `>`
func (le *LedgerEntry) getDescription() string {
	description := le.Type.Description(le.ClaimPayoutOption, le.Amount)

	if le.PolicyName == "" {
		return description + " " + le.RiskCategoryName
	}

	description = fmt.Sprintf(`%s / %s`, description, le.PolicyName)

	if le.PolicyType == api.PolicyTypeHousehold || le.Name == "" {
		return description + " " + le.RiskCategoryName
	}

	return fmt.Sprintf(`%s (%s) %s`, description, le.Name, le.RiskCategoryName)
}

func (le *LedgerEntry) getItemName(tx *pop.Connection) string {
	le.LoadItem(tx, false)
	if le.Item != nil {
		return le.Item.Name
	}

	le.LoadClaim(tx)
	if le.Claim == nil {
		return ""
	}
	le.Claim.LoadClaimItems(tx, false)
	if len(le.Claim.ClaimItems) < 1 {
		return ""
	}

	cItem := le.Claim.ClaimItems[0]
	cItem.LoadItem(tx, false)
	return cItem.Item.Name
}

func (le *LedgerEntry) getItemLocation(tx *pop.Connection) string {
	if le.Item != nil {
		loctn := le.Item.GetAccountablePersonLocation(tx)
		return loctn.Country
	}
	if le.Claim == nil {
		return ""
	}
	le.Claim.LoadClaimItems(tx, false)
	if len(le.Claim.ClaimItems) < 1 {
		return ""
	}

	cItem := le.Claim.ClaimItems[0]
	cItem.LoadItem(tx, false)
	loctn := cItem.Item.GetAccountablePersonLocation(tx)
	return loctn.Country
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
		entity = string(le.PolicyType)
	}

	return fmt.Sprintf("Total %s %s %s", entity, le.RiskCategoryName, premiumsOrClaims)
}

// NewLedgerEntry creates a basic LedgerEntry with common fields completed.
// Requires pre-hydration of policy.EntityCode. If item is not nil, item.RiskCategory must be hydrated.
func NewLedgerEntry(accPersonName string, policy Policy, item *Item, claim *Claim, dateSubmitted time.Time) LedgerEntry {
	costCenter := ""
	if policy.Type == api.PolicyTypeTeam {
		costCenter = policy.CostCenter + accountSeparator + policy.AccountDetail
	}
	le := LedgerEntry{
		PolicyID:      policy.ID,
		PolicyType:    policy.Type,
		EntityCode:    policy.EntityCode.Code,
		DateSubmitted: dateSubmitted,
		AccountNumber: policy.Account,
		IncomeAccount: policy.EntityCode.IncomeAccount,
		CostCenter:    costCenter,
		Name:          accPersonName,
		PolicyName:    policy.Name,
	}
	if policy.HouseholdID.Valid {
		le.HouseholdID = policy.HouseholdID.String
	}

	if item != nil {
		le.ItemID = nulls.NewUUID(item.ID)
		le.RiskCategoryName = item.RiskCategory.Name
		le.RiskCategoryCC = item.RiskCategory.CostCenter
	}
	if claim != nil {
		le.ClaimPayoutOption = string(claim.ClaimItems[0].PayoutOption)
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

// LoadItem - a simple wrapper method for loading the item
func (le *LedgerEntry) LoadItem(tx *pop.Connection, reload bool) {
	if le.Item == nil || reload {
		if err := tx.Load(le, "Item"); err != nil {
			panic("error loading ledger entry item: " + err.Error())
		}
	}
}

// FindRenewals finds the coverage renewal ledger entries for the given year
func (le *LedgerEntries) FindRenewals(tx *pop.Connection, year int) error {
	if err := tx.Where("type = ?", LedgerEntryTypeCoverageRenewal).
		Where("EXTRACT(YEAR FROM date_submitted) = ?", year).
		Where("date_entered IS NULL").
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
		PolicyName:       le.PolicyName,
		Amount:           le.Amount,
		DateSubmitted:    le.DateSubmitted,
		DateEntered:      convertTimeToAPI(le.DateEntered),
		CreatedAt:        le.CreatedAt,
		UpdatedAt:        le.UpdatedAt,
	}
}

func adjustLedgerAmount(amount api.Currency, entryType LedgerEntryType) (api.Currency, error) {
	const minimumAmount = 100

	adjusted := amount
	switch entryType {
	case LedgerEntryTypeCoverageRefund:
		if amount > 0 {
			return adjusted, fmt.Errorf("invalid amount for refund, should not be positive: %d", amount)
		}
		if amount > -minimumAmount {
			adjusted = 0 // don't refund less than $1
		}
	case LedgerEntryTypeNewCoverage, LedgerEntryTypeCoverageRenewal:
		if amount < 0 {
			return adjusted, fmt.Errorf("invalid amount for payment, should not be negative: %d", amount)
		}
	case LedgerEntryTypeCoverageChange:
		if amount > -minimumAmount && amount < 0 {
			adjusted = 0 // don't refund less than $1
		}
	case LedgerEntryTypeClaim, LedgerEntryTypeClaimAdjustment:
		// no adjustments on claims
	default:
		return adjusted, fmt.Errorf("invalid LedgerEntryType specified in adjustLedgerAmount: %s", entryType)
	}
	return adjusted, nil
}
