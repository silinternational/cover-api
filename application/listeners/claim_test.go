package listeners

import (
	"testing"

	"github.com/gobuffalo/events"
	"github.com/gobuffalo/pop/v5"

	"github.com/silinternational/cover-api/api"
	"github.com/silinternational/cover-api/domain"
	"github.com/silinternational/cover-api/models"
	"github.com/silinternational/cover-api/notifications"
)

func getClaimFixtures(db *pop.Connection) models.Fixtures {
	fixConfig := models.FixturesConfig{
		NumberOfPolicies:    1,
		UsersPerPolicy:      2,
		ClaimsPerPolicy:     2,
		ClaimItemsPerClaim:  1,
		DependentsPerPolicy: 0,
		ItemsPerPolicy:      2,
	}
	models.CreateAdminUsers(db)

	return models.CreateItemFixtures(db, fixConfig)
}

func (ts *TestSuite) Test_claimReview1() {
	t := ts.T()
	db := ts.DB

	f := getClaimFixtures(db)

	review1Claim := models.UpdateClaimStatus(db, f.Claims[0], api.ClaimStatusReview1)

	testEmailer := notifications.DummyEmailService{}

	tests := []struct {
		name  string
		event events.Event
	}{
		{
			name: "submitted to review1",
			event: events.Event{
				Kind:    domain.EventApiClaimReview1,
				Payload: newTestPayload(review1Claim.ID, &testEmailer),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testEmailer.DeleteSentMessages()
			claimReview1(tt.event)

			ts.Greater(testEmailer.GetNumberOfMessagesSent(), 0, "no email messages sent")
		})
	}
}

func (ts *TestSuite) Test_claimRevision() {
	t := ts.T()
	db := ts.DB

	f := getClaimFixtures(db)

	revisionClaim := models.UpdateClaimStatus(db, f.Claims[0], api.ClaimStatusRevision)

	testEmailer := notifications.DummyEmailService{}

	tests := []struct {
		name  string
		event events.Event
	}{
		{
			name: "revisions required",
			event: events.Event{
				Kind:    domain.EventApiClaimRevision,
				Payload: newTestPayload(revisionClaim.ID, &testEmailer),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testEmailer.DeleteSentMessages()
			claimRevision(tt.event)

			ts.Greater(testEmailer.GetNumberOfMessagesSent(), 0, "no email messages sent")
		})
	}
}

func (ts *TestSuite) Test_claimPreapproved() {
	t := ts.T()
	db := ts.DB

	f := getClaimFixtures(db)

	receiptClaim := models.UpdateClaimStatus(db, f.Claims[0], api.ClaimStatusReceipt)

	testEmailer := notifications.DummyEmailService{}

	tests := []struct {
		name  string
		event events.Event
	}{
		{
			name: "preapproved",
			event: events.Event{
				Kind:    domain.EventApiClaimPreapproved,
				Payload: newTestPayload(receiptClaim.ID, &testEmailer),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testEmailer.DeleteSentMessages()
			claimPreapproved(tt.event)

			ts.Greater(testEmailer.GetNumberOfMessagesSent(), 0, "no email messages sent")
		})
	}
}

func (ts *TestSuite) Test_claimReceipt() {
	t := ts.T()
	db := ts.DB

	f := getClaimFixtures(db)

	receiptClaim := models.UpdateClaimStatus(db, f.Claims[0], api.ClaimStatusReceipt)

	testEmailer := notifications.DummyEmailService{}

	tests := []struct {
		name  string
		event events.Event
	}{
		{
			name: "receipt required",
			event: events.Event{
				Kind:    domain.EventApiClaimReceipt,
				Payload: newTestPayload(receiptClaim.ID, &testEmailer),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testEmailer.DeleteSentMessages()
			claimReceipt(tt.event)

			ts.Greater(testEmailer.GetNumberOfMessagesSent(), 0, "no email messages sent")
		})
	}
}

func (ts *TestSuite) Test_claimReview2() {
	t := ts.T()
	db := ts.DB

	f := getClaimFixtures(db)

	review2Claim := models.UpdateClaimStatus(db, f.Claims[0], api.ClaimStatusReview2)

	testEmailer := notifications.DummyEmailService{}

	tests := []struct {
		name  string
		event events.Event
	}{
		{
			name: "submitted to review2",
			event: events.Event{
				Kind:    domain.EventApiClaimReview2,
				Payload: newTestPayload(review2Claim.ID, &testEmailer),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testEmailer.DeleteSentMessages()
			claimReview2(tt.event)

			ts.Greater(testEmailer.GetNumberOfMessagesSent(), 0, "no email messages sent")
		})
	}
}

func (ts *TestSuite) Test_claimReview3() {
	t := ts.T()
	db := ts.DB

	f := getClaimFixtures(db)

	review3Claim := models.UpdateClaimStatus(db, f.Claims[0], api.ClaimStatusReview3)

	testEmailer := notifications.DummyEmailService{}

	tests := []struct {
		name  string
		event events.Event
	}{
		{
			name: "submitted to review3",
			event: events.Event{
				Kind:    domain.EventApiClaimReview3,
				Payload: newTestPayload(review3Claim.ID, &testEmailer),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testEmailer.DeleteSentMessages()
			claimReview3(tt.event)

			ts.Greater(testEmailer.GetNumberOfMessagesSent(), 0, "no email messages sent")
		})
	}
}

func (ts *TestSuite) Test_claimApproved() {
	t := ts.T()
	db := ts.DB

	f := getClaimFixtures(db)

	approvedClaim := models.UpdateClaimStatus(db, f.Claims[0], api.ClaimStatusApproved)

	testEmailer := notifications.DummyEmailService{}

	tests := []struct {
		name  string
		event events.Event
	}{
		{
			name: "approved",
			event: events.Event{
				Kind:    domain.EventApiClaimApproved,
				Payload: newTestPayload(approvedClaim.ID, &testEmailer),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testEmailer.DeleteSentMessages()
			claimApproved(tt.event)

			ts.Greater(testEmailer.GetNumberOfMessagesSent(), 0, "no email messages sent")
		})
	}
}

func (ts *TestSuite) Test_claimDenied() {
	t := ts.T()
	db := ts.DB

	f := getClaimFixtures(db)

	deniedClaim := models.UpdateClaimStatus(db, f.Claims[0], api.ClaimStatusDenied)

	testEmailer := notifications.DummyEmailService{}

	tests := []struct {
		name  string
		event events.Event
	}{
		{
			name: "claim denied",
			event: events.Event{
				Kind:    domain.EventApiClaimDenied,
				Payload: newTestPayload(deniedClaim.ID, &testEmailer),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testEmailer.DeleteSentMessages()
			claimDenied(tt.event)

			ts.Greater(testEmailer.GetNumberOfMessagesSent(), 0, "no email messages sent")
		})
	}
}
