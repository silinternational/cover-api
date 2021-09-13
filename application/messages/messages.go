package messages

import (
	"bytes"
	"fmt"

	"github.com/gobuffalo/buffalo/render"

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
