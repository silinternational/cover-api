package models

import (
	"errors"
	"fmt"
	"time"

	"github.com/gobuffalo/nulls"
	"github.com/gobuffalo/pop/v5"
	"github.com/gobuffalo/validate/v3"
	"github.com/gofrs/uuid"

	"github.com/silinternational/cover-api/domain"
)

type NotificationUsers []NotificationUser

type NotificationUser struct {
	ID               uuid.UUID  `db:"id"`
	NotificationID   uuid.UUID  `db:"notification_id"`
	UserID           nulls.UUID `db:"user_id"`
	ToName           string     `db:"to_name"` // Only needed when there is no UserID
	EmailAddress     string     `db:"email_address"`
	ViewedAtUTC      nulls.Time `db:"viewed_at_utc"`
	SendAttemptCount int        `db:"send_attempt_count"`
	SendAfterUTC     time.Time  `db:"send_after_utc"`
	LastAttemptUTC   nulls.Time `db:"last_attempt_utc"`
	SentAtUTC        nulls.Time `db:"sent_at_utc"`

	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`

	Notification Notification `belongs_to:"notifications" validate:"-"`
	User         User         `belongs_to:"users" validate:"-"`
}

// Validate gets run every time you call a "pop.Validate*" (pop.ValidateAndSave, pop.ValidateAndCreate, pop.ValidateAndUpdate) method.
func (n *NotificationUser) Validate(tx *pop.Connection) (*validate.Errors, error) {
	return validateModel(n), nil
}

// Create stores the data as a new record in the database.
func (n *NotificationUser) Create(tx *pop.Connection) error {
	return create(tx, n)
}

// Update writes the NotificationUser data to an existing database record.
func (n *NotificationUser) Update(tx *pop.Connection) error {
	return update(tx, n)
}

func (n *NotificationUser) GetID() uuid.UUID {
	return n.ID
}

func (n *NotificationUser) FindByID(tx *pop.Connection, id uuid.UUID) error {
	return tx.Find(n, id)
}

// Load - a simple wrapper method for loading the notification and the user on the struct
func (n *NotificationUser) Load(tx *pop.Connection) {
	if n.Notification.ID == uuid.Nil {
		if err := tx.Load(n, "Notification"); err != nil {
			panic("database error loading NotificationUser.Notification, " + err.Error())
		}
	}
	if n.User.ID == uuid.Nil {
		if err := tx.Load(n, "User"); err != nil {
			panic("database error loading NotificationUser.User, " + err.Error())
		}
	}
}

func (n *NotificationUsers) GetEmailsToSend(tx *pop.Connection) error {
	q := fmt.Sprintf(`SELECT notification_users.*
  FROM notification_users LEFT JOIN notifications ON notification_users.notification_id = notifications.id
  WHERE notifications.body <> '' AND
     sent_at_utc IS NULL AND
     send_after_utc < now()`) // postgresql appears to use UTC as the timezone for now()

	if err := tx.RawQuery(q).All(n); err != nil {
		if domain.IsOtherThanNoRows(err) {
			return errors.New("error getting queued notification_users to send out: " + err.Error())
		}
	}

	return nil
}
