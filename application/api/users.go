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

	// first name
	FirstName string `json:"first_name"`

	// last name
	LastName string `json:"last_name"`

	// last login date and time (UTC)
	LastLoginUTC time.Time `json:"last_login_utc"`

	// policy ID (temporary, will be replaced with a list of policies)
	// swagger:strfmt uuid4
	PolicyID nulls.UUID `json:"policy_id"` // TODO: provide either a list of IDs or a list of Policies
}
