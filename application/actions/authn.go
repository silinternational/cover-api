package actions

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/silinternational/cover-api/api"
	"github.com/silinternational/cover-api/domain"
	"github.com/silinternational/cover-api/log"
	"github.com/silinternational/cover-api/models"
)

func AuthN(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		if AuthnSkipper(c) {
			return next(c)
		}

		bearerToken := domain.GetBearerTokenFromRequest(c.Request())
		if bearerToken == "" {
			err := errors.New("no bearer token provided")
			return reportError(c, api.NewAppError(err, api.ErrorNotAuthorized, api.CategoryUnauthorized))
		}

		var userAccessToken models.UserAccessToken
		if err := userAccessToken.FindByBearerToken(models.DB, bearerToken); err != nil {
			err := errors.New("invalid bearer token")
			return reportError(c, api.NewAppError(err, api.ErrorNotAuthorized, api.CategoryUnauthorized))
		}

		isExpired, err := userAccessToken.DeleteIfExpired(models.DB)
		if err != nil {
			return reportError(c, err)
		}

		if isExpired {
			err = errors.New("expired bearer token")
			return reportError(c, api.NewAppError(err, api.ErrorNotAuthorized, api.CategoryUnauthorized))
		}

		user, err := userAccessToken.GetUser(models.DB)
		if err != nil {
			err = fmt.Errorf("error finding user by access token, %s", err.Error())
			return reportError(c, err)
		}
		c.Set(domain.ContextKeyCurrentUser, user)

		// set person on log context
		log.SetUser(c.Request().Context(), user.ID.String(), user.GetName().String(), user.Email)

		return next(c)
	}
}

func AuthnSkipper(c echo.Context) bool {
	if c.Request().Method == http.MethodOptions {
		return true
	}
	skipURLs := map[string]struct{}{
		"/":              {},
		"/auth/callback": {},
		"/auth/login":    {},
		"/auth/logout":   {},
		"/robots.txt":    {},
		"/status":        {},
	}
	if _, ok := skipURLs[c.Path()]; ok {
		return true
	}
	return false
}
