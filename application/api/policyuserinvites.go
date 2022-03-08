package api

import "time"

// swagger:model
type PolicyUserInvites []PolicyUserInvite

// swagger:model
type PolicyUserInvite struct {
	// invitee's email
	Email string `json:"email"`

	// invitee's name
	Name string `json:"name"`

	// date and time when invite email was sent (omitted if empty)
	//
	// swagger:strfmt date-time
	EmailSentAt *time.Time `json:"email_sent_at,omitempty"`
}
