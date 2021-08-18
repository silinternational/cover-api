package listeners

import (
	"fmt"
	"time"

	"github.com/gobuffalo/events"
	"github.com/gobuffalo/nulls"
	"github.com/gofrs/uuid"

	"github.com/silinternational/riskman-api/domain"
	"github.com/silinternational/riskman-api/models"
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

	defer panicRecover("createUserPolicy")

	userID, err := getID(e.Payload)
	if err != nil {
		domain.ErrLogger.Printf("Failed to get User ID from event payload, %s", err)
		return
	}

	foundUser := false
	var user models.User
	var findErr error

	i := 1
	for ; i <= domain.Env.ListenerMaxRetries; i++ {
		findErr = models.DB.Find(&user, userID)
		if findErr == nil {
			foundUser = true
			break
		}
		time.Sleep(getDelayDuration(i * i))
	}
	domain.Logger.Printf("listener createUserPolicy required %d retries with delay %d", i-1, domain.Env.ListenerDelayMilliseconds)

	if !foundUser {
		domain.ErrLogger.Printf("Failed to find User in createUserPolicy, %s", findErr)
		return
	}

	if err := user.CreateInitialPolicy(nil); err != nil {
		domain.ErrLogger.Printf("Failed to create initial policy in createUserPolicy, %s", err)
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

func panicRecover(name string) {
	if err := recover(); err != nil {
		domain.Logger.Printf("panic occurred in %s: %s", name, err)
	}
}

// getDelayDuration is a helper function to calculate delay in milliseconds before processing event
func getDelayDuration(multiplier int) time.Duration {
	return time.Duration(domain.Env.ListenerDelayMilliseconds) * time.Millisecond * time.Duration(multiplier)
}
