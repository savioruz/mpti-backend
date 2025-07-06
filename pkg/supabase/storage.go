package supabase

import (
	"context"
	"errors"
	"fmt"
	"mime/multipart"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/google/uuid"
)

const (
	// Storage paths
	FieldsStoragePath = "fields"

	// Constants
	MinURLParts = 2
)

var (
	// Error messages
	ErrInvalidFileURL     = errors.New("invalid file URL")
	ErrFailedToUploadFile = errors.New("failed to upload file to Supabase")
	ErrFailedToDeleteFile = errors.New("failed to delete file from Supabase")
)

type Client struct {
	s3Client    *s3.S3
	bucketName  string
	endpointURL string
	region      string
}

type Config struct {
	AccessKeyID     string
	SecretAccessKey string
	EndpointURL     string
	Region          string
	BucketName      string
}

func NewClient(cfg Config) (*Client, error) {
	sess, err := session.NewSession(&aws.Config{
		Credentials: credentials.NewStaticCredentials(
			cfg.AccessKeyID,
			cfg.SecretAccessKey,
			"",
		),
		Endpoint:         aws.String(cfg.EndpointURL),
		Region:           aws.String(cfg.Region),
		S3ForcePathStyle: aws.Bool(true),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create AWS session: %w", err)
	}

	return &Client{
		s3Client:    s3.New(sess),
		bucketName:  cfg.BucketName,
		endpointURL: cfg.EndpointURL,
		region:      cfg.Region,
	}, nil
}

func (c *Client) UploadFile(ctx context.Context, file multipart.File, filename string) (string, error) {
	// Generate unique filename with fields path
	ext := filepath.Ext(filename)
	uniqueFilename := fmt.Sprintf("%s/%s%s", FieldsStoragePath, uuid.New().String(), ext)

	// Reset file pointer
	if _, err := file.Seek(0, 0); err != nil {
		return "", fmt.Errorf("failed to reset file pointer: %w", err)
	}

	// Determine content type based on file extension
	contentType := "application/octet-stream"

	switch strings.ToLower(ext) {
	case ".jpg", ".jpeg":
		contentType = "image/jpeg"
	case ".png":
		contentType = "image/png"
	case ".gif":
		contentType = "image/gif"
	case ".webp":
		contentType = "image/webp"
	}

	// Upload file to S3
	_, err := c.s3Client.PutObjectWithContext(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(c.bucketName),
		Key:         aws.String(uniqueFilename),
		Body:        file,
		ContentType: aws.String(contentType),
		ACL:         aws.String("public-read"),
	})
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrFailedToUploadFile, err)
	}

	// Return public URL
	publicURL := c.GetPublicURL(uniqueFilename)

	return publicURL, nil
}

func (c *Client) DeleteFile(ctx context.Context, fileURL string) error {
	// Extract key from URL
	key := c.extractKeyFromURL(fileURL)
	if key == "" {
		return fmt.Errorf("%w: %s", ErrInvalidFileURL, fileURL)
	}

	// Delete object from S3
	_, err := c.s3Client.DeleteObjectWithContext(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(c.bucketName),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("%w: %w", ErrFailedToDeleteFile, err)
	}

	return nil
}

func (c *Client) GetPublicURL(key string) string {
	baseURL := strings.Replace(c.endpointURL, "/storage/v1/s3", "", 1)

	return fmt.Sprintf("%s/storage/v1/object/public/%s/%s", baseURL, c.bucketName, key)
}

func (c *Client) extractKeyFromURL(fileURL string) string {
	parts := strings.Split(fileURL, "/")
	if len(parts) < MinURLParts {
		return ""
	}

	for i, part := range parts {
		if part == c.bucketName && i < len(parts)-1 {
			return strings.Join(parts[i+1:], "/")
		}
	}

	return ""
}
