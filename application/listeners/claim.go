package listeners

import (
	"fmt"

	"github.com/gobuffalo/events"

	"github.com/silinternational/cover-api/api"
	"github.com/silinternational/cover-api/domain"
	"github.com/silinternational/cover-api/models"
	"github.com/silinternational/cover-api/notifications"
)

func addMessageClaimData(msg *notifications.Message, claim models.Claim) {
	msg.Data["claimURL"] = fmt.Sprintf("%s/%s/%s", domain.Env.UIURL, domain.TypeClaim, claim.ID)
	msg.Data["claimRefNum"] = claim.ReferenceNumber
	return
}

func claimSubmitted(e events.Event) {
	if e.Kind != domain.EventApiClaimSubmitted {
		return
	}

	defer panicRecover(e.Kind)

	var claim models.Claim
	if err := findObject(e.Payload, &claim, e.Kind); err != nil {
		return
	}

	claim.LoadPolicyMembers(models.DB, false)
	memberName := claim.Policy.Members[0].Name()

	msg := notifications.NewEmailMessage().AddToSteward()
	addMessageClaimData(&msg, claim)
	msg.Template = domain.MessageTemplateClaimSubmittedSteward
	msg.Data["memberName"] = memberName

	if claim.Status == api.ClaimStatusReview1 {
		msg.Subject = "Action Required. " + memberName + " just submitted a new claim for approval"
	} else if claim.Status == api.ClaimStatusReview2 {
		msg.Subject = "Action Required. " + memberName + " just resubmitted a claim for approval"
	} else {
		domain.ErrLogger.Printf(wrongStatusMsg, "claimSubmitted", claim.Status)
	}

	notifiers := getNotifiersFromEventPayload(e.Payload)
	if err := notifications.Send(msg, notifiers...); err != nil {
		domain.ErrLogger.Printf("error sending claim submitted notification, %s", err)
	}

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
