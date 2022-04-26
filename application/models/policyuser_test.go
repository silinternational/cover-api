package models

import (
	"time"

	"github.com/gobuffalo/nulls"

	"github.com/silinternational/cover-api/domain"
)

func (ms *ModelSuite) TestPolicyUser_Delete() {
	invite := PolicyUserInvite{
		ID:           domain.GetUUID(),
		PolicyID:     domain.GetUUID(),
		Email:        "new_user@example.org",
		InviteeName:  "New User",
		InviterName:  "Sam Supervisor",
		InviterEmail: "Join us!",
	}

	// First test with no EmailSentAt
	got := invite.ConvertToAPI()

	ms.Equal(invite.Email, got.Email, "Email is not correct")
	ms.Equal(invite.InviteeName, got.Name, "Name is not correct")
	ms.Nil(got.EmailSentAt, "EmailSentAt is not null")

	// Second test with EmailSenttAt
	now := time.Now().UTC()
	invite.EmailSentAt = nulls.NewTime(now)
	got = invite.ConvertToAPI()

	ms.Equal(now, *got.EmailSentAt, "EmailSentAt is not correct")
}
