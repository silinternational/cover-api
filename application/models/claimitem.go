package models

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net/http"
	"time"

	"github.com/gobuffalo/nulls"
	"github.com/gobuffalo/pop/v5"
	"github.com/gobuffalo/validate/v3"
	"github.com/gofrs/uuid"

	"github.com/silinternational/cover-api/api"
	"github.com/silinternational/cover-api/domain"
)

var ValidPayoutOptions = map[api.PayoutOption]struct{}{
	api.PayoutOptionRepair:        {},
	api.PayoutOptionReplacement:   {},
	api.PayoutOptionFMV:           {},
	api.PayoutOptionFixedFraction: {},
}

type ClaimItems []ClaimItem

type ClaimItem struct {
	ID              uuid.UUID        `db:"id"`
	ClaimID         uuid.UUID        `db:"claim_id"`
	ItemID          uuid.UUID        `db:"item_id"`
	IsRepairable    bool             `db:"is_repairable"`
	RepairEstimate  api.Currency     `db:"repair_estimate" validate:"min=0"`
	RepairActual    api.Currency     `db:"repair_actual" validate:"min=0"`
	ReplaceEstimate api.Currency     `db:"replace_estimate" validate:"min=0"`
	ReplaceActual   api.Currency     `db:"replace_actual" validate:"min=0"`
	PayoutOption    api.PayoutOption `db:"payout_option" validate:"payoutOption"`
	PayoutAmount    api.Currency     `db:"payout_amount" validate:"min=0"`
	FMV             api.Currency     `db:"fmv" validate:"min=0"`
	City            string           `db:"city"`
	State           string           `db:"state"`
	Country         string           `db:"country"`
	LegacyID        nulls.Int        `db:"legacy_id"`
	CreatedAt       time.Time        `db:"created_at"`
	UpdatedAt       time.Time        `db:"updated_at"`

	Claim Claim `belongs_to:"claims" validate:"-"`
	Item  Item  `belongs_to:"items" validate:"-"`
}

// Validate gets run every time you call a "pop.Validate*" (pop.ValidateAndSave, pop.ValidateAndCreate, pop.ValidateAndUpdate) method.
func (c *ClaimItem) Validate(tx *pop.Connection) (*validate.Errors, error) {
	c.LoadClaim(tx, false)

	return validateModel(c), nil
}

// Create validates and stores the data as a new record in the database, assigning a new ID if needed.
func (c *ClaimItem) Create(tx *pop.Connection) error {
	return create(tx, c)
}

// Update saves any changes to the database
func (c *ClaimItem) Update(ctx context.Context) error {
	tx := Tx(ctx)
	c.LoadClaim(tx, false)

	user := CurrentUser(ctx)
	if !c.Claim.canUpdate(user) {
		err := errors.New("user may not edit a ClaimItem that is too far along in the review process")
		appErr := api.NewAppError(err, api.ErrorClaimStatus, api.CategoryUser)
		return appErr
	}

	updates, err := c.getUpdates(ctx)
	if err != nil {
		return err
	}

	if err = c.revertToDraftIfEdited(ctx, updates); err != nil {
		return err
	}

	return update(tx, c)
}

func (c *ClaimItem) getUpdates(ctx context.Context) ([]FieldUpdate, error) {
	tx := Tx(ctx)

	var oldClaimItem ClaimItem
	if err := oldClaimItem.FindByID(tx, c.ID); err != nil {
		return []FieldUpdate{}, appErrorFromDB(err, api.ErrorQueryFailure)
	}

	updates := c.Compare(oldClaimItem)
	for i := range updates {
		history := c.NewHistory(ctx, api.HistoryActionUpdate, updates[i])
		if err := history.Create(tx); err != nil {
			return updates, appErrorFromDB(err, api.ErrorCreateFailure)
		}
	}
	return updates, nil
}

