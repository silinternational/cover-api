package models

import (
	"fmt"
	"time"

	"github.com/gobuffalo/pop/v5"
	"github.com/gobuffalo/validate/v3"
	"github.com/gofrs/uuid"

	"github.com/silinternational/cover-api/api"
)

type ClaimFile struct {
	ID        uuid.UUID `db:"id"`
	ClaimID   uuid.UUID `db:"claim_id" validate:"required"`
	FileID    uuid.UUID `db:"file_id" validate:"required"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`

	File File `belongs_to:"files" validate:"-"`
}

type ClaimFiles []ClaimFile

// NewClaimFile makes a new ClaimFile but does not save it to the database
func NewClaimFile(claimID, fileID uuid.UUID) *ClaimFile {
	return &ClaimFile{ClaimID: claimID, FileID: fileID}
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
		File:      c.File.ConvertToAPI(),
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
