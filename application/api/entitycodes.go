package api

import (
	"github.com/gofrs/uuid"
)

type EntityCode struct {
	ID   uuid.UUID `json:"id"`
	Code string    `json:"code"`
	Name string    `json:"name"`
}
