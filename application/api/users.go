package api

import (
	"time"

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

	// role in the application ('User', 'Steward', 'Signator', 'Admin')
	AppRole string `json:"app_role"`

	// last login date and time (UTC)
	LastLoginUTC time.Time `json:"last_login_utc"`

	// country where user is located
	Country string `json:"country,omitempty"`

	// country code (ISO-3166 alpha-3) where user is located
	CountryCode string `json:"country_code"`

	// all policies in which the user is a member
	Policies Policies `json:"policies,omitempty"`

	// unique id (uuid) for a avatar or photo file
	//
	// swagger:strfmt uuid4
	// example: 63d5b060-1460-4348-bdf0-ad03c105a8d5
	PhotoFileID *uuid.UUID `json:"photo_file_id"`

	// File object that contains the user's photo
	PhotoFile *File `json:"photo_file,omitempty"`
}

// app user update input
// swagger:model
type UserInput struct {
	// email address
	EmailOverride string `json:"email_override,omitempty"`

	// country
	Country string `json:"country,omitempty"`
}

// swagger:model
type UserFileAttachInput struct {
	// File ID to attach to the current user
	//
	// swagger:strfmt uuid4
	FileID uuid.UUID `json:"file_id"`
}
