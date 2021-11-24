package messages

import (
	"testing"

	"github.com/silinternational/cover-api/domain"
	"github.com/silinternational/cover-api/models"
)

func (ts *TestSuite) Test_PolicyUserInviteQueueMessage() {
	t := ts.T()
	db := ts.DB

	f := models.CreatePolicyUserInviteFixtures(db, 2)

	policy := f.Policies[0]
	member := policy.Members[0]
	invite0 := f.PolicyUserInvites[0]

	tests := []testData{
		{
			name:                "ok",
			wantToEmails:        []interface{}{invite0.Email},
			wantSubjectContains: "Invitation to Cover",
			wantBodyContains: []string{
				domain.Env.UIURL,
				"Accept Invite",
				"You've been invited to join",
				member.Name(),
				invite0.GetAcceptURL(),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			PolicyUserInviteQueueMessage(db, invite0)
			validateNotificationUsers(ts, db, tt)
		})
	}
}
