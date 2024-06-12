package models

import (
	"errors"
	"time"

	"github.com/gobuffalo/nulls"

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
		NumberOfPolicies: 1,
	}
	policyFixtures := CreatePolicyFixtures(tx, fixturesConfig)
	policies := policyFixtures.Policies

	createUniqueInvite := func(createdAt time.Time) PolicyUserInvite {
		randomStr := randStr(5)
		return PolicyUserInvite{
			ID:           domain.GetUUID(),
			PolicyID:     policies[0].ID,
			Email:        "test_user" + randomStr + "@example.org",
			InviteeName:  "Test User" + randomStr,
			InviterName:  "Tester" + randomStr,
			InviterEmail: "test" + randomStr + "@example.org",
			CreatedAt:    createdAt,
		}
	}

	countInvites := func() int {
		count, err := tx.Where("policy_id = ?", policies[0].ID).Count(&PolicyUserInvite{})
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
			invite:        createUniqueInvite(now),
			expectedErr:   nil,
			expectedCount: 1,
		},
		{
			name:   "Invite expired",
			invite: createUniqueInvite(cutoff.Add(-time.Hour)),
			expectedErr: api.NewAppError(
				errors.New("attempt to use expired invite, ID: "),
				api.ErrorInviteExpired,
				api.CategoryForbidden,
			),
			expectedCount: 0,
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

			ms.Equal(tt.expectedCount, countInvites())

			if tt.expectedCount == 1 {
				err = tx.Destroy(&tt.invite)
				ms.NoError(err)
			}
		})
	}
}
