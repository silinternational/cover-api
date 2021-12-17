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

// swagger:operation GET /ledger Ledger LedgerList
//
// LedgerList
//
// return the ledger entries not yet reconciled, up to the beginning of the current day (0:00 UTC)
//
// ---
// responses:
//   '200':
//     description: the ledger entries not yet reconciled, in CSV format suitable for use with Sage Accounting
//     content:
//       text/csv:
//         schema:
//           type: string
//           format: text
func ledgerList(c buffalo.Context) error {
	actor := models.CurrentUser(c)
	if !actor.IsAdmin() {
		err := fmt.Errorf("user not allowed to get monthly batch data")
		return reportError(c, api.NewAppError(err, api.ErrorNotAuthorized, api.CategoryForbidden))
	}

	tx := models.Tx(c)

	now := time.Now().UTC()
	firstDay := domain.BeginningOfLastMonth(now)
	var le models.LedgerEntries
	if err := le.AllForMonth(tx, firstDay); err != nil {
		return err
	}

	if len(le) == 0 {
		return c.Render(http.StatusNoContent, nil)
	}

	csvData := le.ToCsv(firstDay)
	filename := fmt.Sprintf("batch_%s.csv", firstDay.Format("2006-01"))

	return renderCsv(c, filename, csvData)
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
// responses:
//   '200':
//     description: batch approval confirmation details
//     schema:
//       "$ref": "#/definitions/BatchApproveResponse"
func ledgerReconcile(c buffalo.Context) error {
	actor := models.CurrentUser(c)
	if !actor.IsAdmin() {
		err := fmt.Errorf("user not allowed to approve monthly batch data")
		return reportError(c, api.NewAppError(err, api.ErrorNotAuthorized, api.CategoryForbidden))
	}

	tx := models.Tx(c)

	now := time.Now().UTC()
	firstDay := domain.BeginningOfLastMonth(now)
	var le models.LedgerEntries
	if err := le.AllForMonth(tx, firstDay); err != nil {
		return err
	}

	if err := le.Reconcile(c); err != nil {
		return reportError(c, err)
	}

	return renderOk(c, api.BatchApproveResponse{NumberOfRecordsApproved: len(le)})
}

// swagger:operation GET /ledger/annual Ledger LedgerAnnual
//
// LedgerAnnual
//
// Get the billing detail for current year's policy renewals
//
// ---
// responses:
//   '200':
//     description: the current year policy renewal ledger entries
//     content:
//       text/csv:
//         schema:
//           type: string
//           format: text
func ledgerAnnual(c buffalo.Context) error {
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

	var le models.LedgerEntries
	if err := le.FindCurrentRenewals(tx, currentYear); err != nil {
		return reportError(c, err)
	}

	if len(le) == 0 {
		return c.Render(http.StatusNoContent, nil)
	}

	date := time.Date(currentYear, 1, 1, 0, 0, 0, 0, time.UTC)
	csvData := le.ToCsv(date)
	filename := fmt.Sprintf("renewal_%d.csv", currentYear)
	return renderCsv(c, filename, csvData)
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
