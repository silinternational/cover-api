package models

import (
	"context"
	"errors"
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
	api.ClaimItemStatusReview1:  {},
	api.ClaimItemStatusReceipt:  {},
	api.ClaimItemStatusRevision: {},
	api.ClaimItemStatusReview2:  {},
	api.ClaimItemStatusReview3:  {},
	api.ClaimItemStatusApproved: {},
	api.ClaimItemStatusPaid:     {},
	api.ClaimItemStatusDenied:   {},
}

var ValidPayoutOptions = map[api.PayoutOption]struct{}{
	api.PayoutOptionRepair:        {},
	api.PayoutOptionReplacement:   {},
	api.PayoutOptionFMV:           {},
	api.PayoutOptionFixedFraction: {},
}

type ClaimItems []ClaimItem

type ClaimItem struct {
	ID              uuid.UUID           `db:"id"`
	ClaimID         uuid.UUID           `db:"claim_id"`
	ItemID          uuid.UUID           `db:"item_id"`
	Status          api.ClaimItemStatus `db:"status" validate:"required,claimItemStatus"`
	IsRepairable    bool                `db:"is_repairable"`
	RepairEstimate  int                 `db:"repair_estimate" validate:"min=0"`
	RepairActual    int                 `db:"repair_actual" validate:"min=0"`
	ReplaceEstimate int                 `db:"replace_estimate" validate:"min=0"`
	ReplaceActual   int                 `db:"replace_actual" validate:"min=0"`
	PayoutOption    api.PayoutOption    `db:"payout_option" validate:"payoutOption"`
	PayoutAmount    int                 `db:"payout_amount" validate:"min=0"`
	FMV             int                 `db:"fmv" validate:"min=0"`
	ReviewDate      nulls.Time          `db:"review_date"`
	ReviewerID      nulls.UUID          `db:"reviewer_id"`
	Location        string              `db:"location"`
	LegacyID        nulls.Int           `db:"legacy_id"`
	CreatedAt       time.Time           `db:"created_at"`
	UpdatedAt       time.Time           `db:"updated_at"`

	Claim    Claim `belongs_to:"claims" validate:"-"`
	Item     Item  `belongs_to:"items" validate:"-"`
	Reviewer User  `belongs_to:"users" validate:"-"`
}

// Validate gets run every time you call a "pop.Validate*" (pop.ValidateAndSave, pop.ValidateAndCreate, pop.ValidateAndUpdate) method.
func (c *ClaimItem) Validate(tx *pop.Connection) (*validate.Errors, error) {
	c.LoadClaim(tx, false)

	return validateModel(c), nil
}

// Create validates and stores the data as a new record in the database, assigning a new ID if needed.
func (c *ClaimItem) Create(tx *pop.Connection) error {
	// Get the parent Claim's status
	c.LoadClaim(tx, false)
	c.Status = api.ClaimItemStatus(c.Claim.Status)

	return create(tx, c)
}

// Update changes the status if it is a valid transition.
func (c *ClaimItem) Update(tx *pop.Connection, oldStatus api.ClaimItemStatus, user User) error {
	// Get the parent Claim's status
	c.LoadClaim(tx, false)
	c.Status = api.ClaimItemStatus(c.Claim.Status)

	// Maybe One day we will want to worry about the status on the ClaimItem itself
	//if !isClaimItemTransitionValid(oldStatus, c.Status) {
	//	err := fmt.Errorf("invalid claim item status transition from %s to %s", oldStatus, c.Status)
	//	appErr := api.NewAppError(err, api.ErrorValidation, api.CategoryUser)
	//	return appErr
	//}

	// Set the Reviewer fields when needed.
	if user.IsAdmin() {
		c.ReviewerID = nulls.NewUUID(user.ID)
		c.ReviewDate = nulls.NewTime(time.Now().UTC())
	}

	return update(tx, c)
}

