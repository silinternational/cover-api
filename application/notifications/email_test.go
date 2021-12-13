package notifications

import (
	"github.com/silinternational/cover-api/domain"
)

func (ts *TestSuite) TestSend() {
	nickname := "nickname"
	const body = "This is the message body."
	msg := Message{
		FromName:  "from name",
		FromEmail: domain.EmailFromAddress(&nickname),
		ToName:    "to name",
		ToEmail:   "to@example.com",
		Template:  "item_pending_steward",
		Body:      body,
	}
	var emailService EmailService
	var testService DummyEmailService
	emailService = &testService

	err := emailService.Send(msg)
	ts.NoError(err, "error sending message")

	n := len(testService.GetSentMessages())
	ts.Require().Equal(1, n, "incorrect number of messages sent")

	ts.Equal(body, testService.GetLastBody())
}
