package models

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gobuffalo/nulls"
	"github.com/gobuffalo/pop/v5"
	"github.com/gobuffalo/validate/v3"
	"github.com/gofrs/uuid"

	"github.com/silinternational/cover-api/api"
	"github.com/silinternational/cover-api/domain"
)

var ValidClaimItemStatus = map[api.ClaimItemStatus]struct{}{
	api.ClaimItemStatusDraft:    {},
	api.ClaimItemStatusPending:  {},
	api.ClaimItemStatusRevision: {},
	api.ClaimItemStatusApproved: {},
	api.ClaimItemStatusDenied:   {},
}

var ValidPayoutOptions = map[api.PayoutOption]struct{}{
	api.PayoutOptionRepair:      {},
	api.PayoutOptionReplacement: {},
	api.PayoutOptionFMV:         {},
}

type ClaimItems []ClaimItem

type ClaimItem struct {
	ID              uuid.UUID           `db:"id"`
	ClaimID         uuid.UUID           `db:"claim_id"`
	ItemID          uuid.UUID           `db:"item_id"`
	Status          api.ClaimItemStatus `db:"status" validate:"required,claimItemStatus"`
	IsRepairable    bool                `db:"is_repairable"`
	RepairEstimate  int                 `db:"repair_estimate"`
	RepairActual    int                 `db:"repair_actual"`
	ReplaceEstimate int                 `db:"replace_estimate"`
	ReplaceActual   int                 `db:"replace_actual"`
	PayoutOption    api.PayoutOption    `db:"payout_option" validate:"payoutOption"`
	PayoutAmount    int                 `db:"payout_amount"`
	FMV             int                 `db:"fmv"`
	ReviewDate      nulls.Time          `db:"review_date"`
	ReviewerID      nulls.UUID          `db:"reviewer_id"`
	LegacyID        nulls.Int           `db:"legacy_id"`
	CreatedAt       time.Time           `db:"created_at"`
	UpdatedAt       time.Time           `db:"updated_at"`

	Claim    Claim `belongs_to:"claims" validate:"-"`
	Item     Item  `belongs_to:"items" validate:"-"`
	Reviewer User  `belongs_to:"users" validate:"-"`
}

// Validate gets run every time you call a "pop.Validate*" (pop.ValidateAndSave, pop.ValidateAndCreate, pop.ValidateAndUpdate) method.
func (c *ClaimItem) Validate(tx *pop.Connection) (*validate.Errors, error) {
	return validateModel(c), nil
}

// Create stores the Policy data as a new record in the database.
func (c *ClaimItem) Create(tx *pop.Connection) error {
	return create(tx, c)
}

// Update writes the Policy data to an existing database record.
func (c *ClaimItem) Update(tx *pop.Connection, oldStatus api.ClaimItemStatus) error {
	validTrans, err := isClaimItemTransitionValid(oldStatus, c.Status)
	if err != nil {
		panic(err)
	}
	if !validTrans {
		err := fmt.Errorf("invalid claim item status transition from %s to %s",
			oldStatus, c.Status)
		appErr := api.NewAppError(err, api.ErrorValidation, api.CategoryUser)
		return appErr
	}
	return update(tx, c)
}

func (c *ClaimItem) GetID() uuid.UUID {
	return c.ID
}

func (c *ClaimItem) FindByID(tx *pop.Connection, id uuid.UUID) error {
	return tx.Find(c, id)
}

// IsActorAllowedTo ensure the actor is either an admin, or a member of this policy to perform any permission
func (c *ClaimItem) IsActorAllowedTo(tx *pop.Connection, actor User, perm Permission, sub SubResource, r *http.Request) bool {
	if actor.IsAdmin() {
		return true
	}

	c.LoadItem(tx, false)

	var policy Policy
	if err := policy.FindByID(tx, c.Item.PolicyID); err != nil {
		domain.ErrLogger.Printf("failed to load Policy for ClaimItem: %s", err)
		return false
	}

	policy.LoadMembers(tx, false)

	for _, m := range policy.Members {
		if m.ID == actor.ID {
			return true
		}
	}

	return false
}

