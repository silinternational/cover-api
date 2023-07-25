package models

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gobuffalo/nulls"
	"github.com/gobuffalo/pop/v6"
	"github.com/gobuffalo/validate/v3"
	"github.com/gofrs/uuid"

	"github.com/silinternational/cover-api/api"
	"github.com/silinternational/cover-api/domain"
)

const (
	ReportTypeMonthly = "Monthly"
	ReportTypeAnnual  = "Annual"
	MinimumYear       = 1971
)

var ValidLedgerReportTypes = map[string]struct{}{
	ReportTypeMonthly: {},
	ReportTypeAnnual:  {},
}

type LedgerReports []LedgerReport

// AllNonPolicy returns all the LedgerReports which have a null policy_id
func (lr *LedgerReports) AllNonPolicy(tx *pop.Connection) error {
	return appErrorFromDB(tx.Where("policy_id IS NULL").All(lr), api.ErrorQueryFailure)
}

// AllForPolicy returns all the LedgerReports which have a matching policy_id
func (lr *LedgerReports) AllForPolicy(tx *pop.Connection, policyID uuid.UUID) error {
	return appErrorFromDB(tx.Where("policy_id = ?", policyID).All(lr), api.ErrorQueryFailure)
}

func (lr *LedgerReports) ConvertToAPI(tx *pop.Connection) api.LedgerReports {
	ledgerReports := make(api.LedgerReports, len(*lr))
	for i, l := range *lr {
		ledgerReports[i] = l.ConvertToAPI(tx)
	}
	return ledgerReports
}

type LedgerReport struct {
	ID        uuid.UUID  `db:"id"`
	FileID    uuid.UUID  `db:"file_id" validate:"required"`
	Type      string     `db:"type"`
	Date      time.Time  `db:"date"`
	PolicyID  nulls.UUID `db:"policy_id"`
	CreatedAt time.Time  `db:"created_at"`
	UpdatedAt time.Time  `db:"updated_at"`

	File          File          `belongs_to:"files" validate:"-"`
	Policy        Policy        `belongs_to:"policies" validate:"-"`
	LedgerEntries LedgerEntries `many_to_many:"ledger_report_entries" validate:"-"`
}

// Validate gets run every time you call a "pop.Validate*" (pop.ValidateAndSave, pop.ValidateAndCreate, pop.ValidateAndUpdate) method.
func (lr *LedgerReport) Validate(tx *pop.Connection) (*validate.Errors, error) {
	return validateModel(lr), nil
}

// Create the LedgerReport, the File, and LedgerEntry junction table records
func (lr *LedgerReport) Create(tx *pop.Connection) error {
	lr.File.Linked = true
	if err := lr.File.Store(tx); err != nil {
		return err
	}
	lr.FileID = lr.File.ID

	// Pop will create junction table records for all connected LedgerEntries
	if err := create(tx, lr); err != nil {
		return appErrorFromDB(err, api.ErrorCreateFailure)
	}

	return nil
}

func (lr *LedgerReport) GetID() uuid.UUID {
	return lr.ID
}

func (lr *LedgerReport) FindByID(tx *pop.Connection, id uuid.UUID) error {
	return tx.Find(lr, id)
}

// IsActorAllowedTo ensures the actor is either an admin or a member of
// the LedgerReport's policy (assuming it has one)
func (lr *LedgerReport) IsActorAllowedTo(tx *pop.Connection, actor User, perm Permission, sub SubResource, r *http.Request) bool {
	if actor.IsAdmin() {
		return true
	}

	if lr.PolicyID.Valid {
		lr.LoadPolicy(tx, false)
		return lr.Policy.isMember(tx, actor.ID)
	}

	return false
}

// ConvertToAPI converts a LedgerReport to api.LedgerReport
func (lr *LedgerReport) ConvertToAPI(tx *pop.Connection) api.LedgerReport {
	lr.LoadFile(tx, false)
	lr.LoadLedgerEntries(tx, false)

	transactionCount := len(lr.LedgerEntries)
	isCleared := true
	for _, e := range lr.LedgerEntries {
		if e.Amount == 0 {
			// TODO: consider whether we should even store ledger entries with a zero amount
			transactionCount--
		}
		if !e.DateEntered.Valid {
			isCleared = false
			break
		}
	}

	return api.LedgerReport{
		ID:               lr.ID,
		File:             lr.File.ConvertToAPI(tx),
		Type:             lr.Type,
		Date:             lr.Date,
		TransactionCount: transactionCount,
		IsCleared:        isCleared,
		CreatedAt:        lr.CreatedAt,
		UpdatedAt:        lr.UpdatedAt,
	}
}

// LoadFile hydrates the File property if necessary or if `reload` is true. The file URL is refreshed
// in any case.
func (lr *LedgerReport) LoadFile(tx *pop.Connection, reload bool) {
	if lr.File.ID == uuid.Nil || reload {
		if err := tx.Load(lr, "File"); err != nil {
			panic("database error loading LedgerReport.File, " + err.Error())
		}
	}
	if err := lr.File.RefreshURL(tx); err != nil {
		panic("failed to refresh LedgerReport.File URL, " + err.Error())
	}
}

