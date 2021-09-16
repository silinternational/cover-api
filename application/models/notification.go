package models

import (
	"fmt"
	"time"

	"github.com/gobuffalo/nulls"
	"github.com/gobuffalo/pop/v5"
	"github.com/gobuffalo/validate/v3"
	"github.com/gofrs/uuid"
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
	return create(tx, n)
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

func (n *Notification) CreateNotificationUser(tx *pop.Connection, user User) {
	notnUser := NotificationUser{
		NotificationID: n.ID,
		UserID:         user.ID,
		EmailAddress:   user.EmailOfChoice(),
		SendAfterUTC:   time.Now().UTC(),
	}

	if err := notnUser.Create(tx); err != nil {
		panic(fmt.Sprintf("error creating new NotificationUser with UserID %s: %s",
			user.ID, err.Error()))
	}
}

func (n *Notification) CreateNotificationUsersForStewards(tx *pop.Connection) {
	var stewards Users
	stewards.FindStewards(tx)

	for _, s := range stewards {
		nu := NotificationUser{
			NotificationID: n.ID,
			UserID:         s.ID,
			EmailAddress:   s.EmailOfChoice(),
			SendAfterUTC:   time.Now().UTC(),
		}

		if err := nu.Create(tx); err != nil {
			panic(fmt.Sprintf("error creating new NotificationUser for steward with UserID %s: %s",
				s.ID, err.Error()))
		}
	}
}

func (n *Notification) CreateNotificationUsersForSignators(tx *pop.Connection) {
	var signators Users
	signators.FindSignators(tx)

	for _, s := range signators {
		nu := NotificationUser{
			NotificationID: n.ID,
			UserID:         s.ID,
			EmailAddress:   s.EmailOfChoice(),
			SendAfterUTC:   time.Now().UTC(),
		}

		if err := nu.Create(tx); err != nil {
			panic(fmt.Sprintf("error creating new NotificationUser for steward with UserID %s: %s",
				s.ID, err.Error()))
		}
	}
}
