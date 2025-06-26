package cachefilesystem

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"zbz/cache"
	"zbz/zlog"
)

// FileSystemProvider implements cache.CacheProvider using the local filesystem
type FileSystemProvider struct {
	baseDir     string
	permissions os.FileMode
	startTime   time.Time
}

// NewFilesystemCache creates a filesystem cache contract with type-safe native client access
// Returns a contract that can be registered as the global singleton or used independently
// Example:
//   contract := cachefilesystem.NewFilesystemCache(config)
//   contract.Register()  // Register as global singleton
//   fsCache := contract.Native()  // Get *FileSystemProvider without casting
func NewFilesystemCache(config cache.CacheConfig) (*cache.CacheContract[*FileSystemProvider], error) {
	baseDir := config.BaseDir
	if baseDir == "" {
		baseDir = "/tmp/zbz-cache"
	}
	permissions := os.FileMode(config.Permissions)
	if permissions == 0 {
		permissions = 0644
	}
	
	// Ensure directory exists
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create cache directory: %w", err)
	}
	
	provider := &FileSystemProvider{
		baseDir:     baseDir,
		permissions: permissions,
		startTime:   time.Now(),
	}
	
	zlog.Info("Filesystem cache provider initialized", 
		zlog.String("base_dir", baseDir))
	
	// Create and return contract
	return cache.NewContract("filesystem", provider, provider, config), nil
}

// Helper to convert cache key to safe filename
func (f *FileSystemProvider) keyToPath(key string) string {
	// Use MD5 hash to create safe filename from key
	hasher := md5.New()
	hasher.Write([]byte(key))
	hash := hex.EncodeToString(hasher.Sum(nil))
	
	// Create subdirectory structure to avoid too many files in one directory
	subDir := hash[:2]
	return filepath.Join(f.baseDir, subDir, hash+".cache")
}

// Helper to read cache file with expiration check
func (f *FileSystemProvider) readCacheFile(path string) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, cache.ErrCacheKeyNotFound
		}
		return nil, err
	}
	defer file.Close()
	
	// Read expiration timestamp (first 8 bytes)
	expirationBytes := make([]byte, 8)
	n, err := file.Read(expirationBytes)
	if err != nil || n != 8 {
		return nil, fmt.Errorf("invalid cache file format")
	}
	
	expirationStr := string(expirationBytes)
	if expirationStr != "00000000" { // "00000000" means no expiration
		expiration, err := strconv.ParseInt(strings.TrimSpace(expirationStr), 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid expiration format")
		}
		
		if time.Now().Unix() > expiration {
			// Expired - delete file and return not found
			os.Remove(path)
			return nil, cache.ErrCacheKeyNotFound
		}
	}
	
	// Read the actual data
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	
	// Return data without the expiration header
	if len(data) < 8 {
		return nil, fmt.Errorf("invalid cache file format")
	}
	
	return data[8:], nil
}

// Helper to write cache file with expiration
func (f *FileSystemProvider) writeCacheFile(path string, data []byte, ttl time.Duration) error {
	// Ensure subdirectory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	
	// Create temporary file
	tempPath := path + ".tmp"
	file, err := os.Create(tempPath)
	if err != nil {
		return err
	}
	defer file.Close()
	
	// Write expiration timestamp (8 bytes)
	var expirationStr string
	if ttl <= 0 {
		expirationStr = "00000000" // No expiration
	} else {
		expiration := time.Now().Add(ttl).Unix()
		expirationStr = fmt.Sprintf("%08d", expiration)
	}
	
	if _, err := file.WriteString(expirationStr); err != nil {
		os.Remove(tempPath)
		return err
	}
	
	// Write actual data
	if _, err := file.Write(data); err != nil {
		os.Remove(tempPath)
		return err
	}
	
	if err := file.Sync(); err != nil {
		os.Remove(tempPath)
		return err
	}
	
	file.Close()
	
	// Atomically move temp file to final location
	if err := os.Rename(tempPath, path); err != nil {
		os.Remove(tempPath)
		return err
	}
	
	// Set permissions
	return os.Chmod(path, f.permissions)
}

// Basic operations

func (f *FileSystemProvider) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	path := f.keyToPath(key)
	return f.writeCacheFile(path, value, ttl)
}

func (f *FileSystemProvider) Get(ctx context.Context, key string) ([]byte, error) {
	path := f.keyToPath(key)
	return f.readCacheFile(path)
}

func (f *FileSystemProvider) Delete(ctx context.Context, key string) error {
	path := f.keyToPath(key)
	err := os.Remove(path)
	if os.IsNotExist(err) {
		return nil // Already deleted
	}
	return err
}

