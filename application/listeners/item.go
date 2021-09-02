package listeners

import (
	"fmt"

	"github.com/gobuffalo/events"

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

func newItemMessageForSteward(item models.Item) notifications.Message {
	msg := notifications.NewEmailMessage().AddToSteward()
	addMessageItemData(&msg, item)
	return msg
}

func newItemMessageForMember(item models.Item, member models.User) notifications.Message {
	msg := notifications.NewEmailMessage()
	addMessageItemData(&msg, item)
	msg.ToName = member.Name()
	msg.ToEmail = member.EmailOfChoice()
	msg.Data["itemMemberName"] = member.Name()

	return msg
}

func itemSubmitted(e events.Event) {
	if e.Kind != domain.EventApiItemSubmitted {
		return
	}

	defer panicRecover(e.Kind)

	var item models.Item
	if err := findObject(e.Payload, &item, e.Kind); err != nil {
		return
	}

	item.LoadPolicyMembers(models.DB, false)
	memberName := item.Policy.Members[0].Name()

	notifiers := getNotifiersFromEventPayload(e.Payload)

	if item.CoverageStatus == api.ItemCoverageStatusApproved {
		notifyItemApprovedMember(item, notifiers)
		notifyItemAutoApprovedSteward(item, memberName, notifiers)
	} else { // Was submitted but not auto approved
		notifyItemSubmitted(item, memberName, notifiers)
	}
}

func notifyItemApprovedMember(item models.Item, notifiers []interface{}) {
	for _, m := range item.Policy.Members {
		msg := newItemMessageForMember(item, m)
		msg.Template = domain.MessageTemplateItemApprovedMember
		msg.Subject = "your new policy item has been approved"
		if err := notifications.Send(msg, notifiers...); err != nil {
			domain.ErrLogger.Printf("error sending item auto approved notification to member, %s", err)
		}
	}
}

func notifyItemAutoApprovedSteward(item models.Item, memberName string, notifiers []interface{}) {
	msg := newItemMessageForSteward(item)
	msg.Template = domain.MessageTemplateItemAutoSteward
	msg.Subject = memberName + " just submitted a new policy item that has been auto approved"
	msg.Data["itemMemberName"] = memberName

	if err := notifications.Send(msg, notifiers...); err != nil {
		domain.ErrLogger.Printf("error sending item auto approved notification to steward, %s", err)
	}
}

func notifyItemSubmitted(item models.Item, memberName string, notifiers []interface{}) {
	msg := newItemMessageForSteward(item)
	msg.Template = domain.MessageTemplateItemSubmittedSteward
	msg.Subject = "Action Required. " + memberName + " just submitted a new policy item for approval"
	msg.Data["itemMemberName"] = memberName

	if err := notifications.Send(msg, notifiers...); err != nil {
		domain.ErrLogger.Printf("error sending item submitted notification, %s", err)
	}
}

func itemRevision(e events.Event) {
	if e.Kind != domain.EventApiItemRevision {
		return
	}

	defer panicRecover(e.Kind)

	var item models.Item
	if err := findObject(e.Payload, &item, e.Kind); err != nil {
		return
	}

	// TODO Notify item creator and do whatever else needs doing
}

func itemApproved(e events.Event) {
	if e.Kind != domain.EventApiItemApproved {
		return
	}

	defer panicRecover(e.Kind)

	var item models.Item
	if err := findObject(e.Payload, &item, e.Kind); err != nil {
		return
	}

	item.LoadPolicyMembers(models.DB, false)

	notifiers := getNotifiersFromEventPayload(e.Payload)
	notifyItemApprovedMember(item, notifiers)
	// TODO do whatever else needs doing
}

func itemDenied(e events.Event) {
	if e.Kind != domain.EventApiItemDenied {
		return
	}

	defer panicRecover(e.Kind)

	var item models.Item
	if err := findObject(e.Payload, &item, e.Kind); err != nil {
		return
	}

	// TODO Notify item creator and do whatever else needs doing
}
