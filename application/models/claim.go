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
	"github.com/gobuffalo/pop/v5"
	"github.com/gobuffalo/validate/v3"
	"github.com/gofrs/uuid"

	"github.com/silinternational/cover-api/api"
	"github.com/silinternational/cover-api/domain"
)

const (
	ClaimReferenceNumberLength = 7
)

var ValidClaimIncidentTypes = map[api.ClaimIncidentType]struct{}{
	api.ClaimIncidentTypeTheft:           {},
	api.ClaimIncidentTypeImpact:          {},
	api.ClaimIncidentTypeElectricalSurge: {},
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
	api.ClaimIncidentTypeImpact: {
		api.PayoutOptionFMV:         {},
		api.PayoutOptionReplacement: {},
		api.PayoutOptionRepair:      {},
	},
	api.ClaimIncidentTypeElectricalSurge: {
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
	ReviewDate          nulls.Time            `db:"review_date"`
	ReviewerID          nulls.UUID            `db:"reviewer_id"`
	PaymentDate         nulls.Time            `db:"payment_date"`
	TotalPayout         api.Currency          `db:"total_payout"`
	LegacyID            nulls.Int             `db:"legacy_id"`
	StatusReason        string                `db:"status_reason" validate:"required_if=Status Revision,required_if=Status Denied"`
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
	updates := c.Compare(oldClaim)
	for i := range updates {
		history := c.NewHistory(ctx, api.HistoryActionUpdate, updates[i])
		if err := history.Create(tx); err != nil {
			return appErrorFromDB(err, api.ErrorCreateFailure)
		}
	}

	return update(tx, c)
}

// UpdateByUser ensures the Claim has an appropriate status for being modified by the user
//  and then writes the Claim data to an existing database record.
func (c *Claim) UpdateByUser(ctx context.Context) error {
	user := CurrentUser(ctx)
	if user.IsAdmin() {
		if c.Status.WasReviewed() {
			c.setReviewer(ctx)
		}
		return c.Update(ctx)
	}

	switch c.Status {
	// OK to modify the Claim when it has one of these statuses but not any others
	case api.ClaimStatusDraft, api.ClaimStatusRevision, api.ClaimStatusReview1:
	default:
		err := errors.New("user may not edit a claim that is too far along in the review process.")
		appErr := api.NewAppError(err, api.ErrorClaimStatus, api.CategoryUser)
		return appErr
	}

	// If the user edits something, it should take it off of the steward's list of things to review and
	//  also force the user to resubmit it.
	if c.Status == api.ClaimStatusReview1 {
		c.Status = api.ClaimStatusDraft
	}

	return c.Update(ctx)
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
// TODO Differentiate between admins (steward and signator)
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
		},
		api.ClaimStatusReview1: {
			api.ClaimStatusDraft,
			api.ClaimStatusRevision,
			api.ClaimStatusReceipt,
			api.ClaimStatusReview3,
			api.ClaimStatusDenied,
		},
		api.ClaimStatusRevision: {
			api.ClaimStatusReview1,
		},
		api.ClaimStatusReceipt: {
			api.ClaimStatusReview2,
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

func (c *Claim) AddItem(tx *pop.Connection, input api.ClaimItemCreateInput) (ClaimItem, error) {
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

	clmItem := ConvertClaimItemCreateInput(input)
	clmItem.ClaimID = c.ID

	if err := clmItem.Create(tx); err != nil {
		return ClaimItem{}, err
	}

	return clmItem, nil
}

// SubmitForApproval changes the status of the claim to either Review1 or Review2
//   depending on its current status.
func (c *Claim) SubmitForApproval(ctx context.Context) error {
	tx := Tx(ctx)

	oldStatus := c.Status
	var eventType string

	switch oldStatus {
	case api.ClaimStatusDraft, api.ClaimStatusRevision:
		c.Status = api.ClaimStatusReview1
		eventType = domain.EventApiClaimReview1
	case api.ClaimStatusReceipt:
		if !c.HasReceiptFile(tx) {
			err := errors.New("submitting this claim at this stage is not allowed until a receipt is attached")
			return api.NewAppError(err, api.ErrorClaimStatus, api.CategoryUser)
		}
		c.Status = api.ClaimStatusReview2
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
//   provided that the current status is Review1 or Review3.
func (c *Claim) RequestRevision(ctx context.Context, message string) error {
	oldStatus := c.Status

	switch oldStatus {
	case api.ClaimStatusReview1, api.ClaimStatusReview3:
		c.Status = api.ClaimStatusRevision
		c.StatusReason = message
		c.setReviewer(ctx)
	default:
		err := fmt.Errorf("invalid claim status for request revision: %s", oldStatus)
		appErr := api.NewAppError(err, api.ErrorClaimStatus, api.CategoryUser)
		return appErr
	}

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

// RequestReceipt changes the status of the claim to Receipt
//   provided that the current status is Review1.
// TODO consider how to communicate what kind of receipt is needed
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

	c.Status = api.ClaimStatusReceipt
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

// Approve changes the status of the claim from either Review1, Review2 to Review3 or
//  from Review3 to Approved. It also adds the ReviewerID and ReviewDate.
// TODO distinguish between admin types (steward vs. signator)
// TODO do whatever post-processing is needed for payment
func (c *Claim) Approve(ctx context.Context) error {
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

	return c.CreateLedgerEntry(Tx(ctx))
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

	c.Status = api.ClaimStatusDenied
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
		ReviewDate:          c.ReviewDate,
		ReviewerID:          c.ReviewerID,
		PaymentDate:         c.PaymentDate,
		TotalPayout:         c.TotalPayout,
		StatusReason:        c.StatusReason,
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

func (c *Claims) All(tx *pop.Connection) error {
	return appErrorFromDB(tx.All(c), api.ErrorQueryFailure)
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

	if c.IncidentDate != old.IncidentDate {
		updates = append(updates, FieldUpdate{
			OldValue:  old.IncidentDate.String(),
			NewValue:  c.IncidentDate.String(),
			FieldName: "IncidentDate",
		})
	}

	if c.IncidentType != old.IncidentType {
		updates = append(updates, FieldUpdate{
			OldValue:  string(old.IncidentType),
			NewValue:  string(c.IncidentType),
			FieldName: "IncidentType",
		})
	}

	if c.IncidentDescription != old.IncidentDescription {
		updates = append(updates, FieldUpdate{
			OldValue:  old.IncidentDescription,
			NewValue:  c.IncidentDescription,
			FieldName: "IncidentDescription",
		})
	}

	if c.Status != old.Status {
		updates = append(updates, FieldUpdate{
			OldValue:  string(old.Status),
			NewValue:  string(c.Status),
			FieldName: "Status",
		})
	}

	if c.ReviewDate != old.ReviewDate {
		updates = append(updates, FieldUpdate{
			OldValue:  old.ReviewDate.Time.String(),
			NewValue:  c.ReviewDate.Time.String(),
			FieldName: "ReviewDate",
		})
	}

	if c.ReviewerID != old.ReviewerID {
		updates = append(updates, FieldUpdate{
			OldValue:  old.ReviewerID.UUID.String(),
			NewValue:  c.ReviewerID.UUID.String(),
			FieldName: "ReviewerID",
		})
	}

	if c.PaymentDate != old.PaymentDate {
		updates = append(updates, FieldUpdate{
			OldValue:  old.PaymentDate.Time.String(),
			NewValue:  c.PaymentDate.Time.String(),
			FieldName: "PaymentDate",
		})
	}

	if c.TotalPayout != old.TotalPayout {
		updates = append(updates, FieldUpdate{
			OldValue:  old.TotalPayout.String(),
			NewValue:  c.TotalPayout.String(),
			FieldName: "TotalPayout",
		})
	}

	if c.StatusReason != old.StatusReason {
		updates = append(updates, FieldUpdate{
			OldValue:  old.StatusReason,
			NewValue:  c.StatusReason,
			FieldName: "StatusReason",
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
	// TODO: decide if we need more review fields for different review steps
	actor := CurrentUser(ctx)
	c.ReviewerID = nulls.NewUUID(actor.ID)
	c.ReviewDate = nulls.NewTime(time.Now().UTC())
}

// ClaimsWithRecentStatusChanges returns the RecentClaims associated with
//  claims that have had their Status changed recently.
//  The slice is sorted by updated time with most recent first.
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

func (c *Claim) CreateLedgerEntry(tx *pop.Connection) error {
	c.LoadPolicy(tx, false)
	c.Policy.LoadEntityCode(tx, false)
	c.Policy.LoadItems(tx, false)

	for _, item := range c.Policy.Items {
		firstName, lastName := item.GetAccountablePersonName(tx)
		item.LoadRiskCategory(tx, false)
		le := LedgerEntry{
			// TODO: check each line below for correctness
			Type:             LedgerEntryTypeClaim,
			RiskCategoryName: item.RiskCategory.Name,
			// ClaimID:            c.ID,
			PolicyID:      c.PolicyID,
			ItemID:        nulls.NewUUID(item.ID),
			EntityCode:    c.Policy.EntityCode.Code,
			Amount:        0,
			DateSubmitted: time.Now().UTC(),
			AccountNumber: c.Policy.Account,
			HouseholdID:   c.Policy.CostCenter,
			FirstName:     firstName,
			LastName:      lastName,
		}
		if err := le.Create(tx); err != nil {
			return err
		}
	}
	return nil
}
