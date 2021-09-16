package messages

import (
	"github.com/gobuffalo/nulls"
	"github.com/gobuffalo/pop/v5"

	"github.com/silinternational/cover-api/models"
)

// ClaimReview1QueueMessage queues messages to the stewards to
//  notify them that a claim has been submitted for preapproval
func ClaimReview1QueueMessage(tx *pop.Connection, claim models.Claim) {
	claim.LoadPolicyMembers(models.DB, false)
	memberName := claim.Policy.Members[0].Name()

	data := newEmailMessageData()
	data.addClaimData(claim)
	data["memberName"] = memberName

	notn := models.Notification{
		ClaimID: nulls.NewUUID(claim.ID),
		Body:    data.renderHTML(MessageTemplateClaimReview1Steward),
		Subject: "Action Required. " + memberName + " just (re)submitted a claim for approval",

		InappText: "A new claim is waiting for your approval",

		// TODO make these constants somewhere
		Event:         "Claim Review1 Notification",
		EventCategory: "Claim",
	}
	if err := notn.Create(tx); err != nil {
		panic("error creating new Claim Review1 Notification: " + err.Error())
	}

	notn.CreateNotificationUsersForStewards(tx)
}

// ClaimRevisionQueueMessage queues messages to the claim's members to
//  notify them that revisions are required on their claim
func ClaimRevisionQueueMessage(tx *pop.Connection, claim models.Claim) {
	claim.LoadPolicyMembers(models.DB, false)

	// TODO figure out how to specify required revisions

	data := newEmailMessageData()
	data.addClaimData(claim)

	notn := models.Notification{
		ClaimID: nulls.NewUUID(claim.ID),
		Body:    data.renderHTML(MessageTemplateClaimRevisionMember),
		Subject: "changes have been requested on your claim",
		// TODO make this more helpful
		InappText: "changes have been requested on your new claim",

		// TODO make these constants somewhere
		Event:         "Claim Revision Required Notification",
		EventCategory: "Claim",
	}
	if err := notn.Create(tx); err != nil {
		panic("error creating new Claim Revision Notification: " + err.Error())
	}

	for _, m := range claim.Policy.Members {
		notn.CreateNotificationUser(tx, m)
	}
}

// ClaimPreapprovedQueueMessage queues messages to the claim's members to
//  notify them that their claim has been preapproved and requires receipts
func ClaimPreapprovedQueueMessage(tx *pop.Connection, claim models.Claim) {
	claim.LoadPolicyMembers(models.DB, false)

	// TODO Figure out how to tell the members what receipts are needed

	data := newEmailMessageData()
	data.addClaimData(claim)

	notn := models.Notification{
		ClaimID: nulls.NewUUID(claim.ID),
		Body:    data.renderHTML(MessageTemplateClaimPreapprovedMember),
		Subject: "receipt(s) needed on your new claim",

		InappText: "receipts are needed on your new claim",

		// TODO make these constants somewhere
		Event:         "Claim Preapproved Notification",
		EventCategory: "Claim",
	}
	if err := notn.Create(tx); err != nil {
		panic("error creating new Claim Preapproved Notification: " + err.Error())
	}

	for _, m := range claim.Policy.Members {
		notn.CreateNotificationUser(tx, m)
	}
}

// ClaimReceiptQueueMessage queues messages to the claim's members to
//  notify them that their claim requires receipts (again)
func ClaimReceiptQueueMessage(tx *pop.Connection, claim models.Claim) {
	claim.LoadPolicyMembers(models.DB, false)

	// TODO Figure out how to tell the members what receipts are needed

	data := newEmailMessageData()
	data.addClaimData(claim)

	notn := models.Notification{
		ClaimID: nulls.NewUUID(claim.ID),
		Body:    data.renderHTML(MessageTemplateClaimReceiptMember),
		Subject: "new receipt(s) needed on your claim",

		InappText: "new/different receipts are needed on your claim",

		// TODO make these constants somewhere
		Event:         "Claim Receipt Notification",
		EventCategory: "Claim",
	}
	if err := notn.Create(tx); err != nil {
		panic("error creating new Claim Receipt Notification: " + err.Error())
	}

	for _, m := range claim.Policy.Members {
		notn.CreateNotificationUser(tx, m)
	}
}

