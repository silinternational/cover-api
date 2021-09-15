package api

import (
	"time"

	"github.com/gobuffalo/nulls"
	"github.com/gofrs/uuid"
)

// swagger:model
type Users []User

// app user
// swagger:model
type User struct {
	// unique ID
	//
	// swagger:strfmt uuid4
	ID uuid.UUID `json:"id"`

	// email address
	Email string `json:"email"`

	// email address
	EmailOverride string `json:"email_override,omitempty"`

	// first name
	FirstName string `json:"first_name"`

	// last name
	LastName string `json:"last_name"`

	// full name
	Name string `json:"name"`

	// role in the application ('user', 'steward', 'signator')
	AppRole string `json:"app_role"`

	// last login date and time (UTC)
	LastLoginUTC time.Time `json:"last_login_utc"`

	// country or something similar
	Location string `json:"location,omitempty"`

	// policy ID (temporary, will be replaced with a list of policies)
	// swagger:strfmt uuid4
	PolicyID nulls.UUID `json:"policy_id"` // TODO: provide either a list of IDs or a list of Policies

	// unique id (uuid) for a avatar or photo file
	//
	// swagger:strfmt uuid4
	// example: 63d5b060-1460-4348-bdf0-ad03c105a8d5
	PhotoFileID nulls.UUID `json:"photo_file_id"`

	// File object that contains the user's photo
	PhotoFile *File `json:"photo_file,omitempty"`
}
