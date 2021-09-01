package models

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gobuffalo/events"
	"github.com/gobuffalo/nulls"
	"github.com/gobuffalo/pop/v5"
	"github.com/gobuffalo/validate/v3"
	"github.com/gofrs/uuid"

	"github.com/silinternational/cover-api/api"
	"github.com/silinternational/cover-api/domain"
)

const (
	ClaimReferenceNumberLength = 7
)

var ValidClaimEventTypes = map[api.ClaimEventType]struct{}{
	api.ClaimEventTypeTheft:           {},
	api.ClaimEventTypeImpact:          {},
	api.ClaimEventTypeElectricalSurge: {},
	api.ClaimEventTypeWaterDamage:     {},
	api.ClaimEventTypeEvacuation:      {},
	api.ClaimEventTypeOther:           {},
}

var ValidClaimStatus = map[api.ClaimStatus]struct{}{
	api.ClaimStatusDraft:    {},
	api.ClaimStatusReview1:  {},
	api.ClaimStatusReview2:  {},
	api.ClaimStatusReview3:  {},
	api.ClaimStatusRevision: {},
	api.ClaimStatusReceipt:  {},
	api.ClaimStatusApproved: {},
	api.ClaimStatusPaid:     {},
	api.ClaimStatusDenied:   {},
	api.ClaimStatusInactive: {},
}

type Claims []Claim

type Claim struct {
	ID               uuid.UUID          `db:"id"`
	PolicyID         uuid.UUID          `db:"policy_id" validate:"required"`
	ReferenceNumber  string             `db:"reference_number" validate:"required,len=7"`
	EventDate        time.Time          `db:"event_date" validate:"required_unless=Status Draft"`
	EventType        api.ClaimEventType `db:"event_type" validate:"claimEventType,required_unless=Status Draft"`
	EventDescription string             `db:"event_description" validate:"required_unless=Status Draft"`
	Status           api.ClaimStatus    `db:"status" validate:"claimStatus"`
	ReviewDate       nulls.Time         `db:"review_date"`
	ReviewerID       nulls.UUID         `db:"reviewer_id"`
	PaymentDate      nulls.Time         `db:"payment_date"`
	TotalPayout      int                `db:"total_payout"`
	LegacyID         nulls.Int          `db:"legacy_id"`
	CreatedAt        time.Time          `db:"created_at"`
	UpdatedAt        time.Time          `db:"updated_at"`

	Policy     Policy     `belongs_to:"policies" validate:"-"`
	ClaimItems ClaimItems `has_many:"claim_items" validate:"-"`
	ClaimFiles ClaimFiles `has_many:"claim_files" validate:"-"`
	Reviewer   User       `belongs_to:"users" validate:"-"`
}

// Validate gets run every time you call a "pop.Validate*" (pop.ValidateAndSave, pop.ValidateAndCreate, pop.ValidateAndUpdate) method.
func (c *Claim) Validate(tx *pop.Connection) (*validate.Errors, error) {
	return validateModel(c), nil
}

// Create stores the Claim data as a new record in the database.
// If its status is not valid, it is created in Draft status.
func (c *Claim) Create(tx *pop.Connection) error {
	c.ReferenceNumber = uniqueClaimReferenceNumber(tx)
	if _, ok := ValidClaimStatus[c.Status]; !ok {
		c.Status = api.ClaimStatusDraft
	}
	return create(tx, c)
}

// Update writes the Claim data to an existing database record.
func (c *Claim) Update(tx *pop.Connection, oldStatus api.ClaimStatus) error {
	validTrans, err := isClaimTransitionValid(oldStatus, c.Status)
	if err != nil {
		panic(err)
	}
	if !validTrans {
		err := fmt.Errorf("invalid claim status transition from %s to %s",
			oldStatus, c.Status)
		appErr := api.NewAppError(err, api.ErrorValidation, api.CategoryUser)
		return appErr
	}

	if c.Status != api.ClaimStatusDraft {
		c.LoadClaimItems(tx, false)
		if len(c.ClaimItems) == 0 {
			err := errors.New("claim must have a claimItem if no longer in draft")
			appErr := api.NewAppError(err, api.ErrorClaimMissingClaimItem, api.CategoryUser)
			return appErr
		}
	}

	return update(tx, c)
}

func (c *Claim) GetID() uuid.UUID {
	return c.ID
}

func (c *Claim) FindByID(tx *pop.Connection, id uuid.UUID) error {
	return tx.Find(c, id)
}

