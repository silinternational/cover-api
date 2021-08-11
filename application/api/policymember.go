package api

import (
	"time"

	"github.com/gofrs/uuid"
)

// swagger:model
type PolicyMembers []PolicyMember

// swagger:model
type PolicyMember struct {
	// unique ID
	//
	// swagger:strfmt uuid4
	ID uuid.UUID `json:"id"`

	// first name
	FirstName string `json:"first_name"`

	// last name
	LastName string `json:"last_name"`

	// email address
	Email string `json:"email"`

	// last login time (UTC)
	//
	// swagger:strfmt date-time
	LastLoginUTC time.Time `json:"last_login_utc"`
}
