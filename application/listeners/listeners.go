package listeners

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gobuffalo/events"
	"github.com/gobuffalo/nulls"
	"github.com/gobuffalo/pop/v5"
	"github.com/gofrs/uuid"

	"github.com/silinternational/cover-api/domain"
	"github.com/silinternational/cover-api/messages"
	"github.com/silinternational/cover-api/models"
)

const EventPayloadNotifier = "notifier"

var eventTypes = map[string]func(event events.Event){
	domain.EventApiItemAutoApproved:        itemAutoApproved,
	domain.EventApiUserCreated:             createUserPolicy,
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
	models.DB.Transaction(func(tx *pop.Connection) error {
		messages.SendQueuedNotifications(tx)
		return nil
	})
}

func listener(e events.Event) {
	if err := recover(); err != nil {
		domain.Logger.Printf("panic occurred in %s: %s", e.Kind, err)
	}

	handler, ok := eventTypes[e.Kind]
	if !ok {
		panic("event '" + e.Kind + "' has no handler")
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

// getDelayDuration is a helper function to calculate delay in milliseconds before processing event
func getDelayDuration(multiplier int) time.Duration {
	return time.Duration(domain.Env.ListenerDelayMilliseconds) * time.Millisecond * time.Duration(multiplier)
}

func GetHHID(staffID string) string {
	if domain.Env.HouseholdIDLookupURL == "" {
		return ""
	}

	req, err := http.NewRequest(http.MethodGet, domain.Env.HouseholdIDLookupURL+staffID, nil)
	if err != nil {
		domain.ErrLogger.Printf("HHID API error, %s", err)
		return ""
	}
	req.SetBasicAuth(domain.Env.HouseholdIDLookupUsername, domain.Env.HouseholdIDLookupPassword)

	client := &http.Client{Timeout: time.Second * 10}
	response, err := client.Do(req)
	if err != nil {
		domain.ErrLogger.Printf("HHID API error, %s", err)
		return ""
	}
	defer response.Body.Close()

	dec := json.NewDecoder(response.Body)
	var v struct {
		ID string `json:"householdIdOut"`
	}
	if err = dec.Decode(&v); err != nil {
		domain.ErrLogger.Printf("HHID API error decoding response, %s", err)
		return ""
	}
	return v.ID
}
