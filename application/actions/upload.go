package actions

import (
	"github.com/labstack/echo/v4"
)

// fileFieldName is the multipart field name for the file upload.
const fileFieldName = "file"

// UploadResponse is a JSON response for the /upload endpoint
// swagger:model
type UploadResponse struct {
	Name        string `json:"filename,omitempty"`
	ID          string `json:"id,omitempty"`
	URL         string `json:"url,omitempty"`
	ContentType string `json:"content_type,omitempty"`
	Size        int    `json:"size,omitempty"`
}

// swagger:operation POST /upload Files UploadFile
// UploadFile
//
// Upload a new File object
// ---
//
//	consumes:
//	  - multipart/form-data
//	parameters:
//	  - name: file
//	    in: formData
//	    type: file
//	    description: file object
//	responses:
//	  '200':
//	    description: uploaded File data
//	    schema:
//	      "$ref": "#/definitions/UploadResponse"
func uploadHandler(c echo.Context) error {
	// FIXME
	//f, err := c.File(fileFieldName)
	//if err != nil {
	//	err := fmt.Errorf("error getting uploaded file from context ... %v", err)
	//	return reportError(c, api.NewAppError(err, api.ErrorReceivingFile, api.CategoryInternal))
	//}
	//
	//if f.Size > int64(domain.MaxFileSize) {
	//	err := fmt.Errorf("file upload size (%v) greater than max (%v)", f.Size, domain.MaxFileSize)
	//	return reportError(c, api.NewAppError(err, api.ErrorStoreFileTooLarge, api.CategoryUser))
	//}
	//
	//content, err := ioutil.ReadAll(f)
	//if err != nil {
	//	err := fmt.Errorf("error reading uploaded file ... %v", err)
	//	return reportError(c, api.NewAppError(err, api.ErrorUnableToReadFile, api.CategoryInternal))
	//}
	//
	//fileObject := models.File{
	//	Name:        f.Filename,
	//	Content:     content,
	//	CreatedByID: models.CurrentUser(c).ID,
	//}
	//if fErr := fileObject.Store(models.Tx(c).Request().Context()); fErr != nil {
	//	return reportError(c, err)
	//}
	//
	//resp := UploadResponse{
	//	Name:        fileObject.Name,
	//	ID:          fileObject.ID.String(),
	//	URL:         fileObject.URL,
	//	ContentType: fileObject.ContentType,
	//	Size:        fileObject.Size,
	//}
	//
	//return c.Render(200, render.JSON(resp))

	return c.JSON(200, nil)
}
