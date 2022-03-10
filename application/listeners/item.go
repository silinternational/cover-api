package listeners

import (
	"fmt"

	"github.com/gobuffalo/events"
	"github.com/gobuffalo/pop/v5"

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

	if item.CoverageStatus != api.ItemCoverageStatusPending {
		domain.ErrLogger.Printf(wrongStatusMsg, "itemSubmitted", item.CoverageStatus)
	}

	models.DB.Transaction(func(tx *pop.Connection) error {
		messages.ItemSubmittedQueueMessage(tx, item)
		messages.ItemPendingQueueMessage(tx, item)
		return nil
	})
}

func itemRevision(e events.Event) {
	var item models.Item
	if err := findObject(e.Payload, &item, e.Kind); err != nil {
		return
	}

	if item.CoverageStatus != api.ItemCoverageStatusRevision {
		panic(fmt.Sprintf(wrongStatusMsg, "itemRevision", item.CoverageStatus))
	}

	models.DB.Transaction(func(tx *pop.Connection) error {
		messages.ItemRevisionQueueMessage(tx, item)
		return nil
	})
}

func itemAutoApproved(e events.Event) {
	var item models.Item
	if err := findObject(e.Payload, &item, e.Kind); err != nil {
		return
	}

	if item.CoverageStatus != api.ItemCoverageStatusApproved {
		domain.ErrLogger.Printf(wrongStatusMsg, "itemApproved", item.CoverageStatus)
		return
	}

	models.DB.Transaction(func(tx *pop.Connection) error {
		messages.ItemAutoApprovedQueueMessage(tx, item)
		return nil
	})
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

	models.DB.Transaction(func(tx *pop.Connection) error {
		messages.ItemApprovedQueueMessage(tx, item)
		return nil
	})
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

	models.DB.Transaction(func(tx *pop.Connection) error {
		messages.ItemDeniedQueueMessage(tx, item)
		return nil
	})
}
