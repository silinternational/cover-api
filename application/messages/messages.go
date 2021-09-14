package messages

import (
	"bytes"
	"fmt"
	"time"

	"github.com/gobuffalo/buffalo/render"
	"github.com/gobuffalo/nulls"
	"github.com/gobuffalo/pop/v5"

	"github.com/silinternational/cover-api/domain"
	"github.com/silinternational/cover-api/models"
	"github.com/silinternational/cover-api/notifications"
)

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

	MessageTemplateItemSubmittedSteward = "item_submitted_steward"
	MessageTemplateItemApprovedMember   = "item_approved_member"
	MessageTemplateItemAutoSteward      = "item_auto_approved_steward"
	MessageTemplateItemRevisionMember   = "item_revision_member"
	MessageTemplateItemDeniedMember     = "item_denied_member"
)

type MessageData render.Data

func newEmailMessageData() MessageData {
	m := MessageData{}

	m["uiURL"] = domain.Env.UIURL
	m["appName"] = domain.Env.AppName

	return m
}

func (m MessageData) addItemData(item models.Item) {
	if m == nil {
		m = map[string]interface{}{}
	}

	m["itemURL"] = fmt.Sprintf("%s/items/%s", domain.Env.UIURL, item.ID)
	m["itemName"] = item.Name
	return
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
	var notnUsers models.NotificationUsers
	if err := notnUsers.GetQueuedEmails(tx); err != nil {
		panic(err.Error())
	}

	for _, n := range notnUsers {
		n.Load(tx)
		msg := notifications.NewEmailMessage()
		msg.ToName = n.User.Name()
		msg.ToEmail = n.EmailAddress
		msg.Body = n.Notification.Body
		msg.Subject = n.Notification.Subject

		if err := notifications.Send(msg); err != nil {
			domain.ErrLogger.Printf("error sending queued notification email, %s", err)
			// TODO figure out how to have a longer delay each time
			n.SendAfterUTC = time.Now().UTC().Add(time.Minute * 10)
			n.LastAttemptUTC = nulls.NewTime(time.Now().UTC())
			n.SendAttemptCount++
		} else {
			n.SentAtUTC = nulls.NewTime(time.Now().UTC())
		}
		if err := n.Update(tx); err != nil {
			domain.ErrLogger.Printf("error updating queued NotificationUser, %s", err)
		}
	}
}
