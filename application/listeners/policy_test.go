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
		NumberOfPolicies: 2,
		UsersPerPolicy:   1,
	}

	f := models.CreateItemFixtures(db, fixConfig)

	// member0 := f.Policies[0].Members[0]
	// member1 := f.Policies[1].Members[0]

	testEmailer := &notifications.TestEmailService

	tests := []struct {
		name                string
		policyID            uuid.UUID
		inviteEmail         string
		wantEmailsSent      int
		wantSubjectsContain []string
	}{
		//{
		//	name:           "existing policy member",
		//	policyID:       f.Policies[0].ID,
		//	inviteEmail:    member0.Email,
		//	wantEmailsSent: 0,
		//},
		//{
		//	name:           "existing user, new to policy",
		//	policyID:       f.Policies[0].ID,
		//	inviteEmail:    member1.Email,
		//	wantEmailsSent: 0,
		//},
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

			msgs := testEmailer.GetSentMessages()
			ts.Len(msgs, tt.wantEmailsSent, "incorrect message count")

			for i, w := range tt.wantSubjectsContain {
				ts.Contains(msgs[i].Subject, w, "incorrect email subject")
			}
		})
	}
}
