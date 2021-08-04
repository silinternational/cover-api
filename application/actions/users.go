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
	if user := getUserFromCxt(c); user != nil {
		return c.Render(http.StatusOK, r.JSON(user))
	}
	return c.Render(http.StatusNotFound, nil)
}

func getUserFromCxt(c buffalo.Context) *models.User {
	user, ok := c.Value(domain.TypeUser).(models.User)
	if !ok {
		return nil
	}
	return &user
}

func usersMe(c buffalo.Context) error {
	if user := getUserFromCxt(c); user != nil {
		return c.Render(http.StatusOK, r.JSON(user))
	}

	return c.Render(http.StatusUnauthorized, nil)
}
