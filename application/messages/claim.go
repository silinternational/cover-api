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

func ClaimReview1(claim models.Claim, notifiers []interface{}) {
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
