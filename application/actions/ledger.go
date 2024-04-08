package actions

import (
	"fmt"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/silinternational/cover-api/fin"
	"github.com/silinternational/cover-api/job"

	"github.com/silinternational/cover-api/api"
	"github.com/silinternational/cover-api/domain"
	"github.com/silinternational/cover-api/models"
)

// swagger:operation GET /ledger-reports LedgerReport LedgerReportList
// LedgerReportList
//
// Return a list of ledger reports that are not associated with a policy
// ---
//
//	responses:
//	  '200':
//	    description: LedgerReport list
//	    schema:
//	      type: array
//	      items:
//	        "$ref": "#/definitions/LedgerReport"
func ledgerReportList(c echo.Context) error {
	var list models.LedgerReports

	tx := models.Tx(c)
	if err := list.AllNonPolicy(tx); err != nil {
		return reportError(c, err)
	}

	return renderOk(c, list.ConvertToAPI(tx))
}

// swagger:operation GET /ledger-reports/{id} LedgerReport LedgerReportView
// LedgerReportView
//
// Return the ledger report specified by `id`. The returned object contains metadata and a File object pointing to
// a CSV file suitable for use with Sage Accounting.
// ---
//
//	parameters:
//	- name: id
//	  in: path
//	  required: true
//	  description: specifies the ID of the report to view
//	responses:
//	  '200':
//	    description: the requested LedgerReport
//	    schema:
//	      "$ref": "#/definitions/LedgerReport"
func ledgerReportView(c echo.Context) error {
	tx := models.Tx(c)

	ledgerReport := getReferencedLedgerReportFromCtx(c)
	return renderOk(c, ledgerReport.ConvertToAPI(tx))
}

// swagger:operation POST /ledger-reports LedgerReport LedgerReportCreate
// LedgerReportCreate
//
// Create and return a report on the ledger entries as specified by the input object. The returned object
// contains metadata and a File object pointing to a CSV file suitable for use with Sage Accounting.
//
// ### Report types:
// + `monthly` - Return all ledger entries not yet reconciled, up to the beginning of the given day (0:00 UTC).
// + `annual` - Return the billing detail for given year's policy renewals.
// ---
//
//	parameters:
//	  - name: input
//	    in: body
//	    description: LedgerReportCreateInput object
//	    required: true
//	    schema:
//	      "$ref": "#/definitions/LedgerReportCreateInput"
//	responses:
//	  '200':
//	    description: the requested LedgerReport
//	    schema:
//	      "$ref": "#/definitions/LedgerReport"
func ledgerReportCreate(c echo.Context) error {
	var input api.LedgerReportCreateInput
	if err := StrictBind(c, &input); err != nil {
		return reportError(c, err)
	}

	date, err := time.Parse(domain.DateFormat, input.Date)
	if err != nil {
		return reportError(c, api.NewAppError(err, api.ErrorInvalidDate, api.CategoryUser))
	}

	// Create Sage report
	report, err := models.NewLedgerReport(c, fin.ReportFormatSage, input.Type, date)
	if err != nil {
		return reportError(c, err)
	}

	tx := models.Tx(c)

	if err = report.Create(tx); err != nil {
		return reportError(c, err)
	}

	// Create NetSuite report
	netsuite := report.LedgerEntries.NewReport(c, fin.ReportFormatNetSuite, input.Type, report.Date)
	if err = netsuite.Create(tx); err != nil {
		return reportError(c, err)
	}

	return renderOk(c, report.ConvertToAPI(tx))
}

// swagger:operation PUT /ledger-reports/{id} LedgerReport LedgerReportReconcile
// LedgerReportReconcile
//
// Mark ledger entries in the report reconciled as of today. Call this only after all transactions in the report
// have been fully loaded into the accounting record.
// ---
//
//	parameters:
//	- name: id
//	  in: path
//	  required: true
//	  description: specifies the ID of the report to reconcile
//	responses:
//	  '200':
//	    description: the requested LedgerReport
//	    schema:
//	      "$ref": "#/definitions/LedgerReport"
func ledgerReportReconcile(c echo.Context) error {
	tx := models.Tx(c)

	ledgerReport := getReferencedLedgerReportFromCtx(c)
	if err := ledgerReport.Reconcile(c); err != nil {
		return reportError(c, err)
	}

	return renderOk(c, ledgerReport.ConvertToAPI(tx))
}

