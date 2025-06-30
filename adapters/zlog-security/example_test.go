package security_test

import (
	"crypto/rand"
	"fmt"
	"regexp"

	"zbz/plugins/zlog-security"
	"zbz/zlog"
)

func Example_basic() {
	// Basic usage - just redaction
	security.Register(security.DefaultConfig())
	
	// Secrets will be redacted
	zlog.Info("User login",
		zlog.String("username", "john.doe"),
		zlog.Secret("password", "super-secret-123"),
	)
	// Output: {"username":"john.doe","password":"***REDACTED***"}
}

func Example_encryption() {
	// Generate a 32-byte key for AES-256
	key := make([]byte, 32)
	rand.Read(key)
	
	// Configure with encryption
	config := security.Config{
		EncryptionKey: key,
		PIIMode:       "hash",
	}
	security.Register(config)
	
	// Secrets will be encrypted
	zlog.Info("API call",
		zlog.String("endpoint", "/api/v1/users"),
		zlog.Secret("api_key", "sk_live_abc123xyz"),
	)
	// Output: {"endpoint":"/api/v1/users","api_key":"enc:base64...","api_key_encrypted":"true"}
}

func Example_piiModes() {
	// Different PII handling modes
	
	// Hash mode (default)
	config := security.Config{PIIMode: "hash"}
	security.Register(config)
	
	zlog.Info("User registered",
		zlog.PII("email", "john.doe@example.com"),
	)
	// Output: {"email_hash":"a1b2c3d4...","email_type":"pii"}
	
	// Redact mode
	config.PIIMode = "redact"
	security.Register(config)
	
	zlog.Info("User info",
		zlog.PII("ssn", "123-45-6789"),
	)
	// Output: {"ssn":"***PII_REDACTED***"}
	
	// Partial mode  
	config.PIIMode = "partial"
	security.Register(config)
	
	zlog.Info("Contact",
		zlog.PII("phone", "555-123-4567"),
	)
	// Output: {"phone":"55*********67"}
}

func Example_customProcessors() {
	// Register basic security
	security.Register(security.DefaultConfig())
	
	// Add custom processor for credit card fields
	zlog.Process(zlog.StringType, func(field zlog.Field) []zlog.Field {
		if field.Key == "credit_card" || field.Key == "cc_number" {
			value := field.Value.(string)
			if len(value) >= 12 {
				// Show only last 4 digits
				masked := fmt.Sprintf("****-****-****-%s", value[len(value)-4:])
				return []zlog.Field{zlog.String(field.Key, masked)}
			}
		}
		return []zlog.Field{field}
	})
	
	// Add IP masking for privacy
	zlog.Process(zlog.StringType, security.MaskIP())
	
	// Add pattern-based sensitive data scanner
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`\b\d{3}-\d{2}-\d{4}\b`), // SSN
		regexp.MustCompile(`key[_-]?[a-zA-Z0-9]{16,}`), // API keys
	}
	zlog.Process(zlog.StringType, security.ScanForSensitive(patterns))
	
	// Now log with various sensitive fields
	zlog.Info("Transaction processed",
		zlog.String("credit_card", "4111111111111111"),
		zlog.String("client_ip", "192.168.1.100"),
		zlog.String("notes", "Customer SSN is 123-45-6789"),
	)
	// Output: {
	//   "credit_card": "****-****-****-1111",
	//   "client_ip": "ip_7f4e9b1a...",
	//   "client_ip_masked": "true",
	//   "notes": "Customer SSN is ***REDACTED***",
	//   "notes_sanitized": true
	// }
}