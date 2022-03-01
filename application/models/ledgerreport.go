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

const (
	ReportTypeMonthly = "Monthly"
	ReportTypeAnnual  = "Annual"
)

var ValidLedgerReportTypes = map[string]struct{}{
	ReportTypeMonthly: {},
	ReportTypeAnnual:  {},
}

type LedgerReports []LedgerReport

func (lr *LedgerReports) All(tx *pop.Connection) error {
	return appErrorFromDB(tx.All(lr), api.ErrorQueryFailure)
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
//    the LedgerReport's policy (assuming it has one)
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
func NewLedgerReport(ctx context.Context, reportType string, date time.Time) (LedgerReport, error) {
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
	report.File.Content = le.ToCsv(report.Date)
	report.File.CreatedByID = CurrentUser(ctx).ID
	report.File.ContentType = domain.ContentCSV
	report.LedgerEntries = le

	return report, nil
}

// NewPolicyLedgerReport creates a new report for one policy by querying the database according
//   to the requested report type and the month and year of the request
func NewPolicyLedgerReport(ctx context.Context, policy Policy, reportType string, month, year int) (LedgerReport, error) {
	tx := Tx(ctx)

	now := time.Now().UTC()
	nowYear := now.Year()

	report := LedgerReport{Type: reportType, PolicyID: nulls.NewUUID(policy.ID)}

	if year < 1971 || year > nowYear {
		err := fmt.Errorf("invalid year requested: %d", year)
		return report, api.NewAppError(err, api.ErrorInvalidDate, api.CategoryUser)
	}

	var le LedgerEntries
	switch reportType {
	case ReportTypeMonthly:
		if err := newMonthlyPolicyLedgerReport(tx, &le, &report, policy, month, year); err != nil {
			return report, err
		}
	case ReportTypeAnnual:
		if err := newAnnualPolicyLedgerReport(tx, &le, &report, policy, year); err != nil {
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

	report.File.Name = fmt.Sprintf("%s_policy_%s_%s_%d-%d.csv",
		domain.Env.AppName, policy.ID.String(), reportType, year, month)

	csvContent, err := le.ToCsvForPolicy()
	if err != nil {
		return LedgerReport{}, err
	}
	report.File.Content = csvContent
	report.File.CreatedByID = CurrentUser(ctx).ID
	report.File.ContentType = domain.ContentCSV
	report.LedgerEntries = le

	return report, nil
}

func newMonthlyPolicyLedgerReport(tx *pop.Connection, lEntries *LedgerEntries, lReport *LedgerReport, policy Policy, month, year int) error {
	if month < 1 || month > 12 {
		err := fmt.Errorf("invalid month requested: %d", month)
		return api.NewAppError(err, api.ErrorInvalidDate, api.CategoryUser)
	}

	now := time.Now().UTC()
	if year == now.Year() && month > int(now.Month()) {
		err := fmt.Errorf("invalid future month requested. Month: %d, Year: %d", month, year)
		return api.NewAppError(err, api.ErrorInvalidDate, api.CategoryUser)
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

func (lr *LedgerReport) Reconcile(ctx context.Context) error {
	tx := Tx(ctx)
	lr.LoadLedgerEntries(tx, false)
	if err := lr.LedgerEntries.Reconcile(ctx); err != nil {
		return api.NewAppError(err, api.ErrorReconcileError, api.CategoryInternal)
	}
	lr.LoadLedgerEntries(tx, true)
	return nil
}
