package api

import (
	"github.com/gofrs/uuid"
)

// swagger:model
type EntityCodes []EntityCode

// swagger:model
type EntityCode struct {
	// unique ID
	//
	// swagger:strfmt uuid4
	ID   uuid.UUID `json:"id"`
	Code string    `json:"code"`
	Name string    `json:"name"`
}
