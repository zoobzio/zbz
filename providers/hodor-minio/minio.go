package minio

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	
	"zbz/hodor"
)

// minioStorage implements HodorProvider interface using MinIO/S3
type minioStorage struct {
	client     *minio.Client
	bucketName string
}

// NewMinIOStorage creates a MinIO storage contract with type-safe native client access
// Returns a contract that can be registered as the global singleton or used independently
// Example:
//   contract := hodorminio.NewMinIOStorage(config)
//   contract.Register()  // Register as global singleton
//   minioClient := contract.Native()  // Get *minio.Client without casting
func NewMinIOStorage(config hodor.HodorConfig) (*hodor.HodorContract[*minio.Client], error) {
	// Apply defaults and use universal config
	endpoint := config.Endpoint
	if endpoint == "" {
		endpoint = "localhost:9000"
	}
	
	accessKey := config.AccessKey
	if accessKey == "" {
		accessKey = "minioadmin"
	}
	
	secretKey := config.SecretKey
	if secretKey == "" {
		secretKey = "minioadmin"
	}
	
	bucketName := config.Bucket
	if bucketName == "" {
		bucketName = "zbz-storage"
	}
	
	useSSL := config.EnableSSL

	// Initialize MinIO client
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create MinIO client: %w", err)
	}

	// Create provider wrapper
	provider := &minioStorage{
		client:     client,
		bucketName: bucketName,
	}
	
	// Ensure bucket exists
	if err := provider.ensureBucket(); err != nil {
		return nil, fmt.Errorf("failed to ensure bucket exists: %w", err)
	}

	// Create and return contract
	return hodor.NewContract("minio", provider, client, config), nil
}

// ensureBucket creates the bucket if it doesn't exist
func (m *minioStorage) ensureBucket() error {
	ctx := context.Background()
	
	exists, err := m.client.BucketExists(ctx, m.bucketName)
	if err != nil {
		return fmt.Errorf("failed to check bucket existence: %w", err)
	}
	
	if !exists {
		err = m.client.MakeBucket(ctx, m.bucketName, minio.MakeBucketOptions{})
		if err != nil {
			return fmt.Errorf("failed to create bucket: %w", err)
		}
	}
	
	return nil
}

// Get retrieves data by key
func (m *minioStorage) Get(key string) ([]byte, error) {
	ctx := context.Background()
	
	obj, err := m.client.GetObject(ctx, m.bucketName, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get object: %w", err)
	}
	defer obj.Close()
	
	data, err := io.ReadAll(obj)
	if err != nil {
		return nil, fmt.Errorf("failed to read object data: %w", err)
	}
	
	return data, nil
}

// Set stores data with optional TTL (TTL implemented via metadata for now)
func (m *minioStorage) Set(key string, data []byte, ttl time.Duration) error {
	ctx := context.Background()
	
	// Prepare metadata for TTL tracking
	metadata := make(map[string]string)
	if ttl > 0 {
		expiresAt := time.Now().Add(ttl).UTC().Format(time.RFC3339)
		metadata["X-Expires-At"] = expiresAt
	}
	
	reader := bytes.NewReader(data)
	
	_, err := m.client.PutObject(ctx, m.bucketName, key, reader, int64(len(data)), minio.PutObjectOptions{
		UserMetadata: metadata,
		ContentType:  "application/octet-stream",
	})
	if err != nil {
		return fmt.Errorf("failed to put object: %w", err)
	}
	
	return nil
}

// Delete removes a key
func (m *minioStorage) Delete(key string) error {
	ctx := context.Background()
	
	err := m.client.RemoveObject(ctx, m.bucketName, key, minio.RemoveObjectOptions{})
	if err != nil {
		return fmt.Errorf("failed to remove object: %w", err)
	}
	
	return nil
}

// Exists checks if a key exists and is not expired
func (m *minioStorage) Exists(key string) (bool, error) {
	ctx := context.Background()
	
	objInfo, err := m.client.StatObject(ctx, m.bucketName, key, minio.StatObjectOptions{})
	if err != nil {
		// Check if error is "object not found"
		if minio.ToErrorResponse(err).Code == "NoSuchKey" {
			return false, nil
		}
		return false, fmt.Errorf("failed to stat object: %w", err)
	}
	
	// Check TTL if set
	if expiresAtStr, exists := objInfo.UserMetadata["X-Expires-At"]; exists {
		expiresAt, err := time.Parse(time.RFC3339, expiresAtStr)
		if err == nil && time.Now().After(expiresAt) {
			// Object expired, delete it
			_ = m.Delete(key)
			return false, nil
		}
	}
	
	return true, nil
}

// List returns all keys with the given prefix
func (m *minioStorage) List(prefix string) ([]string, error) {
	ctx := context.Background()
	
	var keys []string
	now := time.Now()
	
	objectCh := m.client.ListObjects(ctx, m.bucketName, minio.ListObjectsOptions{
		Prefix:    prefix,
		Recursive: true,
	})
	
	for object := range objectCh {
		if object.Err != nil {
			return nil, fmt.Errorf("failed to list objects: %w", object.Err)
		}
		
		// Check TTL if we need object metadata
		objInfo, err := m.client.StatObject(ctx, m.bucketName, object.Key, minio.StatObjectOptions{})
		if err != nil {
			continue // Skip objects we can't stat
		}
		
		// Check TTL if set
		if expiresAtStr, exists := objInfo.UserMetadata["X-Expires-At"]; exists {
			expiresAt, err := time.Parse(time.RFC3339, expiresAtStr)
			if err == nil && now.After(expiresAt) {
				// Object expired, skip it (and optionally delete)
				continue
			}
		}
		
		keys = append(keys, object.Key)
	}
	
	return keys, nil
}

// minioDriver implements BucketDriver for MinIO storage
type minioDriver struct{}

func (d *minioDriver) DriverName() string {
	return "minio"
}

func (d *minioDriver) Connect(config map[string]any) (hodor.BucketService, error) {
	// Override environment with config values if provided
	if endpoint, ok := config["endpoint"].(string); ok && endpoint != "" {
		os.Setenv("ZBZ_MINIO_ENDPOINT", endpoint)
	}
	if accessKey, ok := config["access_key"].(string); ok && accessKey != "" {
		os.Setenv("ZBZ_MINIO_ACCESS_KEY", accessKey)
	}
	if secretKey, ok := config["secret_key"].(string); ok && secretKey != "" {
		os.Setenv("ZBZ_MINIO_SECRET_KEY", secretKey)
	}
	if bucket, ok := config["bucket"].(string); ok && bucket != "" {
		os.Setenv("ZBZ_MINIO_BUCKET", bucket)
	}
	if ssl, ok := config["ssl"].(bool); ok {
		if ssl {
			os.Setenv("ZBZ_MINIO_SSL", "true")
		} else {
			os.Setenv("ZBZ_MINIO_SSL", "false")
		}
	}
	
	return NewMinioProvider(), nil
}

// Auto-register this driver when imported
func init() {
	hodor.RegisterDriver("minio", &minioDriver{})
}