func (f *FileSystemProvider) Exists(ctx context.Context, key string) (bool, error) {
	path := f.keyToPath(key)
	_, err := f.readCacheFile(path)
	if err != nil {
		if err == cache.ErrCacheKeyNotFound {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// Batch operations

func (f *FileSystemProvider) GetMulti(ctx context.Context, keys []string) (map[string][]byte, error) {
	result := make(map[string][]byte)
	
	for _, key := range keys {
		if data, err := f.Get(ctx, key); err == nil {
			result[key] = data
		}
	}
	
	return result, nil
}

func (f *FileSystemProvider) SetMulti(ctx context.Context, items map[string]cache.CacheItem, ttl time.Duration) error {
	for key, item := range items {
		effectiveTTL := ttl
		if item.TTL > 0 {
			effectiveTTL = item.TTL
		}
		
		if err := f.Set(ctx, key, item.Value, effectiveTTL); err != nil {
			return err
		}
	}
	return nil
}

func (f *FileSystemProvider) DeleteMulti(ctx context.Context, keys []string) error {
	for _, key := range keys {
		f.Delete(ctx, key) // Ignore individual errors
	}
	return nil
}

// Advanced operations

func (f *FileSystemProvider) Keys(ctx context.Context, pattern string) ([]string, error) {
	var keys []string
	
	// Walk through all cache files
	err := filepath.WalkDir(f.baseDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // Skip errors
		}
		
		if d.IsDir() || !strings.HasSuffix(path, ".cache") {
			return nil
		}
		
		// For filesystem provider, we can't easily reverse the hash back to original key
		// This is a limitation of this simple implementation
		// In practice, you'd store a mapping file or use a different naming scheme
		
		// For now, just return the filename without extension
		base := filepath.Base(path)
		key := strings.TrimSuffix(base, ".cache")
		
		// Simple pattern matching on the hash (not ideal, but works for demo)
		if matched, _ := filepath.Match(pattern, key); matched {
			keys = append(keys, key)
		}
		
		return nil
	})
	
	return keys, err
}

func (f *FileSystemProvider) TTL(ctx context.Context, key string) (time.Duration, error) {
	path := f.keyToPath(key)
	
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, cache.ErrCacheKeyNotFound
		}
		return 0, err
	}
	defer file.Close()
	
	// Read expiration timestamp
	expirationBytes := make([]byte, 8)
	n, err := file.Read(expirationBytes)
	if err != nil || n != 8 {
		return 0, fmt.Errorf("invalid cache file format")
	}
	
	expirationStr := string(expirationBytes)
	if expirationStr == "00000000" {
		return -1, nil // No expiration
	}
	
	expiration, err := strconv.ParseInt(strings.TrimSpace(expirationStr), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid expiration format")
	}
	
	now := time.Now().Unix()
	if now > expiration {
		return 0, cache.ErrCacheKeyNotFound // Expired
	}
	
	return time.Duration(expiration-now) * time.Second, nil
}

func (f *FileSystemProvider) Expire(ctx context.Context, key string, ttl time.Duration) error {
	// Read current data
	data, err := f.Get(ctx, key)
	if err != nil {
		return err
	}
	
	// Rewrite with new TTL
	return f.Set(ctx, key, data, ttl)
}

// Management operations

func (f *FileSystemProvider) Clear(ctx context.Context) error {
	return os.RemoveAll(f.baseDir)
}

func (f *FileSystemProvider) Stats(ctx context.Context) (cache.CacheStats, error) {
	var fileCount int64
	var totalSize int64
	
	err := filepath.WalkDir(f.baseDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		
		if !d.IsDir() && strings.HasSuffix(path, ".cache") {
			fileCount++
			if info, err := d.Info(); err == nil {
				totalSize += info.Size()
			}
		}
		
		return nil
	})
	
	return cache.CacheStats{
		Provider:   "filesystem",
		Keys:       fileCount,
		Memory:     totalSize,
		Uptime:     time.Since(f.startTime),
		LastAccess: time.Now(),
	}, err
}

func (f *FileSystemProvider) Ping(ctx context.Context) error {
	// Test write access
	testFile := filepath.Join(f.baseDir, ".ping")
	if err := os.WriteFile(testFile, []byte("ping"), 0644); err != nil {
		return err
	}
	return os.Remove(testFile)
}

func (f *FileSystemProvider) Close() error {
	// Nothing to close for filesystem provider
	return nil
}

// Provider metadata

func (f *FileSystemProvider) GetProvider() string {
	return "filesystem"
}

func (f *FileSystemProvider) GetVersion() string {
	return "1.0.0"
}

func (f *FileSystemProvider) NativeClient() interface{} {
	// Return a struct that gives access to filesystem operations
	return struct {
		BaseDir     string
		Permissions os.FileMode
	}{
		BaseDir:     f.baseDir,
		Permissions: f.permissions,
	}
}

// Helper functions for config parsing
func getConfigString(config map[string]interface{}, key string, defaultValue string) string {
	if val, ok := config[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return defaultValue
}

func getConfigInt(config map[string]interface{}, key string, defaultValue int) int {
	if val, ok := config[key]; ok {
		switch v := val.(type) {
		case int:
			return v
		case int64:
			return int(v)
		case float64:
			return int(v)
		}
	}
	return defaultValue
}

