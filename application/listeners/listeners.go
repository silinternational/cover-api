package listeners

import (
	"errors"
	"fmt"
	"time"

	"github.com/gobuffalo/events"
	"github.com/gobuffalo/nulls"
	"github.com/gofrs/uuid"

	"github.com/silinternational/cover-api/domain"
	"github.com/silinternational/cover-api/models"
)

type apiListener struct {
	name     string
	listener func(events.Event)
}

//
// Register new listener functions here.  Remember, though, that these groupings just
// describe what we want.  They don't make it happen this way. The listeners
// themselves still need to verify the event kind
//
var apiListeners = map[string][]apiListener{
	domain.EventApiUserCreated: {
		{
			name:     "user-created-create-policy",
			listener: createUserPolicy,
		},
	},
	domain.EventApiItemSubmitted: {
		{
			name:     "item-submitted",
			listener: itemSubmitted,
		},
	},
	domain.EventApiItemRevision: {
		{
			name:     "item-revision",
			listener: itemRevision,
		},
	},
	domain.EventApiItemApproved: {
		{
			name:     "item-approved",
			listener: itemApproved,
		},
	},
	domain.EventApiItemDenied: {
		{
			name:     "item-denied",
			listener: itemDenied,
		},
	},
}

// RegisterListeners registers all the listeners to be used by the app
func RegisterListeners() {
	for _, listeners := range apiListeners {
		for _, l := range listeners {
			_, err := events.NamedListen(l.name, l.listener)
			if err != nil {
				domain.ErrLogger.Printf("Failed registering listener: %s, err: %s", l.name, err.Error())
			}
		}
	}
}

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

func getID(p events.Payload) (uuid.UUID, error) {
	i, ok := p[domain.EventPayloadID]
	if !ok {
		return uuid.UUID{}, fmt.Errorf("id not in event payload")
	}

	switch id := i.(type) {
	case string:
		return uuid.FromStringOrNil(id), nil
	case uuid.UUID:
		return id, nil
	case nulls.UUID:
		return id.UUID, nil
	default:
		return uuid.UUID{}, fmt.Errorf("id not a valid type: %T", id)
	}
}

func findObject(payload events.Payload, object interface{}, listenerName string) error {
	id, err := getID(payload)
	if err != nil {
		err := errors.New("Failed to get object ID from event payload: " + err.Error())
		domain.ErrLogger.Printf(err.Error())
		return err
	}

	foundObject := false
	var findErr error

	i := 1
	for ; i <= domain.Env.ListenerMaxRetries; i++ {
		findErr = models.DB.Find(object, id)
		if findErr == nil {
			foundObject = true
			break
		}
		time.Sleep(getDelayDuration(i * i))
		if i > 3 {
			return findErr
		}
	}

	if !foundObject {
		err := fmt.Errorf("Failed to find object in %s, %s", listenerName, findErr)
		domain.ErrLogger.Printf("Failed to find object in %s, %s", listenerName, findErr)
		return err
	}
	return nil
}

func panicRecover(name string) {
	if err := recover(); err != nil {
		domain.Logger.Printf("panic occurred in %s: %s", name, err)
	}
}

// getDelayDuration is a helper function to calculate delay in milliseconds before processing event
func getDelayDuration(multiplier int) time.Duration {
	return time.Duration(domain.Env.ListenerDelayMilliseconds) * time.Millisecond * time.Duration(multiplier)
}
