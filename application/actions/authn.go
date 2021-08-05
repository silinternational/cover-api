package actions

import (
	"errors"
	"fmt"

	"github.com/gobuffalo/buffalo"

	"github.com/silinternational/riskman-api/api"
	"github.com/silinternational/riskman-api/domain"
	"github.com/silinternational/riskman-api/models"
)

func AuthN(next buffalo.Handler) buffalo.Handler {
	return func(c buffalo.Context) error {
		bearerToken := domain.GetBearerTokenFromRequest(c.Request())
		if bearerToken == "" {
			err := errors.New("no Bearer token provided")
			return reportError(c, api.NewAppError(err, api.ErrorNotAuthorized, api.CategoryUnauthorized))
		}

		var userAccessToken models.UserAccessToken
		tx := models.Tx(c)
		err := userAccessToken.FindByBearerToken(tx, bearerToken)
		if err != nil {
			if domain.IsOtherThanNoRows(err) {
				return reportError(c, err)
			}
			err = errors.New("invalid bearer token")
			return reportError(c, api.NewAppError(err, api.ErrorNotAuthorized, api.CategoryUnauthorized))
		}

		isExpired, err := userAccessToken.DeleteIfExpired(tx)
		if err != nil {
			return reportError(c, err)
		}

		if isExpired {
			err = errors.New("expired bearer token")
			return reportError(c, api.NewAppError(err, api.ErrorNotAuthorized, api.CategoryUnauthorized))
		}

		user, err := userAccessToken.GetUser(tx)
		if err != nil {
			err = fmt.Errorf("error finding user by access token, %s", err.Error())
			return reportError(c, err)
		}
		c.Set(domain.ContextKeyCurrentUser, user)

		// set person on rollbar session
		domain.RollbarSetPerson(c, user.ID.String(), user.FirstName, user.LastName, user.Email)
		// msg := fmt.Sprintf("user %s authenticated with bearer token from ip %s", user.Email, c.Request().RemoteAddr)
		domain.NewExtra(c, "user_id", user.ID)
		domain.NewExtra(c, "email", user.Email)
		domain.NewExtra(c, "ip", c.Request().RemoteAddr)
		// domain.Info(c, msg)

		return next(c)
	}
}
