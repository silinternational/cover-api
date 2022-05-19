package api

import (
	"time"

	"github.com/gofrs/uuid"
)

// swagger:model
type Strikes []Strike

// swagger:model
type Strike struct {
	// unique ID
	//
	// swagger:strfmt uuid4
	ID uuid.UUID `json:"id"`

	// strike description
	Description string `json:"description"`

	// swagger:strfmt uuid4
	PolicyID uuid.UUID `json:"policy_id"`

	// The time the strike was created
	//
	// swagger:strfmt date-time
	CreatedAt time.Time `json:"created_at"`

	// The time the strike was updated
	//
	// swagger:strfmt date-time
	UpdatedAt time.Time `json:"updated_at"`
}

// swagger:model
type StrikeInput struct {
	// strike description
	Description string `json:"description"`
}
