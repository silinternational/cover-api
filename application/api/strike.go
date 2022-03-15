package api

import (
	"time"

	"github.com/gofrs/uuid"
)

// swagger:model
type Strikes []Strike

// swagger:model
type Strike struct {
	ID          uuid.UUID `json:"id"`
	Description string    `json:"description"`
	PolicyID    uuid.UUID `json:"policy_id"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
