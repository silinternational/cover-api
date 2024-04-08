package actions

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"

	"github.com/gobuffalo/buffalo/render"
	"github.com/labstack/echo/v4"

	"github.com/silinternational/cover-api/api"
	"github.com/silinternational/cover-api/domain"
	"github.com/silinternational/cover-api/log"
)

var r = render.New(render.Options{
	DefaultContentType: domain.ContentJson,
})

// StrictBind hydrates a struct with values from a POST
// REMEMBER the request body must have *exported* fields.
// Otherwise, this will give an empty result without an error.
func StrictBind(c echo.Context, dest any) error {
	dec := json.NewDecoder(c.Request().Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dest); err != nil {
		return api.NewAppError(err, api.ErrorInvalidRequestBody, api.CategoryUser)
	}
	return nil
}

// reportError logs an error with details and renders the error with echo.Render.
// If the HTTP status code provided is in the 300 family, echo.Redirect is used instead.
func reportError(c echo.Context, err error) error {
	var appErr *api.AppError
	if !errors.As(err, &appErr) {
		appErr = appErrorFromErr(err)
	}
	appErr.SetHttpStatusFromCategory()

	if appErr.Extras == nil {
		appErr.Extras = map[string]any{}
	}

	appErr.Extras = domain.MergeExtras([]map[string]any{getExtras(c), appErr.Extras})
	appErr.Extras["function"] = domain.GetFunctionName(2)
	appErr.Extras[domain.ExtrasKey] = appErr.Key
	appErr.Extras[domain.ExtrasStatus] = appErr.HttpStatus
	appErr.Extras["redirectURL"] = appErr.RedirectURL
	appErr.Extras[domain.ExtrasMethod] = c.Request().Method
	appErr.Extras[domain.ExtrasURI] = c.Request().RequestURI

	address, _ := getClientIPAddress(c)
	appErr.Extras[domain.ExtrasIP] = address

	entry := log.WithContext(c.Request().Context()).WithFields(appErr.Extras)
	switch appErr.Category {
	case api.CategoryUnauthorized, api.CategoryUser:
		entry.Warning(err)
	default:
		entry.Error(err)
	}

	appErr.LoadTranslatedMessage(c)

	// clear out debugging info if not in development or test
	if domain.Env.GoEnv == domain.EnvDevelopment || domain.Env.GoEnv == domain.EnvTest {
		appErr.DebugMsg = err.Error()
	} else {
		appErr.Extras = map[string]any{}
	}

	if appErr.HttpStatus >= 300 && appErr.HttpStatus <= 399 {
		if appErr.RedirectURL == "" {
			appErr.RedirectURL = domain.Env.UIURL + "/logged-out?appError=" + appErr.Message
		}
		return c.Redirect(appErr.HttpStatus, appErr.RedirectURL)
	}
	return c.JSON(appErr.HttpStatus, appErr)
}

// reportErrorAndClearSession logs an error with details, clears the session, and renders the error with echo.Render.
// If the HTTP status code provided is in the 300 family, echo.Redirect is used instead.
func reportErrorAndClearSession(c echo.Context, err error) error {
	// FIXME
	// c.Session().Clear()
	return reportError(c, err)
}

func appErrorFromErr(err error) *api.AppError {
	return &api.AppError{
		HttpStatus: http.StatusInternalServerError,
		Key:        api.ErrorUnknown,
		DebugMsg:   err.Error(),
	}
}

func getExtras(c echo.Context) map[string]any {
	extras, _ := c.Get(domain.ContextKeyExtras).(map[string]any)
	if extras == nil {
		extras = map[string]any{}
	}
	return extras
}

func newExtra(c echo.Context, key string, e any) {
	extras := getExtras(c)
	extras[key] = e
	c.Set(domain.ContextKeyExtras, extras)
}

func renderOk(c echo.Context, v any) error {
	return c.JSON(http.StatusOK, v)
}

// getClientIPAddress gets the client IP address from CF-Connecting-IP or RemoteAddr
func getClientIPAddress(c echo.Context) (net.IP, error) {
	req := c.Request()

	// https://developers.cloudflare.com/fundamentals/get-started/reference/http-request-headers/#cf-connecting-ip
	if cf := req.Header.Get("CF-Connecting-IP"); cf != "" {
		return net.ParseIP(cf), nil
	}

	ip, _, err := net.SplitHostPort(req.RemoteAddr)
	if err != nil {
		return nil, fmt.Errorf("userip: %q is not IP:port, %w", req.RemoteAddr, err)
	}

	userIP := net.ParseIP(ip)
	if userIP == nil {
		return nil, fmt.Errorf("userip: %q is not a valid IP address, %w", req.RemoteAddr, err)
	}

	return userIP, nil
}

func robots(c echo.Context) error {
	const body = `User-agent: *
Disallow: /
`
	return c.String(http.StatusOK, body)
}
