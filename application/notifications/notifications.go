package notifications

import (
	"github.com/silinternational/cover-api/domain"
)

var notifiers []Notifier

func init() {
	email := EmailNotifier{} // The type of sender is determined by domain.Env.EmailService
	notifiers = append(notifiers, &email)
}

func Send(msg Message) error {
	for _, n := range notifiers {
		if err := n.Send(msg); err != nil {
			return err
		}
		domain.Logger.Printf("%T: %s message sent", n, msg.Template)
	}

	return nil
}
