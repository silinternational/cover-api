package notifications

import "github.com/silinternational/cover-api/log"

var notifiers []Notifier

func init() {
	email := EmailNotifier{} // The type of sender is determined by domain.Env.EmailService
	notifiers = append(notifiers, &email)
}

// Send loops through the notifiers and calls each of their Send functions
func Send(msg Message) error {
	for _, n := range notifiers {
		if err := n.Send(msg); err != nil {
			return err
		}
		log.Errorf("%T: '%s' message sent to '%s'", n, msg.Subject, msg.ToEmail)
	}

	return nil
}
