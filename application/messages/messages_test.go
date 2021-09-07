package messages

import (
	"testing"

	"github.com/gobuffalo/pop/v5"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/silinternational/cover-api/domain"
	"github.com/silinternational/cover-api/models"
	"github.com/silinternational/cover-api/notifications"
)

// TestSuite establishes a test suite for domain tests
type TestSuite struct {
	suite.Suite
	*require.Assertions
	DB *pop.Connection
}

func (ts *TestSuite) SetupTest() {
	ts.Assertions = require.New(ts.T())
	models.DestroyAll()
}

// Test_TestSuite runs the test suite
func Test_TestSuite(t *testing.T) {
	ts := &TestSuite{}
	c, err := pop.Connect(domain.Env.GoEnv)
	if err == nil {
		ts.DB = c
	}
	suite.Run(t, ts)
}

type testData struct {
	name                string
	wantToEmails        []string
	wantSubjectsContain []string
}

func validateEmails(ts *TestSuite, td testData, testEmailer notifications.DummyEmailService) {
	wantCount := len(td.wantToEmails)

	msgs := testEmailer.GetSentMessages()
	ts.Len(msgs, wantCount, "incorrect message count")

	gotTos := testEmailer.GetAllToAddresses()
	ts.Equal(td.wantToEmails, gotTos)

	for i, w := range td.wantSubjectsContain {
		ts.Contains(msgs[i].Subject, w, "incorrect email subject")
	}
}
