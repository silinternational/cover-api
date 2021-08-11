package aws

import (
	"bytes"
	"os"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/silinternational/riskman-api/domain"
)

// TestSuite establishes a test suite
type TestSuite struct {
	suite.Suite
}

// Test_TestSuite runs the test suite
func Test_TestSuite(t *testing.T) {
	suite.Run(t, new(TestSuite))
}

// TestSendEmail can be used in a local environment for development. Add SES credentials to the appropriate
// environment variables, and change the "To" and "From" email addresses to valid addresses.
func (ts *TestSuite) TestSendEmail() {
	ts.T().Skip("only for use in local environment if configured with SES credentials")
	err := SendEmail(
		"me@example.com",
		domain.Env.EmailFromAddress,
		"AWS Email Test",
		`<h4>body</h4><p>This is a test to see if AWS can send an email.</p><p>End of body</p>`)
	ts.NoError(err)
}

func (ts *TestSuite) TestRawEmail() {
	var buf bytes.Buffer
	domain.ErrLogger.SetOutput(&buf)

	defer domain.ErrLogger.SetOutput(os.Stderr)

	raw := rawEmail(
		"to@example.com",
		domain.Env.EmailFromAddress,
		"AWS Raw Email Test",
		`<h4>body</h4>
		<p>
		This is a test to see if AWS can send a raw email.
		For some reason it needs to be a certain length to pass "make test"
		(as defined by "ts.Greater(len(raw), 1000)" at line 54 
		of riskman-api/application/aws/aws_test.go)
		</p>
		<p>End of body</p>`)

	ts.Greater(len(raw), 1000)

	ts.Equal("", buf.String(), "Got an unexpected error log entry")
}
