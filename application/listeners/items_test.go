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

	models.CreateAdminUsers(db)

	submittedItem := f.Items[0]
	models.UpdateItemStatus(db, submittedItem, api.ItemCoverageStatusPending)

	approvedItem := f.Items[1]
	models.UpdateItemStatus(db, approvedItem, api.ItemCoverageStatusApproved)

	testEmailer := notifications.DummyEmailService{}

	tests := []struct {
		name      string
		event     events.Event
		wantCount int
	}{
		{
			name: "just submitted, not approved",
			event: events.Event{
				Kind:    domain.EventApiItemSubmitted,
				Payload: newTestPayload(submittedItem.ID, &testEmailer),
			},
			wantCount: 1,
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
			wantCount: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testEmailer.DeleteSentMessages()
			itemSubmitted(tt.event)

			var nus models.NotificationUsers
			ts.NoError(db.All(&nus), "error fetching NotificationUsers from db")
			ts.Equal(tt.wantCount, len(nus), "incorrect number of NotificationUsers queued")

			notfns := models.Notifications{}
			ts.NoError(db.All(&notfns), "error fetching all NotificationUsers for destroy")
			ts.NoError(db.Destroy(&notfns), "error destroying all NotificationUsers")
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

	revisionItem := f.Items[0]
	models.UpdateItemStatus(db, revisionItem, api.ItemCoverageStatusRevision)

	testEmailer := notifications.DummyEmailService{}

	tests := []struct {
		name  string
		event events.Event
	}{
		{
			name: "revisions required",
			event: events.Event{
				Kind:    domain.EventApiItemRevision,
				Payload: newTestPayload(revisionItem.ID, &testEmailer),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testEmailer.DeleteSentMessages()
			itemRevision(tt.event)

			var nus models.NotificationUsers
			ts.NoError(db.All(&nus), "error fetching NotificationUsers from db")
			ts.Equal(2, len(nus), "incorrect number of NotificationUsers queued")
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

	revisionItem := f.Items[0]
	models.UpdateItemStatus(db, revisionItem, api.ItemCoverageStatusDenied)

	testEmailer := notifications.DummyEmailService{}

	tests := []struct {
		name  string
		event events.Event
	}{
		{
			name: "coverage denied",
			event: events.Event{
				Kind:    domain.EventApiItemDenied,
				Payload: newTestPayload(revisionItem.ID, &testEmailer),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testEmailer.DeleteSentMessages()
			itemDenied(tt.event)

			var nus models.NotificationUsers
			ts.NoError(db.All(&nus), "error fetching NotificationUsers from db")
			ts.Equal(2, len(nus), "incorrect number of NotificationUsers queued")
		})
	}
}
