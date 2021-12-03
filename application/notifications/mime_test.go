package notifications

import (
	"bytes"
	"os"

	"github.com/silinternational/cover-api/domain"
)

func (ts *TestSuite) TestRawEmail() {
	var buf bytes.Buffer
	domain.ErrLogger.SetOutput(&buf)

	defer domain.ErrLogger.SetOutput(os.Stderr)

	raw := rawEmail(
		"to@example.com",
		domain.Env.EmailFromAddress,
		"test subject",
		`<h4>body</h4><img src="cid:logo"><p>End of body</p>`)

	ts.Greater(len(raw), 1000)

	ts.Equal("", buf.String(), "Got an unexpected error log entry")
}
