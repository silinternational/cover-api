package actions

import (
	"errors"
	"fmt"

	"github.com/gobuffalo/buffalo"

	"github.com/silinternational/cover-api/api"
	"github.com/silinternational/cover-api/domain"
	"github.com/silinternational/cover-api/log"
	"github.com/silinternational/cover-api/models"
)

func AuthN(next buffalo.Handler) buffalo.Handler {
	return func(c buffalo.Context) error {
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
		log.SetUser(c, user.ID.String(), user.GetName().String(), user.Email)

		return next(c)
	}
}