// swagger:operation POST /ledger-reports/annual Ledger LedgerAnnualProcess
// LedgerAnnualProcess
//
// Process billing for current year's policy renewals.
// ---
//
//	responses:
//	  '204':
//	    description: OK but no content in response
func ledgerAnnualRenewalProcess(c echo.Context) error {
	actor := models.CurrentUser(c)
	if !actor.IsAdmin() {
		err := fmt.Errorf("user not allowed to process annual batch data")
		return reportError(c, api.NewAppError(err, api.ErrorNotAuthorized, api.CategoryForbidden))
	}

	if err := job.Submit(job.AnnualRenewal, map[string]any{}); err != nil {
		return reportError(c, api.NewAppError(err, api.ErrorFailedToSubmitJob, api.CategoryInternal))
	}

	return c.JSON(http.StatusNoContent, nil)
}

// swagger:operation GET /ledger-reports/annual Ledger LedgerAnnualRenewalStatus
// LedgerAnnualRenewalStatus
//
// Get the status of the annual billing process.
// ---
//
//	responses:
//	  '200':
//	    description: the status of the annual billing process
//	    schema:
//	      "$ref": "#/definitions/RenewalStatus"
func ledgerAnnualRenewalStatus(c echo.Context) error {
	actor := models.CurrentUser(c)
	if !actor.IsAdmin() {
		err := fmt.Errorf("user not allowed to access annual batch data")
		return reportError(c, api.NewAppError(err, api.ErrorNotAuthorized, api.CategoryForbidden))
	}

	tx := models.Tx(c)

	now := time.Now().UTC()
	endOfYear := domain.EndOfYear(now.Year())

	itemsToRenew, err := models.CountItemsToRenew(tx, endOfYear, domain.BillingPeriodAnnual)
	if err != nil {
		return err
	}

	status := api.RenewalStatus{
		IsComplete:     itemsToRenew == 0,
		ItemsToProcess: itemsToRenew,
		SafeToProcess:  models.IsSafeToRenewAnnual(tx, now),
	}
	return renderOk(c, status)
}

// swagger:operation POST /ledger-reports/monthly Ledger LedgerMonthlyProcess
// LedgerMonthlyProcess
//
// Process billing for current month's policy renewals.
// ---
//
//	responses:
//	  '204':
//	    description: OK but no content in response
func ledgerMonthlyRenewalProcess(c echo.Context) error {
	actor := models.CurrentUser(c)
	if !actor.IsAdmin() {
		err := fmt.Errorf("user not allowed to process monthly batch data")
		return reportError(c, api.NewAppError(err, api.ErrorNotAuthorized, api.CategoryForbidden))
	}

	if err := job.Submit(job.MonthlyRenewal, map[string]any{}); err != nil {
		return reportError(c, api.NewAppError(err, api.ErrorFailedToSubmitJob, api.CategoryInternal))
	}

	return c.JSON(http.StatusNoContent, nil)
}

// swagger:operation GET /ledger-reports/monthly Ledger LedgerMonthlyRenewalStatus
// LedgerMonthlyRenewalStatus
//
// Get the status of the monthly billing process.
// ---
//
//	responses:
//	  '200':
//	    description: the status of the monthly billing process
//	    schema:
//	      "$ref": "#/definitions/RenewalStatus"
func ledgerMonthlyRenewalStatus(c echo.Context) error {
	actor := models.CurrentUser(c)
	if !actor.IsAdmin() {
		err := fmt.Errorf("user not allowed to access monthly batch data")
		return reportError(c, api.NewAppError(err, api.ErrorNotAuthorized, api.CategoryForbidden))
	}

	tx := models.Tx(c)

	now := time.Now().UTC()

	itemsToRenew, err := models.CountItemsToRenew(tx, now, domain.BillingPeriodMonthly)
	if err != nil {
		return err
	}

	status := api.RenewalStatus{
		IsComplete:     itemsToRenew == 0,
		ItemsToProcess: itemsToRenew,
		SafeToProcess:  models.IsSafeToRenewMonthly(tx, now),
	}
	return renderOk(c, status)
}

func getReferencedLedgerReportFromCtx(c echo.Context) *models.LedgerReport {
	lr, ok := c.Get(domain.TypeLedgerReport).(*models.LedgerReport)
	if !ok {
		panic("LedgerReport not found in context")
	}
	return lr
}
