package models

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gobuffalo/pop/v6"
	"github.com/gobuffalo/validate/v3"
	"github.com/gofrs/uuid"

	"github.com/silinternational/cover-api/api"
	"github.com/silinternational/cover-api/log"
	"github.com/silinternational/cover-api/storage"
)

var ValidClaimFilePurpose = map[api.ClaimFilePurpose]struct{}{
	api.ClaimFilePurposeReceipt:        {},
	api.ClaimFilePurposeRepairEstimate: {},
	api.ClaimFilePurposeEvidenceOfFMV:  {},
}

type ClaimFile struct {
	ID        uuid.UUID            `db:"id"`
	ClaimID   uuid.UUID            `db:"claim_id" validate:"required"`
	FileID    uuid.UUID            `db:"file_id" validate:"required"`
	Purpose   api.ClaimFilePurpose `db:"purpose"`
	CreatedAt time.Time            `db:"created_at"`
	UpdatedAt time.Time            `db:"updated_at"`

	File File `belongs_to:"files" validate:"-"`
}

type ClaimFiles []ClaimFile

// NewClaimFile makes a new ClaimFile but does not save it to the database
func NewClaimFile(claimID, fileID uuid.UUID, purpose api.ClaimFilePurpose) *ClaimFile {
	return &ClaimFile{ClaimID: claimID, FileID: fileID, Purpose: purpose}
}

// Validate gets run every time you call a "pop.Validate*" (pop.ValidateAndSave, pop.ValidateAndCreate, pop.ValidateAndUpdate) method.
func (c *ClaimFile) Validate(tx *pop.Connection) (*validate.Errors, error) {
	return validateModel(c), nil
}

// Create stores the file and marks it as linked.
func (c *ClaimFile) Create(tx *pop.Connection) error {
	if err := create(tx, c); err != nil {
		return fmt.Errorf("could not create new ClaimFile, %w", err)
	}

	file := File{ID: c.FileID}
	if err := file.SetLinked(tx); err != nil {
		return fmt.Errorf("could not link new ClaimFile, %w", err)
	}

	return nil
}

// Destroy destroys the claim file and its associated file
func (c *ClaimFile) Destroy(tx *pop.Connection) {
	c.LoadFile(tx, false)
	file := c.File

	if err := tx.Destroy(c); err != nil {
		panic(fmt.Sprintf("database error destroying ClaimFile with ID: %s. %s, ", c.ID.String(), err))
	}

	if err := storage.RemoveFile(file.ID.String()); err != nil {
		log.Errorf("error removing file from S3, id='%s', %s", file.ID.String(), err)
	}

	if err := tx.Destroy(&file); err != nil {
		panic(fmt.Sprintf("database error destroying ClaimFile.File with ID: %s. %s, ", file.ID.String(), err))
	}
}

func (c *ClaimFile) GetID() uuid.UUID {
	return c.ID
}

func (c *ClaimFile) FindByID(tx *pop.Connection, id uuid.UUID) error {
	return tx.Find(c, id)
}

// IsActorAllowedTo ensure the actor is either an admin, or a member of this policy to perform any permission
func (c *ClaimFile) IsActorAllowedTo(tx *pop.Connection, actor User, perm Permission, sub SubResource, r *http.Request) bool {
	if actor.IsAdmin() {
		return true
	}

	var claim Claim
	if err := claim.FindByID(tx, c.ClaimID); err != nil {
		panic(err.Error())
	}

	claim.LoadPolicy(tx, false)
	policy := claim.Policy
	return policy.isMember(tx, actor.ID)
}

// ConvertToAPI converts a ClaimFile to api.ClaimFile
func (c *ClaimFiles) ConvertToAPI(tx *pop.Connection) []api.ClaimFile {
	claims := make([]api.ClaimFile, len(*c))
	for i, cc := range *c {
		claims[i] = cc.ConvertToAPI(tx)
	}
	return claims
}

// ConvertToAPI converts a ClaimFile to api.ClaimFile
func (c *ClaimFile) ConvertToAPI(tx *pop.Connection) api.ClaimFile {
	c.LoadFile(tx, true)

	return api.ClaimFile{
		ID:        c.ID,
		ClaimID:   c.ClaimID,
		FileID:    c.FileID,
		File:      c.File.ConvertToAPI(tx),
		Purpose:   c.Purpose,
		CreatedAt: c.CreatedAt,
		UpdatedAt: c.UpdatedAt,
	}
}

func (c *ClaimFile) LoadFile(tx *pop.Connection, reload bool) {
	if c.File.ID == uuid.Nil || reload {
		if err := tx.Load(c, "File"); err != nil {
			panic("database error loading Claim.File, " + err.Error())
		}
	}
	if err := c.File.RefreshURL(tx); err != nil {
		panic("failed to refresh Claim.File URL, " + err.Error())
	}
}
