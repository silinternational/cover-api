package messages

import (
	"github.com/gobuffalo/nulls"
	"github.com/gobuffalo/pop/v5"

	"github.com/silinternational/cover-api/domain"
	"github.com/silinternational/cover-api/models"
)

func itemApprovedQueueMsg(tx *pop.Connection, item models.Item) {
	data := newEmailMessageData()
	data.addItemData(tx, item)
	data["buttonLabel"] = "Open in " + domain.Env.AppName

	notn := models.Notification{
		ItemID:  nulls.NewUUID(item.ID),
		Body:    data.renderHTML(MessageTemplateItemApprovedMember),
		Subject: "Item Coverage Approved",
		// TODO make this more helpful
		InappText: "Item Coverage Approved",

		// TODO fix these and make them constants somewhere
		Event:         "Item Approved Notification",
		EventCategory: "Item",
	}
	if err := notn.Create(tx); err != nil {
		panic("error creating new Notification: " + err.Error())
	}

	for _, m := range item.Policy.Members {
		notn.CreateNotificationUserForUser(tx, m)
	}
}

func itemAutoApprovedQueueMessage(tx *pop.Connection, item models.Item, member models.User) {
	data := newEmailMessageData()
	data.addItemData(tx, item)
	memberName := member.Name()
	data["memberName"] = memberName
	data["buttonLabel"] = "Open in " + domain.Env.AppName

	notn := models.Notification{
		ItemID:  nulls.NewUUID(item.ID),
		Body:    data.renderHTML(MessageTemplateItemAutoSteward),
		Subject: memberName + " just submitted a new policy item that has been auto approved",

		InappText: "Coverage on a new policy item was just auto approved",

		// TODO make these constants somewhere
		Event:         "Item Auto Approved Notification",
		EventCategory: "Item",
	}
	if err := notn.Create(tx); err != nil {
		panic("error creating new Item Auto Approved Notification: " + err.Error())
	}

	notn.CreateNotificationUsersForStewards(tx)
}

func itemPendingQueueMessage(tx *pop.Connection, item models.Item, member models.User) {
	data := newEmailMessageData()
	data.addItemData(tx, item)
	data["memberName"] = member.Name()
	data["buttonLabel"] = "Open in " + domain.Env.AppName

	notn := models.Notification{
		ItemID:    nulls.NewUUID(item.ID),
		Body:      data.renderHTML(MessageTemplateItemPendingSteward),
		Subject:   "Item Needs Review " + item.Name,
		InappText: "A new policy item is waiting for your approval",

		// TODO make these constants somewhere
		Event:         "Item Pending Notification",
		EventCategory: "Item",
	}
	if err := notn.Create(tx); err != nil {
		panic("error creating new Item Pending Notification: " + err.Error())
	}

	notn.CreateNotificationUsersForStewards(tx)
}

// ItemSubmittedQueueMessage queues messages to the stewards to
//  notify them that an item has been submitted
func ItemSubmittedQueueMessage(tx *pop.Connection, item models.Item) {
	item.LoadPolicyMembers(tx, false)
	itemPendingQueueMessage(tx, item, item.Policy.Members[0])
}

// ItemRevisionQueueMessage queues messages to an item's members to
//  notify them that revisions are required
func ItemRevisionQueueMessage(tx *pop.Connection, item models.Item) {
	item.LoadPolicyMembers(tx, false)

	data := newEmailMessageData()
	data.addItemData(tx, item)
	data["buttonLabel"] = "Change item in " + domain.Env.AppName

	notn := models.Notification{
		ItemID:  nulls.NewUUID(item.ID),
		Body:    data.renderHTML(MessageTemplateItemRevisionMember),
		Subject: "Coverage Needs Attention",
		// TODO make this more helpful
		InappText: "Coverage needs attention",

		// TODO make these constants somewhere
		Event:         "Item Revision Required Notification",
		EventCategory: "Item",
	}
	if err := notn.Create(tx); err != nil {
		panic("error creating new Item Revision Notification: " + err.Error())
	}

	for _, m := range item.Policy.Members {
		notn.CreateNotificationUserForUser(tx, m)
	}
}

// ItemAutoApprovedQueueMessage queues messages to the stewards to
//  notify them that coverage on an item was auto-approved
func ItemAutoApprovedQueueMessage(tx *pop.Connection, item models.Item) {
	item.LoadPolicyMembers(tx, false)
	itemAutoApprovedQueueMessage(tx, item, item.Policy.Members[0])
}

// ItemApprovedQueueMessage queues messages to an item's members to
//  notify them that coverage on their item was approved
func ItemApprovedQueueMessage(tx *pop.Connection, item models.Item) {
	item.LoadPolicyMembers(tx, false)
	itemApprovedQueueMsg(tx, item)
}

// ItemDeniedQueueMessage queues messages to an item's members to
//  notify them that coverage on their item was denied
func ItemDeniedQueueMessage(tx *pop.Connection, item models.Item) {
	item.LoadPolicyMembers(tx, false)

	data := newEmailMessageData()
	data.addItemData(tx, item)
	data["buttonLabel"] = "View in " + domain.Env.AppName

	notn := models.Notification{
		ItemID:    nulls.NewUUID(item.ID),
		Body:      data.renderHTML(MessageTemplateItemDeniedMember),
		Subject:   "An Update on Your Coverage Request",
		InappText: "coverage on your new policy item has been denied",

		// TODO make these constants somewhere
		Event:         "Item Denied Notification",
		EventCategory: "Item",
	}
	if err := notn.Create(tx); err != nil {
		panic("error creating new Item Denied Notification: " + err.Error())
	}

	for _, m := range item.Policy.Members {
		notn.CreateNotificationUserForUser(tx, m)
	}
}
