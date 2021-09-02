package notifications

import (
	"text/template"

	"github.com/silinternational/cover-api/domain"
)

func (ts *TestSuite) TestSend() {
	nickname := "nickname"
	msg := Message{
		FromName:  "from name",
		FromEmail: domain.EmailFromAddress(&nickname),
		ToName:    "to name",
		ToEmail:   "to@example.com",
		Template:  domain.MessageTemplateItemSubmitted,
		Data: map[string]interface{}{
			"uiURL":          "example.com",
			"appName":        "Our App",
			"itemURL":        "my-item.example.com",
			"itemName":       "My Item",
			"itemMemberName": "John Doe",
			"supportEmail":   "support@example.com",
		},
	}
	var emailService EmailService
	testService := NewDummyEmailService()
	emailService = &testService

	err := emailService.Send(msg)
	ts.NoError(err, "error sending message")

	n := len(testService.GetSentMessages())
	ts.Require().Equal(1, n, "incorrect number of messages sent")

	body := testService.GetLastBody()

	ts.Contains(body, template.HTMLEscapeString(msg.Data["itemURL"].(string)))
	ts.NotContains(body, "<script>")
}
