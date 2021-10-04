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
		err := fmt.Errorf("actor not allowed to perform that action on this resource")
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

	response := c.Response()
	response.Header().Set("Content-Type", "text/csv")
	fileHeader := fmt.Sprintf(`attachment; filename="%s.csv"`, firstDay.Format("2006-01"))
	response.Header().Set("Content-Disposition", fileHeader)
	_, err := response.Write(csvData)

	return err
}
