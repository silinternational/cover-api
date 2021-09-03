package listeners

import (
	"testing"

	"github.com/gobuffalo/events"

	"github.com/silinternational/cover-api/api"
	"github.com/silinternational/cover-api/domain"
	"github.com/silinternational/cover-api/models"
	"github.com/silinternational/cover-api/notifications"
)

func (ts *TestSuite) Test_itemSubmitted() {
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

	submittedItem := f.Items[0]
	models.UpdateItemStatus(db, submittedItem, api.ItemCoverageStatusPending)

	approvedItem := f.Items[1]
	models.UpdateItemStatus(db, approvedItem, api.ItemCoverageStatusApproved)

	testEmailer := notifications.DummyEmailService{}

	tests := []struct {
		name                string
		event               events.Event
		wantToEmails        []string
		wantSubjectsContain []string
	}{
		{
			name: "just submitted, not approved",
			event: events.Event{
				Kind: domain.EventApiItemSubmitted,
				Payload: events.Payload{
					domain.EventPayloadID: submittedItem.ID,
					EventPayloadNotifier:  &testEmailer,
				},
			},
			wantToEmails:        []string{steward.EmailOfChoice()},
			wantSubjectsContain: []string{"just submitted a new policy item for approval"},
		},
		{
			name: "auto approved",
			event: events.Event{
				Kind: domain.EventApiItemSubmitted,
				Payload: events.Payload{
					domain.EventPayloadID:                  approvedItem.ID,
					string(api.ItemCoverageStatusApproved): true,
					EventPayloadNotifier:                   &testEmailer,
				},
			},
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

			itemSubmitted(tt.event)

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

func (ts *TestSuite) Test_itemRevision() {
	t := ts.T()
	db := ts.DB

	fixConfig := models.FixturesConfig{
		NumberOfPolicies: 1,
		UsersPerPolicy:   2,
		ItemsPerPolicy:   2,
	}

	f := models.CreateItemFixtures(db, fixConfig)

	member0 := f.Policies[0].Members[0]
	member1 := f.Policies[0].Members[1]

	revisionItem := f.Items[0]
	models.UpdateItemStatus(db, revisionItem, api.ItemCoverageStatusRevision)

	testEmailer := notifications.DummyEmailService{}

	tests := []struct {
		name                string
		event               events.Event
		wantToEmails        []string
		wantSubjectsContain []string
	}{
		{
			name: "revisions required",
			event: events.Event{
				Kind: domain.EventApiItemRevision,
				Payload: events.Payload{
					domain.EventPayloadID: revisionItem.ID,
					EventPayloadNotifier:  &testEmailer,
				},
			},
			wantToEmails: []string{member0.EmailOfChoice(), member1.EmailOfChoice()},
			wantSubjectsContain: []string{
				"changes have been requested on your new policy item",
				"changes have been requested on your new policy item",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testEmailer.DeleteSentMessages()

			itemRevision(tt.event)

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

func (ts *TestSuite) Test_itemDenied() {
	t := ts.T()
	db := ts.DB

	fixConfig := models.FixturesConfig{
		NumberOfPolicies: 1,
		UsersPerPolicy:   2,
		ItemsPerPolicy:   2,
	}

	f := models.CreateItemFixtures(db, fixConfig)

	member0 := f.Policies[0].Members[0]
	member1 := f.Policies[0].Members[1]

	revisionItem := f.Items[0]
	models.UpdateItemStatus(db, revisionItem, api.ItemCoverageStatusDenied)

	testEmailer := notifications.DummyEmailService{}

	tests := []struct {
		name                string
		event               events.Event
		wantToEmails        []string
		wantSubjectsContain []string
	}{
		{
			name: "coverage denied",
			event: events.Event{
				Kind: domain.EventApiItemDenied,
				Payload: events.Payload{
					domain.EventPayloadID: revisionItem.ID,
					EventPayloadNotifier:  &testEmailer,
				},
			},
			wantToEmails: []string{member0.EmailOfChoice(), member1.EmailOfChoice()},
			wantSubjectsContain: []string{
				"coverage on your new policy item has been denied",
				"coverage on your new policy item has been denied",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testEmailer.DeleteSentMessages()

			itemDenied(tt.event)

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