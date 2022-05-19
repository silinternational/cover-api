package actions

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gobuffalo/buffalo"

	"github.com/silinternational/cover-api/api"
	"github.com/silinternational/cover-api/domain"
	"github.com/silinternational/cover-api/models"
)

// swagger:operation GET /ledger-reports LedgerReport LedgerReportList
//
// LedgerReportList
//
// Return a list of ledger reports that are not associated with a policy
//
// ---
// responses:
//   '200':
//     description: LedgerReport list
//     schema:
//       type: array
//       items:
//         "$ref": "#/definitions/LedgerReport"
func ledgerReportList(c buffalo.Context) error {
	var list models.LedgerReports

	tx := models.Tx(c)
	if err := list.AllNonPolicy(tx); err != nil {
		return reportError(c, err)
	}

	return renderOk(c, list.ConvertToAPI(tx))
}

// swagger:operation GET /ledger-reports/{id} LedgerReport LedgerReportView
//
// LedgerReportView
//
// Return the ledger report specified by `id`. The returned object contains metadata and a File object pointing to
// a CSV file suitable for use with Sage Accounting.
//
// ---
// parameters:
// - name: id
//   in: path
//   required: true
//   description: specifies the ID of the report to view
// responses:
//   '200':
//     description: the requested LedgerReport
//     schema:
//       type: array
//       items:
//         "$ref": "#/definitions/LedgerReport"
func ledgerReportView(c buffalo.Context) error {
	tx := models.Tx(c)

	ledgerReport := getReferencedLedgerReportFromCtx(c)
	return renderOk(c, ledgerReport.ConvertToAPI(tx))
}

// swagger:operation POST /ledger-reports LedgerReport LedgerReportCreate
//
// LedgerReportCreate
//
// Create and return a report on the ledger entries as specified by the input object. The returned object
// contains metadata and a File object pointing to a CSV file suitable for use with Sage Accounting.
//
// ### Report types:
// + `monthly` - Return all ledger entries not yet reconciled, up to the beginning of the given day (0:00 UTC).
// + `annual` - Return the billing detail for given year's policy renewals.
//
// ---
// parameters:
//   - name: input
//     in: body
//     description: LedgerReportCreateInput object
//     required: true
//     schema:
//       "$ref": "#/definitions/LedgerReportCreateInput"
// responses:
//   '200':
//     description: the requested LedgerReport
//     schema:
//       type: array
//       items:
//         "$ref": "#/definitions/LedgerReport"
func ledgerReportCreate(c buffalo.Context) error {
	var input api.LedgerReportCreateInput
	if err := StrictBind(c, &input); err != nil {
		return reportError(c, err)
	}

	date, err := time.Parse(domain.DateFormat, input.Date)
	if err != nil {
		return reportError(c, api.NewAppError(err, api.ErrorInvalidDate, api.CategoryUser))
	}

	report, err := models.NewLedgerReport(c, input.Type, date)
	if err != nil {
		return reportError(c, err)
	}

	tx := models.Tx(c)

	if err = report.Create(tx); err != nil {
		return reportError(c, err)
	}

	return renderOk(c, report.ConvertToAPI(tx))
}

// swagger:operation PUT /ledger-reports/{id} LedgerReport LedgerReportReconcile
//
// LedgerReportReconcile
//
// Mark ledger entries in the report reconciled as of today. Call this only after all transactions in the report
// have been fully loaded into the accounting record.
//
// ---
// parameters:
// - name: id
//   in: path
//   required: true
//   description: specifies the ID of the report to reconcile
// responses:
//   '200':
//     description: the requested LedgerReport
//     schema:
//       type: array
//       items:
//         "$ref": "#/definitions/LedgerReport"
func ledgerReportReconcile(c buffalo.Context) error {
	tx := models.Tx(c)

	ledgerReport := getReferencedLedgerReportFromCtx(c)
	if err := ledgerReport.Reconcile(c); err != nil {
		return reportError(c, err)
	}

	return renderOk(c, ledgerReport.ConvertToAPI(tx))
}

// swagger:operation POST /ledger-reports/annual Ledger LedgerAnnualProcess
//
// LedgerAnnualProcess
//
// Process billing for current year's policy renewals.
//
// ---
// responses:
//   '204':
//     description: OK but no content in response
func ledgerAnnualProcess(c buffalo.Context) error {
	actor := models.CurrentUser(c)
	if !actor.IsAdmin() {
		err := fmt.Errorf("user not allowed to process annual batch data")
		return reportError(c, api.NewAppError(err, api.ErrorNotAuthorized, api.CategoryForbidden))
	}

	tx := models.Tx(c)

	currentYear := time.Now().UTC().Year()

	var policies models.Policies
	if err := policies.AllActive(tx); err != nil {
		return reportError(c, err)
	}
	if err := policies.ProcessAnnualCoverage(tx, currentYear); err != nil {
		return reportError(c, err)
	}

	return c.Render(http.StatusNoContent, nil)
}

func getReferencedLedgerReportFromCtx(c buffalo.Context) *models.LedgerReport {
	lr, ok := c.Value(domain.TypeLedgerReport).(*models.LedgerReport)
	if !ok {
		panic("LedgerReport not found in context")
	}
	return lr
}
