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

// swagger:operation GET /batches/latest Batches BatchesLatest
//
// BatchesLatest
//
// return the latest batch of ledger entries
//
// ---
// responses:
//   '200':
//     description: the latest batch of ledger entries
//     content:
//       text/csv:
//         schema:
//           type: string
//           format: text
func batchesGetLatest(c buffalo.Context) error {
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

// swagger:operation POST /batches/approve Batches BatchesApprove
//
// BatchesApprove
//
// Mark the last batch as accepted. Call this only after the recent batch has
// been fully loaded into the accounting record.
//
// ---
// responses:
//   '200':
//     description: batch approval confirmation details
//     schema:
//       "$ref": "#/definitions/BatchApproveResponse"
func batchesApprove(c buffalo.Context) error {
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

// swagger:operation GET /batches/annual Batches BatchesAnnual
//
// BatchesAnnual
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
func batchesAnnual(c buffalo.Context) error {
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
