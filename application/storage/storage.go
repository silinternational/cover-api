package storage

import (
	"bytes"
	// "encoding/base64" (Uncomment to be able to display a logo at the end of an email,
	// see commented out code at the end of function "rawEmail")
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"

	"github.com/silinternational/cover-api/domain"
)

type ObjectUrl struct {
	Url        string
	Expiration time.Time
}

type awsConfig struct {
	awsAccessKeyID     string
	awsSecretAccessKey string
	awsEndpoint        string
	awsRegion          string
	awsS3Bucket        string
	awsDisableSSL      bool
	getPresignedUrl    bool
}

func getS3ConfigFromEnv() awsConfig {
	var a awsConfig
	a.awsAccessKeyID = domain.Env.AwsAccessKeyID
	a.awsSecretAccessKey = domain.Env.AwsSecretAccessKey
	a.awsEndpoint = domain.Env.AwsS3Endpoint
	a.awsRegion = domain.Env.AwsRegion
	a.awsS3Bucket = domain.Env.AwsS3Bucket
	a.awsDisableSSL = domain.Env.AwsS3DisableSSL

	if domain.Env.GoEnv == "development" || domain.Env.GoEnv == "test" {
		a.awsAccessKeyID = "abc123"
		a.awsSecretAccessKey = "abcd1234"
	}

	// a non-empty endpoint means minIO is in use, which doesn't support the S3 object URL scheme
	if !strings.HasPrefix(domain.Env.AwsS3ACL, "public") || len(a.awsEndpoint) > 0 {
		a.getPresignedUrl = true
	}
	return a
}

func createS3Service(config awsConfig) (*s3.S3, error) {
	sess, err := session.NewSession(&aws.Config{
		Credentials:      credentials.NewStaticCredentials(config.awsAccessKeyID, config.awsSecretAccessKey, ""),
		Endpoint:         aws.String(config.awsEndpoint),
		Region:           aws.String(config.awsRegion),
		DisableSSL:       aws.Bool(config.awsDisableSSL),
		S3ForcePathStyle: aws.Bool(len(config.awsEndpoint) > 0),
	})
	svc := s3.New(sess)

	return svc, err
}

func getObjectURL(config awsConfig, svc *s3.S3, key string) (ObjectUrl, error) {
	var objectUrl ObjectUrl

	if !config.getPresignedUrl {
		objectUrl.Url = fmt.Sprintf("https://%s.s3.amazonaws.com/%s", config.awsS3Bucket, url.PathEscape(key))
		objectUrl.Expiration = time.Date(9999, time.December, 31, 0, 0, 0, 0, time.UTC)
		return objectUrl, nil
	}

	req, _ := svc.GetObjectRequest(&s3.GetObjectInput{
		Bucket: aws.String(config.awsS3Bucket),
		Key:    aws.String(key),
	})

	urlLifespan := time.Duration(domain.Env.AwsS3URLLifeMinutes) * time.Minute
	if newUrl, err := req.Presign(urlLifespan); err == nil {
		objectUrl.Url = newUrl
		// return a time slightly before the actual url expiration to account for delays
		objectUrl.Expiration = time.Now().Add(urlLifespan - time.Minute)
	} else {
		return objectUrl, err
	}

	return objectUrl, nil
}

// StoreFile saves content in an AWS S3 bucket or compatible storage, depending on environment configuration.
func StoreFile(key, contentType string, content []byte) (ObjectUrl, error) {
	config := getS3ConfigFromEnv()

	svc, err := createS3Service(config)
	if err != nil {
		return ObjectUrl{}, err
	}

	acl := ""
	if !config.getPresignedUrl {
		acl = domain.Env.AwsS3ACL
	}
	if _, err := svc.PutObject(&s3.PutObjectInput{
		Bucket:      aws.String(config.awsS3Bucket),
		Key:         aws.String(key),
		ContentType: aws.String(contentType),
		ACL:         aws.String(acl),
		Body:        bytes.NewReader(content),
	}); err != nil {
		return ObjectUrl{}, err
	}

	objectUrl, err := getObjectURL(config, svc, key)
	if err != nil {
		return ObjectUrl{}, err
	}

	return objectUrl, nil
}

// GetFileURL retrieves a URL from which a stored object can be loaded. The URL should not require external
// credentials to access. It may reference a file with public_read access or it may be a pre-signed URL.
func GetFileURL(key string) (ObjectUrl, error) {
	config := getS3ConfigFromEnv()

	svc, err := createS3Service(config)
	if err != nil {
		return ObjectUrl{}, err
	}

	return getObjectURL(config, svc, key)
}

// RemoveFile removes a file from the configured AWS S3 bucket.
func RemoveFile(key string) error {
	config := getS3ConfigFromEnv()

	svc, err := createS3Service(config)
	if err != nil {
		return err
	}

	if _, err := svc.DeleteObject(&s3.DeleteObjectInput{
		Bucket: aws.String(config.awsS3Bucket),
		Key:    aws.String(key),
	}); err != nil {
		return err
	}

	return nil
}

// CreateS3Bucket creates an S3 bucket with a name defined by an environment variable. If the bucket already
// exists, it will not return an error.
func CreateS3Bucket() error {
	env := domain.Env.GoEnv
	if env != "test" && env != "development" {
		return errors.New("CreateS3Bucket should only be used in test and development")
	}

	config := getS3ConfigFromEnv()

	svc, err := createS3Service(config)
	if err != nil {
		return err
	}

	c := &s3.CreateBucketInput{Bucket: &domain.Env.AwsS3Bucket}
	if _, err := svc.CreateBucket(c); err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case s3.ErrCodeBucketAlreadyExists:
			case s3.ErrCodeBucketAlreadyOwnedByYou:
			default:
				return err
			}
		}
	}
	return nil
}
