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

func claimReview1(e events.Event) {
	var claim models.Claim
	if err := findObject(e.Payload, &claim, e.Kind); err != nil {
		return
	}

	if claim.Status != api.ClaimStatusReview1 {
		panic(fmt.Sprintf(wrongStatusMsg, "claimReview1", claim.Status))
	}

	models.DB.Transaction(func(tx *pop.Connection) error {
		messages.ClaimReview1QueueMessage(tx, claim)
		return nil
	})
}

func claimRevision(e events.Event) {
	var claim models.Claim
	if err := findObject(e.Payload, &claim, e.Kind); err != nil {
		return
	}

	if claim.Status != api.ClaimStatusRevision {
		panic(fmt.Sprintf(wrongStatusMsg, "claimRevision", claim.Status))
	}

	models.DB.Transaction(func(tx *pop.Connection) error {
		messages.ClaimRevisionQueueMessage(tx, claim)
		return nil
	})
}

func claimPreapproved(e events.Event) {
	var claim models.Claim
	if err := findObject(e.Payload, &claim, e.Kind); err != nil {
		return
	}

	if claim.Status != api.ClaimStatusReceipt {
		panic(fmt.Sprintf(wrongStatusMsg, "claimReceipt", claim.Status))
	}

	models.DB.Transaction(func(tx *pop.Connection) error {
		messages.ClaimPreapprovedQueueMessage(tx, claim)
		return nil
	})
}

func claimReceipt(e events.Event) {
	var claim models.Claim
	if err := findObject(e.Payload, &claim, e.Kind); err != nil {
		return
	}

	if claim.Status != api.ClaimStatusReceipt {
		panic(fmt.Sprintf(wrongStatusMsg, "claimReceipt", claim.Status))
	}

	models.DB.Transaction(func(tx *pop.Connection) error {
		messages.ClaimReceiptQueueMessage(tx, claim)
		return nil
	})
}

func claimReview2(e events.Event) {
	var claim models.Claim
	if err := findObject(e.Payload, &claim, e.Kind); err != nil {
		return
	}

	if claim.Status != api.ClaimStatusReview2 {
		panic(fmt.Sprintf(wrongStatusMsg, "claimReview2", claim.Status))
	}

	models.DB.Transaction(func(tx *pop.Connection) error {
		messages.ClaimReview2QueueMessage(tx, claim)
		return nil
	})
}

func claimReview3(e events.Event) {
	var claim models.Claim
	if err := findObject(e.Payload, &claim, e.Kind); err != nil {
		return
	}

	if claim.Status != api.ClaimStatusReview3 {
		panic(fmt.Sprintf(wrongStatusMsg, "claimReview3", claim.Status))
	}

	models.DB.Transaction(func(tx *pop.Connection) error {
		messages.ClaimReview3QueueMessage(tx, claim)
		return nil
	})
}

func claimApproved(e events.Event) {
	var claim models.Claim
	if err := findObject(e.Payload, &claim, e.Kind); err != nil {
		return
	}

	models.DB.Transaction(func(tx *pop.Connection) error {
		if err := claim.StopItemCoverage(tx); err != nil {
			domain.ErrLogger.Printf("listener error with claim %s: %s", claim.ID.String(), err)
		}
		return nil
	})

	models.DB.Transaction(func(tx *pop.Connection) error {
		messages.ClaimApprovedQueueMessage(tx, claim)
		return nil
	})
}

func claimDenied(e events.Event) {
	var claim models.Claim
	if err := findObject(e.Payload, &claim, e.Kind); err != nil {
		return
	}

	models.DB.Transaction(func(tx *pop.Connection) error {
		messages.ClaimDeniedQueueMessage(tx, claim)
		return nil
	})
}
