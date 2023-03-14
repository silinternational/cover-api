package actions

import (
	"encoding/json"
	"fmt"
	"net/http"
	"runtime/debug"

	"github.com/gobuffalo/buffalo"

	"github.com/silinternational/cover-api/api"
	"github.com/silinternational/cover-api/domain"
)

var httpErrorCodes = map[int]api.ErrorKey{
	http.StatusBadRequest:          api.ErrorBadRequest,
	http.StatusUnauthorized:        api.ErrorNotAuthenticated,
	http.StatusNotFound:            api.ErrorRouteNotFound,
	http.StatusMethodNotAllowed:    api.ErrorMethodNotAllowed,
	http.StatusConflict:            api.ErrorConflict,
	http.StatusUnprocessableEntity: api.ErrorUnprocessableEntity,
}

func registerCustomErrorHandler(app *buffalo.App) {
	for i := 400; i < 600; i++ {
		app.ErrorHandlers[i] = customErrorHandler
	}
}

func customErrorHandler(status int, origErr error, c buffalo.Context) error {
	c.Logger().Error(origErr)
	c.Response().WriteHeader(status)
	c.Response().Header().Set("content-type", domain.ContentJson)

	if domain.Env.GoEnv == domain.EnvDevelopment {
		debug.PrintStack()
	}

	appError := api.AppError{
		HttpStatus: status,
		Key:        getErrorCodeFromStatus(status),
		DebugMsg:   fmt.Sprintf("(%T) %s", origErr, origErr),
	}
	err := json.NewEncoder(c.Response()).Encode(&appError)
	return err
}

func getErrorCodeFromStatus(status int) api.ErrorKey {
	if s, ok := httpErrorCodes[status]; ok {
		return s
	}
	return api.ErrorGenericInternalServer
}
