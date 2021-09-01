package models

import (
	"time"

	"github.com/silinternational/cover-api/api"
)

func (ms *ModelSuite) TestFile_ConvertToAPI() {
	user := CreateUserFixtures(ms.DB, 1).Users[0]
	file := CreateFileFixtures(ms.DB, 1, user.ID).Files[0]

	got := file.ConvertToAPI(ms.DB).(api.File)
	ms.Equal(file.ID, got.ID)
	ms.Equal(file.URL, got.URL)
	ms.Equal(file.URLExpiration, got.URLExpiration)
	ms.WithinDuration(file.URLExpiration, got.URLExpiration, time.Minute)
	ms.Equal(file.Name, got.Name)
	ms.Equal(file.Size, got.Size)
	ms.Equal(file.ContentType, got.ContentType)
	ms.Equal(file.CreatedByID, got.CreatedByID)
}
