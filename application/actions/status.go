package actions

import (
	"net/http"

	"github.com/gobuffalo/buffalo"
)

// swagger:operation GET /status Status Status
// Status
//
// checks the app status
// ---
//
//	responses:
//	  '204':
//	    description: app status is good
func statusHandler(c buffalo.Context) error {
	return c.Render(http.StatusNoContent, nil)
}
