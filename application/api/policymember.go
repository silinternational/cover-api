package api

import (
	"time"

	"github.com/gofrs/uuid"
)

type PolicyMembers []PolicyMember

type PolicyMember struct {
	ID           uuid.UUID `json:"id"`
	FirstName    string    `json:"first_name"`
	LastName     string    `json:"last_name"`
	Email        string    `json:"email"`
	LastLoginUTC time.Time `json:"last_login_utc"`
}
