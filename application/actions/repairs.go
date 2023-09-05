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

// swagger:operation POST /repairs Repairs RepairsRun
//
// RepairsRun
//
// Run a repair
//
// ### Repair types:
// + `renewal` - Repair all items that were incorrectly renewed and billed for another year of coverage. Also issues premium refunds for the incorrect renewal charges.
//
// ---
//	parameters:
//	  - name: input
//	    in: body
//	    description: parameters for the Repair Run
//	    required: true
//	    schema:
//	      "$ref": "#/definitions/RepairRunInput"
//	responses:
//	  '200':
//	    description: the repair result
//	    schema:
//	      "$ref": "#/definitions/RepairResult"
func repairsRun(c buffalo.Context) error {
	actor := models.CurrentUser(c)
	if !actor.IsAdmin() {
		err := fmt.Errorf("user not allowed to run repair")
		return reportError(c, api.NewAppError(err, api.ErrorNotAuthorized, api.CategoryForbidden))
	}

	var input api.RepairRunInput
	if err := StrictBind(c, &input); err != nil {
		return reportError(c, err)
	}

	date, err := time.Parse(domain.DateFormat, input.Date)
	if err != nil {
		return reportError(c, api.NewAppError(err, api.ErrorInvalidDate, api.CategoryUser))
	}

	var result api.RepairResult

	switch input.RepairType {
	case api.RepairTypeRenewal:
		result, err = runRenewalRepair(c, date)

	default:
		err = errors.New("unrecognized repair type, must be " + api.RepairTypeRenewal)
		return reportError(c, api.NewAppError(err, api.ErrorUnrecognizedRepairType, api.CategoryUser))
	}

	if err != nil {
		return reportError(c, err)
	}
	return renderOk(c, result)
}

func runRenewalRepair(c buffalo.Context, date time.Time) (api.RepairResult, error) {
	tx := models.Tx(c)

	var result api.RepairResult
	var items models.Items
	if err := items.RepairItemsIncorrectlyRenewed(c, date); err != nil {
		return result, err
	}

	if len(items) > MaxResultSize {
		err := errors.New("too many rows in the result set")
		return result, api.NewAppError(err, api.ErrorTooManyRows, api.CategoryInternal)
	}

	result.Items = items.ConvertToAPI(tx)
	result.RepairType = api.RepairTypeRenewal
	return result, nil
}
