package actions

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/silinternational/cover-api/domain"

	"github.com/gofrs/uuid"
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
