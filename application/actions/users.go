package actions

import (
	"net/http"

	"github.com/gobuffalo/buffalo"

	"github.com/silinternational/riskman-api/domain"
	"github.com/silinternational/riskman-api/models"
)

func usersList(c buffalo.Context) error {
	return c.Render(http.StatusOK, r.JSON(map[string]string{"users": "This is just a stub"}))
}

func usersView(c buffalo.Context) error {
	if user := getReferencedUserFromCtx(c); user != nil {
		return c.Render(http.StatusOK, r.JSON(user))
	}
	return c.Render(http.StatusNotFound, nil)
}

// getReferencedUserFromCtx pulls the models.User resource from context that was put there
// by the AuthZ middleware based on a url pattern of /users/{id}. This is NOT the authenticated
// API caller
func getReferencedUserFromCtx(c buffalo.Context) *models.User {
	user, ok := c.Value(domain.TypeUser).(models.User)
	if !ok {
		return nil
	}
	return &user
}

func usersMe(c buffalo.Context) error {
	if user := getReferencedUserFromCtx(c); user != nil {
		return c.Render(http.StatusOK, r.JSON(user))
	}

	return c.Render(http.StatusUnauthorized, nil)
}
