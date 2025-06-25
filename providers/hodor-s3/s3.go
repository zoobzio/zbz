package s3

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	
	"zbz/hodor"
)

// s3Storage implements BucketService interface using AWS S3
type s3Storage struct {
	client     *s3.S3
	bucketName string
}

// NewS3Provider creates a new S3-based storage provider
func NewS3Provider() hodor.BucketService {
	region := os.Getenv("ZBZ_S3_REGION")
	if region == "" {
		region = "us-west-2"
	}
	
	bucketName := os.Getenv("ZBZ_S3_BUCKET")
	if bucketName == "" {
		bucketName = "zbz-storage"
	}

	// Create AWS session
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(region),
	})
	if err != nil {
		panic(fmt.Sprintf("Failed to create AWS session: %v", err))
	}

	s3Client := s3.New(sess)

	s3s := &s3Storage{
		client:     s3Client,
		bucketName: bucketName,
	}
	
	// Ensure bucket exists
	if err := s3s.ensureBucket(); err != nil {
		panic(fmt.Sprintf("Failed to ensure bucket exists: %v", err))
	}

	return s3s
}

// ensureBucket creates the bucket if it doesn't exist
func (s *s3Storage) ensureBucket() error {
	// Check if bucket exists
	_, err := s.client.HeadBucket(&s3.HeadBucketInput{
		Bucket: aws.String(s.bucketName),
	})
	
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case s3.ErrCodeNoSuchBucket, "NotFound":
				// Bucket doesn't exist, create it
				_, err = s.client.CreateBucket(&s3.CreateBucketInput{
					Bucket: aws.String(s.bucketName),
				})
				if err != nil {
					return fmt.Errorf("failed to create bucket: %w", err)
				}
			default:
				return fmt.Errorf("failed to check bucket existence: %w", err)
			}
		} else {
			return fmt.Errorf("failed to check bucket existence: %w", err)
		}
	}
	
	return nil
}

// Get retrieves data by key
func (s *s3Storage) Get(key string) ([]byte, error) {
	result, err := s.client.GetObject(&s3.GetObjectInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String(key),
	})
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			if aerr.Code() == s3.ErrCodeNoSuchKey {
				return nil, fmt.Errorf("key '%s' not found", key)
			}
		}
		return nil, fmt.Errorf("failed to get object: %w", err)
	}
	defer result.Body.Close()
	
	data, err := io.ReadAll(result.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read object data: %w", err)
	}
	
	return data, nil
}

// Set stores data with optional TTL (TTL implemented via metadata)
func (s *s3Storage) Set(key string, data []byte, ttl time.Duration) error {
	// Prepare metadata for TTL tracking
	metadata := make(map[string]*string)
	if ttl > 0 {
		expiresAt := time.Now().Add(ttl).UTC().Format(time.RFC3339)
		metadata["X-Expires-At"] = aws.String(expiresAt)
	}
	
	_, err := s.client.PutObject(&s3.PutObjectInput{
		Bucket:      aws.String(s.bucketName),
		Key:         aws.String(key),
		Body:        bytes.NewReader(data),
		ContentType: aws.String("application/octet-stream"),
		Metadata:    metadata,
	})
	if err != nil {
		return fmt.Errorf("failed to put object: %w", err)
	}
	
	return nil
}

// Delete removes a key
func (s *s3Storage) Delete(key string) error {
	_, err := s.client.DeleteObject(&s3.DeleteObjectInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("failed to delete object: %w", err)
	}
	
	return nil
}

// Exists checks if a key exists and is not expired
func (s *s3Storage) Exists(key string) (bool, error) {
	result, err := s.client.HeadObject(&s3.HeadObjectInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String(key),
	})
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			if aerr.Code() == s3.ErrCodeNoSuchKey || aerr.Code() == "NotFound" {
				return false, nil
			}
		}
		return false, fmt.Errorf("failed to check object existence: %w", err)
	}
	
	// Check TTL if set
	if expiresAtStr, exists := result.Metadata["X-Expires-At"]; exists && expiresAtStr != nil {
		expiresAt, err := time.Parse(time.RFC3339, *expiresAtStr)
		if err == nil && time.Now().After(expiresAt) {
			// Object expired, delete it
			_ = s.Delete(key)
			return false, nil
		}
	}
	
	return true, nil
}

// List returns all keys with the given prefix
func (s *s3Storage) List(prefix string) ([]string, error) {
	var keys []string
	now := time.Now()
	
	err := s.client.ListObjectsV2Pages(&s3.ListObjectsV2Input{
		Bucket: aws.String(s.bucketName),
		Prefix: aws.String(prefix),
	}, func(page *s3.ListObjectsV2Output, lastPage bool) bool {
		for _, obj := range page.Contents {
			key := aws.StringValue(obj.Key)
			
			// Check TTL by getting object metadata
			headResult, err := s.client.HeadObject(&s3.HeadObjectInput{
				Bucket: aws.String(s.bucketName),
				Key:    aws.String(key),
			})
			if err != nil {
				continue // Skip objects we can't check
			}
			
			// Check TTL if set
			if expiresAtStr, exists := headResult.Metadata["X-Expires-At"]; exists && expiresAtStr != nil {
				expiresAt, err := time.Parse(time.RFC3339, *expiresAtStr)
				if err == nil && now.After(expiresAt) {
					// Object expired, skip it (and optionally delete)
					continue
				}
			}
			
			keys = append(keys, key)
		}
		return true
	})
	
	if err != nil {
		return nil, fmt.Errorf("failed to list objects: %w", err)
	}
	
	return keys, nil
}

// s3Driver implements BucketDriver for S3 storage
type s3Driver struct{}

func (d *s3Driver) DriverName() string {
	return "s3"
}

func (d *s3Driver) Connect(config map[string]any) (hodor.BucketService, error) {
	// Override environment with config values if provided
	if region, ok := config["region"].(string); ok && region != "" {
		os.Setenv("ZBZ_S3_REGION", region)
	}
	if bucket, ok := config["bucket"].(string); ok && bucket != "" {
		os.Setenv("ZBZ_S3_BUCKET", bucket)
	}
	if accessKey, ok := config["access_key"].(string); ok && accessKey != "" {
		os.Setenv("AWS_ACCESS_KEY_ID", accessKey)
	}
	if secretKey, ok := config["secret_key"].(string); ok && secretKey != "" {
		os.Setenv("AWS_SECRET_ACCESS_KEY", secretKey)
	}
	
	return NewS3Provider(), nil
}

// Auto-register this driver when imported
func init() {
	hodor.RegisterDriver("s3", &s3Driver{})
}