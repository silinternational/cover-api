package messages

import (
	"testing"

	"github.com/gobuffalo/pop/v5"

	"github.com/silinternational/cover-api/api"
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

	return models.CreateItemFixtures(db, fixConfig)
}

func (ts *TestSuite) Test_ClaimReview1Send() {
	t := ts.T()
	db := ts.DB

	f := getClaimFixtures(db)
	steward := models.CreateAdminUser(db)

	review1Claim := f.Claims[0]
	models.UpdateClaimStatus(db, review1Claim, api.ClaimStatusReview1)

	testEmailer := notifications.DummyEmailService{}

	tests := []testData{
		{
			name:                "submitted to review1",
			wantToEmails:        []string{steward.EmailOfChoice()},
			wantSubjectsContain: []string{"just (re)submitted a claim for approval"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testEmailer.DeleteSentMessages()
			ClaimReview1Send(review1Claim, []interface{}{&testEmailer})
			validateEmails(ts, tt, testEmailer)
		})
	}
}

func (ts *TestSuite) Test_ClaimRevisionSend() {
	t := ts.T()
	db := ts.DB

	f := getClaimFixtures(db)
	member0 := f.Policies[0].Members[0]
	member1 := f.Policies[0].Members[1]

	revisionClaim := f.Claims[0]
	models.UpdateClaimStatus(db, revisionClaim, api.ClaimStatusRevision)

	testEmailer := notifications.DummyEmailService{}

	tests := []testData{
		{
			name:         "revisions required",
			wantToEmails: []string{member0.EmailOfChoice(), member1.EmailOfChoice()},
			wantSubjectsContain: []string{
				"changes have been requested on your claim",
				"changes have been requested on your claim",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testEmailer.DeleteSentMessages()
			ClaimRevisionSend(revisionClaim, []interface{}{&testEmailer})
			validateEmails(ts, tt, testEmailer)
		})
	}
}

func (ts *TestSuite) Test_ClaimPreapprovedSend() {
	t := ts.T()
	db := ts.DB

	f := getClaimFixtures(db)
	member0 := f.Policies[0].Members[0]
	member1 := f.Policies[0].Members[1]

	receiptClaim := f.Claims[0]
	models.UpdateClaimStatus(db, receiptClaim, api.ClaimStatusReceipt)

	testEmailer := notifications.DummyEmailService{}

	tests := []testData{
		{
			name:         "preapproved",
			wantToEmails: []string{member0.EmailOfChoice(), member1.EmailOfChoice()},
			wantSubjectsContain: []string{
				"receipts are needed on your new claim",
				"receipts are needed on your new claim",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testEmailer.DeleteSentMessages()
			ClaimPreapprovedSend(receiptClaim, []interface{}{&testEmailer})
			validateEmails(ts, tt, testEmailer)
		})
	}
}

func (ts *TestSuite) Test_ClaimReceiptSend() {
	t := ts.T()
	db := ts.DB

	f := getClaimFixtures(db)
	member0 := f.Policies[0].Members[0]
	member1 := f.Policies[0].Members[1]

	receiptClaim := f.Claims[0]
	models.UpdateClaimStatus(db, receiptClaim, api.ClaimStatusReceipt)

	testEmailer := notifications.DummyEmailService{}

	tests := []testData{
		{
			name:         "receipt required",
			wantToEmails: []string{member0.EmailOfChoice(), member1.EmailOfChoice()},
			wantSubjectsContain: []string{
				"new receipts are needed on your claim",
				"new receipts are needed on your claim",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testEmailer.DeleteSentMessages()
			ClaimReceiptSend(receiptClaim, []interface{}{&testEmailer})
			validateEmails(ts, tt, testEmailer)
		})
	}
}

func (ts *TestSuite) Test_ClaimReview2Send() {
	t := ts.T()
	db := ts.DB

	f := getClaimFixtures(db)
	steward := models.CreateAdminUser(db)

	review2Claim := f.Claims[0]
	models.UpdateClaimStatus(db, review2Claim, api.ClaimStatusReview2)

	testEmailer := notifications.DummyEmailService{}

	tests := []testData{
		{
			name:                "submitted to review2",
			wantToEmails:        []string{steward.EmailOfChoice()},
			wantSubjectsContain: []string{"just resubmitted a claim for approval"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testEmailer.DeleteSentMessages()
			ClaimReview2Send(review2Claim, []interface{}{&testEmailer})
			validateEmails(ts, tt, testEmailer)
		})
	}
}

func (ts *TestSuite) Test_claimReview3() {
	t := ts.T()
	db := ts.DB

	f := getClaimFixtures(db)
	steward := models.CreateAdminUser(db)

	review3Claim := f.Claims[0]
	models.UpdateClaimStatus(db, review3Claim, api.ClaimStatusReview3)

	testEmailer := notifications.DummyEmailService{}

	tests := []testData{
		{
			name:                "submitted to review3",
			wantToEmails:        []string{steward.EmailOfChoice()},
			wantSubjectsContain: []string{"has a claim waiting for your approval"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testEmailer.DeleteSentMessages()
			ClaimReview3Send(review3Claim, []interface{}{&testEmailer})
			validateEmails(ts, tt, testEmailer)
		})
	}
}
