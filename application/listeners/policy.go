package listeners

import (
	"github.com/gobuffalo/events"

	"github.com/silinternational/cover-api/domain"
	"github.com/silinternational/cover-api/models"
)

func createUserPolicy(e events.Event) {
	if e.Kind != domain.EventApiUserCreated {
		return
	}

	defer panicRecover(e.Kind)

	var user models.User
	if err := findObject(e.Payload, &user, e.Kind); err != nil {
		return
	}

	if err := user.CreateInitialPolicy(nil); err != nil {
		domain.ErrLogger.Printf("Failed to create initial policy in %s, %s", e.Kind, err)
		return
	}
}
