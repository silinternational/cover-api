package models

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gobuffalo/buffalo"
	"github.com/gobuffalo/events"
	"github.com/gobuffalo/nulls"
	"github.com/gobuffalo/pop/v6"
	"github.com/gobuffalo/validate/v3"
	"github.com/gofrs/uuid"

	"github.com/silinternational/cover-api/api"
	"github.com/silinternational/cover-api/domain"
	"github.com/silinternational/cover-api/log"
)

const (
	ClaimReferenceNumberLength = 7
)

var ValidClaimIncidentTypes = map[api.ClaimIncidentType]struct{}{
	api.ClaimIncidentTypeTheft:           {},
	api.ClaimIncidentTypePhysicalDamage:  {},
	api.ClaimIncidentTypeElectricalSurge: {},
	api.ClaimIncidentTypeFireDamage:      {},
	api.ClaimIncidentTypeWaterDamage:     {},
	api.ClaimIncidentTypeEvacuation:      {},
	api.ClaimIncidentTypeOther:           {},
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
}

var ValidClaimIncidentTypePayoutOptions = map[api.ClaimIncidentType]map[api.PayoutOption]struct{}{
	api.ClaimIncidentTypeEvacuation: {
		api.PayoutOptionFixedFraction: {},
	},
	api.ClaimIncidentTypeTheft: {
		api.PayoutOptionFMV:         {},
		api.PayoutOptionReplacement: {},
	},
	api.ClaimIncidentTypePhysicalDamage: {
		api.PayoutOptionFMV:         {},
		api.PayoutOptionReplacement: {},
		api.PayoutOptionRepair:      {},
	},
	api.ClaimIncidentTypeElectricalSurge: {
		api.PayoutOptionFMV:         {},
		api.PayoutOptionReplacement: {},
		api.PayoutOptionRepair:      {},
	},
	api.ClaimIncidentTypeFireDamage: {
		api.PayoutOptionFMV:         {},
		api.PayoutOptionReplacement: {},
		api.PayoutOptionRepair:      {},
	},
	api.ClaimIncidentTypeWaterDamage: {
		api.PayoutOptionFMV:         {},
		api.PayoutOptionReplacement: {},
		api.PayoutOptionRepair:      {},
	},
	api.ClaimIncidentTypeOther: {
		api.PayoutOptionFMV:         {},
		api.PayoutOptionReplacement: {},
		api.PayoutOptionRepair:      {},
	},
}

type Claims []Claim

type Claim struct {
	ID                  uuid.UUID             `db:"id"`
	PolicyID            uuid.UUID             `db:"policy_id" validate:"required"`
	ReferenceNumber     string                `db:"reference_number" validate:"required,len=7"`
	IncidentDate        time.Time             `db:"incident_date" validate:"required_unless=Status Draft"`
	IncidentType        api.ClaimIncidentType `db:"incident_type" validate:"claimIncidentType,required_unless=Status Draft"`
	IncidentDescription string                `db:"incident_description" validate:"required_unless=Status Draft"`
	Status              api.ClaimStatus       `db:"status" validate:"claimStatus"`
	StatusChange        string                `db:"status_change"`
	ReviewDate          nulls.Time            `db:"review_date"`
	ReviewerID          nulls.UUID            `db:"reviewer_id"`
	PaymentDate         nulls.Time            `db:"payment_date"`
	TotalPayout         api.Currency          `db:"total_payout"`
	StatusReason        string                `db:"status_reason" validate:"required_if=Status Revision,required_if=Status Denied"`
	City                string                `db:"city"`
	State               string                `db:"state"`
	Country             string                `db:"country"`
	LegacyID            nulls.Int             `db:"legacy_id"`
	CreatedAt           time.Time             `db:"created_at"`
	UpdatedAt           time.Time             `db:"updated_at"`

	Policy     Policy     `belongs_to:"policies" validate:"-"`
	ClaimItems ClaimItems `has_many:"claim_items" validate:"-"`
	ClaimFiles ClaimFiles `has_many:"claim_files" validate:"-"`
	Reviewer   User       `belongs_to:"users" validate:"-"`
}

