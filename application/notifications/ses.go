package notifications

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"mime/multipart"
	"net/textproto"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ses"
	"jaytaylor.com/html2text"

	"github.com/silinternational/cover-api/domain"
)

// SES sends email using Amazon Simple Email Service (SES)
type SES struct{}

type awsConfig struct {
	awsAccessKeyID     string
	awsSecretAccessKey string
	awsRegion          string
}

// Send a message
func (s *SES) Send(msg Message) error {
	body := msg.Body
	if body == "" {
		msg.Data["uiURL"] = domain.Env.UIURL
		msg.Data["appName"] = domain.Env.AppName

		bodyBuf := &bytes.Buffer{}
		if err := EmailRenderer.HTML(msg.Template).Render(bodyBuf, msg.Data); err != nil {
			return errors.New("error rendering ses message body - " + err.Error())
		}
		body = bodyBuf.String()
	}

	to := addressWithName(msg.ToName, msg.ToEmail)
	from := addressWithName(msg.FromName, msg.FromEmail)
	subject := msg.Subject

	return SendEmail(to, from, subject, body)
}

func addressWithName(name, address string) string {
	if name == "" {
		return address
	}
	return fmt.Sprintf("%s <%s>", name, address)
}

// SendEmail sends a message using SES
func SendEmail(to, from, subject, body string) error {
	svc, err := createSESService(getSESConfigFromEnv())
	if err != nil {
		return fmt.Errorf("SendEmail failed creating SES service, %s", err)
	}

	input := &ses.SendRawEmailInput{
		RawMessage: &ses.RawMessage{Data: rawEmail(to, from, subject, body)},
		Source:     aws.String(from),
	}

	result, err := svc.SendRawEmail(input)
	if err != nil {
		return fmt.Errorf("SendEmail failed using SES, %s", err)
	}

	domain.Logger.Printf("Message sent using SES, message ID: %s", *result.MessageId)
	return nil
}

// rawEmail generates a multi-part MIME email message with a plain text, html text, and inline logo attachment as
// follows:
//
// * multipart/alternative
//   * text/plain
//   * multipart/related
//     * text/html
//     * image/png
//
// Abbreviated example of the generated email message:
//  From: from@example.com
//	To: to@example.com
//	Subject: subject text
//	Content-Type: multipart/alternative; boundary="boundary_alternative"
//
//	--boundary_alternative
//	Content-Type: text/plain; charset=utf-8
//
//	Plain text body
//	--boundary_alternative
//	Content-type: multipart/related; boundary="boundary_related"
//
//	--boundary_related
//	Content-Type: text/html; charset=utf-8
//
//	HTML body
//	--boundary_related
//	Content-Type: image/png
//	Content-Transfer-Encoding: base64
//	Content-ID: <logo>
//	--boundary_related--
//	--boundary_alternative--
func rawEmail(to, from, subject, body string) []byte {
	tbody, err := html2text.FromString(body)
	if err != nil {
		domain.Logger.Printf("error converting html email to plain text ... %s", err.Error())
		tbody = body
	}

	b := &bytes.Buffer{}

	b.WriteString("From: " + from + "\n")
	b.WriteString("To: " + to + "\n")
	b.WriteString("Subject: " + subject + "\n")
	b.WriteString("MIME-Version: 1.0\n")

	alternativeWriter := multipart.NewWriter(b)
	b.WriteString(`Content-Type: multipart/alternative; type="text/plain"; boundary="` +
		alternativeWriter.Boundary() + `"` + "\n\n")

	w, err := alternativeWriter.CreatePart(textproto.MIMEHeader{
		"Content-Type":        {"text/plain; charset=utf-8"},
		"Content-Disposition": {"inline"},
	})
	if err != nil {
		domain.ErrLogger.Printf("failed to create MIME text part, %s", err)
	} else {
		_, _ = fmt.Fprint(w, tbody)
	}

	relatedWriter := multipart.NewWriter(b)
	_, err = alternativeWriter.CreatePart(textproto.MIMEHeader{
		"Content-Type": {`multipart/related; type="text/html"; boundary="` + relatedWriter.Boundary() + `"`},
	})
	if err != nil {
		domain.ErrLogger.Printf("failed to create MIME related part, %s", err)
	}

	w, err = relatedWriter.CreatePart(textproto.MIMEHeader{
		"Content-Type":        {"text/html; charset=utf-8"},
		"Content-Disposition": {"inline"},
	})
	if err != nil {
		domain.ErrLogger.Printf("failed to create MIME html part, %s", err)
	} else {
		_, _ = fmt.Fprint(w, body)
	}

	w, err = relatedWriter.CreatePart(textproto.MIMEHeader{
		"Content-Type":              {"image/png"},
		"Content-Disposition":       {"inline"},
		"Content-ID":                {"<logo>"},
		"Content-Transfer-Encoding": {"base64"},
	})
	if err != nil {
		domain.ErrLogger.Printf("failed to create MIME image part, %s", err)
	} else {
		logo, err := domain.Assets.Find("logo-wide.png")
		if err != nil {
			domain.ErrLogger.Printf("failed to read logo file, %s", err)
		}
		encoder := base64.NewEncoder(base64.StdEncoding, b)
		_, err = encoder.Write(logo)
		if err != nil {
			domain.ErrLogger.Printf("failed to write logo to email, %s", err)
		}
		err = encoder.Close()
		if err != nil {
			domain.ErrLogger.Printf("failed to close logo base64 encoder, %s", err)
		}
	}

	if err = relatedWriter.Close(); err != nil {
		domain.ErrLogger.Printf("failed to close MIME related part, %s", err)
	}

	if err = alternativeWriter.Close(); err != nil {
		domain.ErrLogger.Printf("failed to close MIME alternative part, %s", err)
	}

	return b.Bytes()
}

func getSESConfigFromEnv() awsConfig {
	return awsConfig{
		awsAccessKeyID:     domain.Env.AwsAccessKeyID,
		awsSecretAccessKey: domain.Env.AwsSecretAccessKey,
		awsRegion:          domain.Env.AwsRegion,
	}
}

func createSESService(config awsConfig) (*ses.SES, error) {
	sess, err := session.NewSession(&aws.Config{
		Credentials: credentials.NewStaticCredentials(config.awsAccessKeyID, config.awsSecretAccessKey, ""),
		Region:      aws.String(config.awsRegion),
	})
	if err != nil {
		return nil, err
	}
	return ses.New(sess), nil
}
