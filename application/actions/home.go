package actions

import (
	"fmt"

	"github.com/gobuffalo/buffalo"

	"github.com/silinternational/cover-api/domain"
)

// HomeHandler is a default handler to serve up
// a home page.
func HomeHandler(c buffalo.Context) error {
	message := fmt.Sprintf("Welcome to %s API, powered by Buffalo!", domain.Env.AppName)
	return renderOk(c, map[string]string{"message": message})
}
