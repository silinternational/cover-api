package listeners

import (
	"fmt"

	"github.com/gobuffalo/events"

	"github.com/silinternational/cover-api/domain"
	"github.com/silinternational/cover-api/models"
	"github.com/silinternational/cover-api/notifications"
)

// TODO consider sending different email contents depending on auto-approval
func itemSubmitted(e events.Event) {
	if e.Kind != domain.EventApiItemSubmitted {
		return
	}

	defer panicRecover(e.Kind)

	var item models.Item
	if err := findObject(e.Payload, &item, e.Kind); err != nil {
		return
	}

	var steward models.User
	steward.FindSteward(models.DB)

	item.LoadPolicyMembers(models.DB, false)
	memberName := item.Policy.Members[0].Name()

	msg := notifications.Message{
		Template:  domain.MessageTemplateItemSubmitted,
		ToName:    steward.Name(),
		ToEmail:   steward.Email,
		FromEmail: domain.EmailFromAddress(nil),
		Subject:   "a new policy item was submitted for approval",
		Data: map[string]interface{}{
			"appName":        domain.Env.AppName,
			"uiURL":          domain.Env.UIURL,
			"itemMemberName": memberName,
			"itemURL":        fmt.Sprintf("%s/items/%s", domain.Env.UIURL, item.ID),
			"itemName":       item.Name,
		},
	}
	if err := notifications.Send(msg); err != nil {
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

	// TODO Notify item creator and do whatever else needs doing
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
