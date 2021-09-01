package listeners

import (
	"fmt"
	"testing"

	"github.com/gobuffalo/events"
	"github.com/stretchr/testify/suite"

	"github.com/silinternational/cover-api/domain"
	"github.com/silinternational/cover-api/models"
)

// TestSuite establishes a test suite for domain tests
type TestSuite struct {
	suite.Suite
}

// Test_TestSuite runs the test suite
func Test_TestSuite(t *testing.T) {
	suite.Run(t, new(TestSuite))
}

func (ts *TestSuite) Test_findObject() {
	t := ts.T()
	db := models.DB

	fixConfig := models.FixturesConfig{
		NumberOfPolicies:    1,
		UsersPerPolicy:      1,
		ClaimsPerPolicy:     1,
		ClaimItemsPerClaim:  2,
		DependentsPerPolicy: 0,
		ItemsPerPolicy:      2,
	}

	f := models.CreateItemFixtures(db, fixConfig)
	user := f.Users[0]
	item := f.Items[1]
	claim := f.Claims[0]

	tests := []struct {
		name            string
		payload         events.Payload
		object          interface{}
		listenerName    string
		wantErrContains string
		wantContains    []string
	}{
		{
			name:    "find user",
			payload: events.Payload{domain.EventPayloadID: user.ID},
			object:  &models.User{},
			wantContains: []string{
				"ID:" + user.ID.String(),
				"FirstName:" + user.FirstName,
			},
		},
		{
			name:    "find item",
			payload: events.Payload{domain.EventPayloadID: item.ID},
			object:  &models.Item{},
			wantContains: []string{
				"ID:" + item.ID.String(),
				"Name:" + item.Name,
			},
		},
		{
			name:    "find claim",
			payload: events.Payload{domain.EventPayloadID: claim.ID},
			object:  &models.Claim{},
			wantContains: []string{
				"ID:" + claim.ID.String(),
				"EventDescription:" + claim.EventDescription,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := findObject(tt.payload, tt.object, tt.name)
			if tt.wantErrContains != "" {
				ts.Error(err)
				ts.Contains(err, tt.wantErrContains, "incorrect error")
				return
			}

			got := fmt.Sprintf("%+v", tt.object)
			for _, c := range tt.wantContains {
				ts.Contains(got, c, "missing data from test object")
			}
		})
	}
}