// Validate gets run every time you call a "pop.Validate*" (pop.ValidateAndSave, pop.ValidateAndCreate, pop.ValidateAndUpdate) method.
func (c *Claim) Validate(tx *pop.Connection) (*validate.Errors, error) {
	return validateModel(c), nil
}

// CreateWithHistory stores the Claim data as a new record in the database. Also creates a ClaimHistory record.
// If its status is not valid, it is created in Draft status.
func (c *Claim) CreateWithHistory(ctx context.Context) error {
	tx := Tx(ctx)

	if err := c.Create(tx); err != nil {
		return err
	}

	history := c.NewHistory(ctx, api.HistoryActionCreate, FieldUpdate{})
	if err := history.Create(tx); err != nil {
		return appErrorFromDB(err, api.ErrorCreateFailure)
	}
	return nil
}

// Create a Claim but not a history record. Use CreateWithHistory if history is needed.
func (c *Claim) Create(tx *pop.Connection) error {
	c.ReferenceNumber = uniqueClaimReferenceNumber(tx)
	if _, ok := ValidClaimStatus[c.Status]; !ok {
		c.Status = api.ClaimStatusDraft
	}

	return create(tx, c)
}

// Update writes the Claim data to an existing database record.
func (c *Claim) Update(ctx context.Context) error {
	tx := Tx(ctx)

	var oldClaim Claim
	if err := oldClaim.FindByID(tx, c.ID); err != nil {
		return appErrorFromDB(err, api.ErrorQueryFailure)
	}

	validTrans, err := isClaimTransitionValid(oldClaim.Status, c.Status)
	if err != nil {
		panic(err)
	}
	if !validTrans {
		err := fmt.Errorf("invalid claim status transition from %s to %s",
			oldClaim.Status, c.Status)
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

	if err = c.calculatePayout(ctx); err != nil {
		return err
	}

	if err = update(tx, c); err != nil {
		return err
	}

	updates := c.Compare(oldClaim)
	for i := range updates {
		history := c.NewHistory(ctx, api.HistoryActionUpdate, updates[i])
		if err := history.Create(tx); err != nil {
			return appErrorFromDB(err, api.ErrorCreateFailure)
		}
	}

	return nil
}

// UpdateByUser ensures the Claim has an appropriate status for being modified by the user and then writes the Claim
// data to an existing database record.
func (c *Claim) UpdateByUser(ctx context.Context) error {
	user := CurrentUser(ctx)

	// If the user edits something, it should take it off of the steward's list of things to review and
	//  also force the user to resubmit it.
	switch c.Status {
	case api.ClaimStatusReview1, api.ClaimStatusReview2, api.ClaimStatusReview3:
		c.Status = api.ClaimStatusDraft
		c.StatusChange = ClaimStatusChangeReturnedToDraft + user.Name()
	}

	if user.IsAdmin() {
		if c.Status.WasReviewed() {
			c.setReviewer(ctx)
		}
		return c.Update(ctx)
	}

	if !c.canUpdate(user) {
		err := errors.New("user may not edit a claim that is too far along in the review process")
		appErr := api.NewAppError(err, api.ErrorClaimStatus, api.CategoryUser)
		return appErr
	}

	return c.Update(ctx)
}

// Delete ensures the claim does not have a status of approved, denied or paid and then deletes the claim's items and
// the claim itself.
func (c *Claim) Delete(ctx context.Context) error {
	tx := Tx(ctx)

	var oldClaim Claim
	if err := oldClaim.FindByID(tx, c.ID); err != nil {
		return appErrorFromDB(err, api.ErrorQueryFailure)
	}

	if !c.IsRemovable() {
		err := errors.New("claim that has been approved, paid or denied may not be deleted")
		appErr := api.NewAppError(err, api.ErrorClaimStatus, api.CategoryUser)
		return appErr
	}

	c.LoadClaimItems(tx, false)
	for i := range c.ClaimItems {
		if err := tx.Destroy(&c.ClaimItems[i]); err != nil {
			return appErrorFromDB(fmt.Errorf("error destroying claim item: %w", err), api.ErrorQueryFailure)
		}
	}

	if err := tx.Destroy(c); err != nil {
		return appErrorFromDB(fmt.Errorf("error destroying claim: %w", err), api.ErrorQueryFailure)
	}

	return nil
}

// IsRemovable determines whether the claim may be deleted. It may not be if its status is approved, paid or denied.
func (c *Claim) IsRemovable() bool {
	switch c.Status {
	case api.ClaimStatusApproved, api.ClaimStatusPaid, api.ClaimStatusDenied:
		return false
	}
	return true
}

func (c *Claim) canUpdate(user User) bool {
	if user.IsAdmin() {
		return true
	}

	switch c.Status {
	// cannot modify this when the Claim has one of these statuses
	case api.ClaimStatusApproved, api.ClaimStatusDenied, api.ClaimStatusPaid:
		return false
	}

	return true
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
		log.Errorf("failed to load Policy %s for Claim: %s", c.PolicyID, err)
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
		},
		api.ClaimStatusReview1: {
			api.ClaimStatusDraft,
			api.ClaimStatusRevision,
			api.ClaimStatusReceipt,
			api.ClaimStatusReview3,
			api.ClaimStatusDenied,
		},
		api.ClaimStatusRevision: {
			api.ClaimStatusDraft,
			api.ClaimStatusReview1,
		},
		api.ClaimStatusReceipt: {
			api.ClaimStatusDraft,
			api.ClaimStatusReview2,
		},
		api.ClaimStatusReview2: {
			api.ClaimStatusDraft,
			api.ClaimStatusRevision,
			api.ClaimStatusReceipt,
			api.ClaimStatusReview3,
			api.ClaimStatusDenied,
		},
		api.ClaimStatusReview3: {
			api.ClaimStatusDraft,
			api.ClaimStatusRevision,
			api.ClaimStatusReceipt,
			api.ClaimStatusApproved,
			api.ClaimStatusDenied,
		},
		api.ClaimStatusApproved: {
			api.ClaimStatusPaid,
		},
		api.ClaimStatusDenied: {},
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

func (c *Claim) AddItem(ctx context.Context, input api.ClaimItemCreateInput) (ClaimItem, error) {
	tx := Tx(ctx)
	if c == nil {
		panic("claim is nil in AddItem")
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

	if c.PolicyID != item.PolicyID {
		err := fmt.Errorf("claim and item do not have same policy id: %s vs. %s",
			c.PolicyID.String(), item.PolicyID.String())
		appErr := api.NewAppError(err, api.ErrorClaimItemCreateInvalidInput, api.CategoryNotFound)
		return ClaimItem{}, appErr
	}

	claimItem, err := NewClaimItem(tx, input, item, *c)
	if err != nil {
		return claimItem, err
	}

	if err = claimItem.CreateWithHistory(ctx); err != nil {
		return ClaimItem{}, err
	}

	return claimItem, nil
}

// SubmitForApproval changes the status of the claim to either Review1 or Review2 depending on its current status.
func (c *Claim) SubmitForApproval(ctx context.Context) error {
	tx := Tx(ctx)
	user := CurrentUser(ctx)

	oldStatus := c.Status
	var eventType string

	switch oldStatus {
	case api.ClaimStatusDraft, api.ClaimStatusRevision:
		c.Status = api.ClaimStatusReview1
		c.StatusChange = ClaimStatusChangeReview1
		eventType = domain.EventApiClaimReview1
	case api.ClaimStatusReceipt:
		if !c.HasReceiptFile(tx) {
			err := errors.New("submitting this claim at this stage is not allowed until a receipt is attached")
			return api.NewAppError(err, api.ErrorClaimStatus, api.CategoryUser)
		}
		c.Status = api.ClaimStatusReview2
		c.StatusChange = ClaimStatusChangeReview2 + user.Name()
		eventType = domain.EventApiClaimReview2
	default:
		err := fmt.Errorf("invalid claim status for submit: %s", oldStatus)
		return api.NewAppError(err, api.ErrorClaimStatus, api.CategoryUser)
	}

	c.LoadClaimItems(tx, false)
	if len(c.ClaimItems) == 0 {
		err := errors.New("claim must have a claimItem if no longer in draft")
		appErr := api.NewAppError(err, api.ErrorClaimMissingClaimItem, api.CategoryUser)
		return appErr
	}
	for _, ci := range c.ClaimItems {
		if errorKey := ci.ValidateForSubmit(tx); errorKey != "" {
			err := fmt.Errorf("claim item %s is not valid for claim submission", ci.ID)
			return api.NewAppError(err, errorKey, api.CategoryUser)
		}
	}

	if err := c.Update(ctx); err != nil {
		return err
	}

	e := events.Event{
		Kind:    eventType,
		Message: fmt.Sprintf("Claim Submitted: %s  ID: %s", c.IncidentDescription, c.ID.String()),
		Payload: events.Payload{domain.EventPayloadID: c.ID},
	}
	emitEvent(e)

	return nil
}

// RequestRevision changes the status of the claim to Revision
func (c *Claim) RequestRevision(ctx context.Context, message string) error {
	user := CurrentUser(ctx)

	c.Status = api.ClaimStatusRevision
	c.StatusChange = ClaimStatusChangeRevisions + user.Name()
	c.StatusReason = message
	c.setReviewer(ctx)

	if err := c.Update(ctx); err != nil {
		return err
	}

	e := events.Event{
		Kind:    domain.EventApiClaimRevision,
		Message: fmt.Sprintf("Claim Revision: %s  ID: %s", c.IncidentDescription, c.ID.String()),
		Payload: events.Payload{domain.EventPayloadID: c.ID},
	}
	emitEvent(e)

	return nil
}

// RequestReceipt changes the status of the claim to Receipt provided that the current status is Review1.
func (c *Claim) RequestReceipt(ctx buffalo.Context, reason string) error {
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

	user := CurrentUser(ctx)
	c.Status = api.ClaimStatusReceipt
	c.StatusChange = ClaimStatusChangeReceipt + user.Name()
	c.StatusReason = reason
	c.setReviewer(ctx)

	if err := c.Update(ctx); err != nil {
		return err
	}

	e := events.Event{
		Kind:    eventType,
		Message: fmt.Sprintf("Claim Request Receipt: %s  ID: %s", c.IncidentDescription, c.ID.String()),
		Payload: events.Payload{domain.EventPayloadID: c.ID},
	}
	emitEvent(e)

	return nil
}

// Approve changes the status of the claim from either Review1, Review2 to Review3 or from Review3 to Approved. It also
// adds the ReviewerID and ReviewDate.
func (c *Claim) Approve(ctx context.Context) error {
	var eventType string

	user := CurrentUser(ctx)

	switch c.Status {
	case api.ClaimStatusReview1:
		c.LoadClaimItems(Tx(ctx), false)
		payOption := c.ClaimItems[0].PayoutOption
		if payOption != api.PayoutOptionFMV && payOption != api.PayoutOptionFixedFraction {
			err := fmt.Errorf("cannot approve payout option %s from status %s", payOption, c.Status)
			appErr := api.NewAppError(err, api.ErrorClaimItemInvalidPayoutOption, api.CategoryUser)
			return appErr
		}
		c.Status = api.ClaimStatusReview3
		c.StatusChange = ClaimStatusChangeReview3 + user.Name()
		eventType = domain.EventApiClaimReview3
	case api.ClaimStatusReview2:
		c.Status = api.ClaimStatusReview3
		c.StatusChange = ClaimStatusChangeReview3 + user.Name()
		eventType = domain.EventApiClaimReview3
	case api.ClaimStatusReview3:
		if user.ID == c.ReviewerID.UUID {
			err := fmt.Errorf("different approver required for final approval")
			appErr := api.NewAppError(err, api.ErrorClaimInvalidApprover, api.CategoryUser)
			return appErr
		}
		c.Status = api.ClaimStatusApproved
		c.StatusChange = ClaimStatusChangeApproved + user.Name()
		eventType = domain.EventApiClaimApproved
	default:
		err := fmt.Errorf("invalid claim status for approve: %s", c.Status)
		appErr := api.NewAppError(err, api.ErrorClaimStatus, api.CategoryUser)
		return appErr
	}

	c.setReviewer(ctx)

	if err := c.Update(ctx); err != nil {
		return err
	}

	e := events.Event{
		Kind:    eventType,
		Message: fmt.Sprintf("Claim Approved: %s  ID: %s", c.IncidentDescription, c.ID.String()),
		Payload: events.Payload{domain.EventPayloadID: c.ID},
	}
	emitEvent(e)

	if c.Status == api.ClaimStatusApproved {
		return c.CreateLedgerEntry(Tx(ctx))
	}
	return nil
}

// Deny changes the status of the claim to Denied and adds the ReviewerID and ReviewDate.
func (c *Claim) Deny(ctx context.Context, message string) error {
	oldStatus := c.Status

	if oldStatus != api.ClaimStatusReview1 && oldStatus != api.ClaimStatusReview2 &&
		oldStatus != api.ClaimStatusReview3 {
		err := fmt.Errorf("invalid claim status for deny: %s", oldStatus)
		appErr := api.NewAppError(err, api.ErrorClaimStatus, api.CategoryUser)
		return appErr
	}

	user := CurrentUser(ctx)

	c.Status = api.ClaimStatusDenied
	c.StatusChange = ClaimStatusChangeDenied + user.Name()
	c.StatusReason = message
	c.setReviewer(ctx)

	if err := c.Update(ctx); err != nil {
		return err
	}

	e := events.Event{
		Kind:    domain.EventApiClaimDenied,
		Message: fmt.Sprintf("Claim Denied: %s  ID: %s", c.IncidentDescription, c.ID.String()),
		Payload: events.Payload{domain.EventPayloadID: c.ID},
	}
	emitEvent(e)

	return nil
}

func (c *Claim) LoadClaimItems(tx *pop.Connection, reload bool) {
	if len(c.ClaimItems) == 0 || reload {
		if err := tx.Load(c, "ClaimItems", "ClaimItems.Item"); err != nil {
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

func (c *Claim) LoadPolicyMembers(tx *pop.Connection, reload bool) {
	c.LoadPolicy(tx, reload)

	c.Policy.LoadMembers(tx, reload)
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

func (c *Claim) HasReceiptFile(tx *pop.Connection) bool {
	var claimFiles ClaimFiles
	count, err := tx.Where("claim_id = ? AND purpose = ?", c.ID, api.ClaimFilePurposeReceipt).Count(&claimFiles)
	if err != nil {
		panic("error trying to count Claim's receipt files")
	}
	return count > 0
}

func (c *Claim) ConvertToAPI(tx *pop.Connection) api.Claim {
	c.LoadClaimItems(tx, true)
	c.LoadClaimFiles(tx, true)

	return api.Claim{
		ID:                  c.ID,
		PolicyID:            c.PolicyID,
		ReferenceNumber:     c.ReferenceNumber,
		IncidentDate:        c.IncidentDate,
		IncidentType:        c.IncidentType,
		IncidentDescription: c.IncidentDescription,
		Status:              c.Status,
		StatusChange:        c.StatusChange,
		ReviewDate:          convertTimeToAPI(c.ReviewDate),
		ReviewerID:          convertUUIDToAPI(c.ReviewerID),
		PaymentDate:         convertTimeToAPI(c.PaymentDate),
		TotalPayout:         c.TotalPayout,
		StatusReason:        c.StatusReason,
		IsRemovable:         c.IsRemovable(),
		Items:               c.ClaimItems.ConvertToAPI(tx),
		Files:               c.ClaimFiles.ConvertToAPI(tx),
	}
}

func (c *Claims) ConvertToAPI(tx *pop.Connection) api.Claims {
	claims := make(api.Claims, len(*c))
	for i, cc := range *c {
		claims[i] = cc.ConvertToAPI(tx)
	}
	return claims
}

func (c *Claims) ByStatus(tx *pop.Connection, statuses []api.ClaimStatus) error {
	if len(statuses) == 0 {
		statuses = []api.ClaimStatus{
			api.ClaimStatusReview1,
			api.ClaimStatusReview2,
			api.ClaimStatusReview3,
		}
	}

	err := tx.Where("status in (?)", statuses).All(c)
	return appErrorFromDB(err, api.ErrorQueryFailure)
}

func ConvertClaimCreateInput(input api.ClaimCreateInput) Claim {
	return Claim{
		IncidentDate:        input.IncidentDate,
		IncidentType:        input.IncidentType,
		IncidentDescription: input.IncidentDescription,
		Status:              api.ClaimStatusDraft,
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
		if domain.IsOtherThanNoRows(err) {
			panic("database error: " + err.Error())
		}
		if count == 0 && err == nil {
			return ref
		}

		attempts++
		if attempts > 100 {
			panic(fmt.Errorf("failed to find unique claim reference number after %d attempts", attempts-1))
		}
	}
}

// AttachFile adds a previously-stored File to this Claim
func (c *Claim) AttachFile(tx *pop.Connection, input api.ClaimFileAttachInput) (ClaimFile, error) {
	claimFile := NewClaimFile(c.ID, input.FileID, input.Purpose)
	return *claimFile, claimFile.Create(tx)
}

// Compare returns a list of fields that are different between two objects
func (c *Claim) Compare(old Claim) []FieldUpdate {
	var updates []FieldUpdate

	if c.PolicyID != old.PolicyID {
		updates = append(updates, FieldUpdate{
			OldValue:  old.PolicyID.String(),
			NewValue:  c.PolicyID.String(),
			FieldName: FieldClaimPolicyID,
		})
	}

	if c.ReferenceNumber != old.ReferenceNumber {
		updates = append(updates, FieldUpdate{
			OldValue:  old.ReferenceNumber,
			NewValue:  c.ReferenceNumber,
			FieldName: FieldClaimReferenceNumber,
		})
	}

	if c.IncidentDate != old.IncidentDate {
		updates = append(updates, FieldUpdate{
			OldValue:  old.IncidentDate.String(),
			NewValue:  c.IncidentDate.String(),
			FieldName: FieldClaimIncidentDate,
		})
	}

	if c.IncidentType != old.IncidentType {
		updates = append(updates, FieldUpdate{
			OldValue:  string(old.IncidentType),
			NewValue:  string(c.IncidentType),
			FieldName: FieldClaimIncidentType,
		})
	}

	if c.IncidentDescription != old.IncidentDescription {
		updates = append(updates, FieldUpdate{
			OldValue:  old.IncidentDescription,
			NewValue:  c.IncidentDescription,
			FieldName: FieldClaimIncidentDescription,
		})
	}

	if c.Status != old.Status {
		updates = append(updates, FieldUpdate{
			OldValue:  string(old.Status),
			NewValue:  string(c.Status),
			FieldName: FieldClaimStatus,
		})
	}

	if c.ReviewDate != old.ReviewDate {
		updates = append(updates, FieldUpdate{
			OldValue:  old.ReviewDate.Time.String(),
			NewValue:  c.ReviewDate.Time.String(),
			FieldName: FieldClaimReviewDate,
		})
	}

	if c.ReviewerID != old.ReviewerID {
		updates = append(updates, FieldUpdate{
			OldValue:  old.ReviewerID.UUID.String(),
			NewValue:  c.ReviewerID.UUID.String(),
			FieldName: FieldClaimReviewerID,
		})
	}

	if c.PaymentDate != old.PaymentDate {
		updates = append(updates, FieldUpdate{
			OldValue:  old.PaymentDate.Time.String(),
			NewValue:  c.PaymentDate.Time.String(),
			FieldName: FieldClaimPaymentDate,
		})
	}

	if c.TotalPayout != old.TotalPayout {
		updates = append(updates, FieldUpdate{
			OldValue:  old.TotalPayout.String(),
			NewValue:  c.TotalPayout.String(),
			FieldName: FieldClaimTotalPayout,
		})
	}

	if c.StatusReason != old.StatusReason {
		updates = append(updates, FieldUpdate{
			OldValue:  old.StatusReason,
			NewValue:  c.StatusReason,
			FieldName: FieldClaimStatusReason,
		})
	}

	if c.City != old.City {
		updates = append(updates, FieldUpdate{
			OldValue:  old.City,
			NewValue:  c.City,
			FieldName: FieldClaimCity,
		})
	}

	if c.State != old.State {
		updates = append(updates, FieldUpdate{
			OldValue:  old.State,
			NewValue:  c.State,
			FieldName: FieldClaimState,
		})
	}

	if c.Country != old.Country {
		updates = append(updates, FieldUpdate{
			OldValue:  old.Country,
			NewValue:  c.Country,
			FieldName: FieldClaimCountry,
		})
	}

	return updates
}

func (c *Claim) NewHistory(ctx context.Context, action string, fieldUpdate FieldUpdate) ClaimHistory {
	return ClaimHistory{
		Action:    action,
		ClaimID:   c.ID,
		UserID:    CurrentUser(ctx).ID,
		FieldName: fieldUpdate.FieldName,
		OldValue:  fmt.Sprintf("%s", fieldUpdate.OldValue),
		NewValue:  fmt.Sprintf("%s", fieldUpdate.NewValue),
	}
}

func (c *Claim) setReviewer(ctx context.Context) {
	actor := CurrentUser(ctx)
	c.ReviewerID = nulls.NewUUID(actor.ID)
	c.ReviewDate = nulls.NewTime(time.Now().UTC())
}

// ClaimsWithRecentStatusChanges returns the RecentClaims associated with claims that have had their Status changed
// recently. The slice is sorted by updated time with most recent first.
func ClaimsWithRecentStatusChanges(tx *pop.Connection) (api.RecentClaims, error) {
	var cHistories ClaimHistories

	if err := cHistories.RecentClaimStatusChanges(tx); err != nil {
		return api.RecentClaims{}, err
	}

	// Fetch the actual claims from the database and convert them to api types
	claims := make(api.RecentClaims, len(cHistories))
	for i, next := range cHistories {
		var claim Claim
		if err := claim.FindByID(tx, next.ClaimID); err != nil {
			panic("error finding claim by ID: " + err.Error())
		}

		apiClaim := claim.ConvertToAPI(tx)
		claims[i] = api.RecentClaim{Claim: apiClaim, StatusUpdatedAt: next.CreatedAt}
	}

	return claims, nil
}

// CreateLedgerEntry does nothing if the TotalPayout is zero
func (c *Claim) CreateLedgerEntry(tx *pop.Connection) error {
	if c.Status != api.ClaimStatusApproved {
		return errors.New("cannot pay out a claim that is not approved")
	}

	if c.TotalPayout == 0 {
		return nil
	}

	adjustedAmount, err := adjustLedgerAmount(c.TotalPayout, LedgerEntryTypeClaim)
	if err != nil {
		return err
	}

	c.LoadClaimItems(tx, false)
	c.LoadPolicy(tx, false)
	c.Policy.LoadEntityCode(tx, false)

	now := time.Now().UTC()

	for i := range c.ClaimItems {
		c.ClaimItems[i].LoadItem(tx, false)
		item := c.ClaimItems[i].Item
		item.LoadRiskCategory(tx, false)
		name := item.GetAccountablePersonName(tx).String()

		le := NewLedgerEntry(name, c.Policy, &item, c, now)
		le.Type = LedgerEntryTypeClaim
		le.Amount = adjustedAmount

		le.RiskCategoryName = item.RiskCategory.Name
		le.RiskCategoryCC = item.RiskCategory.CostCenter
		le.IncomeAccount = domain.Env.ClaimIncomeAccount

		if err := le.Create(tx); err != nil {
			return err
		}
	}
	return nil
}

func (c *Claim) UpdateStatus(ctx context.Context, newStatus api.ClaimStatus) error {
	if newStatus == c.Status {
		return nil
	}
	oldStatus := c.Status
	c.Status = newStatus
	tx := Tx(ctx)
	if err := tx.UpdateColumns(c, "status", "updated_at"); err != nil {
		return appErrorFromDB(err, api.ErrorUpdateFailure)
	}

	history := c.NewHistory(ctx, api.HistoryActionUpdate, FieldUpdate{
		FieldName: FieldClaimStatus,
		OldValue:  string(oldStatus),
		NewValue:  string(newStatus),
	})
	if err := history.Create(tx); err != nil {
		return appErrorFromDB(err, api.ErrorCreateFailure)
	}

	return nil
}

// Based on the claim's UpdatedAt field, unless there is a ClaimHistory for this claim that is for a Status Update where
// the new field is Review1.  Uses the CreatedAt time of the earliest history with Status change to Review1.
func (c *Claim) SubmittedAt(tx *pop.Connection) time.Time {
	var histories ClaimHistories

	err := tx.RawQuery(`
		SELECT created_at
		FROM claim_histories
		WHERE claim_id = ? AND field_name = ? AND action = ? AND new_value = ?
		ORDER BY created_at ASC
		LIMIT 1
		`, c.ID, FieldClaimStatus, api.HistoryActionUpdate, api.ClaimStatusReview1).All(&histories)
	if err != nil {
		log.Error("error finding claim's histories:", err)
		return c.UpdatedAt
	}

	if len(histories) == 0 {
		return c.UpdatedAt
	}

	return histories[0].CreatedAt
}

func (c *Claim) calculatePayout(ctx context.Context) error {
	switch c.Status {
	case api.ClaimStatusPaid, api.ClaimStatusDenied, api.ClaimStatusApproved:
		return nil
	}

	c.LoadClaimItems(Tx(ctx), false)
	var payout api.Currency
	for _, claimItem := range c.ClaimItems {
		if err := claimItem.updatePayoutAmount(ctx); err != nil {
			return err
		}
		payout += claimItem.PayoutAmount
	}
	c.TotalPayout = payout
	return nil
}

func (c *Claim) GetDeductibleRate(tx *pop.Connection) float64 {
	cutOff := c.IncidentDate
	if c.IncidentDate.IsZero() {
		cutOff = c.CreatedAt
	}

	c.LoadPolicy(tx, false)
	var strikes Strikes
	err := strikes.RecentForPolicy(tx, c.PolicyID, cutOff)

	if domain.IsOtherThanNoRows(err) {
		log.Errorf("error retrieving recent strikes for claim %s: %s", c.ID.String(), err)
		return domain.Env.Deductible
	}

	extra := domain.Env.DeductibleIncrease * float64(len(strikes))

	d := domain.Env.Deductible + extra
	if d >= domain.Env.DeductibleMaximum {
		return domain.Env.DeductibleMaximum
	}
	return d
}

// StopItemCoverage sets the claim's items' statuses to `Inactive` and creates refund ledger entries for them.
//
// Returns an error if the claim's status or item's coverage status is not `Approved`
func (c *Claim) StopItemCoverage(tx *pop.Connection) error {
	if c.Status != api.ClaimStatusApproved {
		return errors.New("cannot auto-stop coverage on an item the claim of which is not approved")
	}

	c.LoadClaimItems(tx, true)

	for _, ci := range c.ClaimItems {
		if ci.PayoutOption == api.PayoutOptionRepair {
			continue
		}

		reason := "replacement claim approved on item"
		if err := ci.Item.cancelCoverageAfterClaim(tx, reason); err != nil {
			return err
		}
	}

	return nil
}
