package api

import (
	"time"

	"github.com/gobuffalo/nulls"
	"github.com/gofrs/uuid"
)

type PolicyUserInvite struct {
	// invite ID
	//
	// required: true
	// read only: true
	// swagger:strfmt uuid4
	ID uuid.UUID `json:"id"`

	// unique id (uuid) for the policy being invited to co-manage
	//
	// required: true
	// swagger:strfmt uuid4
	PolicyID uuid.UUID `json:"policy_id"`

	// user's email address
	//
	// required: true
	Email string `json:"email"`

	// Time last invite email was sent
	//
	// read-only: true
	// example: 2020-10-02T15:00:00Z
	// swagger:strfmt date-time
	EmailSentAt nulls.Time `json:"email_sent_at"`

	// Count of times invite email has been sent
	//
	// read-only: true
	EmailSendCount int `json:"email_send_count"`

	// Name of person creating invite
	//
	// required: true
	InviterName string `json:"inviter_name"`

	// Email address of person creating invite
	InviterEmail string `json:"inviter_email"`

	// A personal message from inviter to include in invite email
	InviterMessage string `json:"inviter_message"`

	// Datetime when invite was created
	//
	// read-only: true
	// example: 2020-10-02T15:00:00Z
	CreatedAt time.Time `json:"created_at"`

	// Datetime when invite was last updated
	//
	// read-only: true
	// example: 2020-10-02T15:00:00Z
	UpdatedAt time.Time `json:"updated_at"`
}

type PolicyUserInviteCreate struct {
	// unique id (uuid) for the policy being invited to co-manage
	//
	// required: true
	// swagger:strfmt uuid4
	PolicyID uuid.UUID `json:"policy_id"`

	// user's email address
	//
	// required: true
	Email string `json:"email"`

	// A personal message from inviter to include in invite email
	InviterMessage string `json:"inviter_message"`
}
