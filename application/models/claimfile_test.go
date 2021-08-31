package models

import (
	"testing"

	"github.com/silinternational/cover-api/domain"
)

func (ms *ModelSuite) TestNewClaimFile() {
	claimID := domain.GetUUID()
	fileID := domain.GetUUID()
	got := NewClaimFile(claimID, fileID)
	ms.NotNil(got, "UUT returned a nil pointer")
	ms.Equal(claimID, got.ClaimID)
	ms.Equal(fileID, got.FileID)
}

func (ms *ModelSuite) TestClaimFile_Create() {
	db := ms.DB
	policy := CreatePolicyFixtures(db, FixturesConfig{NumberOfPolicies: 1}).Policies[0]
	claim := createClaimFixture(db, policy.ID, FixturesConfig{})

	files := CreateFileFixtures(db, 3, CreateAdminUser(db).ID).Files
	claim1File := files[0]
	ms.NoError(NewClaimFile(claim.ID, claim1File.ID).Create(db))
	linkedFile := files[1]
	ms.NoError(linkedFile.SetLinked(db))
	newFile := files[2]

	tests := []struct {
		name      string
		claimFile ClaimFile
		wantErr   string
	}{
		{
			name: "attempt to add the same file twice on the same claim",
			claimFile: ClaimFile{
				ClaimID: claim.ID,
				FileID:  claim1File.ID,
			},
			wantErr: "duplicate key value violates unique constraint \"claim_files_file_id_idx\"",
		},
		{
			name: "attempt to reuse a linked file",
			claimFile: ClaimFile{
				ClaimID: claim.ID,
				FileID:  linkedFile.ID,
			},
			wantErr: "already linked",
		},
		{
			name: "no claim ID",
			claimFile: ClaimFile{
				FileID: newFile.ID,
			},
			wantErr: "Field validation for 'ClaimID' failed on the 'required' tag",
		},
		{
			name: "no file ID",
			claimFile: ClaimFile{
				ClaimID: claim.ID,
			},
			wantErr: "Field validation for 'FileID' failed on the 'required' tag",
		},
		{
			name: "ok",
			claimFile: ClaimFile{
				ClaimID: claim.ID,
				FileID:  newFile.ID,
			},
		},
	}
	for _, tt := range tests {
		ms.T().Run(tt.name, func(t *testing.T) {
			var claimFile ClaimFile
			claimID := tt.claimFile.ClaimID
			fileID := tt.claimFile.FileID

			if tt.wantErr == "" {
				ms.Error(db.Where("claim_id = ? AND file_id = ?", claimID, fileID).First(&claimFile),
					"ClaimFile should not exist yet")
			}

			err := tt.claimFile.Create(db)
			if tt.wantErr != "" {
				ms.Error(err)
				ms.Contains(err.Error(), tt.wantErr)
				return
			}
			ms.NoError(err)
			ms.NoError(db.Where("claim_id = ? AND file_id = ?", claimID, fileID).First(&claimFile),
				"new ClaimFile not found in database")
		})
	}
}
