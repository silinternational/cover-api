package actions

import (
	"github.com/gobuffalo/buffalo"

	"github.com/silinternational/cover-api/models"
)

// swagger:operation GET /config/entity-codes Config EntityCodesList
//
// EntityCodesList
//
// list Entity Codes
//
// ---
// responses:
//   '200':
//     description: list of Entity Codes
//     schema:
//       type: array
//       items:
//         "$ref": "#/definitions/EntityCodes"
func entityCodesList(c buffalo.Context) error {
	tx := models.Tx(c)
	var entityCodes models.EntityCodes
	if err := entityCodes.AllActive(tx); err != nil {
		return reportError(c, err)
	}

	return renderOk(c, entityCodes.ConvertToAPI(tx))
}
