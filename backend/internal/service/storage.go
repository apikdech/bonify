package service

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/receipt-manager/backend/internal/config"
)

// StorageService provides S3-compatible object storage using RustFS
type StorageService struct {
	client   *s3.Client
	bucket   string
	endpoint string
}

// NewStorageService creates a new storage service with RustFS/S3 configuration
func NewStorageService(cfg *config.Config) (*StorageService, error) {
	// Create static credentials provider
	creds := credentials.NewStaticCredentialsProvider(
		cfg.RustFS.AccessKey,
		cfg.RustFS.SecretKey,
		"",
	)

	// Load AWS configuration with custom endpoint for RustFS
	awsCfg, err := awsconfig.LoadDefaultConfig(
		context.Background(),
		awsconfig.WithRegion(cfg.RustFS.Region),
		awsconfig.WithCredentialsProvider(creds),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Create S3 client with custom endpoint resolver for RustFS
	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(cfg.RustFS.Endpoint)
		o.UsePathStyle = true // Required for S3-compatible services like RustFS
	})

	service := &StorageService{
		client:   client,
		bucket:   cfg.RustFS.Bucket,
		endpoint: cfg.RustFS.Endpoint,
	}

	// Ensure bucket exists
	if err := service.ensureBucket(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to ensure bucket exists: %w", err)
	}

	return service, nil
}

// ensureBucket checks if the bucket exists and creates it if necessary
func (s *StorageService) ensureBucket(ctx context.Context) error {
	// Check if bucket exists
	_, err := s.client.HeadBucket(ctx, &s3.HeadBucketInput{
		Bucket: aws.String(s.bucket),
	})
	if err == nil {
		// Bucket exists
		return nil
	}

	// Bucket doesn't exist, try to create it
	_, err = s.client.CreateBucket(ctx, &s3.CreateBucketInput{
		Bucket: aws.String(s.bucket),
		CreateBucketConfiguration: &types.CreateBucketConfiguration{
			LocationConstraint: types.BucketLocationConstraint("us-east-1"),
		},
	})
	if err != nil {
		return fmt.Errorf("failed to create bucket: %w", err)
	}

	return nil
}

// Upload uploads an object to the storage service
// Returns the object key (filename) on success
func (s *StorageService) Upload(ctx context.Context, reader io.Reader, size int64, contentType, filename string) (string, error) {
	// Generate unique object key using timestamp and filename
	objectKey := fmt.Sprintf("%d_%s", time.Now().Unix(), filename)

	// Upload the object
	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:        aws.String(s.bucket),
		Key:           aws.String(objectKey),
		Body:          reader,
		ContentLength: aws.Int64(size),
		ContentType:   aws.String(contentType),
	})
	if err != nil {
		return "", fmt.Errorf("failed to upload object: %w", err)
	}

	return objectKey, nil
}

// GetPresignedURL generates a presigned URL for accessing an object
func (s *StorageService) GetPresignedURL(ctx context.Context, objectName string, expiry time.Duration) (string, error) {
	// Create presign client
	presignClient := s3.NewPresignClient(s.client)

	// Generate presigned URL
	req, err := presignClient.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(objectName),
	}, func(opts *s3.PresignOptions) {
		opts.Expires = expiry
	})
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned URL: %w", err)
	}

	return req.URL, nil
}

// Delete removes an object from the storage service
func (s *StorageService) Delete(ctx context.Context, objectName string) error {
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(objectName),
	})
	if err != nil {
		return fmt.Errorf("failed to delete object: %w", err)
	}

	return nil
}
