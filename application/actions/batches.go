package actions

import (
	"net/http"
	"time"

	"github.com/gobuffalo/buffalo"

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
	tx := models.Tx(c)

	now := time.Date(2021, 07, 01, 0, 0, 0, 0, time.UTC)
	// now := time.Now().UTC()
	today := now.Truncate(time.Hour * 24)
	firstDay := domain.BeginningOfLastMonth(today)
	var le models.LedgerEntries
	if err := le.FindBatch(tx, firstDay); err != nil {
		return err
	}

	if len(le) == 0 {
		return c.Render(http.StatusNoContent, nil)
	}

	csvData, err := le.ToCsv(firstDay)
	if err != nil {
		return reportError(c, err)
	}

	response := c.Response()
	response.Header().Set("Content-Type", "text/csv")
	response.Header().Set("Content-Disposition", `attachment; filename="batch.csv"`)
	_, err = response.Write(csvData)

	return err
}
