package messages

import (
	"github.com/gobuffalo/nulls"
	"github.com/gobuffalo/pop/v5"

	"github.com/silinternational/cover-api/models"
)

func itemApprovedQueueMsg(tx *pop.Connection, item models.Item) {
	data := newEmailMessageData()
	data.addItemData(tx, item)

	notn := models.Notification{
		ItemID:        nulls.NewUUID(item.ID),
		Body:          data.renderHTML(MessageTemplateItemApprovedMember),
		Subject:       "Item Coverage Approved",
		InappText:     "Item Coverage Approved",
		Event:         "Item Approved Notification",
		EventCategory: EventCategoryItem,
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

	notn := models.Notification{
		ItemID:        nulls.NewUUID(item.ID),
		Body:          data.renderHTML(MessageTemplateItemAutoSteward),
		Subject:       "Item has been auto approved: " + item.Name,
		InappText:     "Coverage on a new policy item was just auto approved",
		Event:         "Item Auto Approved Notification",
		EventCategory: EventCategoryItem,
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

	notn := models.Notification{
		ItemID:        nulls.NewUUID(item.ID),
		Body:          data.renderHTML(MessageTemplateItemPendingSteward),
		Subject:       "Item Needs Review " + item.Name,
		InappText:     "A new policy item is waiting for your approval",
		Event:         "Item Pending Notification",
		EventCategory: EventCategoryItem,
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

	notn := models.Notification{
		ItemID:        nulls.NewUUID(item.ID),
		Body:          data.renderHTML(MessageTemplateItemRevisionMember),
		Subject:       "Coverage Needs Attention",
		InappText:     "Coverage needs attention",
		Event:         "Item Revision Required Notification",
		EventCategory: EventCategoryItem,
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

	notn := models.Notification{
		ItemID:        nulls.NewUUID(item.ID),
		Body:          data.renderHTML(MessageTemplateItemDeniedMember),
		Subject:       "An Update on Your Coverage Request",
		InappText:     "coverage on your new policy item has been denied",
		Event:         "Item Denied Notification",
		EventCategory: EventCategoryItem,
	}
	if err := notn.Create(tx); err != nil {
		panic("error creating new Item Denied Notification: " + err.Error())
	}

	for _, m := range item.Policy.Members {
		notn.CreateNotificationUserForUser(tx, m)
	}
}
