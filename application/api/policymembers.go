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

	// email address
	EmailOverride string `json:"email_override,omitempty"`

	// last login time (UTC)
	//
	// swagger:strfmt date-time
	LastLoginUTC time.Time `json:"last_login_utc"`

	// a country
	Country string `json:"country,omitempty"`

	// ID of the PolicyUser object that is related to this policy and user
	//
	// swagger:strfmt uuid4
	PolicyUserID uuid.UUID `json:"policy_user_id"`
}
