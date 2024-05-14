package listeners

import (
	"fmt"
	"strings"
	"time"

	"github.com/gobuffalo/events"
	"github.com/gobuffalo/nulls"
	"github.com/gobuffalo/pop/v6"
	"github.com/gofrs/uuid"

	"github.com/silinternational/cover-api/domain"
	"github.com/silinternational/cover-api/log"
	"github.com/silinternational/cover-api/messages"
	"github.com/silinternational/cover-api/models"
)

const EventPayloadNotifier = "notifier"

var eventTypes = map[string]func(event events.Event){
	domain.EventApiItemAutoApproved:        itemAutoApproved,
	domain.EventApiUserCreated:             userCreated,
	domain.EventApiItemSubmitted:           itemSubmitted,
	domain.EventApiItemRevision:            itemRevision,
	domain.EventApiItemApproved:            itemApproved,
	domain.EventApiItemDenied:              itemDenied,
	domain.EventApiClaimReview1:            claimReview1,
	domain.EventApiClaimRevision:           claimRevision,
	domain.EventApiClaimPreapproved:        claimPreapproved,
	domain.EventApiClaimReceipt:            claimReceipt,
	domain.EventApiClaimReview2:            claimReview2,
	domain.EventApiClaimReview3:            claimReview3,
	domain.EventApiClaimApproved:           claimApproved,
	domain.EventApiClaimDenied:             claimDenied,
	domain.EventApiNotificationCreated:     notificationCreated,
	domain.EventApiPolicyUserInviteCreated: policyUserInviteCreated,
}

func notificationCreated(e events.Event) {
	_ = models.DB.Transaction(func(tx *pop.Connection) error {
		messages.SendQueuedNotifications(tx)
		return nil
	})
}

func listener(e events.Event) {
	defer func() {
		if err := recover(); err != nil {
			log.Errorf("panic in event %s: %s", e.Kind, err)
		}
	}()

	handler, ok := eventTypes[e.Kind]
	if !ok {
		if strings.HasPrefix(e.Kind, "app") {
			panic("event '" + e.Kind + "' has no handler")
		}
		return
	}

	time.Sleep(time.Second * 5) // a rough guess at the longest time it takes for the database transaction to close

	handler(e)
}

// RegisterListener registers the event listener
func RegisterListener() {
	if _, err := events.Listen(listener); err != nil {
		panic("failed to register event listener " + err.Error())
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
		if id.Valid {
			return id.UUID, nil
		}
		return uuid.UUID{}, fmt.Errorf("id is not valid")
	default:
		return uuid.UUID{}, fmt.Errorf("id not a valid type: %T", id)
	}
}

func findObject(payload events.Payload, object any, listenerName string) error {
	id, err := getID(payload)
	if err != nil {
		err := fmt.Errorf("failed to get object ID from event payload: %w", err)
		log.Error(err)
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
		err := fmt.Errorf("failed to find object in %s, %w", listenerName, findErr)
		log.Error(err)
		return err
	}
	return nil
}

// getDelayDuration is a helper function to calculate delay in milliseconds before processing event
func getDelayDuration(multiplier int) time.Duration {
	return time.Duration(domain.Env.ListenerDelayMilliseconds) * time.Millisecond * time.Duration(multiplier)
}