func claimItemStatusTransitions() map[api.ClaimItemStatus][]api.ClaimItemStatus {
	return map[api.ClaimItemStatus][]api.ClaimItemStatus{
		api.ClaimItemStatusPending: {
			api.ClaimItemStatusRevision,
			api.ClaimItemStatusApproved,
			api.ClaimItemStatusDenied,
		},
		api.ClaimItemStatusRevision: {
			api.ClaimItemStatusPending,
		},
		api.ClaimItemStatusApproved: {},
		api.ClaimItemStatusDenied:   {},
	}
}

func isClaimItemTransitionValid(status1, status2 api.ClaimItemStatus) (bool, error) {
	if status1 == status2 {
		return true, nil
	}
	targets, ok := claimItemStatusTransitions()[status1]
	if !ok {
		return false, errors.New("unexpected initial claim item status - " + string(status1))
	}

	for _, target := range targets {
		if status2 == target {
			return true, nil
		}
	}

	return false, nil
}

func (c *ClaimItem) LoadClaim(tx *pop.Connection, reload bool) {
	if c.Claim.ID == uuid.Nil || reload {
		if err := tx.Load(c, "Claim"); err != nil {
			panic("database error loading ClaimItem.Claim, " + err.Error())
		}
	}
}

func (c *ClaimItem) LoadItem(tx *pop.Connection, reload bool) {
	if c.Item.ID == uuid.Nil || reload {
		if err := tx.Load(c, "Item"); err != nil {
			panic("database error loading ClaimItem.Item, " + err.Error())
		}
	}
}

func (c *ClaimItem) LoadReviewer(tx *pop.Connection, reload bool) {
	if c.ReviewerID.Valid && (c.Reviewer.ID == uuid.Nil || reload) {
		if err := tx.Load(c, "Reviewer"); err != nil {
			panic("database error loading ClaimItem.Reviewer, " + err.Error())
		}
	}
}

func (c *ClaimItem) ConvertToAPI(tx *pop.Connection) interface{} {
	c.LoadItem(tx, false)
	item := c.Item.ConvertToAPI(tx).(api.Item)
	return api.ClaimItem{
		ItemID:            c.ItemID,
		Name:              item.Name,
		Category:          item.Category,
		InStorage:         item.InStorage,
		Country:           item.Country,
		Description:       item.Description,
		PolicyID:          item.PolicyID,
		Make:              item.Make,
		Model:             item.Model,
		SerialNumber:      item.SerialNumber,
		CoverageAmount:    item.CoverageAmount,
		PurchaseDate:      item.PurchaseDate,
		CoverageStatus:    item.CoverageStatus,
		CoverageStartDate: item.CoverageStartDate,
		ClaimID:           c.ClaimID,
		Status:            c.Status,
		IsRepairable:      c.IsRepairable,
		RepairEstimate:    c.RepairEstimate,
		RepairActual:      c.RepairActual,
		ReplaceEstimate:   c.ReplaceEstimate,
		ReplaceActual:     c.ReplaceActual,
		PayoutOption:      c.PayoutOption,
		PayoutAmount:      c.PayoutAmount,
		FMV:               c.FMV,
		ReviewDate:        c.ReviewDate.Time,
		ReviewerID:        c.ReviewerID.UUID,
		CreatedAt:         c.CreatedAt,
		UpdatedAt:         c.UpdatedAt,
	}
}

func (c *ClaimItems) ConvertToAPI(tx *pop.Connection) interface{} {
	claimItems := make(api.ClaimItems, len(*c))
	for i, cc := range *c {
		claimItems[i] = cc.ConvertToAPI(tx).(api.ClaimItem)
	}
	return claimItems
}

func ConvertClaimItemCreateInput(input api.ClaimItemCreateInput) ClaimItem {
	item := ClaimItem{
		ItemID:          input.ItemID,
		IsRepairable:    input.IsRepairable,
		RepairEstimate:  input.RepairEstimate,
		RepairActual:    input.RepairActual,
		ReplaceEstimate: input.ReplaceEstimate,
		ReplaceActual:   input.ReplaceActual,
		PayoutOption:    input.PayoutOption,
		PayoutAmount:    input.PayoutAmount,
		FMV:             input.FMV,
	}

	item.Status = api.ClaimItemStatusDraft

	return item
}
