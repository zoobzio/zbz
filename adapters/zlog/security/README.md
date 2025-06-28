# zlog-security

Security plugin for zlog that provides encryption, PII handling, and sensitive data protection.

## Features

- **Secret Encryption**: AES-256-GCM encryption for secret fields
- **PII Protection**: Multiple modes for handling personally identifiable information
- **Pattern Detection**: Automatic detection and redaction of sensitive patterns
- **IP Masking**: Privacy-preserving IP address handling
- **Custom Processors**: Extensible architecture for custom security rules

## Installation

```go
import (
    "zbz/plugins/zlog-security"
    "zbz/zlog"
)
```

## Quick Start

### Basic Usage (Redaction Only)

```go
// Register with default config (no encryption key)
security.Register(security.DefaultConfig())

// Secrets will be redacted
zlog.Info("Login attempt", 
    zlog.Secret("password", "my-password"))
// Output: {"password":"***REDACTED***"}
```

### With Encryption

```go
// Generate or load a 32-byte encryption key
key := make([]byte, 32)
rand.Read(key)

config := security.Config{
    EncryptionKey: key,
    PIIMode: "hash",
}
security.Register(config)

// Secrets will be encrypted
zlog.Info("API call",
    zlog.Secret("api_key", "sk_live_secret123"))
// Output: {"api_key":"enc:base64...","api_key_encrypted":"true"}
```

## PII Handling Modes

The plugin supports three modes for handling PII:

### Hash Mode (Default)
```go
config := security.Config{PIIMode: "hash"}
zlog.Info("User", zlog.PII("email", "user@example.com"))
// Output: {"email_hash":"a1b2c3d4...","email_type":"pii"}
```

### Redact Mode
```go
config := security.Config{PIIMode: "redact"}
zlog.Info("User", zlog.PII("ssn", "123-45-6789"))
// Output: {"ssn":"***PII_REDACTED***"}
```

### Partial Mode
```go
config := security.Config{PIIMode: "partial"}
zlog.Info("User", zlog.PII("phone", "555-123-4567"))
// Output: {"phone":"55*********67"}
```

## Advanced Features

### Pattern-Based Detection

Automatically detect and redact sensitive patterns in any string field:

```go
patterns := []*regexp.Regexp{
    regexp.MustCompile(`\b\d{3}-\d{2}-\d{4}\b`), // SSN
    regexp.MustCompile(`key[_-]?[a-zA-Z0-9]{16,}`), // API keys
}
zlog.Process(zlog.StringType, security.ScanForSensitive(patterns))
```

### IP Address Masking

Privacy-preserving IP logging:

```go
zlog.Process(zlog.StringType, security.MaskIP())

zlog.Info("Request", zlog.String("client_ip", "192.168.1.100"))
// Output: {"client_ip":"ip_7f4e9b1a...","client_ip_masked":"true"}
```

## Security Considerations

1. **Key Management**: Store encryption keys securely (e.g., environment variables, key management service)
2. **Performance**: Encryption adds overhead - use selectively for truly sensitive data
3. **Compliance**: Choose PII mode based on your compliance requirements (GDPR, CCPA, etc.)
4. **Patterns**: Regularly update sensitive data patterns based on your data types

## Configuration

```go
type Config struct {
    // 32-byte key for AES-256 encryption (nil for redaction only)
    EncryptionKey []byte
    
    // PII handling: "hash", "redact", or "partial"
    PIIMode string
    
    // Regex patterns for sensitive data detection
    SensitivePatterns []*regexp.Regexp
}
```

## Best Practices

1. Use `zlog.Secret()` for passwords, API keys, tokens
2. Use `zlog.PII()` for emails, phone numbers, SSNs, names
3. Register security processors early in application startup
4. Consider using different PII modes for development vs production
5. Regularly rotate encryption keys