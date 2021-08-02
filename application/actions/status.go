package actions

import (
	"net/http"

	"github.com/gobuffalo/buffalo"
)

func statusHandler(c buffalo.Context) error {
	return c.Render(http.StatusNoContent, nil)
}
