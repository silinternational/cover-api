package messages

import (
	"fmt"

	"github.com/gobuffalo/nulls"
	"github.com/gobuffalo/pop/v5"

	"github.com/silinternational/cover-api/api"
	"github.com/silinternational/cover-api/domain"
	"github.com/silinternational/cover-api/models"
)

// ClaimReview1QueueMessage queues messages to the stewards to
//  notify them that a claim has been submitted for preapproval
func ClaimReview1QueueMessage(tx *pop.Connection, claim models.Claim) {
	claim.LoadPolicyMembers(tx, false)
	memberName := claim.Policy.Members[0].Name()

	data := newEmailMessageData()
	data.addClaimData(tx, claim)
	data["memberName"] = memberName

	item := data["item"].(models.Item)

	notn := models.Notification{
		ClaimID: nulls.NewUUID(claim.ID),
		Body:    data.renderHTML(MessageTemplateClaimReview1Steward),
		Subject: "New claim on " + item.Name,

		InappText:     "A new claim is waiting for your approval",
		Event:         "Claim Review1 Notification",
		EventCategory: EventCategoryClaim,
	}
	if err := notn.Create(tx); err != nil {
		panic("error creating new Claim Review1 Notification: " + err.Error())
	}

	notn.CreateNotificationUsersForStewards(tx)
}

// ClaimRevisionQueueMessage queues messages to the claim's members to
//  notify them that revisions are required on their claim
func ClaimRevisionQueueMessage(tx *pop.Connection, claim models.Claim) {
	claim.LoadPolicyMembers(tx, false)

	data := newEmailMessageData()
	data.addClaimData(tx, claim)

	notn := models.Notification{
		ClaimID:       nulls.NewUUID(claim.ID),
		Body:          data.renderHTML(MessageTemplateClaimRevisionMember),
		Subject:       "Please provide more information",
		InappText:     "Please provide more information on your new claim",
		Event:         "Claim Revision Required Notification",
		EventCategory: EventCategoryClaim,
	}
	if err := notn.Create(tx); err != nil {
		panic("error creating new Claim Revision Notification: " + err.Error())
	}

	for _, m := range claim.Policy.Members {
		notn.CreateNotificationUserForUser(tx, m)
	}
}

// ClaimPreapprovedQueueMessage queues messages to the claim's members to
//  notify them that their claim has been preapproved and requires receipts
func ClaimPreapprovedQueueMessage(tx *pop.Connection, claim models.Claim) {
	claim.LoadPolicyMembers(tx, false)

	data := newEmailMessageData()
	data.addClaimData(tx, claim)

	claim.LoadClaimItems(tx, false)

	if len(claim.ClaimItems) == 0 {
		msg := fmt.Sprintf("claim %s has no claim_item", claim.ID)
		panic(msg)
	}

	claimItem := claim.ClaimItems[0]
	claimItem.LoadItem(tx, false)

	data["estimate"] = "$-"
	data["deductible"] = domain.Env.DeductibleString
	data["repairThreshold"] = domain.Env.RepairThresholdString

	switch claimItem.PayoutOption {
	case api.PayoutOptionRepair:
		data["estimate"] = "$" + claimItem.RepairEstimate.String()
	case api.PayoutOptionReplacement:
		data["estimate"] = "$" + claimItem.ReplaceEstimate.String()
	}

	notn := models.Notification{
		ClaimID:       nulls.NewUUID(claim.ID),
		Body:          data.renderHTML(MessageTemplateClaimPreapprovedMember),
		Subject:       "Claim Approved for " + string(claimItem.PayoutOption),
		InappText:     "receipts are needed on your new claim",
		Event:         "Claim Preapproved Notification",
		EventCategory: EventCategoryClaim,
	}
	if err := notn.Create(tx); err != nil {
		panic("error creating new Claim Preapproved Notification: " + err.Error())
	}

	for _, m := range claim.Policy.Members {
		notn.CreateNotificationUserForUser(tx, m)
	}
}

// ClaimReceiptQueueMessage queues messages to the claim's members to
//  notify them that their claim requires receipts (again)
func ClaimReceiptQueueMessage(tx *pop.Connection, claim models.Claim) {
	claim.LoadPolicyMembers(tx, false)
	claim.LoadClaimItems(tx, false)

	if len(claim.ClaimItems) == 0 {
		msg := fmt.Sprintf("claim %s has no claim_item", claim.ID)
		panic(msg)
	}

	claimItem := claim.ClaimItems[0]

	data := newEmailMessageData()
	data.addClaimData(tx, claim)

	data["estimate"] = "$-"
	data["deductible"] = domain.Env.DeductibleString

	switch claimItem.PayoutOption {
	case api.PayoutOptionRepair:
		data["estimate"] = "$" + claimItem.RepairEstimate.String()
	case api.PayoutOptionReplacement:
		data["estimate"] = "$" + claimItem.ReplaceEstimate.String()
	}

	notn := models.Notification{
		ClaimID: nulls.NewUUID(claim.ID),
		Body:    data.renderHTML(MessageTemplateClaimReceiptMember),
		Subject: "Claim Needs Receipt",

		InappText:     "Please provide a receipt",
		Event:         "Claim Receipt Notification",
		EventCategory: EventCategoryClaim,
	}
	if err := notn.Create(tx); err != nil {
		panic("error creating new Claim Receipt Notification: " + err.Error())
	}

	for _, m := range claim.Policy.Members {
		notn.CreateNotificationUserForUser(tx, m)
	}
}