func (lr *LedgerReport) LoadLedgerEntries(tx *pop.Connection, reload bool) {
	if len(lr.LedgerEntries) == 0 || reload {
		if err := tx.Load(lr, "LedgerEntries"); err != nil {
			panic("database error loading LedgerReport.LedgerEntries, " + err.Error())
		}
	}
}

// LoadPolicy hydrates the Policy property if necessary or if `reload` is true.
func (lr *LedgerReport) LoadPolicy(tx *pop.Connection, reload bool) {
	if lr.PolicyID.Valid && (lr.Policy.ID == uuid.Nil || reload) {
		if err := tx.Load(lr, "Policy"); err != nil {
			panic("database error loading LedgerReport.Policy, " + err.Error())
		}
	}
}

// NewLedgerReport creates a new report by querying the database according to the requested report type
func NewLedgerReport(ctx context.Context, format, reportType string, date time.Time) (LedgerReport, error) {
	tx := Tx(ctx)

	report := LedgerReport{Type: reportType}

	if date.After(time.Now().UTC()) {
		err := errors.New("future date requested for ledger report: " + date.Format(domain.DateFormat))
		return report, api.NewAppError(err, api.ErrorInvalidDate, api.CategoryUser)
	}

	var le LedgerEntries
	switch reportType {
	case ReportTypeMonthly:
		report.Date = domain.BeginningOfDay(date)
		if err := le.AllNotEntered(tx, report.Date); err != nil {
			return report, err
		}
	case ReportTypeAnnual:
		year := date.Year()
		report.Date = time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC)
		if err := le.FindRenewals(tx, year); err != nil {
			return report, err
		}
	default:
		err := errors.New("invalid report type: " + reportType)
		return report, api.NewAppError(err, api.ErrorInvalidReportType, api.CategoryUser)
	}

	if len(le) == 0 {
		err := errors.New("no LedgerEntries found")
		return LedgerReport{}, api.NewAppError(err, api.ErrorNoLedgerEntries, api.CategoryNotFound)
	}

	report.File.Name = fmt.Sprintf("%s_%s_%s.csv",
		domain.Env.AppName, reportType, report.Date.Format(domain.DateFormat))
	report.File.Content = le.ToCsv(format, report.Date)
	report.File.CreatedByID = CurrentUser(ctx).ID
	report.File.ContentType = domain.ContentCSV
	report.LedgerEntries = le

	return report, nil
}

// NewPolicyLedgerReport creates a new report for one policy by querying the database according
// to the requested report type and the month and year of the request.
// If no ledger entries are found, it returns an empty LedgerReport.
func NewPolicyLedgerReport(ctx context.Context, policy Policy, reportType string, month, year int) (LedgerReport, error) {
	tx := Tx(ctx)

	report := LedgerReport{Type: reportType, PolicyID: nulls.NewUUID(policy.ID)}

	var le LedgerEntries
	switch reportType {
	case ReportTypeMonthly:
		if err := validateMonthYearForReport(month, year); err != nil {
			return LedgerReport{}, err
		}
		if err := newMonthlyPolicyLedgerReport(tx, &le, &report, policy, month, year); err != nil {
			return report, err
		}
	case ReportTypeAnnual:
		if err := validateYearForReport(year); err != nil {
			return LedgerReport{}, err
		}
		if err := newAnnualPolicyLedgerReport(tx, &le, &report, policy, year); err != nil {
			return report, err
		}
	default:
		err := errors.New("invalid report type: " + reportType)
		return report, api.NewAppError(err, api.ErrorInvalidReportType, api.CategoryUser)
	}

	if len(le) == 0 {
		return LedgerReport{}, nil
	}

	report.File.Name = fmt.Sprintf("%s_policy_%s_%s_%d-%d.csv",
		domain.Env.AppName, policy.ID.String(), reportType, year, month)
	report.File.Content = le.ToCsvForPolicy()
	report.File.CreatedByID = CurrentUser(ctx).ID
	report.File.ContentType = domain.ContentCSV
	report.LedgerEntries = le

	return report, nil
}

func newMonthlyPolicyLedgerReport(tx *pop.Connection, lEntries *LedgerEntries, lReport *LedgerReport, policy Policy, month, year int) error {
	if err := validateMonthYearForReport(month, year); err != nil {
		return err
	}

	startDate := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	endDate := startDate.AddDate(0, 1, -1)

	err := tx.Where("policy_id = ?", policy.ID).
		Where("date_entered >= ? AND date_entered <= ?", startDate, endDate).All(lEntries)

	lReport.Date = startDate
	if domain.IsOtherThanNoRows(err) {
		return api.NewAppError(err, api.ErrorUnknown, api.CategoryDatabase)
	}
	return nil
}

