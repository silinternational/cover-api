package listeners

import (
	"fmt"

	"github.com/gobuffalo/events"

	"github.com/silinternational/cover-api/api"
	"github.com/silinternational/cover-api/domain"
	"github.com/silinternational/cover-api/messages"
	"github.com/silinternational/cover-api/models"
	"github.com/silinternational/cover-api/notifications"
)

func addMessageClaimData(msg *notifications.Message, claim models.Claim) {
	msg.Data["claimURL"] = fmt.Sprintf("%s/claims/%s", domain.Env.UIURL, claim.ID)
	msg.Data["claimRefNum"] = claim.ReferenceNumber
	return
}

func newClaimMessageForMember(claim models.Claim, member models.User) notifications.Message {
	msg := notifications.NewEmailMessage()
	addMessageClaimData(&msg, claim)
	msg.ToName = member.Name()
	msg.ToEmail = member.EmailOfChoice()
	msg.Data["memberName"] = member.Name()

	return msg
}

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

	notifiers := getNotifiersFromEventPayload(e.Payload)
	messages.ClaimReview1(claim, notifiers)
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

	claim.LoadPolicyMembers(models.DB, false)
	notifiers := getNotifiersFromEventPayload(e.Payload)

	// TODO figure out how to specify required revisions

	for _, m := range claim.Policy.Members {
		msg := newClaimMessageForMember(claim, m)
		msg.Template = domain.MessageTemplateClaimRevisionMember
		msg.Subject = "changes have been requested on your claim"
		if err := notifications.Send(msg, notifiers...); err != nil {
			domain.ErrLogger.Printf("error sending claim revision notification to member, %s", err)
		}
	}
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

	claim.LoadPolicyMembers(models.DB, false)
	notifiers := getNotifiersFromEventPayload(e.Payload)

	// TODO Figure out how to tell the members what receipts are needed

	for _, m := range claim.Policy.Members {
		msg := newClaimMessageForMember(claim, m)
		msg.Template = domain.MessageTemplateClaimPreapprovedMember
		msg.Subject = "receipts are needed on your new claim"
		if err := notifications.Send(msg, notifiers...); err != nil {
			domain.ErrLogger.Printf("error sending claim preapproved notification to member, %s", err)
		}
	}
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

	claim.LoadPolicyMembers(models.DB, false)
	notifiers := getNotifiersFromEventPayload(e.Payload)

	// TODO Figure out how to tell the members what receipts are needed

	for _, m := range claim.Policy.Members {
		msg := newClaimMessageForMember(claim, m)
		msg.Template = domain.MessageTemplateClaimReceiptMember
		msg.Subject = "new receipts are needed on your claim"
		if err := notifications.Send(msg, notifiers...); err != nil {
			domain.ErrLogger.Printf("error sending claim receipt notification to member, %s", err)
		}
	}
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

	claim.LoadPolicyMembers(models.DB, false)
	memberName := claim.Policy.Members[0].Name()

	msg := notifications.NewEmailMessage().AddToSteward()
	addMessageClaimData(&msg, claim)
	msg.Template = domain.MessageTemplateClaimReview2Steward
	msg.Data["memberName"] = memberName
	msg.Subject = "Action Required. " + memberName + " just resubmitted a claim for approval"

	notifiers := getNotifiersFromEventPayload(e.Payload)
	if err := notifications.Send(msg, notifiers...); err != nil {
		domain.ErrLogger.Printf("error sending claim review2 notification, %s", err)
	}
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

	claim.LoadPolicyMembers(models.DB, false)
	memberName := claim.Policy.Members[0].Name()

	msg := notifications.NewEmailMessage().AddToSteward()
	addMessageClaimData(&msg, claim)
	msg.Template = domain.MessageTemplateClaimReview3Boss
	msg.Data["memberName"] = memberName
	msg.Subject = "Action Required. " + memberName + " has a claim waiting for your approval"

	notifiers := getNotifiersFromEventPayload(e.Payload)
	if err := notifications.Send(msg, notifiers...); err != nil {
		domain.ErrLogger.Printf("error sending claim review3 notification, %s", err)
	}
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
