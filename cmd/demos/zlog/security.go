package zlog

import (
	"crypto/rand"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"zbz/plugins/zlog-security"
	zapProvider "zbz/providers/zlog-zap"
	"zbz/zlog"
)

// SecurityDemo demonstrates zlog security features with encryption and PII protection
func SecurityDemo() {
	fmt.Println("🛡️  ZBZ Framework zlog Security Plugin Demo")
	fmt.Println(strings.Repeat("=", 55))

	// Step 1: Initialize logging
	fmt.Println("\n📊 Step 1: Initialize zap development logger")
	_ = zapProvider.NewDevelopment()

	// Step 2: Set up file output
	fmt.Println("\n📝 Step 2: Set up secure log output")
	file, _ := os.OpenFile("/tmp/zlog-security-demo.log", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	zlog.Pipe(file)
	defer file.Close()

	// Step 3: Configure security plugin
	fmt.Println("\n🔐 Step 3: Configure security plugin with encryption")

	// Generate encryption key
	key := make([]byte, 32)
	rand.Read(key)

	securityConfig := security.Config{
		EncryptionKey: key,
		PIIMode:       "partial", // Show partial data
		SensitivePatterns: []*regexp.Regexp{
			regexp.MustCompile(`\b\d{3}-\d{2}-\d{4}\b`),                      // SSN
			regexp.MustCompile(`\b\d{4}[-\s]?\d{4}[-\s]?\d{4}[-\s]?\d{4}\b`), // Credit card
			regexp.MustCompile(`api[_-]?key[_-]?[a-zA-Z0-9]{16,}`),           // API keys
		},
	}

	err := security.Register(securityConfig)
	if err != nil {
		fmt.Printf("❌ Failed to register security plugin: %v\n", err)
		return
	}

	// Step 4: Demo secret encryption
	fmt.Println("\n🔒 Step 4: Secret field encryption")
	zlog.Info("API authentication attempt",
		zlog.String("user_id", "user_12345"),
		zlog.Secret("api_key", "sk_live_1234567890abcdefghijk"),
		zlog.Secret("password", "super-secret-password-123"),
		zlog.String("endpoint", "/api/v1/users"))

	// Step 5: Demo PII protection
	fmt.Println("\n👤 Step 5: PII protection (partial mode)")
	zlog.Info("User registration",
		zlog.String("action", "user_created"),
		zlog.PII("email", "john.doe@company.com"),
		zlog.PII("phone", "555-123-4567"),
		zlog.PII("full_name", "John Michael Doe"),
		zlog.String("user_id", "user_12345"),
		zlog.Time("created_at", time.Now()))

	// Step 6: Custom field processors
	fmt.Println("\n⚙️  Step 6: Custom field processors")

	// Credit card masking
	zlog.Process(zlog.StringType, func(field zlog.Field) []zlog.Field {
		if field.Key == "credit_card" || field.Key == "cc_number" {
			value := field.Value.(string)
			if len(value) >= 12 {
				masked := "****-****-****-" + value[len(value)-4:]
				return []zlog.Field{zlog.String(field.Key, masked)}
			}
		}
		return []zlog.Field{field}
	})

	// IP address privacy
	zlog.Process(zlog.StringType, security.MaskIP())

	// Pattern-based sensitive data detection
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`\b\d{3}-\d{2}-\d{4}\b`), // SSN in any field
	}
	zlog.Process(zlog.StringType, security.ScanForSensitive(patterns))

	// Step 7: Demo custom processors in action
	fmt.Println("\n💳 Step 7: Payment processing with security")
	zlog.Info("Payment processed",
		zlog.String("transaction_id", "txn_abc123"),
		zlog.String("credit_card", "4111111111111111"),
		zlog.Float64("amount", 99.99),
		zlog.String("currency", "USD"),
		zlog.String("client_ip", "192.168.1.100"),
		zlog.String("notes", "Customer SSN 123-45-6789 verified for large purchase"),
		zlog.Bool("success", true))

	// Step 8: Error handling with sensitive data
	fmt.Println("\n❌ Step 8: Error logging with sensitive data sanitization")
	sensitiveError := fmt.Errorf("authentication failed for api_key sk_test_abc123xyz789")
	zlog.Error("Authentication error",
		zlog.Err(sensitiveError),
		zlog.String("user_agent", "MyApp/1.0"),
		zlog.String("ip_address", "203.0.113.42"),
		zlog.Time("timestamp", time.Now()))

	// Step 9: Different PII modes demonstration
	fmt.Println("\n🔄 Step 9: Switching PII modes")

	// Switch to hash mode
	fmt.Println("   Switching to hash mode...")
	security.Register(security.Config{
		EncryptionKey: key,
		PIIMode:       "hash",
	})

	zlog.Info("User lookup (hash mode)",
		zlog.PII("email", "jane.smith@example.com"),
		zlog.String("operation", "user_search"))

	// Switch to redact mode
	fmt.Println("   Switching to redact mode...")
	security.Register(security.Config{
		EncryptionKey: key,
		PIIMode:       "redact",
	})

	zlog.Info("User data export (redact mode)",
		zlog.PII("ssn", "987-65-4321"),
		zlog.PII("phone", "555-987-6543"),
		zlog.String("operation", "data_export"))

	// Step 10: Security audit logging
	fmt.Println("\n📋 Step 10: Security audit logging")
	zlog.Info("Security audit event",
		zlog.String("event_type", "privilege_escalation"),
		zlog.String("user_id", "admin_001"),
		zlog.Secret("session_token", "sess_1234567890abcdef"),
		zlog.String("resource", "/admin/users"),
		zlog.String("action", "DELETE"),
		zlog.Bool("authorized", true),
		zlog.Time("timestamp", time.Now()))

	fmt.Println("\n✅ Security demo complete!")
	fmt.Println("🔐 Secrets encrypted, PII protected, patterns detected")
	fmt.Println("📁 Check /tmp/zlog-security-demo.log for secure output")
}
