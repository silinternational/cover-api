package listeners

import (
	"testing"

	"github.com/gobuffalo/events"
	"github.com/gofrs/uuid"

	"github.com/silinternational/cover-api/domain"
	"github.com/silinternational/cover-api/models"
	"github.com/silinternational/cover-api/notifications"
)

func (ts *TestSuite) Test_PolicyUserInviteSend() {
	t := ts.T()
	db := ts.DB

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
			wantSubjectsContain: []string{"invited you to manage a policy on Cover"},
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
