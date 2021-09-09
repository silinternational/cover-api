package messages

import (
	"github.com/silinternational/cover-api/domain"
	"github.com/silinternational/cover-api/models"
	"github.com/silinternational/cover-api/notifications"
)

func PolicyUserInviteSend(invite models.PolicyUserInvite, notifiers []interface{}) {
	invite.LoadPolicy(models.DB, false)

	msg := notifications.NewEmailMessage()
	msg.ToEmail = invite.Email
	msg.Template = MessageTemplatePolicyUserInvite
	msg.Data["acceptURL"] = invite.GetAcceptURL()
	msg.Data["inviterName"] = invite.InviterName
	msg.Data["inviterEmail"] = invite.InviterEmail
	msg.Data["policyName"] = invite.Policy.CostCenter // TODO update with policy name once added to model for corporate policies
	msg.Subject = "Action Required. " + invite.InviterName + " invited you to manage a policy on Cover"

	if err := notifications.Send(msg, notifiers...); err != nil {
		domain.ErrLogger.Printf("error sending claim review1 notification, %s", err)
	}
}