func (c *Claim) FindByReferenceNumber(tx *pop.Connection, ref string) error {
	return tx.Where("reference_number = ?", ref).First(c)
}

// IsActorAllowedTo ensure the actor is either an admin, or a member of this policy to perform any permission
// TODO Differentiate between admins (steward and boss)
func (c *Claim) IsActorAllowedTo(tx *pop.Connection, actor User, perm Permission, sub SubResource, r *http.Request) bool {
	if actor.IsAdmin() {
		return true
	}

	// Only admin can do these
	adminSubs := []string{
		api.ResourceRevision, api.ResourceApprove,
		api.ResourcePreapprove, api.ResourceReceipt, api.ResourceDeny,
	}
	if domain.IsStringInSlice(string(sub), adminSubs) {
		return false
	}

	if perm == PermissionList || (perm == PermissionCreate && sub == "") {
		return true
	}

	var policy Policy
	if err := policy.FindByID(tx, c.PolicyID); err != nil {
		domain.ErrLogger.Printf("failed to load Policy %s for Claim: %s", c.PolicyID, err)
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

func claimStatusTransitions() map[api.ClaimStatus][]api.ClaimStatus {
	return map[api.ClaimStatus][]api.ClaimStatus{
		api.ClaimStatusDraft: {
			api.ClaimStatusReview1,
			api.ClaimStatusInactive,
		},
		api.ClaimStatusReview1: {
			api.ClaimStatusRevision,
			api.ClaimStatusReceipt,
			api.ClaimStatusReview3,
			api.ClaimStatusDenied,
		},
		api.ClaimStatusRevision: {
			api.ClaimStatusReview1,
			api.ClaimStatusInactive,
		},
		api.ClaimStatusReceipt: {
			api.ClaimStatusReview2,
			api.ClaimStatusInactive,
		},
		api.ClaimStatusReview2: {
			api.ClaimStatusReceipt,
			api.ClaimStatusReview3,
			api.ClaimStatusDenied,
		},
		api.ClaimStatusReview3: {
			api.ClaimStatusReceipt,
			api.ClaimStatusRevision,
			api.ClaimStatusApproved,
			api.ClaimStatusDenied,
		},
		api.ClaimStatusApproved: {
			api.ClaimStatusPaid,
		},
		api.ClaimStatusDenied:   {},
		api.ClaimStatusInactive: {},
	}
}

func isClaimTransitionValid(status1, status2 api.ClaimStatus) (bool, error) {
	if status1 == status2 {
		return true, nil
	}
	targets, ok := claimStatusTransitions()[status1]
	if !ok {
		return false, errors.New("unexpected initial status - " + string(status1))
	}

	for _, target := range targets {
		if status2 == target {
			return true, nil
		}
	}

	return false, nil
}

func (c *Claim) AddItem(tx *pop.Connection, claim Claim, input api.ClaimItemCreateInput) (ClaimItem, error) {
	if c == nil {
		return ClaimItem{}, errors.New("claim is nil in AddItem")
	}

	// ensure item and claim belong to the same policy
	var item Item
	if err := item.FindByID(tx, input.ItemID); err != nil {
		err = fmt.Errorf("failed to load item: %s", err)
		appErr := api.NewAppError(err, api.ErrorResourceNotFound, api.CategoryNotFound)
		if domain.IsOtherThanNoRows(err) {
			appErr.Category = api.CategoryInternal
		}
		return ClaimItem{}, appErr
	}

	if claim.PolicyID != item.PolicyID {
		err := fmt.Errorf("claim and item do not have same policy id: %s vs. %s",
			claim.PolicyID.String(), item.PolicyID.String())
		appErr := api.NewAppError(err, api.ErrorClaimItemCreateInvalidInput, api.CategoryUser)
		return ClaimItem{}, appErr
	}

	clmItem := ConvertClaimItemCreateInput(input)
	clmItem.ClaimID = claim.ID

	if err := clmItem.Create(tx); err != nil {
		return ClaimItem{}, err
	}

	return clmItem, nil
}

// SubmitForApproval changes the status of the claim to either Review1 or Review2
//   depending on its current status.
// TODO ensure the associated claimItem is valid
func (c *Claim) SubmitForApproval(tx *pop.Connection) error {
	oldStatus := c.Status
	var eventType string

	switch oldStatus {
	case api.ClaimStatusDraft, api.ClaimStatusRevision:
		c.Status = api.ClaimStatusReview1
		eventType = domain.EventApiClaimSubmitted
	case api.ClaimStatusReceipt:
		// TODO ensure there is a file attached for a receipt
		c.Status = api.ClaimStatusReview2
		eventType = domain.EventApiClaimReview2
	default:
		err := fmt.Errorf("invalid claim status for submit: %s", oldStatus)
		appErr := api.NewAppError(err, api.ErrorClaimStatus, api.CategoryUser)
		return appErr
	}

	if err := c.Update(tx, oldStatus); err != nil {
		return err
	}

	e := events.Event{
		Kind:    eventType,
		Message: fmt.Sprintf("Claim Submitted: %s  ID: %s", c.EventDescription, c.ID.String()),
		Payload: events.Payload{domain.EventPayloadID: c.ID},
	}
	emitEvent(e)

	return nil
}

// RequestRevision changes the status of the claim to Revision
//   provided that the current status is Review1 or Review3.
// TODO record the particular revisions that are needed
func (c *Claim) RequestRevision(tx *pop.Connection) error {
	oldStatus := c.Status

	switch oldStatus {
	case api.ClaimStatusReview1, api.ClaimStatusReview3:
		c.Status = api.ClaimStatusRevision
	default:
		err := fmt.Errorf("invalid claim status for request revision: %s", oldStatus)
		appErr := api.NewAppError(err, api.ErrorClaimStatus, api.CategoryUser)
		return appErr
	}

	if err := c.Update(tx, oldStatus); err != nil {
		return err
	}

	e := events.Event{
		Kind:    domain.EventApiClaimRevision,
		Message: fmt.Sprintf("Claim Revision: %s  ID: %s", c.EventDescription, c.ID.String()),
		Payload: events.Payload{domain.EventPayloadID: c.ID},
	}
	emitEvent(e)

	return nil
}

// RequestReceipt changes the status of the claim to Receipt
//   provided that the current status is Review1.
// TODO consider how to communicate what kind of receipt is needed
func (c *Claim) RequestReceipt(tx *pop.Connection) error {
	oldStatus := c.Status
	var eventType string

	switch oldStatus {
	case api.ClaimStatusReview1:
		eventType = domain.EventApiClaimPreapproved
	case api.ClaimStatusReview2, api.ClaimStatusReview3:
		eventType = domain.EventApiClaimReceipt
	default:
		err := fmt.Errorf("invalid claim status for request receipt: %s", oldStatus)
		appErr := api.NewAppError(err, api.ErrorClaimStatus, api.CategoryUser)
		return appErr
	}

	c.Status = api.ClaimStatusReceipt

	if err := c.Update(tx, oldStatus); err != nil {
		return err
	}

	e := events.Event{
		Kind:    eventType,
		Message: fmt.Sprintf("Claim Request Receipt: %s  ID: %s", c.EventDescription, c.ID.String()),
		Payload: events.Payload{domain.EventPayloadID: c.ID},
	}
	emitEvent(e)

	return nil
}

// Approve changes the status of the claim from either Review1, Review2 to Review3 or
//  from Review3 to Approved. It also adds the ReviewerID and ReviewDate.
// TODO distinguish between admin types (steward vs. boss)
// TODO do whatever post-processing is needed for payment
func (c *Claim) Approve(tx *pop.Connection, actor User) error {
	oldStatus := c.Status
	var eventType string

	switch oldStatus {
	case api.ClaimStatusReview1, api.ClaimStatusReview2:
		c.Status = api.ClaimStatusReview3
		eventType = domain.EventApiClaimReview3
	case api.ClaimStatusReview3:
		c.Status = api.ClaimStatusApproved
		eventType = domain.EventApiClaimApproved
	default:
		err := fmt.Errorf("invalid claim status for approve: %s", oldStatus)
		appErr := api.NewAppError(err, api.ErrorClaimStatus, api.CategoryUser)
		return appErr
	}

	c.ReviewerID = nulls.NewUUID(actor.ID)
	c.ReviewDate = nulls.NewTime(time.Now().UTC())

	if err := c.Update(tx, oldStatus); err != nil {
		return err
	}

	e := events.Event{
		Kind:    eventType,
		Message: fmt.Sprintf("Claim Approved: %s  ID: %s", c.EventDescription, c.ID.String()),
		Payload: events.Payload{domain.EventPayloadID: c.ID},
	}
	emitEvent(e)

	return nil
}

// Deny changes the status of the claim to Denied and adds the ReviewerID and ReviewDate.
func (c *Claim) Deny(tx *pop.Connection, actor User) error {
	oldStatus := c.Status

	if oldStatus != api.ClaimStatusReview1 && oldStatus != api.ClaimStatusReview2 &&
		oldStatus != api.ClaimStatusReview3 {
		err := fmt.Errorf("invalid claim status for deny: %s", oldStatus)
		appErr := api.NewAppError(err, api.ErrorClaimStatus, api.CategoryUser)
		return appErr
	}

	c.Status = api.ClaimStatusDenied

	c.ReviewerID = nulls.NewUUID(actor.ID)
	c.ReviewDate = nulls.NewTime(time.Now().UTC())

	if err := c.Update(tx, oldStatus); err != nil {
		return err
	}

	e := events.Event{
		Kind:    domain.EventApiClaimDenied,
		Message: fmt.Sprintf("Claim Denied: %s  ID: %s", c.EventDescription, c.ID.String()),
		Payload: events.Payload{domain.EventPayloadID: c.ID},
	}
	emitEvent(e)

	return nil
}

func (c *Claim) LoadClaimItems(tx *pop.Connection, reload bool) {
	if len(c.ClaimItems) == 0 || reload {
		if err := tx.Load(c, "ClaimItems"); err != nil {
			panic("database error loading Claim.ClaimItems, " + err.Error())
		}
	}
}

func (c *Claim) LoadPolicy(tx *pop.Connection, reload bool) {
	if c.Policy.ID == uuid.Nil || reload {
		if err := tx.Load(c, "Policy"); err != nil {
			panic("database error loading Claim.Policy, " + err.Error())
		}
	}
}

func (c *Claim) LoadReviewer(tx *pop.Connection, reload bool) {
	if c.ReviewerID.Valid && (c.Reviewer.ID == uuid.Nil || reload) {
		if err := tx.Load(c, "Reviewer"); err != nil {
			panic("database error loading Claim.Reviewer, " + err.Error())
		}
	}
}

func (c *Claim) LoadClaimFiles(tx *pop.Connection, reload bool) {
	if len(c.ClaimFiles) == 0 || reload {
		if err := tx.Load(c, "ClaimFiles"); err != nil {
			panic("database error loading Claim.ClaimFiles, " + err.Error())
		}
	}
}

func (c *Claim) ConvertToAPI(tx *pop.Connection) api.Claim {
	c.LoadClaimItems(tx, true)

	return api.Claim{
		ID:               c.ID,
		PolicyID:         c.PolicyID,
		EventDate:        c.EventDate,
		EventType:        c.EventType,
		EventDescription: c.EventDescription,
		Status:           c.Status,
		ReviewDate:       c.ReviewDate,
		ReviewerID:       c.ReviewerID,
		PaymentDate:      c.PaymentDate,
		TotalPayout:      c.TotalPayout,
		Items:            c.ClaimItems.ConvertToAPI(tx),
	}
}

func (c *Claims) ConvertToAPI(tx *pop.Connection) api.Claims {
	claims := make(api.Claims, len(*c))
	for i, cc := range *c {
		claims[i] = cc.ConvertToAPI(tx)
	}
	return claims
}

func ConvertClaimCreateInput(input api.ClaimCreateInput) Claim {
	return Claim{
		EventDate:        input.EventDate,
		EventType:        input.EventType,
		EventDescription: input.EventDescription,
		Status:           api.ClaimStatusDraft,
	}
}

func uniqueClaimReferenceNumber(tx *pop.Connection) string {
	attempts := 0
	for {
		// create reference number in format CAB1234
		ref := fmt.Sprintf("C%s%s",
			domain.RandomString(2, "ABCDEFGHIJKLMNOPQRSTUVWXYZ"),
			domain.RandomString(ClaimReferenceNumberLength-3, "1234567890"))

		count, err := tx.Where("reference_number = ?", ref).Count(Claim{})
		if count == 0 && err == nil {
			return ref
		}

		attempts++
		if attempts > 100 {
			panic(fmt.Errorf("failed to find unique claim reference number after 100 attempts"))
		}
	}
}

// AttachFile adds a previously-stored File to this Claim
func (c *Claim) AttachFile(tx *pop.Connection, input api.ClaimFileAttachInput) (ClaimFile, error) {
	claimFile := NewClaimFile(c.ID, input.FileID)
	return *claimFile, claimFile.Create(tx)
}
