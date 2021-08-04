package actions

import (
	"encoding/json"
	"net/http"

	"github.com/gobuffalo/buffalo"
	"github.com/silinternational/riskman-api/api"
)

func registerCustomErrorHandler(app *buffalo.App) {
	app.ErrorHandlers[http.StatusInternalServerError] = customErrorHandler
}

func customErrorHandler(status int, origErr error, c buffalo.Context) error {
	c.Logger().Error(origErr)
	c.Response().WriteHeader(status)
	c.Response().Header().Set("content-type", "application/json")

	appError := api.AppError{
		HttpStatus: status,
		Key:        api.ErrorGenericInternalServer,
		DebugMsg:   origErr.Error(),
		Message:    "An internal system error has occurred",
	}
	err := json.NewEncoder(c.Response()).Encode(&appError)
	return err
}
