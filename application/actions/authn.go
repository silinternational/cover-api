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
		accessToken, ok := c.Session().Get(AccessTokenSessionKey).(string)
		if !ok {
			log.Error("failed to retrieve access token from session")
		}

		if accessToken == "" {
			err := errors.New("no access token provided")
			return reportError(c, api.NewAppError(err, api.ErrorNotAuthorized, api.CategoryUnauthorized))
		}

		var userAccessToken models.UserAccessToken
		if err := userAccessToken.FindByAccessToken(models.DB, accessToken); err != nil {
			err := errors.New("invalid access token")
			return reportError(c, api.NewAppError(err, api.ErrorNotAuthorized, api.CategoryUnauthorized))
		}

		isExpired, err := userAccessToken.DeleteIfExpired(models.DB)
		if err != nil {
			return reportError(c, err)
		}

		if isExpired {
			err = errors.New("expired access token")
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
