package models

import (
	"fmt"
	"time"

	"github.com/gobuffalo/events"
	"github.com/gobuffalo/nulls"
	"github.com/gobuffalo/pop/v6"
	"github.com/gobuffalo/validate/v3"
	"github.com/gofrs/uuid"

	"github.com/silinternational/cover-api/domain"
)

type Notifications []Notification

type Notification struct {
	ID            uuid.UUID  `db:"id"`
	PolicyID      nulls.UUID `db:"policy_id"`
	ItemID        nulls.UUID `db:"item_id"`
	ClaimID       nulls.UUID `db:"claim_id"`
	Event         string     `db:"event"`
	EventCategory string     `db:"event_category"`
	Subject       string     `db:"subject"` // validation is checked at the struct level
	InappText     string     `db:"inapp_text"`
	Body          string     `db:"body"` // validation is checked at the struct level
	CreatedAt     time.Time  `db:"created_at"`
	UpdatedAt     time.Time  `db:"updated_at"`

	Policy Policy `belongs_to:"policies" validate:"-"`
	Item   Item   `belongs_to:"items" validate:"-"`
	Claim  Claim  `belongs_to:"claims" validate:"-"`
}

// Validate gets run every time you call a "pop.Validate*" (pop.ValidateAndSave, pop.ValidateAndCreate, pop.ValidateAndUpdate) method.
func (n *Notification) Validate(tx *pop.Connection) (*validate.Errors, error) {
	return validateModel(n), nil
}

// Create stores the Notification data as a new record in the database.
func (n *Notification) Create(tx *pop.Connection) error {
	if err := create(tx, n); err != nil {
		return err
	}

	if err := events.Emit(events.Event{Kind: domain.EventApiNotificationCreated}); err != nil {
		domain.ErrLogger.Printf("error emitting event %s ... %v", domain.EventApiNotificationCreated, err)
	}

	return nil
}

// Update writes the Notification data to an existing database record.
func (n *Notification) Update(tx *pop.Connection) error {
	return update(tx, n)
}

func (n *Notification) GetID() uuid.UUID {
	return n.ID
}

func (n *Notification) FindByID(tx *pop.Connection, id uuid.UUID) error {
	return tx.Find(n, id)
}

// LoadPolicy - a simple wrapper method for loading the policy on the struct
func (n *Notification) LoadPolicy(tx *pop.Connection, reload bool) {
	if n.PolicyID.Valid && (n.Policy.ID == uuid.Nil || reload) {
		if err := tx.Load(n, "Policy"); err != nil {
			panic("database error loading Notification.Policy, " + err.Error())
		}
	}
}

// LoadItem - a simple wrapper method for loading the item on the struct
func (n *Notification) LoadItem(tx *pop.Connection, reload bool) {
	if n.ItemID.Valid && (n.Item.ID == uuid.Nil || reload) {
		if err := tx.Load(n, "Item"); err != nil {
			panic("database error loading Notification.Item, " + err.Error())
		}
	}
}

// LoadClaim - a simple wrapper method for loading the claim on the struct
func (n *Notification) LoadClaim(tx *pop.Connection, reload bool) {
	if n.ClaimID.Valid && (n.Claim.ID == uuid.Nil || reload) {
		if err := tx.Load(n, "Claim"); err != nil {
			panic("database error loading Notification.Claim, " + err.Error())
		}
	}
}

func (n *Notification) CreateNotificationUser(tx *pop.Connection, userID nulls.UUID, emailAddress, toName string) {
	notnUser := NotificationUser{
		NotificationID: n.ID,
		UserID:         userID,
		EmailAddress:   emailAddress,
		ToName:         toName,
		SendAfterUTC:   time.Now().UTC(),
	}

	if err := notnUser.Create(tx); err != nil {
		panic(fmt.Sprintf("error creating new NotificationUser with UserID %s: %s",
			userID.UUID, err.Error()))
	}
}

func (n *Notification) CreateNotificationUserForUser(tx *pop.Connection, user User) {
	n.CreateNotificationUser(tx, nulls.NewUUID(user.ID), user.EmailOfChoice(), "")
}

func (n *Notification) CreateNotificationUsersForStewards(tx *pop.Connection) {
	var stewards Users
	stewards.FindStewards(tx)

	for _, s := range stewards {
		n.CreateNotificationUser(tx, nulls.NewUUID(s.ID), s.EmailOfChoice(), "")
	}
}

func (n *Notification) CreateNotificationUsersForSignators(tx *pop.Connection) {
	var signators Users
	signators.FindSignators(tx)

	for _, s := range signators {
		n.CreateNotificationUser(tx, nulls.NewUUID(s.ID), s.EmailOfChoice(), "")
	}
}
