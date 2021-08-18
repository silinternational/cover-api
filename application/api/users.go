package api

import (
	"time"

	"github.com/gobuffalo/nulls"
	"github.com/gofrs/uuid"
)

type Users []User

// swagger:model
type User struct {
	ID           uuid.UUID `json:"id"`
	Email        string    `json:"email"`
	FirstName    string    `json:"first_name"`
	LastName     string    `json:"last_name"`
	LastLoginUTC time.Time `json:"last_login_utc"`

	// unique id (uuid) for a avatar or photo file
	//
	// swagger:strfmt uuid4
	// example: 63d5b060-1460-4348-bdf0-ad03c105a8d5
	PhotoFileID nulls.UUID `json:"photo_file_id"`

	PolicyID nulls.UUID `json:"policy_id"` // TODO: provide either a list of IDs or a list of Policies

	// File object that contains the user's photo
	PhotoFile *File `json:"photo_file,omitempty"`
}
