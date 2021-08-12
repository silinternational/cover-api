package models

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"mime"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/gobuffalo/pop/v5"
	"github.com/gobuffalo/validate/v3"
	"github.com/gobuffalo/validate/v3/validators"
	"github.com/gofrs/uuid"
	"github.com/silinternational/riskman-api/api"
	_ "golang.org/x/image/webp" // enable decoding of WEBP images

	"github.com/silinternational/riskman-api/aws"
	"github.com/silinternational/riskman-api/domain"
)

type FileUploadError struct {
	HttpStatus int
	ErrorCode  api.ErrorKey
	Message    string
}

func (f *FileUploadError) Error() string {
	return fmt.Sprintf("%d: %s ... %s", f.HttpStatus, f.ErrorCode, f.Message)
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

// String can be helpful for serializing the model
func (f File) String() string {
	jf, _ := json.Marshal(f)
	return string(jf)
}

// Files is merely for convenience and brevity
type Files []File

// String can be helpful for serializing the model
func (f Files) String() string {
	jf, _ := json.Marshal(f)
	return string(jf)
}

// Validate gets run every time you call a "pop.Validate*" (pop.ValidateAndSave, pop.ValidateAndCreate, pop.ValidateAndUpdate) method.
func (f *File) Validate(tx *pop.Connection) (*validate.Errors, error) {
	return validate.Validate(
		&validators.UUIDIsPresent{Field: f.UUID, Name: "UUID"},
	), nil
}

// ValidateCreate gets run every time you call "pop.ValidateAndCreate" method.
func (f *File) ValidateCreate(tx *pop.Connection) (*validate.Errors, error) {
	return validate.NewErrors(), nil
}

// ValidateUpdate gets run every time you call "pop.ValidateAndUpdate" method.
func (f *File) ValidateUpdate(tx *pop.Connection) (*validate.Errors, error) {
	return validate.NewErrors(), nil
}

// Store takes a byte slice and stores it into S3 and saves the metadata in the database file table.
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

// removeMetadata removes, if possible, all EXIF metadata by re-encoding the image. If the encoding type changes,
// `contentType` will be modified accordingly.
func (f *File) removeMetadata() {
	img, _, err := image.Decode(bytes.NewReader(f.Content))
	if err != nil {
		return
	}
	buf := new(bytes.Buffer)
	switch f.ContentType {
	case "image/jpg":
		if err := jpeg.Encode(buf, img, nil); err == nil {
			f.Content = buf.Bytes()
		}
	case "image/gif":
		if err := gif.Encode(buf, img, nil); err == nil {
			f.Content = buf.Bytes()
		}
	case "image/png":
		if err := png.Encode(buf, img); err == nil {
			f.Content = buf.Bytes()
		}
	case "image/webp":
		if err := png.Encode(buf, img); err == nil {
			f.Content = buf.Bytes()
			f.ContentType = "image/png"
		}
	}
}

// changeFileExtension attempts to make the file extension match the given content type
func (f *File) changeFileExtension() {
	ext, err := mime.ExtensionsByType(f.ContentType)
	if err != nil || len(ext) < 1 {
		return
	}
	f.Name = strings.TrimSuffix(f.Name, filepath.Ext(f.Name)) + ext[0]
}

// FindByUUID locates a file by UUID and returns the result, including a valid URL.
// None of the struct members of f are used as input, but are updated if the function is successful.
func (f *File) FindByUUID(tx *pop.Connection, fileUUID string) error {
	var file File
	if err := tx.Where("uuid = ?", fileUUID).First(&file); err != nil {
		return err
	}

	if err := file.RefreshURL(tx); err != nil {
		return err
	}

	*f = file
	return nil
}

// RefreshURL ensures the file URL is good for at least a few minutes
func (f *File) RefreshURL(tx *pop.Connection) error {
	if f.URLExpiration.After(time.Now().Add(time.Minute * 5)) {
		return nil
	}

	newURL, err := aws.GetFileURL(f.UUID.String())
	if err != nil {
		return err
	}
	f.URL = newURL.Url
	f.URLExpiration = newURL.Expiration
	if err = f.Update(tx); err != nil {
		return err
	}
	return nil
}

func validateContentType(content []byte) (string, error) {
	detectedType := http.DetectContentType(content)
	if domain.IsStringInSlice(detectedType, domain.AllowedFileUploadTypes) {
		return detectedType, nil
	}
	return "", fmt.Errorf("invalid file type %s", detectedType)
}

// Create stores the File data as a new record in the database.
func (f *File) Create(tx *pop.Connection) error {
	return create(tx, f)
}

// Update writes the File data to an existing database record.
func (f *File) Update(tx *pop.Connection) error {
	return update(tx, f)
}

// DeleteUnlinked removes all files that are no longer linked to any database records
func (f *Files) DeleteUnlinked(tx *pop.Connection) error {
	var files Files
	if err := tx.Select("id", "uuid").
		Where("linked = FALSE AND updated_at < ?", time.Now().Add(-4*domain.DurationWeek)).
		All(&files); err != nil {
		return err
	}
	domain.Logger.Printf("unlinked files: %d", len(files))
	if len(files) > domain.Env.MaxFileDelete {
		return fmt.Errorf("attempted to delete too many files, MaxFileDelete=%d", domain.Env.MaxFileDelete)
	}
	if len(files) == 0 {
		return nil
	}

	nRemovedFromDB := 0
	nRemovedFromS3 := 0
	for _, file := range files {
		if err := aws.RemoveFile(file.UUID.String()); err != nil {
			domain.ErrLogger.Printf("error removing from S3, id='%s', %s", file.UUID.String(), err)
			continue
		}
		nRemovedFromS3++

		f := file
		if err := tx.Destroy(&f); err != nil {
			domain.ErrLogger.Printf("file %d destroy error, %s", file.ID, err)
			continue
		}
		nRemovedFromDB++
	}

	if nRemovedFromDB < len(files) || nRemovedFromS3 < len(files) {
		domain.ErrLogger.Printf("not all unlinked files were removed")
	}
	domain.Logger.Printf("removed %d from S3, %d from file table", nRemovedFromS3, nRemovedFromDB)
	return nil
}

// SetLinked marks the file as linked. If already linked, return an error since it may be attempting to link a file to
// multiple records.
// The File struct need not be hydrated; only the ID is needed.
func (f *File) SetLinked(tx *pop.Connection) error {
	if err := tx.Reload(f); err != nil {
		return fmt.Errorf("failed to load file for setting linked flag, %w", err)
	}
	if f.Linked {
		return fmt.Errorf("cannot link file, it is already linked")
	}
	f.Linked = true
	return tx.UpdateColumns(f, "linked", "updated_at")
}

// ClearLinked marks the file as unlinked. The struct need not be hydrated; only the ID is needed.
func (f *File) ClearLinked(tx *pop.Connection) error {
	f.Linked = false
	return tx.UpdateColumns(f, "linked", "updated_at")
}

// FindByIDs finds all Files associated with the given IDs and loads them from the database
func (f *Files) FindByIDs(tx *pop.Connection, ids []int) error {
	return tx.Where("id in (?)", ids).All(f)
}

// convertFile converts a models.File to an api.File
func convertFile(file File) api.File {
	return api.File{
		ID:            file.UUID,
		URL:           file.URL,
		URLExpiration: file.URLExpiration,
		Name:          file.Name,
		Size:          file.Size,
		ContentType:   file.ContentType,
	}
}
