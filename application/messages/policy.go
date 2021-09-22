package messages

import (
	"github.com/gobuffalo/nulls"
	"github.com/gobuffalo/pop/v5"

	"github.com/silinternational/cover-api/models"
)

// PolicyUserInviteQueueMessage queues messages to an invited policy user
func PolicyUserInviteQueueMessage(tx *pop.Connection, invite models.PolicyUserInvite) {

	data := newEmailMessageData()
	data["policyName"] = invite.Policy.CostCenter // TODO update with policy name once added to model for corporate policies
	data["acceptURL"] = invite.GetAcceptURL()
	data["inviterEmail"] = invite.InviterEmail
	data["inviterName"] = invite.InviterName

	notn := models.Notification{
		PolicyID: nulls.NewUUID(invite.PolicyID),
		Body:     data.renderHTML(MessageTemplatePolicyUserInvite),
		Subject:  "Action Required. " + invite.InviterName + " invited you to manage a policy on Cover",

		// TODO make these constants somewhere
		Event:         "Policy User Invite Notification",
		EventCategory: "PolicyUserInvite",
	}
	if err := notn.Create(tx); err != nil {
		panic("error creating new Policy User Invite Notification: " + err.Error())
	}

	notn.CreateNotificationUser(tx, nulls.UUID{}, invite.Email, invite.InviteeName)

}
