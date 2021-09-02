package notifications

import (
	"bytes"
	"errors"
	"time"

	"github.com/silinternational/cover-api/domain"
)

var sentMessages = map[int][]dummyMessage{}

type DummyEmailService struct {
	timestamp int
}

func NewDummyEmailService() DummyEmailService {
	return DummyEmailService{timestamp: time.Now().Nanosecond()}
}

var TestEmailService DummyEmailService

type dummyMessage struct {
	subject, body, fromName, fromEmail, toName, toEmail string
}

type DummyMessageInfo struct {
	Subject, ToName, ToEmail string
}

func (t DummyEmailService) Send(msg Message) error {
	eTemplate := msg.Template
	bodyBuf := &bytes.Buffer{}
	if err := eR.HTML(eTemplate).Render(bodyBuf, msg.Data); err != nil {
		errMsg := "error rendering message body - " + err.Error()
		domain.ErrLogger.Printf(errMsg)
		return errors.New(errMsg)
	}

	domain.Logger.Printf("dummy message subject: %s, recipient: %s",
		msg.Subject, msg.ToName)

	sentMsgs := sentMessages[t.timestamp]
	sentMsgs = append(sentMsgs, dummyMessage{
		subject:   msg.Subject,
		body:      bodyBuf.String(),
		fromName:  msg.FromName,
		fromEmail: msg.FromEmail,
		toName:    msg.ToName,
		toEmail:   msg.ToEmail,
	})
	sentMessages[t.timestamp] = sentMsgs
	return nil
}

// GetNumberOfMessagesSent returns the number of messages sent since initialization or the last call to
// DeleteSentMessages
func (t *DummyEmailService) GetNumberOfMessagesSent() int {
	return len(sentMessages[t.timestamp])
}

// DeleteSentMessages erases the store of sent messages
func (t *DummyEmailService) DeleteSentMessages() {
	sentMessages[t.timestamp] = []dummyMessage{}
}

func (t *DummyEmailService) GetLastToEmail() string {
	sentMsgs := sentMessages[t.timestamp]
	if len(sentMsgs) == 0 {
		return ""
	}

	return sentMsgs[len(sentMsgs)-1].toEmail
}

func (t *DummyEmailService) GetToEmailByIndex(i int) string {
	sentMsgs := sentMessages[t.timestamp]
	if len(sentMsgs) <= i {
		return ""
	}

	return sentMsgs[i].toEmail
}

func (t *DummyEmailService) GetAllToAddresses() []string {
	sentMsgs := sentMessages[t.timestamp]
	emailAddresses := make([]string, len(sentMsgs))
	for i := range sentMsgs {
		emailAddresses[i] = sentMsgs[i].toEmail
	}
	return emailAddresses
}

func (t *DummyEmailService) GetLastBody() string {
	sentMsgs := sentMessages[t.timestamp]
	if len(sentMsgs) == 0 {
		return ""
	}

	return sentMsgs[len(sentMsgs)-1].body
}

func (t *DummyEmailService) GetSentMessages() []DummyMessageInfo {
	sentMsgs := sentMessages[t.timestamp]
	messages := make([]DummyMessageInfo, len(sentMsgs))
	for i, m := range sentMsgs {
		messages[i] = DummyMessageInfo{
			Subject: m.subject,
			ToName:  m.toName,
			ToEmail: m.toEmail,
		}
	}
	return messages
}
