package api

import (
	"time"

	"github.com/gobuffalo/nulls"
	"github.com/gofrs/uuid"
)

type Users []User

type User struct {
	ID           uuid.UUID  `json:"id"`
	Email        string     `json:"email"`
	FirstName    string     `json:"first_name"`
	LastName     string     `json:"last_name"`
	LastLoginUTC time.Time  `json:"last_login_utc"`
	PolicyID     nulls.UUID `json:"policy_id"` // TODO: provide either a list of IDs or a list of Policies
}
