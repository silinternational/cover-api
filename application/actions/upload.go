package actions

import (
	"fmt"
	"io/ioutil"

	"github.com/silinternational/riskman-api/api"
	"github.com/silinternational/riskman-api/domain"

	"github.com/gobuffalo/buffalo"
	"github.com/gobuffalo/buffalo/render"
	"github.com/silinternational/riskman-api/models"
)

// fileFieldName is the multipart field name for the file upload.
const fileFieldName = "file"

// UploadResponse is a JSON response for the /upload endpoint
type UploadResponse struct {
	Name        string `json:"filename,omitempty"`
	UUID        string `json:"id,omitempty"`
	URL         string `json:"url,omitempty"`
	ContentType string `json:"content_type,omitempty"`
	Size        int    `json:"size,omitempty"`
}

// uploadHandler responds to POST requests at /upload
func uploadHandler(c buffalo.Context) error {
	f, err := c.File(fileFieldName)
	if err != nil {
		err := fmt.Errorf("error getting uploaded file from context ... %v", err)
		return reportError(c, api.NewAppError(err, api.ErrorReceivingFile, api.CategoryInternal))
	}

	if f.Size > int64(domain.MaxFileSize) {
		err := fmt.Errorf("file upload size (%v) greater than max (%v)", f.Size, domain.MaxFileSize)
		return reportError(c, api.NewAppError(err, api.ErrorStoreFileTooLarge, api.CategoryUser))
	}

	content, err := ioutil.ReadAll(f)
	if err != nil {
		err := fmt.Errorf("error reading uploaded file ... %v", err)
		return reportError(c, api.NewAppError(err, api.ErrorUnableToReadFile, api.CategoryInternal))
	}

	fileObject := models.File{
		Name:    f.Filename,
		Content: content,
	}
	if fErr := fileObject.Store(models.Tx(c)); fErr != nil {
		domain.Error(c, fmt.Sprintf("error storing uploaded file ... %v", fErr))
		return c.Render(fErr.HttpStatus, render.JSON(api.AppError{
			HttpStatus: fErr.HttpStatus,
			Message:  fErr.Message,
		}))
	}

	resp := UploadResponse{
		Name:        fileObject.Name,
		UUID:        fileObject.UUID.String(),
		URL:         fileObject.URL,
		ContentType: fileObject.ContentType,
		Size:        fileObject.Size,
	}

	return c.Render(200, render.JSON(resp))
}