package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"regexp"
	"strings"

	"zbz/zlog"
)

// Security field types defined by this adapter
const (
	SecretType = zlog.FieldType("secret") // Encrypt sensitive data
	PIIType    = zlog.FieldType("pii")    // Redact/hash PII based on compliance
)

// Security field constructors
func Secret(key, value string) zlog.Field {
	return zlog.Field{Key: key, Type: SecretType, Value: value}
}

func PII(key, value string) zlog.Field {
	return zlog.Field{Key: key, Type: PIIType, Value: value}
}

// Config holds security processor configuration
type Config struct {
	// Encryption key for secrets (32 bytes for AES-256)
	EncryptionKey []byte
	
	// PII handling mode: "hash", "redact", "partial"
	PIIMode string
	
	// Patterns to detect sensitive data
	SensitivePatterns []*regexp.Regexp
}

// DefaultConfig returns a basic security configuration
func DefaultConfig() Config {
	return Config{
		PIIMode: "hash",
		SensitivePatterns: []*regexp.Regexp{
			regexp.MustCompile(`\b\d{3}-\d{2}-\d{4}\b`),              // SSN
			regexp.MustCompile(`\b\d{4}[-\s]?\d{4}[-\s]?\d{4}[-\s]?\d{4}\b`), // Credit card
			regexp.MustCompile(`\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Z|a-z]{2,}\b`), // Email
		},
	}
}

// Register installs security processors for secret and PII field types
func Register(config Config) error {
	// Validate configuration
	if config.EncryptionKey != nil && len(config.EncryptionKey) != 32 {
		return fmt.Errorf("encryption key must be 32 bytes for AES-256")
	}
	
	// Register secret processor - CRITICAL: This runs before event emission
	zlog.RegisterFieldProcessor(SecretType, secretProcessor(config))
	
	// Register PII processor - CRITICAL: This runs before event emission  
	zlog.RegisterFieldProcessor(PIIType, piiProcessor(config))
	
	return nil
}

// secretProcessor handles encryption of secret fields
// CRITICAL: Returns StringType fields so downstream systems only see processed values
func secretProcessor(config Config) zlog.FieldProcessor {
	return func(field zlog.Field) []zlog.Field {
		value, ok := field.Value.(string)
		if !ok {
			// Non-string secrets are just redacted
			return []zlog.Field{zlog.String(field.Key, "***REDACTED***")}
		}
		
		// If no encryption key, redact
		if config.EncryptionKey == nil {
			return []zlog.Field{zlog.String(field.Key, "***REDACTED***")}
		}
		
		// Encrypt the value
		encrypted, err := encrypt(value, config.EncryptionKey)
		if err != nil {
			// On encryption failure, redact
			return []zlog.Field{
				zlog.String(field.Key, "***ENCRYPTION_FAILED***"),
				zlog.String(field.Key+"_error", err.Error()),
			}
		}
		
		return []zlog.Field{
			zlog.String(field.Key, "enc:"+encrypted),
			zlog.String(field.Key+"_encrypted", "true"),
		}
	}
}

// piiProcessor handles PII based on configured mode
// CRITICAL: Returns StringType fields so downstream systems only see processed values
func piiProcessor(config Config) zlog.FieldProcessor {
	return func(field zlog.Field) []zlog.Field {
		value, ok := field.Value.(string)
		if !ok {
			// Non-string PII - hash it
			return []zlog.Field{
				zlog.String(field.Key+"_hash", hash(fmt.Sprintf("%v", field.Value))),
			}
		}
		
		switch config.PIIMode {
		case "redact":
			return []zlog.Field{zlog.String(field.Key, "***PII_REDACTED***")}
			
		case "partial":
			// Show partial data (first and last few chars)
			masked := partialMask(value)
			return []zlog.Field{zlog.String(field.Key, masked)}
			
		case "hash":
			fallthrough
		default:
			// Hash the PII
			return []zlog.Field{
				zlog.String(field.Key+"_hash", hash(value)),
				zlog.String(field.Key+"_type", "pii"),
			}
		}
	}
}

// encrypt performs AES-256-GCM encryption
func encrypt(plaintext string, key []byte) (string, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	
	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// hash creates a consistent hash of the input
func hash(input string) string {
	h := sha256.Sum256([]byte(input))
	// Return first 16 chars of hex for brevity
	return fmt.Sprintf("%x", h)[:16]
}

// partialMask shows only parts of sensitive data
func partialMask(value string) string {
	length := len(value)
	if length <= 4 {
		return "****"
	}
	if length <= 8 {
		return value[:2] + strings.Repeat("*", length-2)
	}
	// Show first 2 and last 2 characters
	return value[:2] + strings.Repeat("*", length-4) + value[length-2:]
}

// Additional security utilities

// ScanForSensitive checks any string field for sensitive patterns
func ScanForSensitive(patterns []*regexp.Regexp) zlog.FieldProcessor {
	return func(field zlog.Field) []zlog.Field {
		// Only process string fields
		if field.Type != zlog.StringType {
			return []zlog.Field{field}
		}
		
		value, ok := field.Value.(string)
		if !ok {
			return []zlog.Field{field}
		}
		
		// Check each pattern
		for _, pattern := range patterns {
			if pattern.MatchString(value) {
				// Found sensitive data - redact it
				return []zlog.Field{
					zlog.String(field.Key, pattern.ReplaceAllString(value, "***REDACTED***")),
					zlog.Bool(field.Key+"_sanitized", true),
				}
			}
		}
		
		return []zlog.Field{field}
	}
}

// MaskIP replaces IP addresses with hashed versions for privacy
func MaskIP() zlog.FieldProcessor {
	ipPattern := regexp.MustCompile(`\b(?:\d{1,3}\.){3}\d{1,3}\b`)
	
	return func(field zlog.Field) []zlog.Field {
		if field.Key != "ip" && field.Key != "client_ip" && field.Key != "remote_addr" {
			return []zlog.Field{field}
		}
		
		value, ok := field.Value.(string)
		if !ok {
			return []zlog.Field{field}
		}
		
		if ipPattern.MatchString(value) {
			return []zlog.Field{
				zlog.String(field.Key, "ip_"+hash(value)),
				zlog.String(field.Key+"_masked", "true"),
			}
		}
		
		return []zlog.Field{field}
	}
}