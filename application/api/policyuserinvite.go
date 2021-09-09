package api

// PolicyUserInviteCreate
//
// input model for creating policy user invites
//
// swagger:model
type PolicyUserInviteCreate struct {
	// user's email address
	//
	// required: true
	Email string `json:"email"`

	// A personal message from inviter to include in invite email
	InviterMessage string `json:"inviter_message"`
}
