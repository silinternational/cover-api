package actions

import (
	"net/http"

	"github.com/labstack/echo/v4"
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
func statusHandler(c echo.Context) error {
	return c.JSON(http.StatusNoContent, nil)
}
