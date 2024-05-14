package listeners

import (
	"fmt"

	"github.com/gobuffalo/events"
	"github.com/gobuffalo/pop/v6"

	"github.com/silinternational/cover-api/api"
	"github.com/silinternational/cover-api/log"
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
		log.Errorf(wrongStatusMsg, "itemSubmitted", item.CoverageStatus)
	}

	_ = models.DB.Transaction(func(tx *pop.Connection) error {
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

	_ = models.DB.Transaction(func(tx *pop.Connection) error {
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
		log.Errorf(wrongStatusMsg, "itemApproved", item.CoverageStatus)
		return
	}

	_ = models.DB.Transaction(func(tx *pop.Connection) error {
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
		log.Errorf(wrongStatusMsg, "itemApproved", item.CoverageStatus)
		return
	}

	_ = models.DB.Transaction(func(tx *pop.Connection) error {
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
		log.Errorf(wrongStatusMsg, "itemDenied", item.CoverageStatus)
		return
	}

	_ = models.DB.Transaction(func(tx *pop.Connection) error {
		messages.ItemDeniedQueueMessage(tx, item)
		return nil
	})
}
