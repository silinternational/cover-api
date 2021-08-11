package actions

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"testing"
	"time"

	"github.com/gobuffalo/nulls"
	"github.com/gofrs/uuid"
	"github.com/silinternational/riskman-api/api"

	"github.com/gobuffalo/buffalo/binding"

	"github.com/silinternational/riskman-api/aws"
	"github.com/silinternational/riskman-api/domain"

	"github.com/gobuffalo/httptest"
	"github.com/silinternational/riskman-api/models"
)


// UploadFixtures is for returning fixtures from Fixtures_Upload
type UploadFixtures struct {
	ClientID    string
	AccessToken string
}

// Fixtures_Upload creates fixtures for the Test_Upload test
func Fixtures_Upload(as *ActionSuite, t *testing.T) UploadFixtures {
	// Load Org test fixtures
	org := &models.Organization{
		AuthConfig: "{}",
		UUID:       domain.GetUUID(),
	}
	err := as.DB.Create(org)
	if err != nil {
		t.Errorf("could not create organization for test, error: %s", err)
		t.FailNow()
	}

	// Load User test fixture
	user := models.User{
		UUID: domain.GetUUID(),
	}

	if err := as.DB.Create(&user); err != nil {
		t.Errorf("could not create test user ... %v", err)
		t.FailNow()
	}

	// Load UserOrganization test fixture
	userOrg := models.UserOrganization{
		OrganizationID: org.ID,
		UserID:         user.ID,
	}

	if err := as.DB.Create(&userOrg); err != nil {
		t.Errorf("could not create test user org ... %v", err)
		t.FailNow()
	}

	clientID := "12345678"
	accessToken := "ABCDEFGHIJKLMONPQRSTUVWXYZ123456"
	hash := models.HashClientIdAccessToken(clientID + accessToken)

	userAccessToken := models.UserAccessToken{
		UserID:             user.ID,
		UserOrganizationID: nulls.NewInt(userOrg.ID),
		AccessToken:        hash,
		ExpiresAt:          time.Now().Add(time.Hour),
	}

	if err := as.DB.Create(&userAccessToken); err != nil {
		t.Errorf("could not create test userAccessToken ... %v", err)
		t.FailNow()
	}

	if err := aws.CreateS3Bucket(); err != nil {
		t.Errorf("failed to create S3 bucket, %s", err)
		t.FailNow()
	}

	return UploadFixtures{
		ClientID:    clientID,
		AccessToken: accessToken,
	}
}

// Test_Upload tests the actions.uploadHandler function
func (as *ActionSuite) Test_Upload() {
	t := as.T()
	fixtures := Fixtures_Upload(as, t)

	type meta struct {
		File binding.File
	}

	const filename = "test.gif"

	f := httptest.File{
		ParamName: fileFieldName,
		FileName:  filename,
		Reader:    bytes.NewReader([]byte("GIF87a")),
	}

	req := as.HTML("/upload")
	req.Headers = map[string]string{
		"Authorization": fmt.Sprintf("Bearer %s", fixtures.ClientID+fixtures.AccessToken),
	}
	resp, err := req.MultiPartPost(&meta{}, f)
	as.NoError(err)
	as.Equal(200, resp.Code, "bad response code, body: \n%s", resp.Body)

	body, err := ioutil.ReadAll(resp.Body)
	as.NoError(err, "error reading response body")

	var u UploadResponse
	err = json.Unmarshal(body, &u)
	as.NoError(err, "error unmarshalling response")

	as.Equal(filename, u.Name)
	fileUUID, err := uuid.FromString(u.UUID)
	as.NoError(err)
	as.True(fileUUID.Version() != 0)
	as.Regexp("^https?", u.URL)
	as.Equal("image/gif", u.ContentType)
	as.Equal(6, u.Size)

	// Test Error

	// File too big
	readerBytes := []byte("GIF87a")
	massive := make([]byte, domain.Megabyte*10) // ten megabytes
	readerBytes = append(readerBytes, massive...)

	f.Reader = bytes.NewReader(readerBytes)
	resp, err = req.MultiPartPost(&meta{}, f)
	as.NoError(err)

	body, err = ioutil.ReadAll(resp.Body)
	as.NoError(err, "error reading response body")

	var appErr api.AppError
	err = json.Unmarshal(body, &appErr)
	as.NoError(err, "error unmarshalling response")

	as.Equal(http.StatusBadRequest, appErr.Code, "incorrect error code")
	as.Equal(api.ErrorStoreFileTooLarge, appErr.Key, "incorrect error code")

	// Bad Content Type
	const badFilename = "test.bad"

	f = httptest.File{
		ParamName: fileFieldName,
		FileName:  badFilename,
		Reader:    bytes.NewReader([]byte("RIFF1111")),
	}

	resp, err = req.MultiPartPost(&meta{}, f)
	as.NoError(err)

	body, err = ioutil.ReadAll(resp.Body)
	as.NoError(err, "error reading response body")

	err = json.Unmarshal(body, &appErr)
	as.NoError(err, "error unmarshalling response")

	as.Equal(http.StatusBadRequest, appErr.Code, "incorrect error code")
	as.Equal(api.ErrorStoreFileBadContentType, appErr.Key, "incorrect error code")
}