// ClaimReview2QueueMessage queues messages to the stewards to
//  notify them that a claim has been submitted to Review2 status
func ClaimReview2QueueMessage(tx *pop.Connection, claim models.Claim) {
	claim.LoadPolicyMembers(models.DB, false)
	memberName := claim.Policy.Members[0].Name()

	data := newEmailMessageData()
	data.addClaimData(claim)
	data["memberName"] = memberName

	notn := models.Notification{
		ClaimID: nulls.NewUUID(claim.ID),
		Body:    data.renderHTML(MessageTemplateClaimReview2Steward),
		Subject: "Action Required. " + memberName + " just resubmitted a claim for approval",

		InappText: "A claim is waiting for your approval",

		// TODO make these constants somewhere
		Event:         "Claim Review2 Notification",
		EventCategory: "Claim",
	}
	if err := notn.Create(tx); err != nil {
		panic("error creating new Claim Review2 Notification: " + err.Error())
	}

	notn.CreateNotificationUsersForStewards(tx)
}

// ClaimReview3QueueMessage queues messages to the signators to
//  notify them that a claim has been submitted to Review3 status
func ClaimReview3QueueMessage(tx *pop.Connection, claim models.Claim) {
	claim.LoadPolicyMembers(models.DB, false)
	memberName := claim.Policy.Members[0].Name()

	data := newEmailMessageData()
	data.addClaimData(claim)
	data["memberName"] = memberName

	notn := models.Notification{
		ClaimID: nulls.NewUUID(claim.ID),
		Body:    data.renderHTML(MessageTemplateClaimReview3Signator),
		Subject: "Action Required. " + memberName + " has a claim waiting for your approval",

		InappText: "A claim is waiting for your approval",

		// TODO make these constants somewhere
		Event:         "Claim Review3 Notification",
		EventCategory: "Claim",
	}
	if err := notn.Create(tx); err != nil {
		panic("error creating new Claim Review3 Notification: " + err.Error())
	}

	notn.CreateNotificationUsersForSignators(tx)
}

// ClaimApprovedQueueMessage queues messages to a claim's members to
//  notify them that it has been approved
func ClaimApprovedQueueMessage(tx *pop.Connection, claim models.Claim) {
	claim.LoadPolicyMembers(models.DB, false)

	// TODO figure out how to specify required revisions

	data := newEmailMessageData()
	data.addClaimData(claim)

	notn := models.Notification{
		ClaimID:   nulls.NewUUID(claim.ID),
		Body:      data.renderHTML(MessageTemplateClaimApprovedMember),
		Subject:   "your claim has been approved",
		InappText: "your claim has been approved",

		// TODO make these constants somewhere
		Event:         "Claim Approved Notification",
		EventCategory: "Claim",
	}
	if err := notn.Create(tx); err != nil {
		panic("error creating new Claim Approved Notification: " + err.Error())
	}

	for _, m := range claim.Policy.Members {
		notn.CreateNotificationUser(tx, m)
	}
}

// ClaimDeniedQueueMessage queues messages to a claim's members to
//  notify them that it has been denied
func ClaimDeniedQueueMessage(tx *pop.Connection, claim models.Claim) {
	claim.LoadPolicyMembers(models.DB, false)

	// TODO check if it was denied by the signator and if so, email the steward
	// TODO figure out how to notify the members of the reason for the denial

	data := newEmailMessageData()
	data.addClaimData(claim)

	notn := models.Notification{
		ClaimID:   nulls.NewUUID(claim.ID),
		Body:      data.renderHTML(MessageTemplateClaimDeniedMember),
		Subject:   "your claim has been denied",
		InappText: "your claim has been denied",

		// TODO make these constants somewhere
		Event:         "Claim Denied Notification",
		EventCategory: "Claim",
	}
	if err := notn.Create(tx); err != nil {
		panic("error creating new Claim Denied Notification: " + err.Error())
	}

	for _, m := range claim.Policy.Members {
		notn.CreateNotificationUser(tx, m)
	}
}
