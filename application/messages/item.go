package messages

import (
	"fmt"
	"time"

	"github.com/gobuffalo/nulls"
	"github.com/gobuffalo/pop/v5"

	"github.com/silinternational/cover-api/api"
	"github.com/silinternational/cover-api/domain"
	"github.com/silinternational/cover-api/models"
	"github.com/silinternational/cover-api/notifications"
)

func addMessageItemData(msg *notifications.Message, item models.Item) {
	msg.Data["itemURL"] = fmt.Sprintf("%s/items/%s", domain.Env.UIURL, item.ID)
	msg.Data["itemName"] = item.Name
	return
}

func newItemMessageForMember(item models.Item, member models.User) notifications.Message {
	msg := notifications.NewEmailMessage()
	addMessageItemData(&msg, item)
	msg.ToName = member.Name()
	msg.ToEmail = member.EmailOfChoice()
	msg.Data["memberName"] = member.Name()

	return msg
}

func notifyItemApprovedMember(item models.Item, notifiers []interface{}) {
	for _, m := range item.Policy.Members {
		msg := newItemMessageForMember(item, m)
		msg.Template = MessageTemplateItemApprovedMember
		msg.Subject = "your new policy item has been approved"
		if err := notifications.Send(msg, notifiers...); err != nil {
			domain.ErrLogger.Printf("error sending item auto approved notification to member, %s", err)
		}
	}
}

func notifyItemAutoApprovedSteward(item models.Item, memberName string, notifiers []interface{}) {
	msg := notifications.NewEmailMessage()
	addMessageItemData(&msg, item)
	msg.Template = MessageTemplateItemAutoSteward
	msg.Subject = memberName + " just submitted a new policy item that has been auto approved"
	msg.Data["memberName"] = memberName

	msgs := msg.CopyToStewards()

	for _, m := range msgs {
		if err := notifications.Send(m, notifiers...); err != nil {
			domain.ErrLogger.Printf("error sending item auto approved notification to steward, %s", err)
		}
	}
}

func notifyItemSubmitted(item models.Item, memberName string, notifiers []interface{}) {
	msg := notifications.NewEmailMessage()
	addMessageItemData(&msg, item)
	msg.Template = MessageTemplateItemSubmittedSteward
	msg.Subject = "Action Required. " + memberName + " just submitted a new policy item for approval"
	msg.Data["memberName"] = memberName

	msgs := msg.CopyToStewards()
	for _, m := range msgs {
		if err := notifications.Send(m, notifiers...); err != nil {
			domain.ErrLogger.Printf("error sending item submitted notification, %s", err)
		}
	}
}
func ItemSubmittedSend(item models.Item, notifiers []interface{}) {
	item.LoadPolicyMembers(models.DB, false)
	memberName := item.Policy.Members[0].Name()

	if item.CoverageStatus == api.ItemCoverageStatusApproved {
		notifyItemApprovedMember(item, notifiers)
		notifyItemAutoApprovedSteward(item, memberName, notifiers)
	} else if item.CoverageStatus == api.ItemCoverageStatusPending { // Was submitted but not auto approved
		notifyItemSubmitted(item, memberName, notifiers)
	}
}

func ItemRevisionQueueMessage(tx *pop.Connection, item models.Item) {
	item.LoadPolicyMembers(models.DB, false)

	// TODO figure out how to specify required revisions

	notn := models.Notification{
		ItemID:  nulls.NewUUID(item.ID),
		Subject: "changes have been requested on your new policy item",
		// TODO make this more helpful
		InappText: "changes have been requested on your new policy item",
	}

	data := newEmailMessageData()
	data.addItemData(item)

	for _, m := range item.Policy.Members {
		data["memberName"] = m.Name()

		notn2 := notn
		notn2.Body = data.renderHTML(MessageTemplateItemRevisionMember)
		// TODO Make these constants somewhere
		notn2.Event = "Item Revision Required Notification"
		notn2.EventCategory = "Item"

		if err := notn2.Create(tx); err != nil {
			panic("error creating new Notification: " + err.Error())
		}

		notnUser := models.NotificationUser{
			NotificationID: notn2.ID,
			UserID:         m.ID,
			EmailAddress:   m.EmailOfChoice(),
			SendAfterUTC:   time.Now().UTC(),
		}

		if err := notnUser.Create(tx); err != nil {
			panic("error creating new NotificationUser: " + err.Error())
		}
	}
}

func ItemApprovedSend(item models.Item, notifiers []interface{}) {
	item.LoadPolicyMembers(models.DB, false)
	notifyItemApprovedMember(item, notifiers)
}

func ItemDeniedSend(item models.Item, notifiers []interface{}) {

	item.LoadPolicyMembers(models.DB, false)

	// TODO figure out how to give a reason for the denial

	for _, m := range item.Policy.Members {
		msg := newItemMessageForMember(item, m)
		msg.Template = MessageTemplateItemDeniedMember
		msg.Subject = "coverage on your new policy item has been denied"
		if err := notifications.Send(msg, notifiers...); err != nil {
			domain.ErrLogger.Printf("error sending item denied notification to member, %s", err)
		}
	}
}