// UpdateByUser ensures the parent Claim has an appropriate status for being modified by the user
//  and then writes the ClaimItem data to an existing database record.
func (c *ClaimItem) UpdateByUser(ctx context.Context, oldStatus api.ClaimItemStatus, user User) error {
	tx := Tx(ctx)
	if user.IsAdmin() {
		return c.Update(tx, oldStatus, user)
	}

	c.LoadClaim(tx, false)

	switch c.Claim.Status {
	// OK to modify this when the parent Claim has one of these statuses but not any others
	case api.ClaimStatusDraft, api.ClaimStatusRevision, api.ClaimStatusReview1:
	default:
		err := errors.New("user may not edit a claim item that is too far along in the review process.")
		appErr := api.NewAppError(err, api.ErrorClaimStatus, api.CategoryUser)
		return appErr
	}

	// If the user edits something, it should take the claim off of the steward's list of things to review and
	//  also force the user to resubmit it.
	if c.Claim.Status == api.ClaimStatusReview1 {
		c.Claim.Status = api.ClaimStatusDraft
	}

	if err := c.Claim.Update(ctx); err != nil {
		return err
	}

	return c.Update(tx, oldStatus, user)
}

// Maybe one day we will want to do something like this on a ClaimItem
//func isClaimItemTransitionValid(status1, status2 api.ClaimItemStatus) bool {
//	if status1 == status2 {
//		return true
//	}
//	cStatus1 := api.ClaimStatus(status1)
//	cStatus2 := api.ClaimStatus(status2)
//	valid, _ := isClaimTransitionValid(cStatus1, cStatus2)
//	return valid
//}

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

func (c *ClaimItem) ConvertToAPI(tx *pop.Connection) api.ClaimItem {
	c.LoadItem(tx, false)

	return api.ClaimItem{
		ID:              c.ID,
		ItemID:          c.ItemID,
		Item:            c.Item.ConvertToAPI(tx),
		ClaimID:         c.ClaimID,
		Status:          c.Status,
		IsRepairable:    c.IsRepairable,
		RepairEstimate:  c.RepairEstimate,
		RepairActual:    c.RepairActual,
		ReplaceEstimate: c.ReplaceEstimate,
		ReplaceActual:   c.ReplaceActual,
		PayoutOption:    c.PayoutOption,
		PayoutAmount:    c.PayoutAmount,
		FMV:             c.FMV,
		ReviewDate:      c.ReviewDate.Time,
		ReviewerID:      c.ReviewerID.UUID,
		CreatedAt:       c.CreatedAt,
		UpdatedAt:       c.UpdatedAt,
	}
}

func (c *ClaimItem) ValidateForSubmit(tx *pop.Connection) api.ErrorKey {
	c.LoadClaim(tx, false)

	if c.PayoutOption == "" {
		return api.ErrorClaimItemMissingPayoutOption
	}

	if c.IsRepairable && !c.Claim.IncidentType.IsRepairable() {
		return api.ErrorClaimItemNotRepairable
	}

	switch c.Claim.IncidentType {
	case api.ClaimIncidentTypeTheft:
		if c.replaceEstimateMissing() {
			return api.ErrorClaimItemMissingReplaceEstimate
		}
		if c.fmvMissing() {
			return api.ErrorClaimItemMissingFMV
		}
	case api.ClaimIncidentTypeImpact, api.ClaimIncidentTypeElectricalSurge,
		api.ClaimIncidentTypeWaterDamage, api.ClaimIncidentTypeOther:
		if c.IsRepairable {
			if c.RepairEstimate == 0 {
				return api.ErrorClaimItemMissingRepairEstimate
			}
			if c.FMV == 0 {
				return api.ErrorClaimItemMissingFMV
			}
		} else {
			if c.PayoutOption == api.PayoutOptionRepair {
				return api.ErrorClaimItemInvalidPayoutOption
			}
			if c.replaceEstimateMissing() {
				return api.ErrorClaimItemMissingReplaceEstimate
			}
			if c.fmvMissing() {
				return api.ErrorClaimItemMissingFMV
			}
		}
	case api.ClaimIncidentTypeEvacuation:
		if c.PayoutOption != api.PayoutOptionFixedFraction {
			return api.ErrorClaimItemInvalidPayoutOption
		}
	}
	return ""
}

func (c *ClaimItem) replaceEstimateMissing() bool {
	if c.PayoutOption == api.PayoutOptionReplacement && c.ReplaceEstimate == 0 {
		return true
	}
	return false
}

func (c *ClaimItem) fmvMissing() bool {
	if c.PayoutOption == api.PayoutOptionFMV && c.FMV == 0 {
		return true
	}
	return false
}

func (c *ClaimItems) ConvertToAPI(tx *pop.Connection) api.ClaimItems {
	claimItems := make(api.ClaimItems, len(*c))
	for i, cc := range *c {
		claimItems[i] = cc.ConvertToAPI(tx)
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
		FMV:             input.FMV,
	}

	item.Status = api.ClaimItemStatusDraft

	return item
}
