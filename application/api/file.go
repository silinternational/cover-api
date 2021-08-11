package api

import (
	"time"

	"github.com/gofrs/uuid"
)

// File metadata for images and other supported file types.
// If the URL expiration time passes, a new query will refresh
// the URL and the URL expiration time.
//
// swagger:model
type File struct {
	// unique identifier for the File object
	//
	// swagger:strfmt uuid4
	// unique: true
	// example: 63d5b060-1460-4348-bdf0-ad03c105a8d5
	ID uuid.UUID `json:"id"`

	// file content can be loaded from the given URL if the expiration time has not passed, limited to 1,024 characters
	URL string `json:"url"`

	// expiration time of the URL, re-issue the API request to get a new URL and expiration time
	URLExpiration time.Time `json:"url_expiration"`

	// filename with extension, limited to 255 characters, e.g. `image.jpg`
	Name string `json:"name"`

	// file size in bytes
	Size int `json:"size"`

	// MIME content type, limited to 255 characters, e.g. 'image/jpeg'
	ContentType string `json:"content_type"`
}