func validateMonthYearForReport(month, year int) error {
	if err := validateYearForReport(year); err != nil {
		return err
	}

	if month < 1 || month > 12 {
		err := fmt.Errorf("invalid month requested: %d", month)
		return api.NewAppError(err, api.ErrorInvalidDate, api.CategoryUser)
	}

	now := time.Now().UTC()
	if year == now.Year() && month > int(now.Month()) {
		err := fmt.Errorf("invalid future month requested. Month: %d, Year: %d", month, year)
		return api.NewAppError(err, api.ErrorInvalidDate, api.CategoryUser)
	}
	return nil
}

func validateYearForReport(year int) error {
	now := time.Now().UTC()
	nowYear := now.Year()
	if year < MinimumYear || year > nowYear {
		err := fmt.Errorf("invalid year requested: %d", year)
		return api.NewAppError(err, api.ErrorInvalidDate, api.CategoryUser)
	}
	return nil
}

func newAnnualPolicyLedgerReport(tx *pop.Connection, lEntries *LedgerEntries, lReport *LedgerReport, policy Policy, year int) error {
	startDate := time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := startDate.AddDate(1, 0, -1)
	lReport.Date = startDate

	err := tx.Where("policy_id = ?", policy.ID).
		Where("date_entered >= ? AND date_entered <= ?", startDate, endDate).All(lEntries)
	if domain.IsOtherThanNoRows(err) {
		return api.NewAppError(err, api.ErrorUnknown, api.CategoryDatabase)
	}
	return nil
}

func PolicyLedgerTable(c context.Context, policy Policy, month, year int) (api.LedgerTable, error) {
	tx := Tx(c)

	if err := validateMonthYearForReport(month, year); err != nil {
		return api.LedgerTable{}, err
	}

	var ledgerEntries LedgerEntries
	startDate := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	endDate := startDate.AddDate(0, 1, -1)

	const q = "policy_id = ? AND date_entered BETWEEN ? AND ?"
	if err := tx.Where(q, policy.ID, startDate, endDate).All(&ledgerEntries); err != nil {
		if domain.IsOtherThanNoRows(err) {
			return api.LedgerTable{}, api.NewAppError(err, api.ErrorUnknown, api.CategoryDatabase)
		}
		return api.LedgerTable{}, nil
	}

	// TODO: hydrate LastChanged date field when we figure out what that should be based on
	lTable := api.LedgerTable{
		PremiumTotal:  policy.calculateAnnualPremium(tx),
		PremiumRate:   domain.Env.PremiumFactor,
		CoverageValue: policy.currentCoverageTotal(tx),
		ReportMonth:   month,
		ReportYear:    year,
		Entries:       make([]api.LedgerTableEntry, len(ledgerEntries)),
	}

	payoutTotal := api.Currency(0)
	netTransactions := api.Currency(0)

	for i, le := range ledgerEntries {

		var statusBefore, statusAfter string

		switch le.Type {
		case LedgerEntryTypeClaim, LedgerEntryTypeClaimAdjustment:
			statusBefore = string(api.ClaimStatusReview3)
			statusAfter = string(api.ClaimStatusApproved)
		case LedgerEntryTypeNewCoverage:
			statusBefore = string(api.ItemCoverageStatusPending)
			statusAfter = string(api.ItemCoverageStatusApproved)
		case LedgerEntryTypePolicyAdjustment:
			statusAfter = api.PolicyStatusActive
		case LedgerEntryTypeCoverageRefund:
			statusBefore = string(api.ItemCoverageStatusApproved)
			statusAfter = string(api.ItemCoverageStatusInactive)
		case LedgerEntryTypeCoverageChange, LedgerEntryTypeCoverageRenewal:
			statusBefore = string(api.ItemCoverageStatusApproved)
			statusAfter = string(api.ItemCoverageStatusApproved)
		}

		lTable.Entries[i].ItemName = le.getItemName(tx)
		lTable.Entries[i].Type = le.Type.Description(le.ClaimPayoutOption, le.Amount)
		lTable.Entries[i].Value = le.Amount
		lTable.Entries[i].Date = le.DateEntered.Time
		lTable.Entries[i].AssignedTo = le.Name
		lTable.Entries[i].Location = le.getItemLocation(tx)
		lTable.Entries[i].StatusBefore = statusBefore
		lTable.Entries[i].StatusAfter = statusAfter
		netTransactions += le.Amount
		if le.Amount > 0 { // reimbursements/reductions are positive and charges are negative
			payoutTotal += le.Amount
		}
	}

	lTable.PayoutTotal = payoutTotal
	lTable.NetTransactions = netTransactions

	return lTable, nil
}

func (lr *LedgerReport) Reconcile(ctx context.Context) error {
	tx := Tx(ctx)
	lr.LoadLedgerEntries(tx, false)
	if err := lr.LedgerEntries.Reconcile(ctx); err != nil {
		return api.NewAppError(err, api.ErrorReconcileError, api.CategoryInternal)
	}
	lr.LoadLedgerEntries(tx, true)
	return nil
}
