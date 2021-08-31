package listeners

import (
	"github.com/gobuffalo/events"

	"github.com/silinternational/cover-api/domain"
	"github.com/silinternational/cover-api/models"
)

func claimSubmitted(e events.Event) {
	if e.Kind != domain.EventApiClaimSubmitted {
		return
	}

	defer panicRecover(e.Kind)

	var claim models.Claim
	if err := findObject(e.Payload, &claim, e.Kind); err != nil {
		return
	}

	// TODO Notify admin and do whatever else needs doing
}

func claimRevision(e events.Event) {
	if e.Kind != domain.EventApiClaimRevision {
		return
	}

	defer panicRecover(e.Kind)

	var claim models.Claim
	if err := findObject(e.Payload, &claim, e.Kind); err != nil {
		return
	}

	// TODO Notify user and do whatever else needs doing
}

func claimPreapproved(e events.Event) {
	if e.Kind != domain.EventApiClaimPreapproved {
		return
	}

	defer panicRecover(e.Kind)

	var claim models.Claim
	if err := findObject(e.Payload, &claim, e.Kind); err != nil {
		return
	}

	// TODO Notify user and do whatever else needs doing
}

func claimReceipt(e events.Event) {
	if e.Kind != domain.EventApiClaimReceipt {
		return
	}

	defer panicRecover(e.Kind)

	var claim models.Claim
	if err := findObject(e.Payload, &claim, e.Kind); err != nil {
		return
	}

	// TODO Notify user and do whatever else needs doing
}

func claimReview2(e events.Event) {
	if e.Kind != domain.EventApiClaimReview2 {
		return
	}

	defer panicRecover(e.Kind)

	var claim models.Claim
	if err := findObject(e.Payload, &claim, e.Kind); err != nil {
		return
	}

	// TODO Notify admin and do whatever else needs doing
}

func claimReview3(e events.Event) {
	if e.Kind != domain.EventApiClaimReview3 {
		return
	}

	defer panicRecover(e.Kind)

	var claim models.Claim
	if err := findObject(e.Payload, &claim, e.Kind); err != nil {
		return
	}

	// TODO Notify admin boss and do whatever else needs doing
}

func claimApproved(e events.Event) {
	if e.Kind != domain.EventApiClaimApproved {
		return
	}

	defer panicRecover(e.Kind)

	var claim models.Claim
	if err := findObject(e.Payload, &claim, e.Kind); err != nil {
		return
	}

	// TODO Notify user and do whatever else needs doing
}

func claimDenied(e events.Event) {
	if e.Kind != domain.EventApiClaimDenied {
		return
	}

	defer panicRecover(e.Kind)

	var claim models.Claim
	if err := findObject(e.Payload, &claim, e.Kind); err != nil {
		return
	}

	// TODO Notify user and do whatever else needs doing
}
