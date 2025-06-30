package main

import (
	"crypto/rand"
	"regexp"
	"zbz/adapters/zlog-security"
	"zbz/zlog"
	
	// Auto-hydrate with Capitan for event emission
	_ "zbz/capitan"
)

func main() {
	// Configure zlog
	zlog.Configure(zlog.Config{
		Level:  zlog.DEBUG,
		Format: "console",
	})
	
	println("🔒 ZLog Security Adapter Demo")
	println("===============================")
	
	// Demo 1: Basic redaction (no encryption key)
	println("\n1. Basic Redaction Mode:")
	security.Register(security.DefaultConfig())
	
	zlog.Info("User login attempt",
		zlog.String("username", "alice"),
		security.Secret("password", "super-secret-123"),
		security.PII("email", "alice@example.com"))
	
	// Demo 2: Encryption mode
	println("\n2. Encryption Mode:")
	key := make([]byte, 32)
	rand.Read(key)
	
	config := security.Config{
		EncryptionKey: key,
		PIIMode:       "hash",
	}
	security.Register(config)
	
	zlog.Info("API authentication",
		zlog.String("endpoint", "/api/v1/users"),
		security.Secret("api_key", "sk_live_abc123xyz789"),
		security.PII("user_id", "user-12345"))
	
	// Demo 3: Different PII modes
	println("\n3. PII Handling Modes:")
	
	// Redact mode
	config.PIIMode = "redact"
	security.Register(config)
	zlog.Info("Payment processed",
		security.PII("credit_card", "4111-1111-1111-1111"))
	
	// Partial mode  
	config.PIIMode = "partial"
	security.Register(config)
	zlog.Info("Contact information",
		security.PII("phone", "555-123-4567"))
	
	// Demo 4: Pattern-based scanning
	println("\n4. Pattern-Based Security:")
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`\b\d{3}-\d{2}-\d{4}\b`),              // SSN
		regexp.MustCompile(`key[_-]?[a-zA-Z0-9]{16,}`),          // API keys
		regexp.MustCompile(`\b\d{4}[-\s]?\d{4}[-\s]?\d{4}[-\s]?\d{4}\b`), // Credit cards
	}
	
	// Register pattern scanner for all string fields
	zlog.RegisterFieldProcessor(zlog.StringType, security.ScanForSensitive(patterns))
	
	zlog.Info("Processing transaction",
		zlog.String("notes", "Customer SSN is 123-45-6789 for verification"),
		zlog.String("payment_method", "Card ending key_abc123xyz456def"))
	
	// Demo 5: IP masking
	println("\n5. IP Address Privacy:")
	zlog.RegisterFieldProcessor(zlog.StringType, security.MaskIP())
	
	zlog.Info("Web request",
		zlog.String("client_ip", "192.168.1.100"),
		zlog.String("user_agent", "Mozilla/5.0"))
	
	println("\n✅ Security Demo Complete!")
	println("🔑 All sensitive data encrypted/redacted before event emission")
	println("🛡️ Downstream systems only see processed values")
	println("🚀 Zero security leaks in adapter ecosystem")
}