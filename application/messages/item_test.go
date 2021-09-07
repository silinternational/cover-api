package messages

import (
	"testing"

	"github.com/silinternational/cover-api/api"
	"github.com/silinternational/cover-api/models"
	"github.com/silinternational/cover-api/notifications"
)

func (ts *TestSuite) Test_ItemSubmittedSend() {
	t := ts.T()
	db := ts.DB

	fixConfig := models.FixturesConfig{
		NumberOfPolicies:    1,
		UsersPerPolicy:      2,
		ClaimsPerPolicy:     1,
		DependentsPerPolicy: 0,
		ItemsPerPolicy:      2,
	}

	f := models.CreateItemFixtures(db, fixConfig)

	steward := models.CreateAdminUser(db)
	member0 := f.Policies[0].Members[0]
	member1 := f.Policies[0].Members[1]

	submittedItem := models.UpdateItemStatus(db, f.Items[0], api.ItemCoverageStatusPending)
	approvedItem := models.UpdateItemStatus(db, f.Items[1], api.ItemCoverageStatusApproved)

	testEmailer := notifications.DummyEmailService{}

	tests := []struct {
		name                string
		item                models.Item
		wantToEmails        []string
		wantSubjectsContain []string
	}{
		{
			name:                "just submitted, not approved",
			item:                submittedItem,
			wantToEmails:        []string{steward.EmailOfChoice()},
			wantSubjectsContain: []string{"just submitted a new policy item for approval"},
		},
		{
			name:         "auto approved",
			item:         approvedItem,
			wantToEmails: []string{member0.EmailOfChoice(), member1.EmailOfChoice(), steward.EmailOfChoice()},
			wantSubjectsContain: []string{
				"your new policy item has been approved",
				"your new policy item has been approved",
				"a new policy item that has been auto approved",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testEmailer.DeleteSentMessages()

			ItemSubmittedSend(tt.item, []interface{}{&testEmailer})

			wantCount := len(tt.wantToEmails)

			msgs := testEmailer.GetSentMessages()
			ts.Len(msgs, wantCount, "incorrect message count")

			gotTos := testEmailer.GetAllToAddresses()
			ts.Equal(tt.wantToEmails, gotTos)

			for i, w := range tt.wantSubjectsContain {
				ts.Contains(msgs[i].Subject, w, "incorrect email subject")
			}
		})
	}
}
