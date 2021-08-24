package actions

import (
	"github.com/gobuffalo/buffalo"
)

// HomeHandler is a default handler to serve up
// a home page.
func HomeHandler(c buffalo.Context) error {
	return renderOk(c, map[string]string{"message": "Welcome to Cover API, powered by Buffalo!"})
}
