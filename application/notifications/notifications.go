package notifications

import (
	"github.com/silinternational/cover-api/domain"
)

var notifiers []Notifier

func init() {
	email := EmailNotifier{} // The type of sender is determined by domain.Env.EmailService
	notifiers = append(notifiers, &email)
}

// Send loops through the default notifiers (or custom ones if they are provided)
//  and calls each of their Send functions
func Send(msg Message, customNotifiers ...interface{}) error {
	notrs := []Notifier{}

	for _, n := range customNotifiers {
		notr, ok := n.(Notifier)
		if ok {
			notrs = append(notrs, notr)
		}
	}

	if len(notrs) == 0 {
		notrs = notifiers
	}

	for _, n := range notrs {
		if err := n.Send(msg); err != nil {
			return err
		}
		domain.Logger.Printf("%T: %s message sent", n, msg.Template)
	}

	return nil
}
