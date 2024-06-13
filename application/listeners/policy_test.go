package listeners

import (
	"testing"
	"time"

	"github.com/gobuffalo/events"
	"github.com/gofrs/uuid"

	"github.com/silinternational/cover-api/domain"
	"github.com/silinternational/cover-api/models"
	"github.com/silinternational/cover-api/notifications"
)

func (ts *TestSuite) TestPolicy_UserInviteSend() {
	t := ts.T()
	db := ts.DB

	models.CreateAdminUsers(db)

	fixConfig := models.FixturesConfig{
		NumberOfPolicies: 1,
		UsersPerPolicy:   1,
	}
	f := models.CreateItemFixtures(db, fixConfig)

	testEmailer := &notifications.TestEmailService

	tests := []struct {
		name                string
		policyID            uuid.UUID
		inviteEmail         string
		wantEmailsSent      int
		wantSubjectsContain []string
	}{
		{
			name:                "new user",
			policyID:            f.Policies[0].ID,
			inviteEmail:         "new_user_invite@testing123.com",
			wantEmailsSent:      1,
			wantSubjectsContain: []string{"invited you to manage a policy"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testEmailer.DeleteSentMessages()
			invite := models.PolicyUserInvite{
				PolicyID: tt.policyID,
				Email:    tt.inviteEmail,
			}
			ts.NoError(invite.Create(ts.DB))

			e := events.Event{
				Kind:    domain.EventApiPolicyUserInviteCreated,
				Message: "PolicyUserInvite created",
				Payload: events.Payload{"id": invite.ID},
			}

			policyUserInviteCreated(e)

			var nus models.NotificationUsers
			ts.NoError(db.All(&nus), "error fetching NotificationUsers from db")
			ts.Len(nus, 1, "incorrect number of NotificationUsers queued")
		})
	}
}

func (ts *TestSuite) TestPolicyUser_InviteExpired() {
	t := ts.T()
	db := ts.DB

	fixConfig := models.FixturesConfig{
		NumberOfPolicies: 2,
	}
	f := models.CreateItemFixtures(db, fixConfig)
	pID1 := f.Policies[0].ID
	pID2 := f.Policies[1].ID

	countInvites := func(id uuid.UUID) int {
		count, err := db.Where("policy_id = ?", id).Count(&models.PolicyUserInvite{})
		ts.NoError(err)
		return count
	}

	now := time.Now().UTC()
	cutoff := now.Add(time.Duration(-domain.Env.InviteLifetimeDays) * domain.DurationDay)

	tests := []struct {
		name          string
		invite        models.PolicyUserInvite
		expectedCount int
	}{
		{
			name:          "Invite not expired",
			invite:        models.CreateUniqueInvite(now, pID1),
			expectedCount: 1,
		},
		{
			name:          "Invite expired",
			invite:        models.CreateUniqueInvite(cutoff.Add(-time.Hour), pID2),
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := db.Create(&tt.invite)
			ts.NoError(err)

			isExpired := tt.invite.CreatedAt.Before(cutoff)
			if isExpired {
				policyUserInviteExpired(events.Event{
					Payload: events.Payload{"id": tt.invite.ID},
				})
			}
			ts.Equal(tt.expectedCount, countInvites(tt.invite.PolicyID))
		})
	}
}