// If a customer edits something other than ReceiptActual or ReplaceActual, it should take the claim off
// of the steward's list of things to review and also force the user to resubmit it.
func (c *ClaimItem) revertToDraftIfEdited(ctx context.Context, updates []FieldUpdate) error {
	user := CurrentUser(ctx)
	if user.IsAdmin() {
		return nil
	}

	for _, u := range updates {
		switch u.FieldName {
		case FieldClaimItemReplaceActual, FieldClaimItemRepairActual:
			continue
		default:
			return c.Claim.UpdateStatus(ctx, api.ClaimStatusDraft)
		}
	}
	return nil
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

func (c *ClaimItem) ConvertToAPI(tx *pop.Connection) api.ClaimItem {
	c.LoadItem(tx, false)
	c.LoadClaim(tx, false)

	return api.ClaimItem{
		ID:              c.ID,
		ItemID:          c.ItemID,
		Item:            c.Item.ConvertToAPI(tx),
		ClaimID:         c.ClaimID,
		Status:          c.Claim.Status,
		IsRepairable:    c.IsRepairable,
		RepairEstimate:  c.RepairEstimate,
		RepairActual:    c.RepairActual,
		ReplaceEstimate: c.ReplaceEstimate,
		ReplaceActual:   c.ReplaceActual,
		PayoutOption:    c.PayoutOption,
		PayoutAmount:    c.PayoutAmount,
		FMV:             c.FMV,
		ReviewDate:      c.Claim.ReviewDate.Time,
		ReviewerID:      c.Claim.ReviewerID.UUID,
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

// NewClaimItem makes a new ClaimItem, but does not do a database create
func NewClaimItem(tx *pop.Connection, input api.ClaimItemCreateInput, item Item, claim Claim) (ClaimItem, error) {
	claimItem := ClaimItem{
		ItemID:          input.ItemID,
		IsRepairable:    input.IsRepairable,
		RepairEstimate:  input.RepairEstimate,
		RepairActual:    input.RepairActual,
		ReplaceEstimate: input.ReplaceEstimate,
		ReplaceActual:   input.ReplaceActual,
		PayoutOption:    input.PayoutOption,
		FMV:             input.FMV,
	}

	claimItem.ClaimID = claim.ID
	loc := item.GetAccountablePersonLocation(tx)
	claimItem.City = loc.City
	claimItem.State = loc.State
	claimItem.Country = loc.Country
	return claimItem, nil
}

// Compare returns a list of fields that are different between two objects
func (c *ClaimItem) Compare(old ClaimItem) []FieldUpdate {
	var updates []FieldUpdate

	if c.ItemID != old.ItemID {
		updates = append(updates, FieldUpdate{
			OldValue:  old.ItemID.String(),
			NewValue:  c.ItemID.String(),
			FieldName: FieldClaimItemItemID,
		})
	}

	if c.IsRepairable != old.IsRepairable {
		updates = append(updates, FieldUpdate{
			OldValue:  fmt.Sprintf("%t", old.IsRepairable),
			NewValue:  fmt.Sprintf("%t", c.IsRepairable),
			FieldName: FieldClaimItemIsRepairable,
		})
	}

	if c.RepairEstimate != old.RepairEstimate {
		updates = append(updates, FieldUpdate{
			OldValue:  api.Currency(old.RepairEstimate).String(),
			NewValue:  api.Currency(c.RepairEstimate).String(),
			FieldName: FieldClaimItemRepairEstimate,
		})
	}

	if c.RepairActual != old.RepairActual {
		updates = append(updates, FieldUpdate{
			OldValue:  api.Currency(old.RepairActual).String(),
			NewValue:  api.Currency(c.RepairActual).String(),
			FieldName: FieldClaimItemRepairActual,
		})
	}

	if c.ReplaceEstimate != old.ReplaceEstimate {
		updates = append(updates, FieldUpdate{
			OldValue:  api.Currency(old.ReplaceEstimate).String(),
			NewValue:  api.Currency(c.ReplaceEstimate).String(),
			FieldName: FieldClaimItemReplaceEstimate,
		})
	}

	if c.ReplaceActual != old.ReplaceActual {
		updates = append(updates, FieldUpdate{
			OldValue:  api.Currency(old.ReplaceActual).String(),
			NewValue:  api.Currency(c.ReplaceActual).String(),
			FieldName: FieldClaimItemReplaceActual,
		})
	}

	if c.PayoutOption != old.PayoutOption {
		updates = append(updates, FieldUpdate{
			OldValue:  string(old.PayoutOption),
			NewValue:  string(c.PayoutOption),
			FieldName: FieldClaimItemPayoutOption,
		})
	}

	if c.PayoutAmount != old.PayoutAmount {
		updates = append(updates, FieldUpdate{
			OldValue:  api.Currency(old.PayoutAmount).String(),
			NewValue:  api.Currency(c.PayoutAmount).String(),
			FieldName: FieldClaimItemPayoutAmount,
		})
	}

	if c.FMV != old.FMV {
		updates = append(updates, FieldUpdate{
			OldValue:  api.Currency(old.FMV).String(),
			NewValue:  api.Currency(c.FMV).String(),
			FieldName: FieldClaimItemFMV,
		})
	}

	if c.GetLocation() != old.GetLocation() {
		updates = append(updates, FieldUpdate{
			OldValue:  old.GetLocation().String(),
			NewValue:  c.GetLocation().String(),
			FieldName: FieldClaimItemLocation,
		})
	}

	return updates
}

func (c *ClaimItem) GetLocation() Location {
	return Location{
		City:    c.City,
		State:   c.State,
		Country: c.Country,
	}
}

// NewHistory makes a new ClaimHistory object based on the current ClaimItem.
func (c *ClaimItem) NewHistory(ctx context.Context, action string, fieldUpdate FieldUpdate) ClaimHistory {
	return ClaimHistory{
		Action:      action,
		ClaimID:     c.ClaimID,
		ClaimItemID: nulls.NewUUID(c.ID),
		UserID:      CurrentUser(ctx).ID,
		FieldName:   fieldUpdate.FieldName,
		OldValue:    fieldUpdate.OldValue,
		NewValue:    fieldUpdate.NewValue,
	}
}

func (c *ClaimItem) updatePayoutAmount(ctx context.Context) error {
	c.LoadItem(Tx(ctx), false)

	coverageAmount := c.Item.CoverageAmount

	deductible := 0.05
	maxValue := 0.0
	switch c.PayoutOption {
	case api.PayoutOptionRepair:
		maxValue = float64(c.RepairEstimate)
		if c.RepairActual > 0 {
			maxValue = float64(c.RepairActual)
		}
	case api.PayoutOptionReplacement:
		maxValue = float64(c.ReplaceEstimate)
		if c.ReplaceActual > 0 {
			maxValue = float64(c.ReplaceActual)
		}
	case api.PayoutOptionFMV:
		maxValue = float64(c.FMV)
	case api.PayoutOptionFixedFraction:
		deductible = 1.0 / 3.0
		maxValue = float64(coverageAmount)
	}

	before := c.PayoutAmount
	c.PayoutAmount = api.Currency(math.Round(math.Min(maxValue, float64(coverageAmount)) * (1.0 - deductible)))

	if c.PayoutAmount == before {
		return nil
	}

	return c.Update(ctx)
}
