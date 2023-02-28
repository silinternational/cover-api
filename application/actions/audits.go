package actions

import (
	"errors"
	"fmt"
	"time"

	"github.com/gobuffalo/buffalo"
	"github.com/silinternational/cover-api/api"
	"github.com/silinternational/cover-api/domain"
	"github.com/silinternational/cover-api/models"
)

// MaxResultSize is the maximum number of rows that can be returned from an API call
// TODO: apply this to all API result sets and move this const to actions.go
const MaxResultSize = 1000

// swagger:operation POST /audits Audit AuditRun
//
// AuditRun
//
// Run an audit
//
// ### Audit types:
// + `renewal` - Return all items that were incorrectly renewed and billed for another year of coverage.
//
// ---
// parameters:
//   - name: input
//     in: body
//     description: parameters for the Audit Run
//     required: true
//     schema:
//       "$ref": "#/definitions/AuditRunInput"
// responses:
//   '200':
//     description: the audit result
//     schema:
//       "$ref": "#/definitions/AuditResult"
func auditRun(c buffalo.Context) error {
	actor := models.CurrentUser(c)
	if !actor.IsAdmin() {
		err := fmt.Errorf("user not allowed to run audit")
		return reportError(c, api.NewAppError(err, api.ErrorNotAuthorized, api.CategoryForbidden))
	}

	var input api.AuditRunInput
	if err := StrictBind(c, &input); err != nil {
		return reportError(c, err)
	}

	date, err := time.Parse(domain.DateFormat, input.Date)
	if err != nil {
		return reportError(c, api.NewAppError(err, api.ErrorInvalidDate, api.CategoryUser))
	}

	var result api.AuditResult

	switch input.AuditType {
	case api.AuditTypeRenewal:
		result, err = runRenewalAudit(c, date)

	default:
		err = errors.New("unrecognized audit type, must be " + api.AuditTypeRenewal)
		return reportError(c, api.NewAppError(err, api.ErrorUnrecognizedAuditType, api.CategoryUser))
	}

	if err != nil {
		return reportError(c, err)
	}
	return renderOk(c, result)
}

func runRenewalAudit(c buffalo.Context, date time.Time) (api.AuditResult, error) {
	tx := models.Tx(c)

	var result api.AuditResult
	var items models.Items
	if err := items.FindItemsIncorrectlyRenewed(tx, date); err != nil {
		return result, err
	}

	if len(items) > MaxResultSize {
		err := errors.New("too many rows in the result set")
		return result, api.NewAppError(err, api.ErrorTooManyRows, api.CategoryInternal)
	}

	result.Items = items.ConvertToAPI(tx)
	result.AuditType = api.AuditTypeRenewal
	return result, nil
}
