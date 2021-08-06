package actions

import (
	"errors"

	"github.com/gobuffalo/buffalo"

	"github.com/silinternational/riskman-api/api"
	"github.com/silinternational/riskman-api/domain"
	"github.com/silinternational/riskman-api/models"
)

func usersList(c buffalo.Context) error {
	var users models.Users
	if err := users.GetAll(models.Tx(c)); err != nil {
		if domain.IsOtherThanNoRows(err) {
			return reportError(c, err)
		}
		return reportError(c, api.NewAppError(err, api.ErrorNoRows, api.CategoryNotFound))
	}
	return renderOk(c, models.ConvertUsers(users))
}

func usersView(c buffalo.Context) error {
	user := getReferencedUserFromCtx(c)
	if user == nil {
		err := errors.New("user not found in context")
		return reportError(c, api.NewAppError(err, "", api.CategoryInternal))
	}
	return renderUser(c, *user)
}

func usersMe(c buffalo.Context) error {
	return renderUser(c, models.CurrentUser(c))
}

func renderUser(c buffalo.Context, user models.User) error {
	user.LoadPolicies(models.Tx(c), false)
	return renderOk(c, models.ConvertUser(user))
}

// getReferencedUserFromCtx pulls the models.User resource from context that was put there
// by the AuthZ middleware based on a url pattern of /users/{id}. This is NOT the authenticated
// API caller
func getReferencedUserFromCtx(c buffalo.Context) *models.User {
	user, ok := c.Value(domain.TypeUser).(*models.User)
	if !ok {
		return nil
	}
	return user
}
