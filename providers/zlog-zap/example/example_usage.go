package main

import (
	"crypto/rand"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"go.uber.org/zap"
	"zbz/plugins/zlog-security"
	zapProvider "zbz/providers/zlog-zap"
	"zbz/zlog"
)

func main() {
	fmt.Println("🚀 ZBZ Framework zlog + zap + security plugin demo")
	fmt.Println(strings.Repeat("=", 50))
	
	// Step 1: Initialize zap provider
	fmt.Println("\n📊 Step 1: Initialize zap development logger")
	zapContract := zapProvider.NewDevelopment()
	
	// Step 2: Set up output piping
	fmt.Println("\n📝 Step 2: Set up log piping")
	file, _ := os.OpenFile("/tmp/zlog-demo.log", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	zlog.Pipe(file)
	zlog.Pipe(&prefixWriter{prefix: "[INTERCEPTED] "})
	
	// Step 3: Demo basic logging
	fmt.Println("\n🎯 Step 3: Basic structured logging")
	zlog.Info("Application starting",
		zlog.String("version", "1.0.0"),
		zlog.Int("port", 8080),
		zlog.Bool("debug", true),
		zlog.Duration("startup_time", 150*time.Millisecond))
	
	// Step 4: Demo security plugin - setup
	fmt.Println("\n🔐 Step 4: Register security plugin")
	key := make([]byte, 32)
	rand.Read(key)
	
	securityConfig := security.Config{
		EncryptionKey: key,
		PIIMode:       "partial",
		SensitivePatterns: []*regexp.Regexp{
			regexp.MustCompile(`\b\d{3}-\d{2}-\d{4}\b`), // SSN
			regexp.MustCompile(`api[_-]?key[_-]?[a-zA-Z0-9]{16,}`), // API keys
		},
	}
	security.Register(securityConfig)
	
	// Step 5: Demo security features
	fmt.Println("\n🛡️  Step 5: Security features in action")
	
	// Encrypted secrets
	zlog.Info("API authentication",
		zlog.Secret("api_key", "sk_live_1234567890abcdef"),
		zlog.Secret("password", "super-secret-password"))
	
	// PII handling (partial mode)
	zlog.Info("User registration",
		zlog.PII("email", "john.doe@example.com"),
		zlog.PII("phone", "555-123-4567"),
		zlog.String("name", "John Doe"))
	
	// Step 6: Demo custom processors
	fmt.Println("\n⚙️  Step 6: Custom field processors")
	
	// Add custom processor for credit card fields
	zlog.Process(zlog.StringType, func(field zlog.Field) []zlog.Field {
		if field.Key == "credit_card" {
			value := field.Value.(string)
			if len(value) >= 12 {
				masked := "****-****-****-" + value[len(value)-4:]
				return []zlog.Field{zlog.String(field.Key, masked)}
			}
		}
		return []zlog.Field{field}
	})
	
	// Add IP masking
	zlog.Process(zlog.StringType, security.MaskIP())
	
	// Add pattern-based scanner for any string field
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`\b\d{3}-\d{2}-\d{4}\b`), // SSN in any field
	}
	zlog.Process(zlog.StringType, security.ScanForSensitive(patterns))
	
	// Demo the custom processors
	zlog.Info("Payment processed",
		zlog.String("credit_card", "4111111111111111"),
		zlog.String("client_ip", "192.168.1.100"),
		zlog.String("notes", "Customer SSN is 123-45-6789 for verification"),
		zlog.Float64("amount", 99.99))
	
	// Step 7: Demo all field types with processing
	fmt.Println("\n🎨 Step 7: All field types with processing")
	zlog.Info("Comprehensive logging demo",
		zlog.String("string_field", "regular string"),
		zlog.Int("user_id", 12345),
		zlog.Bool("is_premium", true),
		zlog.Time("timestamp", time.Now()),
		zlog.Duration("latency", 42*time.Millisecond),
		zlog.Any("metadata", map[string]any{
			"source": "demo",
			"build":  "v1.0.0",
		}))
	
	// Step 8: Error handling with security
	fmt.Println("\n❌ Step 8: Error logging with sensitive data")
	err := fmt.Errorf("authentication failed for api_key sk_test_abc123xyz")
	zlog.Error("Authentication error",
		zlog.Err(err),
		zlog.String("user_agent", "MyApp/1.0"))
	
	// Step 9: Native zap usage
	fmt.Println("\n⚡ Step 9: Direct zap logger access")
	zapLogger := zapContract.Logger()
	zapLogger.Info("Direct zap usage - bypasses zlog processors",
		zap.String("note", "This goes directly to zap without field processing"))
	
	// Step 10: Service usage simulation
	fmt.Println("\n🔧 Step 10: Service integration example")
	simulateServiceUsage(zapContract.Provider())
	
	fmt.Println("\n✅ Demo complete! Check /tmp/zlog-demo.log for file output")
}

// simulateServiceUsage shows how a service would use the provider interface
func simulateServiceUsage(logger zlog.Provider) {
	logger.Info("Service is starting", []zlog.Field{
		zlog.String("service", "user-auth"),
		zlog.String("component", "initialization"),
	})
	
	logger.Debug("Service configuration loaded", []zlog.Field{
		zlog.String("config_path", "/etc/app/config.yaml"),
		zlog.Int("timeout_ms", 5000),
	})
	
	logger.Warn("Service is using default configuration", []zlog.Field{
		zlog.String("reason", "config file not found"),
	})
}

// prefixWriter is a demo io.Writer that adds a prefix to all output
type prefixWriter struct {
	prefix string
}

func (p *prefixWriter) Write(data []byte) (n int, err error) {
	fmt.Printf("%s%s", p.prefix, string(data))
	return len(data), nil
}