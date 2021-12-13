package notifications

import (
	"errors"

	"github.com/silinternational/cover-api/domain"
)

type DummyEmailService struct {
	sentMessages []dummyMessage
}

var TestEmailService DummyEmailService

type dummyMessage struct {
	subject, body, fromName, fromEmail, toName, toEmail string
}

type DummyMessageInfo struct {
	Body, Subject, ToName, ToEmail string
}

func (t *DummyEmailService) Send(msg Message) error {
	body := msg.Body

	if body == "ERROR" {
		return errors.New("mock error for testing sending email message")
	}

	domain.Logger.Printf("dummy message subject: %s, recipient: %s",
		msg.Subject, msg.ToName)

	t.sentMessages = append(t.sentMessages,
		dummyMessage{
			subject:   msg.Subject,
			body:      body,
			fromName:  msg.FromName,
			fromEmail: msg.FromEmail,
			toName:    msg.ToName,
			toEmail:   msg.ToEmail,
		})
	return nil
}

// GetNumberOfMessagesSent returns the number of messages sent since initialization or the last call to
// DeleteSentMessages
func (t *DummyEmailService) GetNumberOfMessagesSent() int {
	return len(t.sentMessages)
}

// DeleteSentMessages erases the store of sent messages
func (t *DummyEmailService) DeleteSentMessages() {
	t.sentMessages = []dummyMessage{}
}

func (t *DummyEmailService) GetLastToEmail() string {
	if len(t.sentMessages) == 0 {
		return ""
	}

	return t.sentMessages[len(t.sentMessages)-1].toEmail
}

func (t *DummyEmailService) GetToEmailByIndex(i int) string {
	if len(t.sentMessages) <= i {
		return ""
	}

	return t.sentMessages[i].toEmail
}

func (t *DummyEmailService) GetAllToAddresses() []string {
	emailAddresses := make([]string, len(t.sentMessages))
	for i := range t.sentMessages {
		emailAddresses[i] = t.sentMessages[i].toEmail
	}
	return emailAddresses
}

func (t *DummyEmailService) GetLastBody() string {
	if len(t.sentMessages) == 0 {
		return ""
	}

	return t.sentMessages[len(t.sentMessages)-1].body
}

func (t *DummyEmailService) GetSentMessages() []DummyMessageInfo {
	messages := make([]DummyMessageInfo, len(t.sentMessages))
	for i, m := range t.sentMessages {
		messages[i] = DummyMessageInfo{
			Body:    m.body,
			Subject: m.subject,
			ToName:  m.toName,
			ToEmail: m.toEmail,
		}
	}
	return messages
}
