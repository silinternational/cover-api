package actions

import (
	"github.com/gobuffalo/buffalo"

	"github.com/silinternational/cover-api/api"
)

// swagger:operation GET /config/claim-event-types Config ClaimEventTypes
//
// ClaimEventTypes
//
// list all valid Claim Event Types
//
// ---
// responses:
//   '200':
//     description: list of valid Claim Event Types
//     schema:
//       type: array
//       items:
//         "$ref": "#/definitions/ClaimEventTypeStruct"
func claimEventTypes(c buffalo.Context) error {
	return renderOk(c, api.AllClaimEventTypes)
}
