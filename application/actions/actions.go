package actions

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"runtime"

	"github.com/gobuffalo/buffalo"
	"github.com/gobuffalo/buffalo/render"

	"github.com/silinternational/cover-api/api"
	"github.com/silinternational/cover-api/domain"
	"github.com/silinternational/cover-api/models"
)

var r *render.Engine

func init() {
	r = render.New(render.Options{
		DefaultContentType: "application/json",
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
	appErr.Extras["function"] = GetFunctionName(2)
	appErr.Extras["key"] = appErr.Key
	appErr.Extras["status"] = appErr.HttpStatus
	appErr.Extras["redirectURL"] = appErr.RedirectURL
	appErr.Extras["method"] = c.Request().Method
	appErr.Extras["URI"] = c.Request().RequestURI
	appErr.Extras["IP"] = c.Request().RemoteAddr
	domain.Error(c, appErr.Error(), appErr.Extras)

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
			appErr.RedirectURL = domain.Env.UIURL + "/login?appError=" + appErr.Message
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

// GetFunctionName provides the filename, line number, and function name of the caller, skipping the top `skip`
// functions on the stack.
func GetFunctionName(skip int) string {
	pc, file, line, ok := runtime.Caller(skip)
	if !ok {
		return "?"
	}

	fn := runtime.FuncForPC(pc)
	return fmt.Sprintf("%s:%d %s", file, line, fn.Name())
}

func renderOk(c buffalo.Context, v interface{}) error {
	convertable, ok := v.(models.Convertable)
	if ok {
		return c.Render(http.StatusOK, r.JSON(convertable.ConvertToAPI(models.Tx(c))))
	}

	// Return an error to protect against rendering model data directly
	return c.Render(http.StatusInternalServerError, nil)
}
