package messages

import (
	"bytes"
	"fmt"
	"strings"
	"time"

	"github.com/gobuffalo/buffalo/render"
	"github.com/gobuffalo/nulls"
	"github.com/gobuffalo/pop/v5"

	"github.com/silinternational/cover-api/api"
	"github.com/silinternational/cover-api/domain"
	"github.com/silinternational/cover-api/models"
	"github.com/silinternational/cover-api/notifications"
)

const Greetings_Placeholder = "[Greetings]"

//  Email templates
const (
	MessageTemplateClaimReview1Steward    = "claim_review1_steward"
	MessageTemplateClaimRevisionMember    = "claim_revision_member"
	MessageTemplateClaimPreapprovedMember = "claim_preapproved_member"
	MessageTemplateClaimReceiptMember     = "claim_receipt_member"
	MessageTemplateClaimReview2Steward    = "claim_review2_steward"
	MessageTemplateClaimReview3Signator   = "claim_review3_signator"
	MessageTemplateClaimApprovedMember    = "claim_approved_member"
	MessageTemplateClaimDeniedMember      = "claim_denied_member"

	MessageTemplateItemPendingSteward = "item_pending_steward"
	MessageTemplateItemApprovedMember = "item_approved_member"
	MessageTemplateItemAutoSteward    = "item_auto_approved_steward"
	MessageTemplateItemRevisionMember = "item_revision_member"
	MessageTemplateItemDeniedMember   = "item_denied_member"

	MessageTemplatePolicyUserInvite = "policy_user_invite"
	MessageTemplateUserWelcome      = "user_welcome"
)

// blockSending is used to avoid having duplicate emails sent out when
// two notifications are created one after the other.
var blockSending bool

func unBlockSending() {
	blockSending = false
}

type MessageData render.Data

func newEmailMessageData() MessageData {
	m := MessageData{}

	m["uiURL"] = domain.Env.UIURL
	m["appName"] = domain.Env.AppName
	m["premiumPercentage"] = fmt.Sprintf("%.2g%%", domain.Env.PremiumFactor*100)

	return m
}

func (m MessageData) addClaimData(tx *pop.Connection, claim models.Claim) {
	if m == nil {
		m = map[string]interface{}{}
	}

	m.addStewardData(tx)

	m["claimURL"] = fmt.Sprintf("%s/policies/%s/claims/%s", domain.Env.UIURL, claim.PolicyID, claim.ID)
	m["claim"] = claim

	m["incidentDate"] = claim.IncidentDate.Format(domain.LocalizedDate)
	m["incidentType"] = string(claim.IncidentType)
	m["incidentTypeDescription"] = claim.IncidentType.Description()

	claim.LoadClaimItems(tx, false)
	item := claim.ClaimItems[0].Item
	m["item"] = item
	m["coverageAmount"] = "$" + api.Currency(item.CoverageAmount).String()

	item.LoadPolicy(tx, false)
	m["policy"] = item.Policy

	person := item.GetAccountablePersonName(tx)
	m["accountablePerson"] = person.String()
	m["personFirstName"] = person.First

	m["payoutOption"] = string(claim.ClaimItems[0].PayoutOption)
	m["payoutOptionLower"] = strings.ToLower(string(claim.ClaimItems[0].PayoutOption))
	m["totalPayout"] = "$" + claim.TotalPayout.String()
	m["submitted"] = domain.TimeBetween(time.Now().UTC(), claim.SubmittedAt(tx))
}

func (m MessageData) addItemData(tx *pop.Connection, item models.Item) {
	if m == nil {
		m = map[string]interface{}{}
	}
	m.addStewardData(tx)

	m["itemURL"] = fmt.Sprintf("%s/policies/%s/items/%s", domain.Env.UIURL, item.PolicyID, item.ID)

	item.Load(tx)
	item.LoadPolicy(tx, false)
	m["item"] = item

	person := item.GetAccountablePersonName(tx)
	m["accountablePerson"] = person.String()
	m["personFirstName"] = person.First

	m["policy"] = item.Policy
	m["policyType"] = string(item.Policy.Type)

	m["coverageAmount"] = "$" + api.Currency(item.CoverageAmount).String()
	m["coverageStartDate"] = item.CoverageStartDate.Format(domain.LocalizedDate)
	if item.CoverageEndDate.Valid {
		m["coverageEndDate"] = item.CoverageEndDate.Time.Format(domain.LocalizedDate)
	} else {
		m["coverageEndDate"] = ""
	}
	m["annualPremium"] = "$" + item.CalculateAnnualPremium().String()
	m["proratedPremium"] = "$" + item.CalculateProratedPremium(item.CoverageStartDate).String()
}

func (m MessageData) addStewardData(tx *pop.Connection) {
	if m == nil {
		m = map[string]interface{}{}
	}

	steward := models.GetDefaultSteward(tx)
	m["supportEmail"] = domain.Env.SupportEmail
	m["supportName"] = steward.Name()
	m["supportFirstName"] = steward.FirstName
}

func (m MessageData) renderHTML(template string) string {
	bodyBuf := &bytes.Buffer{}
	data := render.Data(m)
	if err := notifications.EmailRenderer.HTML(template).Render(bodyBuf, data); err != nil {
		panic("error rendering message body - " + err.Error())
	}
	return bodyBuf.String()
}

func SendQueuedNotifications(tx *pop.Connection) {
	// Wait up to two minutes to see if it's OK to try sending emails
	for i := 0; i < 24; i++ {
		if !blockSending {
			break
		}
		time.Sleep(5 * time.Second)
	}

	blockSending = true
	defer unBlockSending()

	var notnUsers models.NotificationUsers
	if err := notnUsers.GetEmailsToSend(tx); err != nil {
		panic(err.Error())
	}

	for _, n := range notnUsers {
		n.Load(tx)
		userName := n.ToName
		if n.UserID.Valid {
			userName = n.User.Name()
		}

		msg := notifications.NewEmailMessage()

		msg.ToName = userName
		msg.ToEmail = n.EmailAddress
		msg.Subject = n.Notification.Subject
		msg.Body = strings.Replace(n.Notification.Body,
			Greetings_Placeholder, fmt.Sprintf("Greetings %s,", userName), 1)

		if err := notifications.Send(msg); err != nil {
			domain.ErrLogger.Printf("error sending queued notification email, %s", err)
			n.LastAttemptUTC = nulls.NewTime(time.Now().UTC())
			n.SendAfterUTC = nextAttemptTime(n.SendAttemptCount)
			n.SendAttemptCount++
		} else {
			n.SentAtUTC = nulls.NewTime(time.Now().UTC())
		}
		if err := n.Update(tx); err != nil {
			domain.ErrLogger.Printf("error updating queued NotificationUser, %s", err)
		}
	}
}

func nextAttemptTime(attemptCount int) time.Time {
	delayMinutes := 100
	if attemptCount < 10 {
		delayMinutes = attemptCount * attemptCount
	}

	delay := time.Duration(delayMinutes) * time.Minute
	return time.Now().UTC().Add(delay)
}
