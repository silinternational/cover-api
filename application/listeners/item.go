package listeners

import (
	"fmt"

	"github.com/gobuffalo/events"

	"github.com/silinternational/cover-api/api"
	"github.com/silinternational/cover-api/domain"
	"github.com/silinternational/cover-api/messages"
	"github.com/silinternational/cover-api/models"
)

const wrongStatusMsg = "error with %s listener. Object has wrong status: %s"

func itemSubmitted(e events.Event) {
	var item models.Item
	if err := findObject(e.Payload, &item, e.Kind); err != nil {
		return
	}

	if item.CoverageStatus == api.ItemCoverageStatusApproved {
		// TODO any business rules to deal with here
	} else if item.CoverageStatus == api.ItemCoverageStatusPending { // Was submitted but not auto approved
		// TODO any business rules to deal with here
	} else {
		domain.ErrLogger.Printf(wrongStatusMsg, "itemSubmitted", item.CoverageStatus)
	}

	messages.ItemSubmittedSend(item, getNotifiersFromEventPayload(e.Payload))
}

func itemRevision(e events.Event) {
	var item models.Item
	if err := findObject(e.Payload, &item, e.Kind); err != nil {
		return
	}

	if item.CoverageStatus != api.ItemCoverageStatusRevision {
		panic(fmt.Sprintf(wrongStatusMsg, "itemRevision", item.CoverageStatus))
	}

	messages.ItemRevisionSend(item, getNotifiersFromEventPayload(e.Payload))
}

func itemApproved(e events.Event) {
	var item models.Item
	if err := findObject(e.Payload, &item, e.Kind); err != nil {
		return
	}

	if item.CoverageStatus != api.ItemCoverageStatusApproved {
		domain.ErrLogger.Printf(wrongStatusMsg, "itemApproved", item.CoverageStatus)
		return
	}

	messages.ItemApprovedSend(item, getNotifiersFromEventPayload(e.Payload))
	// TODO do whatever else needs doing
}

func itemDenied(e events.Event) {
	var item models.Item
	if err := findObject(e.Payload, &item, e.Kind); err != nil {
		return
	}

	if item.CoverageStatus != api.ItemCoverageStatusDenied {
		domain.ErrLogger.Printf(wrongStatusMsg, "itemDenied", item.CoverageStatus)
		return
	}

	messages.ItemDeniedSend(item, getNotifiersFromEventPayload(e.Payload))
}
