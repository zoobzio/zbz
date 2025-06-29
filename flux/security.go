package flux

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// Security constants
const (
	MaxFileSize int64 = 10 * 1024 * 1024 // 10MB maximum file size
)

// Allowed file extensions for security
var AllowedExtensions = map[string]bool{
	".yaml": true,
	".yml":  true,
	".json": true,
	".txt":  true,
	".md":   true,
	".toml": true,
	".ini":  true,
	".env":  true,
	".conf": true,
	".cfg":  true,
}

// Allowed MIME types for security
var AllowedMimeTypes = map[string]bool{
	"text/plain":            true,
	"text/yaml":             true,
	"application/x-yaml":    true,
	"application/json":      true,
	"text/markdown":         true,
	"application/toml":      true,
	"text/x-ini":            true,
	"application/octet-stream": true,  // For binary config files
}

// validateFile performs comprehensive security validation with default settings
func validateFile(filePath string) error {
	return validateFileWithOptions(filePath, nil)
}

// validateFileWithOptions performs comprehensive security validation with custom options
func validateFileWithOptions(filePath string, maxFileSize *int64) error {
	// Check extension
	ext := strings.ToLower(filepath.Ext(filePath))
	if !AllowedExtensions[ext] {
		return fmt.Errorf("file type not allowed: %s", ext)
	}
	
	// Check file exists and get size
	info, err := os.Stat(filePath)
	if err != nil {
		return fmt.Errorf("file does not exist: %w", err)
	}
	
	// Determine max file size (custom or default)
	maxSize := MaxFileSize
	if maxFileSize != nil {
		maxSize = *maxFileSize
	}
	
	if info.Size() > maxSize {
		return fmt.Errorf("file too large: %d bytes (max %d)", info.Size(), maxSize)
	}
	
	// MIME type check (read first 512 bytes)
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file for MIME check: %w", err)
	}
	defer file.Close()
	
	buffer := make([]byte, 512)
	n, err := file.Read(buffer)
	if err != nil && err != io.EOF {
		return fmt.Errorf("failed to read file for MIME check: %w", err)
	}
	
	mimeType := http.DetectContentType(buffer[:n])
	
	// Extract base MIME type (remove charset and other parameters)
	baseMimeType := strings.Split(mimeType, ";")[0]
	baseMimeType = strings.TrimSpace(baseMimeType)
	
	if !AllowedMimeTypes[baseMimeType] {
		return fmt.Errorf("MIME type not allowed: %s for file %s", mimeType, filePath)
	}
	
	return nil
}

// loadFileSecurely loads a file with security validation using default settings
func loadFileSecurely(filePath string) ([]byte, error) {
	return loadFileSecurelyWithOptions(filePath, nil)
}

// loadFileSecurelyWithOptions loads a file with security validation using custom options
func loadFileSecurelyWithOptions(filePath string, maxFileSize *int64) ([]byte, error) {
	// Validate file security (this also checks if file exists)
	if err := validateFileWithOptions(filePath, maxFileSize); err != nil {
		return nil, fmt.Errorf("security validation failed: %w", err)
	}
	
	// Read file content
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}
	
	return content, nil
}

// validateContentWithSecurity performs comprehensive security validation on content bytes
func validateContentWithSecurity(content []byte, key string, maxFileSize *int64) error {
	// 1. Check file extension from key
	ext := strings.ToLower(filepath.Ext(key))
	if !AllowedExtensions[ext] {
		return fmt.Errorf("file type not allowed: %s for key '%s'", ext, key)
	}
	
	// 2. Check content size
	size := int64(len(content))
	maxSize := MaxFileSize
	if maxFileSize != nil {
		maxSize = *maxFileSize
	}
	
	if size > maxSize {
		return fmt.Errorf("content size %d bytes exceeds maximum %d bytes for key '%s'", size, maxSize, key)
	}
	
	// 3. MIME type detection from content
	mimeType := http.DetectContentType(content)
	
	// Extract base MIME type (remove charset and other parameters)
	baseMimeType := strings.Split(mimeType, ";")[0]
	baseMimeType = strings.TrimSpace(baseMimeType)
	
	if !AllowedMimeTypes[baseMimeType] {
		return fmt.Errorf("MIME type not allowed: %s for key %s", mimeType, key)
	}
	
	return nil
}

// validateFileWithMaps performs validation with explicit maps
func validateFileWithMaps(filePath string, maxFileSize *int64, allowedExt, allowedMime map[string]bool, defaultMax int64) error {
	// Check extension
	ext := strings.ToLower(filepath.Ext(filePath))
	if !allowedExt[ext] {
		return fmt.Errorf("file type not allowed: %s", ext)
	}
	
	// Check file exists and get size
	info, err := os.Stat(filePath)
	if err != nil {
		return fmt.Errorf("file does not exist: %w", err)
	}
	
	// Determine max file size (custom or service default)
	maxSize := defaultMax
	if maxFileSize != nil {
		maxSize = *maxFileSize
	}
	
	if info.Size() > maxSize {
		return fmt.Errorf("file too large: %d bytes (max %d)", info.Size(), maxSize)
	}
	
	// MIME type check (read first 512 bytes)
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file for MIME check: %w", err)
	}
	defer file.Close()
	
	buffer := make([]byte, 512)
	n, err := file.Read(buffer)
	if err != nil && err != io.EOF {
		return fmt.Errorf("failed to read file for MIME check: %w", err)
	}
	
	mimeType := http.DetectContentType(buffer[:n])
	
	// Extract base MIME type (remove charset and other parameters)
	baseMimeType := strings.Split(mimeType, ";")[0]
	baseMimeType = strings.TrimSpace(baseMimeType)
	
	if !allowedMime[baseMimeType] {
		return fmt.Errorf("MIME type not allowed: %s for file %s", mimeType, filePath)
	}
	
	return nil
}