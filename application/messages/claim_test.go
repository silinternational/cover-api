package messages

import (
	"testing"

	"github.com/gobuffalo/pop/v6"
	"github.com/silinternational/cover-api/api"
	"github.com/silinternational/cover-api/domain"
	"github.com/silinternational/cover-api/models"
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

func (ts *TestSuite) Test_ClaimReview1QueueMessage() {
	t := ts.T()
	db := ts.DB

	f := getClaimFixtures(db)

	steward := models.CreateAdminUsers(db)[models.AppRoleSteward]

	review1Claim := models.UpdateClaimStatus(db, f.Claims[0], api.ClaimStatusReview1, "")

	tests := []testData{
		{
			name:                  "submitted to review1",
			wantToEmails:          []any{steward.EmailOfChoice()},
			wantSubjectContains:   "New claim on " + review1Claim.ClaimItems[0].Item.Name,
			wantInappTextContains: "A new claim is waiting for your approval",
			wantBodyContains: []string{
				domain.Env.UIURL,
				review1Claim.ReferenceNumber,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ClaimReview1QueueMessage(db, review1Claim)
			validateNotificationUsers(ts, db, tt)
		})
	}
}

func (ts *TestSuite) Test_ClaimRevisionQueueMessage() {
	t := ts.T()
	db := ts.DB

	f := getClaimFixtures(db)

	member0 := f.Policies[0].Members[0]
	member1 := f.Policies[0].Members[1]
	item := f.Policies[0].Items[0]

	models.CreateAdminUsers(db)

	revisionClaim := models.UpdateClaimStatus(db, f.Claims[0], api.ClaimStatusRevision, "too many typos")

	tests := []testData{
		{
			name:                  "revisions required",
			wantToEmails:          []any{member0.EmailOfChoice(), member1.EmailOfChoice()},
			wantSubjectContains:   "Please provide more information",
			wantInappTextContains: "Please provide more information on your new claim",
			wantBodyContains: []string{
				domain.Env.UIURL,
				item.Name,
				revisionClaim.StatusReason,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ClaimRevisionQueueMessage(db, revisionClaim)
			validateNotificationUsers(ts, db, tt)
		})
	}
}

func (ts *TestSuite) Test_ClaimPreapprovedQueueMessage() {
	t := ts.T()
	db := ts.DB

	f := getClaimFixtures(db)

	member0 := f.Policies[0].Members[0]
	member1 := f.Policies[0].Members[1]

	models.CreateAdminUsers(db)

	receiptClaim := models.UpdateClaimStatus(db, f.Claims[0], api.ClaimStatusReceipt, "")

	tests := []testData{
		{
			name:                  "preapproved",
			wantToEmails:          []any{member0.EmailOfChoice(), member1.EmailOfChoice()},
			wantSubjectContains:   "Claim Approved for Repair",
			wantInappTextContains: "receipts are needed on your new claim",
			wantBodyContains: []string{
				domain.Env.UIURL,
				receiptClaim.ReferenceNumber,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ClaimPreapprovedQueueMessage(db, receiptClaim)
			validateNotificationUsers(ts, db, tt)
		})
	}
}

func (ts *TestSuite) Test_ClaimReceiptQueueMessage() {
	t := ts.T()
	db := ts.DB

	f := getClaimFixtures(db)

	member0 := f.Policies[0].Members[0]
	member1 := f.Policies[0].Members[1]

	models.CreateAdminUsers(db)

	receiptClaim := models.UpdateClaimStatus(db, f.Claims[0], api.ClaimStatusReceipt, "")

	tests := []testData{
		{
			name:                  "receipts required again",
			wantToEmails:          []any{member0.EmailOfChoice(), member1.EmailOfChoice()},
			wantSubjectContains:   "Claim Needs Receipt",
			wantInappTextContains: "Please provide a receipt",
			wantBodyContains: []string{
				domain.Env.UIURL,
				"we need a receipt",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ClaimReceiptQueueMessage(db, receiptClaim)
			validateNotificationUsers(ts, db, tt)
		})
	}
}

func (ts *TestSuite) Test_ClaimReview2QueueMessage() {
	t := ts.T()
	db := ts.DB

	f := getClaimFixtures(db)

	steward := models.CreateAdminUsers(db)[models.AppRoleSteward]

	review2Claim := models.UpdateClaimStatus(db, f.Claims[0], api.ClaimStatusReview2, "")

	tests := []testData{
		{
			name:                  "submitted to review2",
			wantToEmails:          []any{steward.EmailOfChoice()},
			wantSubjectContains:   "Consider payout for claim on " + review2Claim.ClaimItems[0].Item.Name,
			wantInappTextContains: "A claim is waiting for your approval",
			wantBodyContains: []string{
				domain.Env.UIURL,
				review2Claim.ReferenceNumber,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ClaimReview2QueueMessage(db, review2Claim)
			validateNotificationUsers(ts, db, tt)
		})
	}
}

func (ts *TestSuite) Test_ClaimReview3QueueMessage() {
	t := ts.T()
	db := ts.DB

	f := getClaimFixtures(db)

	signator := models.CreateAdminUsers(db)[models.AppRoleSignator]

	review3Claim := models.UpdateClaimStatus(db, f.Claims[0], api.ClaimStatusReview3, "")

	tests := []testData{
		{
			name:                  "submitted to review3",
			wantToEmails:          []any{signator.EmailOfChoice()},
			wantSubjectContains:   "Final approval for claim on " + review3Claim.ClaimItems[0].Item.Name,
			wantInappTextContains: "A claim is waiting for your approval",
			wantBodyContains: []string{
				domain.Env.UIURL,
				review3Claim.ReferenceNumber,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ClaimReview3QueueMessage(db, review3Claim)
			validateNotificationUsers(ts, db, tt)
		})
	}
}

func (ts *TestSuite) Test_ClaimApprovedQueueMessage() {
	t := ts.T()
	db := ts.DB

	f := getClaimFixtures(db)

	member0 := f.Policies[0].Members[0]
	member1 := f.Policies[0].Members[1]

	models.CreateAdminUsers(db)

	approvedClaim := models.UpdateClaimStatus(db, f.Claims[0], api.ClaimStatusApproved, "")

	tests := []testData{
		{
			name:                  "coverage approved",
			wantToEmails:          []any{member0.EmailOfChoice(), member1.EmailOfChoice()},
			wantSubjectContains:   "Claim Payout Approved",
			wantInappTextContains: "your claim has been approved",
			wantBodyContains: []string{
				domain.Env.UIURL,
				approvedClaim.ReferenceNumber,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ClaimApprovedQueueMessage(db, approvedClaim)
			validateNotificationUsers(ts, db, tt)
		})
	}
}

func (ts *TestSuite) Test_ClaimDeniedQueueMessage() {
	t := ts.T()
	db := ts.DB

	f := getClaimFixtures(db)

	member0 := f.Policies[0].Members[0]
	member1 := f.Policies[0].Members[1]
	item := f.Policies[0].Items[0]

	models.CreateAdminUsers(db)

	deniedClaim := models.UpdateClaimStatus(db, f.Claims[0], api.ClaimStatusDenied, "Try again next year")

	tests := []testData{
		{
			name:                  "coverage denied",
			wantToEmails:          []any{member0.EmailOfChoice(), member1.EmailOfChoice()},
			wantSubjectContains:   "An Update on Your Coverage Request",
			wantInappTextContains: "your claim has been denied",
			wantBodyContains: []string{
				domain.Env.UIURL,
				item.Name,
				"has been denied.",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ClaimDeniedQueueMessage(db, deniedClaim)
			validateNotificationUsers(ts, db, tt)
		})
	}
}
