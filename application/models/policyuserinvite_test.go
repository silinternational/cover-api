package models

import (
	"errors"
	"time"

	"github.com/gobuffalo/nulls"
	"github.com/gofrs/uuid"

	"github.com/silinternational/cover-api/api"
	"github.com/silinternational/cover-api/domain"
)

func (ms *ModelSuite) TestPolicyUserInvite_ConvertToAPI() {
	invite := PolicyUserInvite{
		ID:           domain.GetUUID(),
		PolicyID:     domain.GetUUID(),
		Email:        "new_user@example.org",
		InviteeName:  "New User",
		InviterName:  "Sam Supervisor",
		InviterEmail: "Join us!",
	}

	// First test with no EmailSentAt
	got := invite.ConvertToAPI()

	ms.Equal(invite.Email, got.Email, "Email is not correct")
	ms.Equal(invite.InviteeName, got.Name, "Name is not correct")
	ms.Nil(got.EmailSentAt, "EmailSentAt is not null")

	// Second test with EmailSenttAt
	now := time.Now().UTC()
	invite.EmailSentAt = nulls.NewTime(now)
	got = invite.ConvertToAPI()

	ms.Equal(now, *got.EmailSentAt, "EmailSentAt is not correct")
}

func (ms *ModelSuite) TestDestroyIfExpired() {
	tx := ms.DB

	fixturesConfig := FixturesConfig{
		NumberOfPolicies: 2,
	}
	policyFixtures := CreatePolicyFixtures(tx, fixturesConfig)
	policies := policyFixtures.Policies
	pID := policies[0].ID
	pID2 := policies[1].ID

	countInvites := func(id uuid.UUID) int {
		count, err := tx.Where("policy_id = ?", id).Count(&PolicyUserInvite{})
		ms.NoError(err)
		return count
	}

	now := time.Now().UTC()
	cutoff := now.Add(time.Duration(-domain.Env.InviteLifetimeDays) * domain.DurationDay)

	tests := []struct {
		name          string
		invite        PolicyUserInvite
		expectedErr   error
		expectedCount int
	}{
		{
			name:          "Invite not expired",
			invite:        CreateUniqueInvite(now, pID),
			expectedErr:   nil,
			expectedCount: 1,
		},
		{
			name:   "Invite expired",
			invite: CreateUniqueInvite(cutoff.Add(-time.Hour), pID2),
			expectedErr: api.NewAppError(
				errors.New("attempt to use expired invite, ID: "),
				api.ErrorInviteExpired,
				api.CategoryForbidden,
			),
			expectedCount: 1,
		},
	}

	for _, tt := range tests {
		ms.Run(tt.name, func() {
			err := tx.Create(&tt.invite)
			ms.NoError(err)

			err = tt.invite.DestroyIfExpired(tx)
			if tt.expectedErr != nil {
				ms.Error(err)
				ms.Contains(err.Error(), tt.expectedErr.Error())
			} else {
				ms.NoError(err)
			}

			ms.Equal(tt.expectedCount, countInvites(tt.invite.PolicyID))
		})
	}
}
