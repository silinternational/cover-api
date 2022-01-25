package actions

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gobuffalo/buffalo"

	"github.com/silinternational/cover-api/api"
	"github.com/silinternational/cover-api/domain"
	"github.com/silinternational/cover-api/models"
)

const (
	reportTypeParam   = "report-type"
	reportTypeMonthly = "monthly"
	reportTypeAnnual  = "annual"
)

// swagger:operation GET /ledger Ledger LedgerList
//
// LedgerList
//
// Return the ledger entries as specified by the `report-type` paramater. If `text/csv` is specified in the `Accept`
// header, the response will be in CSV format suitable for use with Sage Accounting
//
// ### Report types:
// + `monthly` - Return all ledger entries not yet reconciled, up to the beginning of the current day (0:00 UTC).
// + `annual` - Return the billing detail for current year's policy renewals.
//
// ---
// parameters:
// - name: report-type
//   in: query
//   required: true
//   description: specifies the report type, which controls which ledger entries are returned
// responses:
//   '200':
//     description: the ledger entries requested
//     schema:
//       type: array
//       items:
//         "$ref": "#/definitions/LedgerEntry"
// produces:
//   - application/json
//   - text/csv
func ledgerList(c buffalo.Context) error {
	tx := models.Tx(c)

	var le models.LedgerEntries
	var date time.Time

	reportType := c.Params().Get(reportTypeParam)
	switch reportType {
	case reportTypeMonthly:
		date = domain.BeginningOfDay(time.Now().UTC())
		if err := le.AllNotEntered(tx, date); err != nil {
			return err
		}

	case reportTypeAnnual:
		currentYear := time.Now().UTC().Year()
		date = time.Date(currentYear, 1, 1, 0, 0, 0, 0, time.UTC)
		if err := le.FindCurrentRenewals(tx, currentYear); err != nil {
			return reportError(c, err)
		}

	default:
		err := errors.New("invalid " + reportTypeParam)
		return reportError(c, api.NewAppError(err, api.ErrorInvalidReportType, api.CategoryUser))
	}

	if domain.IsStringInSlice("text/csv", c.Request().Header["Accept"]) {
		if len(le) == 0 {
			return c.Render(http.StatusNoContent, nil)
		}

		filename := fmt.Sprintf("cover_%s_%s.csv", reportType, date.Format(domain.DateFormat))
		return renderCsv(c, filename, le.ToCsv(date))
	}

	return renderOk(c, le.ConvertToAPI(tx))
}

// swagger:operation POST /ledger Ledger LedgerReconcile
//
// LedgerReconcile
//
// Mark ledger entries as reconciled as of today. Call this only after all transactions returned by
// LedgerList have been fully loaded into the accounting record. Today's transactions
// (entered after 0:00 UTC) are not marked as reconciled.
//
// ---
// parameters:
//   - name: ledger reconcile input
//     in: body
//     description: ledger reconcile input
//     required: true
//     schema:
//       "$ref": "#/definitions/LedgerReconcileInput"
// responses:
//   '200':
//     description: batch approval confirmation details
//     schema:
//       "$ref": "#/definitions/BatchApproveResponse"
func ledgerReconcile(c buffalo.Context) error {
	var input api.LedgerReconcileInput
	if err := StrictBind(c, &input); err != nil {
		return reportError(c, err)
	}

	tx := models.Tx(c)

	date, err := time.Parse(domain.DateFormat, input.EndDate)
	if err != nil {
		return reportError(c, api.NewAppError(err, api.ErrorItemInvalidEndDate, api.CategoryUser))
	}

	var le models.LedgerEntries
	if err := le.AllNotEntered(tx, date); err != nil {
		return err
	}

	if err := le.Reconcile(c); err != nil {
		return reportError(c, err)
	}

	return renderOk(c, api.BatchApproveResponse{NumberOfRecordsApproved: len(le)})
}

// swagger:operation POST /ledger/annual Ledger LedgerAnnualProcess
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

	if err := models.ProcessAnnualCoverage(tx, currentYear); err != nil {
		return reportError(c, err)
	}

	return c.Render(http.StatusNoContent, nil)
}

func renderCsv(c buffalo.Context, filename string, csvData []byte) error {
	response := c.Response()
	response.Header().Set("Content-Type", "text/csv")
	response.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; %s"`, filename))
	_, err := response.Write(csvData)
	if err != nil {
		return err
	}

	return c.Render(http.StatusOK, nil)
}
