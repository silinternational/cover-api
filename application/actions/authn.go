package actions

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gobuffalo/buffalo"

	"github.com/silinternational/riskman-api/domain"
	"github.com/silinternational/riskman-api/models"
)

func AuthN(next buffalo.Handler) buffalo.Handler {
	return func(c buffalo.Context) error {
		bearerToken := domain.GetBearerTokenFromRequest(c.Request())
		if bearerToken == "" {
			return c.Error(http.StatusUnauthorized, errors.New("no Bearer token provided"))
		}

		var userAccessToken models.UserAccessToken
		tx := models.Tx(c)
		err := userAccessToken.FindByBearerToken(tx, bearerToken)
		if err != nil {
			if domain.IsOtherThanNoRows(err) {
				return c.Error(http.StatusInternalServerError, err)
			}
			return c.Error(http.StatusUnauthorized, errors.New("invalid bearer token"))
		}

		isExpired, err := userAccessToken.DeleteIfExpired(tx)
		if err != nil {
			return c.Error(http.StatusInternalServerError, err)
		}

		if isExpired {
			return c.Error(http.StatusUnauthorized, errors.New("expired bearer token"))
		}

		user, err := userAccessToken.GetUser(tx)
		if err != nil {
			return c.Error(http.StatusInternalServerError, fmt.Errorf("error finding user by access token, %s", err.Error()))
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
