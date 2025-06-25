package gcs

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"cloud.google.com/go/storage"
	"google.golang.org/api/iterator"
	
	"zbz/hodor"
)

// gcsStorage implements BucketService interface using Google Cloud Storage
type gcsStorage struct {
	client     *storage.Client
	bucketName string
	ctx        context.Context
}

// NewGCSProvider creates a new GCS-based storage provider
func NewGCSProvider() hodor.BucketService {
	projectID := os.Getenv("ZBZ_GCS_PROJECT_ID")
	if projectID == "" {
		projectID = os.Getenv("GOOGLE_CLOUD_PROJECT")
	}
	
	bucketName := os.Getenv("ZBZ_GCS_BUCKET")
	if bucketName == "" {
		bucketName = "zbz-storage"
	}

	ctx := context.Background()
	
	// Create GCS client (automatically uses service account or ADC)
	client, err := storage.NewClient(ctx)
	if err != nil {
		panic(fmt.Sprintf("Failed to create GCS client: %v", err))
	}

	gcss := &gcsStorage{
		client:     client,
		bucketName: bucketName,
		ctx:        ctx,
	}
	
	// Ensure bucket exists
	if err := gcss.ensureBucket(); err != nil {
		panic(fmt.Sprintf("Failed to ensure bucket exists: %v", err))
	}

	return gcss
}

// ensureBucket creates the bucket if it doesn't exist
func (g *gcsStorage) ensureBucket() error {
	bucket := g.client.Bucket(g.bucketName)
	
	// Check if bucket exists
	_, err := bucket.Attrs(g.ctx)
	if err != nil {
		if err == storage.ErrBucketNotExist {
			// Bucket doesn't exist, create it
			projectID := os.Getenv("ZBZ_GCS_PROJECT_ID")
			if projectID == "" {
				projectID = os.Getenv("GOOGLE_CLOUD_PROJECT")
			}
			if projectID == "" {
				return fmt.Errorf("project ID is required to create bucket")
			}
			
			err = bucket.Create(g.ctx, projectID, &storage.BucketAttrs{
				Location: "US", // Default location
			})
			if err != nil {
				return fmt.Errorf("failed to create bucket: %w", err)
			}
		} else {
			return fmt.Errorf("failed to check bucket existence: %w", err)
		}
	}
	
	return nil
}

// Get retrieves data by key
func (g *gcsStorage) Get(key string) ([]byte, error) {
	obj := g.client.Bucket(g.bucketName).Object(key)
	
	reader, err := obj.NewReader(g.ctx)
	if err != nil {
		if err == storage.ErrObjectNotExist {
			return nil, fmt.Errorf("key '%s' not found", key)
		}
		return nil, fmt.Errorf("failed to create object reader: %w", err)
	}
	defer reader.Close()
	
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read object data: %w", err)
	}
	
	return data, nil
}

// Set stores data with optional TTL (TTL implemented via metadata)
func (g *gcsStorage) Set(key string, data []byte, ttl time.Duration) error {
	obj := g.client.Bucket(g.bucketName).Object(key)
	
	// Prepare metadata for TTL tracking
	metadata := make(map[string]string)
	if ttl > 0 {
		expiresAt := time.Now().Add(ttl).UTC().Format(time.RFC3339)
		metadata["X-Expires-At"] = expiresAt
	}
	
	writer := obj.NewWriter(g.ctx)
	writer.ContentType = "application/octet-stream"
	writer.Metadata = metadata
	
	_, err := writer.Write(data)
	if err != nil {
		writer.Close()
		return fmt.Errorf("failed to write object data: %w", err)
	}
	
	if err := writer.Close(); err != nil {
		return fmt.Errorf("failed to close object writer: %w", err)
	}
	
	return nil
}

// Delete removes a key
func (g *gcsStorage) Delete(key string) error {
	obj := g.client.Bucket(g.bucketName).Object(key)
	
	err := obj.Delete(g.ctx)
	if err != nil && err != storage.ErrObjectNotExist {
		return fmt.Errorf("failed to delete object: %w", err)
	}
	
	return nil
}

// Exists checks if a key exists and is not expired
func (g *gcsStorage) Exists(key string) (bool, error) {
	obj := g.client.Bucket(g.bucketName).Object(key)
	
	attrs, err := obj.Attrs(g.ctx)
	if err != nil {
		if err == storage.ErrObjectNotExist {
			return false, nil
		}
		return false, fmt.Errorf("failed to get object attributes: %w", err)
	}
	
	// Check TTL if set
	if expiresAtStr, exists := attrs.Metadata["X-Expires-At"]; exists {
		expiresAt, err := time.Parse(time.RFC3339, expiresAtStr)
		if err == nil && time.Now().After(expiresAt) {
			// Object expired, delete it
			_ = g.Delete(key)
			return false, nil
		}
	}
	
	return true, nil
}

// List returns all keys with the given prefix
func (g *gcsStorage) List(prefix string) ([]string, error) {
	var keys []string
	now := time.Now()
	
	bucket := g.client.Bucket(g.bucketName)
	query := &storage.Query{Prefix: prefix}
	
	it := bucket.Objects(g.ctx, query)
	for {
		attrs, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to iterate objects: %w", err)
		}
		
		// Check TTL if set
		if expiresAtStr, exists := attrs.Metadata["X-Expires-At"]; exists {
			expiresAt, err := time.Parse(time.RFC3339, expiresAtStr)
			if err == nil && now.After(expiresAt) {
				// Object expired, skip it (and optionally delete)
				continue
			}
		}
		
		keys = append(keys, attrs.Name)
	}
	
	return keys, nil
}

// gcsDriver implements BucketDriver for GCS storage
type gcsDriver struct{}

func (d *gcsDriver) DriverName() string {
	return "gcs"
}

func (d *gcsDriver) Connect(config map[string]any) (hodor.BucketService, error) {
	// Override environment with config values if provided
	if projectID, ok := config["project_id"].(string); ok && projectID != "" {
		os.Setenv("ZBZ_GCS_PROJECT_ID", projectID)
		os.Setenv("GOOGLE_CLOUD_PROJECT", projectID)
	}
	if bucket, ok := config["bucket"].(string); ok && bucket != "" {
		os.Setenv("ZBZ_GCS_BUCKET", bucket)
	}
	if credentials, ok := config["credentials"].(string); ok && credentials != "" {
		os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", credentials)
	}
	
	return NewGCSProvider(), nil
}

// Auto-register this driver when imported
func init() {
	hodor.RegisterDriver("gcs", &gcsDriver{})
}