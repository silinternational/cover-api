package messages

import (
	"fmt"
	"html/template"

	"github.com/gobuffalo/nulls"
	"github.com/gobuffalo/pop/v5"

	"github.com/silinternational/cover-api/domain"
	"github.com/silinternational/cover-api/models"
)

// PolicyUserInviteQueueMessage queues messages to an invited policy user
func PolicyUserInviteQueueMessage(tx *pop.Connection, invite models.PolicyUserInvite) {
	data := newEmailMessageData()
	data["policyName"] = invite.Policy.Name
	data["acceptURL"] = invite.GetAcceptURL()
	data["inviterEmail"] = invite.InviterEmail
	data["inviterName"] = invite.InviterName
	data["inviteeName"] = invite.InviteeName

	invite.LoadPolicy(tx, false)
	data["policy"] = invite.Policy

	data["emailIntro"] = template.HTML(domain.Env.UserWelcomeEmailIntro) // #nosec G203

	data.addStewardData(tx)

	notn := models.Notification{
		PolicyID: nulls.NewUUID(invite.PolicyID),
		Body:     data.renderHTML(MessageTemplatePolicyUserInvite),
		Subject:  fmt.Sprintf("Invitation to %s policy on %s", invite.Policy.Name, domain.Env.AppName),

		// TODO make these constants somewhere
		Event:         "Policy User Invite Notification",
		EventCategory: "PolicyUserInvite",
	}
	if err := notn.Create(tx); err != nil {
		panic("error creating new Policy User Invite Notification: " + err.Error())
	}

	notn.CreateNotificationUser(tx, nulls.UUID{}, invite.Email, invite.InviteeName)
}
