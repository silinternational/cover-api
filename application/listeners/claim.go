package listeners

import (
	"fmt"

	"github.com/gobuffalo/events"

	"github.com/silinternational/cover-api/api"
	"github.com/silinternational/cover-api/domain"
	"github.com/silinternational/cover-api/messages"
	"github.com/silinternational/cover-api/models"
)

func claimReview1(e events.Event) {
	if e.Kind != domain.EventApiClaimReview1 {
		return
	}

	defer panicRecover(e.Kind)

	var claim models.Claim
	if err := findObject(e.Payload, &claim, e.Kind); err != nil {
		return
	}

	if claim.Status != api.ClaimStatusReview1 {
		panic(fmt.Sprintf(wrongStatusMsg, "claimReview1", claim.Status))
	}

	messages.ClaimReview1Send(claim, getNotifiersFromEventPayload(e.Payload))
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

	if claim.Status != api.ClaimStatusRevision {
		panic(fmt.Sprintf(wrongStatusMsg, "claimRevision", claim.Status))
	}

	messages.ClaimRevisionSend(claim, getNotifiersFromEventPayload(e.Payload))
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

	if claim.Status != api.ClaimStatusReceipt {
		panic(fmt.Sprintf(wrongStatusMsg, "claimReceipt", claim.Status))
	}

	messages.ClaimPreapprovedSend(claim, getNotifiersFromEventPayload(e.Payload))
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

	if claim.Status != api.ClaimStatusReceipt {
		panic(fmt.Sprintf(wrongStatusMsg, "claimReceipt", claim.Status))
	}

	messages.ClaimReceiptSend(claim, getNotifiersFromEventPayload(e.Payload))
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

	if claim.Status != api.ClaimStatusReview2 {
		panic(fmt.Sprintf(wrongStatusMsg, "claimReview2", claim.Status))
	}

	messages.ClaimReview2Send(claim, getNotifiersFromEventPayload(e.Payload))
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

	if claim.Status != api.ClaimStatusReview3 {
		panic(fmt.Sprintf(wrongStatusMsg, "claimReview3", claim.Status))
	}

	messages.ClaimReview3Send(claim, getNotifiersFromEventPayload(e.Payload))
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