// ClaimReview2QueueMessage queues messages to the stewards to
//  notify them that a claim has been submitted to Review2 status
func ClaimReview2QueueMessage(tx *pop.Connection, claim models.Claim) {
	claim.LoadPolicyMembers(tx, false)
	memberName := claim.Policy.Members[0].Name()

	data := newEmailMessageData()
	data.addClaimData(tx, claim)
	data["memberName"] = memberName

	item := data["item"].(models.Item)

	notn := models.Notification{
		ClaimID:       nulls.NewUUID(claim.ID),
		Body:          data.renderHTML(MessageTemplateClaimReview2Steward),
		Subject:       "Consider payout for claim on " + item.Name,
		InappText:     "A claim is waiting for your approval",
		Event:         "Claim Review2 Notification",
		EventCategory: EventCategoryClaim,
	}
	if err := notn.Create(tx); err != nil {
		panic("error creating new Claim Review2 Notification: " + err.Error())
	}

	notn.CreateNotificationUsersForStewards(tx)
}

// ClaimReview3QueueMessage queues messages to the signators to
//  notify them that a claim has been submitted to Review3 status
func ClaimReview3QueueMessage(tx *pop.Connection, claim models.Claim) {
	claim.LoadPolicyMembers(tx, false)
	memberName := claim.Policy.Members[0].Name()

	data := newEmailMessageData()
	data.addClaimData(tx, claim)
	data["memberName"] = memberName

	item := data["item"].(models.Item)

	claim.LoadReviewer(tx, false)
	data["firstReviewer"] = claim.Reviewer.Name()

	notn := models.Notification{
		ClaimID:       nulls.NewUUID(claim.ID),
		Body:          data.renderHTML(MessageTemplateClaimReview3Signator),
		Subject:       "Final approval for claim on " + item.Name,
		InappText:     "A claim is waiting for your approval",
		Event:         "Claim Review3 Notification",
		EventCategory: EventCategoryClaim,
	}
	if err := notn.Create(tx); err != nil {
		panic("error creating new Claim Review3 Notification: " + err.Error())
	}

	notn.CreateNotificationUsersForSignators(tx)
}

// ClaimApprovedQueueMessage queues messages to a claim's members to
//  notify them that it has been approved
func ClaimApprovedQueueMessage(tx *pop.Connection, claim models.Claim) {
	claim.LoadPolicyMembers(tx, false)

	data := newEmailMessageData()
	data.addClaimData(tx, claim)

	notn := models.Notification{
		ClaimID:       nulls.NewUUID(claim.ID),
		Body:          data.renderHTML(MessageTemplateClaimApprovedMember),
		Subject:       "Claim Payout Approved",
		InappText:     "your claim has been approved",
		Event:         "Claim Approved Notification",
		EventCategory: EventCategoryClaim,
	}
	if err := notn.Create(tx); err != nil {
		panic("error creating new Claim Approved Notification: " + err.Error())
	}

	for _, m := range claim.Policy.Members {
		notn.CreateNotificationUserForUser(tx, m)
	}
}

// ClaimDeniedQueueMessage queues messages to a claim's members to
//  notify them that it has been denied
func ClaimDeniedQueueMessage(tx *pop.Connection, claim models.Claim) {
	claim.LoadPolicyMembers(tx, false)

	// TODO check if it was denied by the signator and if so, email the steward

	data := newEmailMessageData()
	data.addClaimData(tx, claim)

	notn := models.Notification{
		ClaimID:       nulls.NewUUID(claim.ID),
		Body:          data.renderHTML(MessageTemplateClaimDeniedMember),
		Subject:       "An Update on Your Coverage Request",
		InappText:     "your claim has been denied",
		Event:         "Claim Denied Notification",
		EventCategory: EventCategoryClaim,
	}
	if err := notn.Create(tx); err != nil {
		panic("error creating new Claim Denied Notification: " + err.Error())
	}

	for _, m := range claim.Policy.Members {
		notn.CreateNotificationUserForUser(tx, m)
	}
}
