package api

import (
	"time"

	"github.com/gofrs/uuid"
)

// RiskCategories is a slice of RiskCategory objects
// swagger:model
type RiskCategories []RiskCategory

// RiskCategory represents an item category's risk category
// swagger:model
type RiskCategory struct {
	ID               uuid.UUID `json:"id"`
	Name             string    `json:"name"`
	PolicyMax        int       `json:"policy_max"`
	RequireMakeModel bool      `json:"require_make_model"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}
