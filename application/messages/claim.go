package messages

import (
	"fmt"

	"github.com/silinternational/cover-api/domain"
	"github.com/silinternational/cover-api/models"
	"github.com/silinternational/cover-api/notifications"
)

func addMessageClaimData(msg *notifications.Message, claim models.Claim) {
	msg.Data["claimURL"] = fmt.Sprintf("%s/%s/%s", domain.Env.UIURL, domain.TypeClaim, claim.ID)
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

func ClaimReview1Send(claim models.Claim, notifiers []interface{}) {
	claim.LoadPolicyMembers(models.DB, false)
	memberName := claim.Policy.Members[0].Name()

	msg := notifications.NewEmailMessage().AddToSteward()
	addMessageClaimData(&msg, claim)
	msg.Template = MessageTemplateClaimReview1Steward
	msg.Data["memberName"] = memberName
	msg.Subject = "Action Required. " + memberName + " just (re)submitted a claim for approval"

	if err := notifications.Send(msg, notifiers...); err != nil {
		domain.ErrLogger.Printf("error sending claim review1 notification, %s", err)
	}
}

func ClaimRevisionSend(claim models.Claim, notifiers []interface{}) {
	claim.LoadPolicyMembers(models.DB, false)

	// TODO figure out how to specify required revisions

	for _, m := range claim.Policy.Members {
		msg := newClaimMessageForMember(claim, m)
		msg.Template = MessageTemplateClaimRevisionMember
		msg.Subject = "changes have been requested on your claim"
		if err := notifications.Send(msg, notifiers...); err != nil {
			domain.ErrLogger.Printf("error sending claim revision notification to member, %s", err)
		}
	}
}

func ClaimPreapprovedSend(claim models.Claim, notifiers []interface{}) {
	claim.LoadPolicyMembers(models.DB, false)

	// TODO Figure out how to tell the members what receipts are needed

	for _, m := range claim.Policy.Members {
		msg := newClaimMessageForMember(claim, m)
		msg.Template = MessageTemplateClaimPreapprovedMember
		msg.Subject = "receipts are needed on your new claim"
		if err := notifications.Send(msg, notifiers...); err != nil {
			domain.ErrLogger.Printf("error sending claim preapproved notification to member, %s", err)
		}
	}
}

func ClaimReceiptSend(claim models.Claim, notifiers []interface{}) {
	claim.LoadPolicyMembers(models.DB, false)

	// TODO Figure out how to tell the members what receipts are needed

	for _, m := range claim.Policy.Members {
		msg := newClaimMessageForMember(claim, m)
		msg.Template = MessageTemplateClaimReceiptMember
		msg.Subject = "new receipts are needed on your claim"
		if err := notifications.Send(msg, notifiers...); err != nil {
			domain.ErrLogger.Printf("error sending claim receipt notification to member, %s", err)
		}
	}
}

func ClaimReview2Send(claim models.Claim, notifiers []interface{}) {
	claim.LoadPolicyMembers(models.DB, false)
	memberName := claim.Policy.Members[0].Name()

	msg := notifications.NewEmailMessage().AddToSteward()
	addMessageClaimData(&msg, claim)
	msg.Template = MessageTemplateClaimReview2Steward
	msg.Data["memberName"] = memberName
	msg.Subject = "Action Required. " + memberName + " just resubmitted a claim for approval"
	if err := notifications.Send(msg, notifiers...); err != nil {
		domain.ErrLogger.Printf("error sending claim review2 notification, %s", err)
	}
}

func ClaimReview3Send(claim models.Claim, notifiers []interface{}) {
	claim.LoadPolicyMembers(models.DB, false)
	memberName := claim.Policy.Members[0].Name()

	msg := notifications.NewEmailMessage().AddToSteward()
	addMessageClaimData(&msg, claim)
	msg.Template = MessageTemplateClaimReview3Boss
	msg.Data["memberName"] = memberName
	msg.Subject = "Action Required. " + memberName + " has a claim waiting for your approval"

	if err := notifications.Send(msg, notifiers...); err != nil {
		domain.ErrLogger.Printf("error sending claim review3 notification, %s", err)
	}
}
