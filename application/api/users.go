package api

import (
	"github.com/gofrs/uuid"
)

type Users []User

type User struct {
	ID uuid.UUID `json:"id"`
}
