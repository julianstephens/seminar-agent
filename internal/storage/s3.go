// Package storage provides an S3-compatible object storage client used for
// uploading export files and generating presigned download URLs.
package storage

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"

	appConfig "github.com/julianstephens/formation/internal/config"
)

// PresignExpiry is the lifetime of a presigned download URL.
const PresignExpiry = 15 * time.Minute

// S3Client wraps the AWS S3 client and presign client with the configured
// bucket name so callers do not need to pass the bucket on every call.
type S3Client struct {
	client  *s3.Client
	presign *s3.PresignClient
	bucket  string
}

// NewS3Client constructs an S3Client from application configuration.
// It configures a custom endpoint URL to support S3-compatible services such
// as Garage or MinIO.
func NewS3Client(cfg *appConfig.Config) *S3Client {
	creds := credentials.NewStaticCredentialsProvider(
		cfg.S3AccessKeyID,
		cfg.S3SecretKey,
		"",
	)

	// Create S3 client directly without LoadDefaultConfig to avoid AWS defaults
	client := s3.New(s3.Options{
		Region:       cfg.S3Region,
		Credentials:  creds,
		BaseEndpoint: aws.String(cfg.S3EndpointURL),
		UsePathStyle: true, // Required for S3-compatible services like Garage/MinIO

		// Disable features that can interfere with Cloudflare tunnels
		EndpointOptions: s3.EndpointResolverOptions{
			DisableHTTPS: false,
		},
	})

	s3Client := &S3Client{
		client:  client,
		presign: s3.NewPresignClient(client),
		bucket:  cfg.S3BucketName,
	}

	// Skip bucket existence check for now - let upload operations fail if bucket doesn't exist
	// if err := s3Client.ensureBucketExists(ctx); err != nil {
	// 	panic(fmt.Sprintf("unable to ensure S3 bucket exists: %v", err))
	// }

	return s3Client
}

func (c *S3Client) ensureBucketExists(ctx context.Context) error {
	_, err := c.client.HeadBucket(ctx, &s3.HeadBucketInput{
		Bucket: aws.String(c.bucket),
	}, func(o *s3.Options) {})
	exists := true
	if err != nil {
		var apiError smithy.APIError
		if errors.As(err, &apiError) {
			switch apiError.(type) {
			case *types.NotFound:
				exists = false
				err = nil
			default:
				return fmt.Errorf("checking bucket existence: %w", err)
			}
		}
	}
	if err != nil {
		return fmt.Errorf("checking bucket existence: %w", err)
	}
	if exists {
		return nil
	}

	// Bucket does not exist, attempt to create it
	_, err = c.client.CreateBucket(ctx, &s3.CreateBucketInput{
		Bucket: aws.String(c.bucket),
	})
	if err != nil {
		return fmt.Errorf("creating bucket: %w", err)
	}

	return nil
}

// Upload puts content into the bucket under key with the given content type.
// Returns an error if the upload fails.
func (c *S3Client) Upload(ctx context.Context, key string, content []byte, contentType string) error {
	_, err := c.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(c.bucket),
		Key:         aws.String(key),
		Body:        bytes.NewReader(content),
		ContentType: aws.String(contentType),
	})
	if err != nil {
		return fmt.Errorf("s3 upload %q: %w", key, err)
	}
	return nil
}

// PresignURL returns a presigned GET URL for the given object key that is
// valid for PresignExpiry.
func (c *S3Client) PresignURL(ctx context.Context, key string) (string, error) {
	req, err := c.presign.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	}, s3.WithPresignExpires(PresignExpiry))
	if err != nil {
		return "", fmt.Errorf("s3 presign %q: %w", key, err)
	}
	return req.URL, nil
}
