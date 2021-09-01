package notifications

import (
	"github.com/silinternational/cover-api/domain"
)

const (
	EmailServiceSES   = "ses"
	EmailServiceDummy = "dummy"
)

// Notifier is an abstraction layer for multiple types of notifications: email, mobile, and push (TBD).
type Notifier interface {
	Send(msg Message) error
}

// EmailNotifier is an email notifier that conforms to the Notifier interface.
type EmailNotifier struct{}

// Send a notification using an email notifier.
func (e *EmailNotifier) Send(msg Message) error {
	var emailService EmailService

	emailServiceType := domain.Env.EmailService
	switch emailServiceType {
	case EmailServiceDummy:
		emailService = &TestEmailService
	case EmailServiceSES:
		emailService = &SES{}
	default:
		emailService = &TestEmailService
	}

	emailMessage := Message{
		FromName:  msg.FromName,
		FromEmail: msg.FromEmail,
		ToName:    msg.ToName,
		ToEmail:   msg.ToEmail,
		Template:  msg.Template,
		Data:      msg.Data,
		Subject:   msg.Subject,
	}

	return emailService.Send(emailMessage)
}
