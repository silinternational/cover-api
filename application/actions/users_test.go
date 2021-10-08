package actions

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/gofrs/uuid"

	"github.com/silinternational/cover-api/api"
	"github.com/silinternational/cover-api/domain"
	"github.com/silinternational/cover-api/models"
)

func (as *ActionSuite) Test_usersMe() {
	db := as.DB

	f := models.CreateUserFixtures(db, 2)
	userWithPhoto := f.Users[0]
	userNoPhoto := f.Users[1]

	fileFixtures := models.CreateFileFixtures(db, 1, userWithPhoto.ID)
	fileID := fileFixtures.Files[0].ID

	as.NoError(userWithPhoto.AttachPhotoFile(as.DB, fileID))

	tests := []struct {
		name        string
		userID      string
		token       string
		user        models.User
		wantPhotoID uuid.UUID
		wantStatus  int
		wantInBody  []string
	}{
		{
			name:       "Unauthenticated",
			token:      "doesnt-exist",
			user:       models.User{},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "User without Photo",
			token:      userNoPhoto.Email,
			user:       userNoPhoto,
			wantStatus: http.StatusOK,
			wantInBody: []string{
				`{"id":"` + userNoPhoto.ID.String(),
				`"email":"` + userNoPhoto.Email,
				`"first_name":"` + userNoPhoto.FirstName,
				`"last_name":"` + userNoPhoto.LastName,
				`"app_role":"` + string(userNoPhoto.AppRole),
				`"last_login_utc":"` + userNoPhoto.LastLoginUTC.Format(domain.DateFormat),
			},
		},
		{
			name:        "User with Photo",
			token:       userWithPhoto.Email,
			user:        userWithPhoto,
			wantPhotoID: fileID,
			wantStatus:  http.StatusOK,
			wantInBody: []string{
				`{"id":"` + userWithPhoto.ID.String(),
				`"email":"` + userWithPhoto.Email,
				`"first_name":"` + userWithPhoto.FirstName,
				`"last_name":"` + userWithPhoto.LastName,
				`"last_login_utc":"` + userWithPhoto.LastLoginUTC.Format(domain.DateFormat),
			},
		},
	}

	for _, tt := range tests {
		as.T().Run(tt.name, func(t *testing.T) {
			req := as.JSON("/users/me")
			req.Headers["Authorization"] = fmt.Sprintf("Bearer %s", tt.token)

			res := req.Get()

			as.Require().Equal(tt.wantStatus, res.Code, "incorrect status code returned: %d", res.Code)
			if tt.wantStatus != http.StatusOK {
				return
			}

			body := res.Body.String()

			if tt.wantPhotoID != uuid.Nil {
				want := `"photo_file":{"id":"` + tt.wantPhotoID.String()
				as.Contains(body, want, "didn't get the photo file in the response")
				want = `"photo_file_id":"` + tt.wantPhotoID.String()
				as.Contains(body, want, "didn't get the photo ID in the response")
			}

			as.verifyResponseData(tt.wantInBody, body, "Users Me fields")
		})
	}
}

func (as *ActionSuite) Test_UsersMeUpdate() {
	db := as.DB
	f := models.CreateUserFixtures(db, 3)
	userAddEmail := f.Users[0]
	userAddLocation := f.Users[1]
	userAddBoth := f.Users[2]

	inputAddEmail := api.UserInput{EmailOverride: "new_email0@example.org"}
	inputAddLocation := api.UserInput{Location: "New York, NY"}
	inputAddBoth := api.UserInput{EmailOverride: "new_email2@example.org", Location: "Tucson, AZ"}

	tests := []struct {
		name       string
		actor      models.User
		oldUser    models.User
		input      api.UserInput
		wantStatus int
		wantInBody []string
	}{
		{
			name:       "unauthenticated",
			actor:      models.User{},
			oldUser:    userAddEmail,
			input:      inputAddEmail,
			wantStatus: http.StatusUnauthorized,
			wantInBody: []string{
				api.ErrorNotAuthorized.String(),
				"no bearer token provided",
			},
		},
		{
			name:       "just add email",
			actor:      userAddEmail,
			oldUser:    userAddEmail,
			input:      inputAddEmail,
			wantStatus: http.StatusOK,
			wantInBody: []string{
				`"first_name":"` + userAddEmail.FirstName,
				`"email_override":"` + inputAddEmail.EmailOverride,
			},
		},
		{
			name:       "just location",
			actor:      userAddLocation,
			oldUser:    userAddLocation,
			input:      inputAddLocation,
			wantStatus: http.StatusOK,
			wantInBody: []string{
				`"first_name":"` + userAddLocation.FirstName,
				`"location":"` + inputAddLocation.EmailOverride,
			},
		},
		{
			name:       "just add email",
			actor:      userAddBoth,
			oldUser:    userAddBoth,
			input:      inputAddBoth,
			wantStatus: http.StatusOK,
			wantInBody: []string{
				`"first_name":"` + userAddBoth.FirstName,
				`"email_override":"` + inputAddBoth.EmailOverride,
				`"location":"` + inputAddBoth.Location,
			},
		},
	}

	for _, tt := range tests {
		as.T().Run(tt.name, func(t *testing.T) {
			req := as.JSON("/users/me")
			req.Headers["Authorization"] = fmt.Sprintf("Bearer %s", tt.actor.Email)
			req.Headers["content-type"] = "application/json"
			res := req.Put(tt.input)

			body := res.Body.String()
			as.Equal(tt.wantStatus, res.Code, "incorrect status code returned, body: %s", body)

			as.verifyResponseData(tt.wantInBody, body, "")

			if res.Code != http.StatusOK {
				return
			}

			var apiUser api.User
			err := json.Unmarshal([]byte(body), &apiUser)
			as.NoError(err)

			var user models.User
			as.NoError(as.DB.Where(`first_name = ?`, tt.oldUser.FirstName).First(&user),
				"error finding newly updated user.")
			as.Equal(tt.oldUser.LastName, user.LastName, "incorrect LastName")
			as.Equal(tt.input.EmailOverride, user.EmailOverride, "incorrect EmailOverride")
			as.Equal(tt.input.Location, user.Location, "incorrect Location")
		})
	}
}
