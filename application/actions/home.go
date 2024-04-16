package actions

import (
	"fmt"

	"github.com/labstack/echo/v4"

	"github.com/silinternational/cover-api/domain"
)

// HomeHandler is a default handler to serve up
// a home page.
func HomeHandler(c echo.Context) error {
	message := fmt.Sprintf("Welcome to %s API, powered by Echo!", domain.Env.AppName)
	return renderOk(c, map[string]string{"message": message})
}
