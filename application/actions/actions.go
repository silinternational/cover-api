package actions

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/gobuffalo/buffalo"
	"github.com/gobuffalo/buffalo/render"
	"github.com/gofrs/uuid"
	"github.com/rollbar/rollbar-go"

	"github.com/silinternational/cover-api/api"
	"github.com/silinternational/cover-api/domain"
)

var r *render.Engine

func init() {
	r = render.New(render.Options{
		DefaultContentType: domain.ContentJson,
	})

	checkSamlConfig()
}

// StrictBind hydrates a struct with values from a POST
// REMEMBER the request body must have *exported* fields.
//  Otherwise, this will give an empty result without an error.
func StrictBind(c buffalo.Context, dest interface{}) error {
	dec := json.NewDecoder(c.Request().Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dest); err != nil {
		return api.NewAppError(err, api.ErrorInvalidRequestBody, api.CategoryUser)
	}
	return nil
}

// reportError logs an error with details and renders the error with buffalo.Render.
// If the HTTP status code provided is in the 300 family, buffalo.Redirect is used instead.
func reportError(c buffalo.Context, err error) error {
	var appErr *api.AppError
	if !errors.As(err, &appErr) {
		appErr = appErrorFromErr(err)
	}
	appErr.SetHttpStatusFromCategory()

	if appErr.Extras == nil {
		appErr.Extras = map[string]interface{}{}
	}

	appErr.Extras = domain.MergeExtras([]map[string]interface{}{getExtras(c), appErr.Extras})
	appErr.Extras["function"] = domain.GetFunctionName(2)
	appErr.Extras["key"] = appErr.Key
	appErr.Extras["status"] = appErr.HttpStatus
	appErr.Extras["redirectURL"] = appErr.RedirectURL
	appErr.Extras["method"] = c.Request().Method
	appErr.Extras["URI"] = c.Request().RequestURI
	appErr.Extras["IP"] = c.Request().RemoteAddr

	level := rollbar.ERR
	switch appErr.Category {
	case api.CategoryUnauthorized, api.CategoryUser:
		level = rollbar.WARN
	}
	domain.LogErrorMessage(c, appErr.Error(), level, appErr.Extras)

	appErr.LoadTranslatedMessage(c)

	// clear out debugging info if not in development or test
	if domain.Env.GoEnv == "development" || domain.Env.GoEnv == "test" {
		if appErr.Err != nil {
			appErr.DebugMsg = appErr.Err.Error()
		}
	} else {
		appErr.Extras = map[string]interface{}{}
	}

	if appErr.HttpStatus >= 300 && appErr.HttpStatus <= 399 {
		if appErr.RedirectURL == "" {
			appErr.RedirectURL = domain.Env.UIURL + "/logged-out?appError=" + appErr.Message
		}
		return c.Redirect(appErr.HttpStatus, appErr.RedirectURL)
	}
	return c.Render(appErr.HttpStatus, r.JSON(appErr))
}

// reportErrorAndClearSession logs an error with details, clears the session, and renders the error with buffalo.Render.
// If the HTTP status code provided is in the 300 family, buffalo.Redirect is used instead.
func reportErrorAndClearSession(c buffalo.Context, err error) error {
	c.Session().Clear()
	return reportError(c, err)
}

func appErrorFromErr(err error) *api.AppError {
	return &api.AppError{
		HttpStatus: http.StatusInternalServerError,
		Key:        api.ErrorUnknown,
		DebugMsg:   err.Error(),
	}
}

func getExtras(c buffalo.Context) map[string]interface{} {
	extras, _ := c.Value(domain.ContextKeyExtras).(map[string]interface{})
	if extras == nil {
		extras = map[string]interface{}{}
	}
	return extras
}

func newExtra(c buffalo.Context, key string, e interface{}) {
	extras := getExtras(c)
	extras[key] = e
	c.Set(domain.ContextKeyExtras, extras)
}

func renderOk(c buffalo.Context, v interface{}) error {
	return c.Render(http.StatusOK, r.JSON(v))
}

func getUUIDFromParam(c buffalo.Context, param string) (uuid.UUID, error) {
	s := c.Param(param)
	id := uuid.FromStringOrNil(s)
	if id == uuid.Nil {
		newExtra(c, param, s)
		err := fmt.Errorf("invalid %s provided: '%s'", param, s)
		return uuid.UUID{}, api.NewAppError(err, api.ErrorMustBeAValidUUID, api.CategoryUser)
	}
	return id, nil
}
