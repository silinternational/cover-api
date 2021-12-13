package notifications

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ses"

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
	to := addressWithName(msg.ToName, msg.ToEmail)
	from := addressWithName(msg.FromName, msg.FromEmail)

	return SendRaw(from, rawEmail(to, from, msg.Subject, msg.Body))
}

func addressWithName(name, address string) string {
	if name == "" {
		return address
	}
	return fmt.Sprintf("%s <%s>", name, address)
}

// SendRaw sends a message using SES, given a pre-built raw byte stream
func SendRaw(from string, data []byte) error {
	svc, err := createSESService(getSESConfigFromEnv())
	if err != nil {
		return fmt.Errorf("SendEmail failed creating SES service, %s", err)
	}

	input := &ses.SendRawEmailInput{
		RawMessage: &ses.RawMessage{Data: data},
		Source:     aws.String(from),
	}

	result, err := svc.SendRawEmail(input)
	if err != nil {
		return fmt.Errorf("SendEmail failed using SES, %s", err)
	}

	domain.Logger.Printf("Message sent using SES, message ID: %s", *result.MessageId)
	return nil
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
