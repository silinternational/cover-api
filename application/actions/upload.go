package actions

import (
	"fmt"
	"io/ioutil"
	"time"
	"net/http"

	"github.com/silinternational/riskman-api/api"
	"github.com/silinternational/riskman-api/domain"

	"github.com/gofrs/uuid"

	"github.com/gobuffalo/buffalo"
	"github.com/gobuffalo/buffalo/render"
	"github.com/gobuffalo/pop/v5"
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

type FileUploadError struct {
	HttpStatus int
	ErrorCode  api.ErrorKey
	Message    string
}

type File struct {
	ID            int       `json:"-" db:"id"`
	UUID          uuid.UUID `json:"uuid" db:"uuid"`
	URL           string    `json:"url" db:"url"`
	URLExpiration time.Time `json:"url_expiration" db:"url_expiration"`
	Name          string    `json:"name" db:"name"`
	Size          int       `json:"size" db:"size"`
	ContentType   string    `json:"content_type" db:"content_type"`
	Linked        bool      `json:"linked" db:"linked"`
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time `json:"updated_at" db:"updated_at"`
	Content       []byte    `json:"-" db:"-"`
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

	fileObject := File{
		Name:    f.Filename,
		Content: content,
	}
	if fErr := fileObject.store(models.Tx(c)); fErr != nil {
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


// Ported from WeCarry's models/file.go
// Store takes a byte slice and stores it into S3 and saves the metadata in the database file table.\
func (f *File) Store(tx *pop.Connection) *FileUploadError {
	if len(f.Content) > domain.MaxFileSize {
		e := FileUploadError{
			HttpStatus: http.StatusBadRequest,
			ErrorCode:  api.ErrorStoreFileTooLarge,
			Message:    fmt.Sprintf("file too large (%d bytes), max is %d bytes", len(f.Content), domain.MaxFileSize),
		}
		return &e
	}

	contentType, err := validateContentType(f.Content)
	if err != nil {
		e := FileUploadError{
			HttpStatus: http.StatusBadRequest,
			ErrorCode:  api.ErrorStoreFileBadContentType,
			Message:    err.Error(),
		}
		return &e
	}

	f.ContentType = contentType
	f.removeMetadata()
	f.changeFileExtension()

	f.UUID = domain.GetUUID()

	url, err := aws.StoreFile(f.UUID.String(), contentType, f.Content)
	if err != nil {
		e := FileUploadError{
			HttpStatus: http.StatusInternalServerError,
			ErrorCode:  api.ErrorUnableToStoreFile,
			Message:    err.Error(),
		}
		return &e
	}

	f.URL = url.Url
	f.URLExpiration = url.Expiration
	f.Size = len(f.Content)
	if err := f.Create(tx); err != nil {
		e := FileUploadError{
			HttpStatus: http.StatusInternalServerError,
			ErrorCode:  api.ErrorUnableToStoreFile,
			Message:    err.Error(),
		}
		return &e
	}

	return nil
}