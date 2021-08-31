package listeners

import (
	"github.com/gobuffalo/events"

	"github.com/silinternational/cover-api/domain"
	"github.com/silinternational/cover-api/models"
)

func itemSubmitted(e events.Event) {
	if e.Kind != domain.EventApiItemSubmitted {
		return
	}

	defer panicRecover(e.Kind)

	var item models.Item
	if err := findObject(e.Payload, &item, e.Kind); err != nil {
		return
	}

	// TODO Notify admin and do whatever else needs doing
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
