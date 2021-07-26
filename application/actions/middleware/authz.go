package middleware

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gobuffalo/buffalo"

	"github.com/silinternational/riskman-api/domain"
	"github.com/silinternational/riskman-api/models"
)

var authableResources = map[string]models.Authable{
	"user": &models.User{},
	//"policy": &models.Policy{},
	//"item":   &models.Item{},
	//"claim":  &models.Claim{},
}

func AuthZ(next buffalo.Handler) buffalo.Handler {
	return func(c buffalo.Context) error {
		user, ok := c.Value(domain.ContextKeyCurrentUser).(models.User)
		if !ok {
			return c.Error(http.StatusUnauthorized, fmt.Errorf("user must be authenticated to proceed"))
		}

		pathParts := strings.Split(strings.TrimLeft(c.Request().URL.Path, "/"), "/")

		var resource models.Authable
		var isAuthable bool
		if resource, isAuthable = authableResources[pathParts[0]]; !isAuthable {
			return c.Error(http.StatusInternalServerError, fmt.Errorf("resource expected to be authable but isn't"))
		}

		id := c.Param("id")
		if id != "" {
			tx := models.Tx(c)
			if tx == nil {
				return c.Error(http.StatusInternalServerError, fmt.Errorf("failed to intialize db connection"))
			}
			if err := resource.FindByID(tx, id); err != nil {
			}
		}

		var p models.Permission

		switch c.Request().Method {
		case http.MethodGet:
			p = models.PermissionList
			if id != "" {
				p = models.PermissionView
			}
		case http.MethodPost:
			p = models.PermissionCreate
		case http.MethodPut:
			p = models.PermissionUpdate
		case http.MethodDelete:
			p = models.PermissionDelete
		default:
			p = models.PermissionDenied
		}

		if !resource.IsUserAllowedTo(user, p, c.Request()) {
			return c.Error(http.StatusForbidden, fmt.Errorf("user not allowed to perform that action on this resource"))
		}

		return next(c)
	}
}
