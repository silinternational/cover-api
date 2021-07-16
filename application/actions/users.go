package actions

import (
	"net/http"

	"github.com/gobuffalo/buffalo"
)

func usersList(c buffalo.Context) error {
	return c.Render(http.StatusOK, r.JSON(map[string]string{"users": "This is just a stub"}))
}
