package listeners

import (
	"github.com/gobuffalo/events"
	"github.com/gobuffalo/pop/v5"

	"github.com/silinternational/cover-api/domain"
	"github.com/silinternational/cover-api/messages"
	"github.com/silinternational/cover-api/models"
)

func createUserPolicy(e events.Event) {
	var user models.User
	if err := findObject(e.Payload, &user, e.Kind); err != nil {
		return
	}

	if err := user.CreateInitialPolicy(nil); err != nil {
		domain.ErrLogger.Printf("Failed to create initial policy in %s, %s", e.Kind, err)
		return
	}
}

func policyUserInviteCreated(e events.Event) {
	var invite models.PolicyUserInvite
	if err := findObject(e.Payload, &invite, e.Kind); err != nil {
		return
	}

	err := models.DB.Transaction(func(tx *pop.Connection) error {
		messages.PolicyUserInviteQueueMessage(tx, invite)
		return nil
	})
	if err != nil {
		domain.ErrLogger.Printf("error queuing policy user invite: %s", err)
	}
}
