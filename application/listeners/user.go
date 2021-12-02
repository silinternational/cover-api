package listeners

import (
	"github.com/gobuffalo/events"
	"github.com/gobuffalo/pop/v5"

	"github.com/silinternational/cover-api/domain"
	"github.com/silinternational/cover-api/messages"
	"github.com/silinternational/cover-api/models"
)

func userCreated(e events.Event) {
	var user models.User
	if err := findObject(e.Payload, &user, e.Kind); err != nil {
		return
	}

	var householdID string
	if user.StaffID.Valid {
		householdID = GetHHID(user.StaffID.String)
	}

	if err := user.CreateInitialPolicy(nil, householdID); err != nil {
		domain.ErrLogger.Printf("Failed to create initial policy in %s, %s", e.Kind, err)
		return
	}

	userWelcome(e)
}

func userWelcome(e events.Event) {
	var user models.User
	if err := findObject(e.Payload, &user, e.Kind); err != nil {
		return
	}

	models.DB.Transaction(func(tx *pop.Connection) error {
		messages.UserWelcomeQueueMessage(tx, user)
		return nil
	})
}
