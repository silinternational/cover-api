package actions

import (
	"encoding/json"
	"fmt"
	"net/http"
	"runtime/debug"

	"github.com/gobuffalo/buffalo"

	"github.com/silinternational/riskman-api/api"
	"github.com/silinternational/riskman-api/domain"
)

func registerCustomErrorHandler(app *buffalo.App) {
	app.ErrorHandlers[http.StatusInternalServerError] = customErrorHandler
}

func customErrorHandler(status int, origErr error, c buffalo.Context) error {
	c.Logger().Error(origErr)
	c.Response().WriteHeader(status)
	c.Response().Header().Set("content-type", "application/json")

	if domain.Env.GoEnv == "development" {
		debug.PrintStack()
	}

	appError := api.AppError{
		HttpStatus: status,
		Key:        api.ErrorGenericInternalServer,
		DebugMsg:   fmt.Sprintf("(%T) %s", origErr, origErr),
		Message:    "An internal system error has occurred",
	}
	err := json.NewEncoder(c.Response()).Encode(&appError)
	return err
}
