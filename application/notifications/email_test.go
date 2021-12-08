package notifications

import (
	"text/template"

	"github.com/gobuffalo/nulls"

	"github.com/silinternational/cover-api/domain"
	"github.com/silinternational/cover-api/models"
)

func (ts *TestSuite) TestSend() {
	item := models.Item{
		Name: "My Item",
	}

	nickname := "nickname"
	msg := Message{
		FromName:  "from name",
		FromEmail: domain.EmailFromAddress(&nickname),
		ToName:    "to name",
		ToEmail:   "to@example.com",
		Template:  "item_pending_steward",
		Data: map[string]interface{}{
			"uiURL":             "example.com",
			"appName":           "Our App",
			"buttonLabel":       "Open in Our App",
			"itemURL":           "https://my-item.example.com",
			"item":              item,
			"memberName":        "John Doe",
			"supportEmail":      "support@example.com",
			"coverageAmount":    "$100.00",
			"coverageEndDate":   "2021-12-31",
			"coverageStartDate": "2021-01-01",
			"annualPremium":     "$3.50",
			"accountablePerson": "John Doe",
			"policy":            models.Policy{HouseholdID: nulls.NewString("007")},
		},
	}
	var emailService EmailService
	var testService DummyEmailService
	emailService = &testService

	err := emailService.Send(msg)
	ts.NoError(err, "error sending message")

	n := len(testService.GetSentMessages())
	ts.Require().Equal(1, n, "incorrect number of messages sent")

	body := testService.GetLastBody()

	ts.Contains(body, template.HTMLEscapeString(msg.Data["itemURL"].(string)))
	ts.NotContains(body, "<script>")
}
