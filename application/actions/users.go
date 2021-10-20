package actions

import (
	"github.com/gobuffalo/buffalo"

	"github.com/silinternational/cover-api/api"
	"github.com/silinternational/cover-api/domain"
	"github.com/silinternational/cover-api/models"
)

// swagger:operation GET /users Users UsersList
//
// UsersList
//
// gets the data for all Users.
//
// ---
// responses:
//   '200':
//     description: all users
//     schema:
//       type: array
//       items:
//         "$ref": "#/definitions/User"
func usersList(c buffalo.Context) error {
	var users models.Users
	tx := models.Tx(c)
	if err := users.GetAll(tx); err != nil {
		if domain.IsOtherThanNoRows(err) {
			return reportError(c, err)
		}
		return reportError(c, api.NewAppError(err, api.ErrorNoRows, api.CategoryNotFound))
	}
	return renderOk(c, users.ConvertToAPI(tx))
}

// swagger:operation GET /users/{id} Users UsersView
//
// UsersView
//
// gets the data for a specific User.
//
// ---
// parameters:
//   - name: id
//     in: path
//     required: true
//     description: user ID
// responses:
//   '200':
//     description: a user
//     schema:
//       "$ref": "#/definitions/User"
func usersView(c buffalo.Context) error {
	user := getReferencedUserFromCtx(c)
	return renderUser(c, *user)
}

// swagger:operation GET /users/me Users UsersMe
//
// UsersMe
//
// gets the data for authenticated User.
//
// ---
// responses:
//   '200':
//     description: authenticated user
//     schema:
//       "$ref": "#/definitions/User"
func usersMe(c buffalo.Context) error {
	return renderUser(c, models.CurrentUser(c))
}

// swagger:operation PUT /users/me Users UserMeUpdate
//
// UserMeUpdate
//
// update the current user's personal settings
//
// ---
// parameters:
//   - name: user's settings input
//     in: body
//     description: the editable settings for a user
//     required: true
//     schema:
//       "$ref": "#/definitions/UserInput"
// responses:
//   '200':
//     description: updated User
//     schema:
//       "$ref": "#/definitions/User"
func usersMeUpdate(c buffalo.Context) error {
	tx := models.Tx(c)
	user := models.CurrentUser(c)

	var input api.UserInput
	if err := StrictBind(c, &input); err != nil {
		return reportError(c, err)
	}

	user.EmailOverride = input.EmailOverride

	if input.Country != "" {
		user.Country = input.Country
	}

	if err := user.Update(tx); err != nil {
		return reportError(c, err)
	}

	return renderUser(c, user)
}

// swagger:operation POST /users/me/files Users UsersMeFileAttach
//
// UsersMeFileAttach
//
// attach a File to the current user
//
// ---
// parameters:
//   - name: user file input
//     in: body
//     description: photo/avatar to attach to the current user
//     required: true
//     schema:
//       "$ref": "#/definitions/UserFileAttachInput"
// responses:
//   '200':
//     description: the User
//     schema:
//       "$ref": "#/definitions/User"
func usersMeFilesAttach(c buffalo.Context) error {
	var input api.UserFileAttachInput
	if err := StrictBind(c, &input); err != nil {
		return reportError(c, err)
	}

	tx := models.Tx(c)

	user := models.CurrentUser(c)
	if err := user.AttachPhotoFile(tx, input.FileID); err != nil {
		return reportError(c, err)
	}

	return renderUser(c, user)
}

func renderUser(c buffalo.Context, user models.User) error {
	tx := models.Tx(c)
	user.LoadPolicies(tx, false)
	return renderOk(c, user.ConvertToAPI(tx))
}

// getReferencedUserFromCtx pulls the models.User resource from context that was put there
// by the AuthZ middleware based on a url pattern of /users/{id}. This is NOT the authenticated
// API caller
func getReferencedUserFromCtx(c buffalo.Context) *models.User {
	user, ok := c.Value(domain.TypeUser).(*models.User)
	if !ok {
		panic("user not found in context")
	}
	return user
}
