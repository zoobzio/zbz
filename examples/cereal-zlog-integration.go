package main

import (
	"fmt"
	"time"
)

// This example demonstrates how zlog integrates with cereal for structured logging and field serialization

// Mock interfaces for demonstration
type ZlogProvider interface {
	Info(msg string, fields []Field)
	Error(msg string, fields []Field)
	Debug(msg string, fields []Field)
	Warn(msg string, fields []Field)
	Fatal(msg string, fields []Field)
	Close() error
}

type CerealProvider interface {
	Marshal(data any) ([]byte, error)
	Unmarshal(data []byte, target any) error
	MarshalScoped(data any, userPermissions []string) ([]byte, error)
	UnmarshalScoped(data []byte, target any, userPermissions []string, operation string) error
	ContentType() string
	Format() string
}

// Field represents a structured log field
type Field struct {
	Key   string
	Value interface{}
	Type  string
}

// FieldType constants
const (
	StringType = "string"
	IntType    = "int"
	AnyType    = "any"
)

// CerealZlog demonstrates zlog integration with cereal for structured data serialization
type CerealZlog struct {
	zlogProvider   ZlogProvider
	cerealProvider CerealProvider
}

// NewCerealZlog creates a new zlog service with cereal integration
func NewCerealZlog(zlog ZlogProvider, cereal CerealProvider) *CerealZlog {
	return &CerealZlog{
		zlogProvider:   zlog,
		cerealProvider: cereal,
	}
}

// Info logs with structured data serialization via cereal
func (cz *CerealZlog) Info(msg string, fields ...Field) {
	processedFields := cz.processFields(fields)
	cz.zlogProvider.Info(msg, processedFields)
}

// InfoScoped logs with field-level scoping (filters sensitive data)
func (cz *CerealZlog) InfoScoped(msg string, userPermissions []string, fields ...Field) {
	processedFields := cz.processScopedFields(fields, userPermissions)
	cz.zlogProvider.Info(msg, processedFields)
}

// Error logs with structured data serialization
func (cz *CerealZlog) Error(msg string, fields ...Field) {
	processedFields := cz.processFields(fields)
	cz.zlogProvider.Error(msg, processedFields)
}

// ErrorScoped logs with field-level scoping
func (cz *CerealZlog) ErrorScoped(msg string, userPermissions []string, fields ...Field) {
	processedFields := cz.processScopedFields(fields, userPermissions)
	cz.zlogProvider.Error(msg, processedFields)
}

// processFields serializes complex data structures using cereal
func (cz *CerealZlog) processFields(fields []Field) []Field {
	var processed []Field
	
	for _, field := range fields {
		switch field.Type {
		case AnyType:
			// Use cereal to serialize complex data structures
			if serialized, err := cz.cerealProvider.Marshal(field.Value); err == nil {
				processed = append(processed, Field{
					Key:   field.Key,
					Value: string(serialized),
					Type:  StringType,
				})
			} else {
				// Fall back to string representation if serialization fails
				processed = append(processed, Field{
					Key:   field.Key,
					Value: fmt.Sprintf("%v", field.Value),
					Type:  StringType,
				})
			}
		default:
			// Pass through simple types
			processed = append(processed, field)
		}
	}
	
	return processed
}

// processScopedFields serializes data with field-level permission filtering
func (cz *CerealZlog) processScopedFields(fields []Field, userPermissions []string) []Field {
	var processed []Field
	
	for _, field := range fields {
		switch field.Type {
		case AnyType:
			// Use cereal to serialize with scoping
			if serialized, err := cz.cerealProvider.MarshalScoped(field.Value, userPermissions); err == nil {
				processed = append(processed, Field{
					Key:   field.Key,
					Value: string(serialized),
					Type:  StringType,
				})
			} else {
				// Log error but continue (field will be omitted)
				fmt.Printf("Scoped serialization failed for field %s: %v\n", field.Key, err)
			}
		default:
			// Pass through simple types (no scoping needed)
			processed = append(processed, field)
		}
	}
	
	return processed
}

// LogEvent serializes and logs an entire event structure
func (cz *CerealZlog) LogEvent(level string, msg string, event interface{}) {
	field := Field{
		Key:   "event",
		Value: event,
		Type:  AnyType,
	}
	
	switch level {
	case "info":
		cz.Info(msg, field)
	case "error":
		cz.Error(msg, field)
	default:
		cz.Info(msg, field)
	}
}

// LogEventScoped serializes and logs an event with field-level scoping
func (cz *CerealZlog) LogEventScoped(level string, msg string, event interface{}, userPermissions []string) {
	field := Field{
		Key:   "event",
		Value: event,
		Type:  AnyType,
	}
	
	switch level {
	case "info":
		cz.InfoScoped(msg, userPermissions, field)
	case "error":
		cz.ErrorScoped(msg, userPermissions, field)
	default:
		cz.InfoScoped(msg, userPermissions, field)
	}
}

// Test data structures for logging
type UserEvent struct {
	UserID    int    `json:"user_id"`
	Action    string `json:"action"`
	Timestamp string `json:"timestamp"`
	Email     string `json:"email" scope:"read:admin"`
	IPAddress string `json:"ip_address" scope:"read:security"`
	SessionID string `json:"session_id" scope:"read:admin"`
}

type SystemEvent struct {
	Component string                 `json:"component"`
	Level     string                 `json:"level"`
	Message   string                 `json:"message"`
	Metadata  map[string]interface{} `json:"metadata"`
}

// Mock implementations
type MockZlog struct{}

func (m *MockZlog) Info(msg string, fields []Field) {
	fmt.Printf("[INFO] %s", msg)
	for _, field := range fields {
		fmt.Printf(" %s=%v", field.Key, field.Value)
	}
	fmt.Println()
}

func (m *MockZlog) Error(msg string, fields []Field) {
	fmt.Printf("[ERROR] %s", msg)
	for _, field := range fields {
		fmt.Printf(" %s=%v", field.Key, field.Value)
	}
	fmt.Println()
}

func (m *MockZlog) Debug(msg string, fields []Field) {
	fmt.Printf("[DEBUG] %s", msg)
	for _, field := range fields {
		fmt.Printf(" %s=%v", field.Key, field.Value)
	}
	fmt.Println()
}

func (m *MockZlog) Warn(msg string, fields []Field) {
	fmt.Printf("[WARN] %s", msg)
	for _, field := range fields {
		fmt.Printf(" %s=%v", field.Key, field.Value)
	}
	fmt.Println()
}

func (m *MockZlog) Fatal(msg string, fields []Field) {
	fmt.Printf("[FATAL] %s", msg)
	for _, field := range fields {
		fmt.Printf(" %s=%v", field.Key, field.Value)
	}
	fmt.Println()
}

func (m *MockZlog) Close() error {
	return nil
}

type MockCereal struct{}

func (m *MockCereal) Marshal(data any) ([]byte, error) {
	return []byte(fmt.Sprintf(`{"serialized": "%T", "data": "mock_serialized"}`)), nil
}

func (m *MockCereal) Unmarshal(data []byte, target any) error {
	return nil
}

func (m *MockCereal) MarshalScoped(data any, userPermissions []string) ([]byte, error) {
	// Simulate field filtering based on permissions
	hasAdmin := contains(userPermissions, "admin")
	hasSecurity := contains(userPermissions, "security")
	
	if hasAdmin && hasSecurity {
		return []byte(`{"user_id": 123, "action": "login", "email": "user@example.com", "ip_address": "192.168.1.1", "session_id": "abc123"}`), nil
	} else if hasAdmin {
		return []byte(`{"user_id": 123, "action": "login", "email": "user@example.com", "session_id": "abc123"}`), nil
	} else if hasSecurity {
		return []byte(`{"user_id": 123, "action": "login", "ip_address": "192.168.1.1"}`), nil
	} else {
		return []byte(`{"user_id": 123, "action": "login"}`), nil
	}
}

func (m *MockCereal) UnmarshalScoped(data []byte, target any, userPermissions []string, operation string) error {
	return nil
}

func (m *MockCereal) ContentType() string { return "application/json" }
func (m *MockCereal) Format() string { return "json" }

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// Convenience functions for creating log fields
func String(key, value string) Field {
	return Field{Key: key, Value: value, Type: StringType}
}

func Int(key string, value int) Field {
	return Field{Key: key, Value: value, Type: IntType}
}

func Any(key string, value interface{}) Field {
	return Field{Key: key, Value: value, Type: AnyType}
}

func main() {
	fmt.Println("📝 Cereal-Zlog Integration Demo")
	fmt.Println("===============================")
	
	// Set up services
	zlog := &MockZlog{}
	cereal := &MockCereal{}
	logger := NewCerealZlog(zlog, cereal)
	
	fmt.Println("\n📋 Basic Structured Logging:")
	
	// Basic logging with simple fields
	logger.Info("User logged in",
		String("user_id", "123"),
		String("action", "login"),
		Int("attempts", 1),
	)
	
	// Logging with complex structured data
	userEvent := UserEvent{
		UserID:    123,
		Action:    "login",
		Timestamp: time.Now().Format(time.RFC3339),
		Email:     "user@example.com",
		IPAddress: "192.168.1.1",
		SessionID: "abc123def456",
	}
	
	fmt.Println("\n📦 Complex Data Logging:")
	logger.LogEvent("info", "User event occurred", userEvent)
	
	// System event logging
	systemEvent := SystemEvent{
		Component: "auth-service",
		Level:     "info",
		Message:   "Authentication successful",
		Metadata: map[string]interface{}{
			"duration_ms": 45,
			"method":      "oauth2",
			"provider":    "google",
		},
	}
	
	logger.LogEvent("info", "System event", systemEvent)
	
	fmt.Println("\n🔒 Scoped Logging (Public User):")
	
	// Log with public permissions (limited fields)
	publicPermissions := []string{"public"}
	logger.LogEventScoped("info", "User event (public view)", userEvent, publicPermissions)
	
	fmt.Println("\n🔒 Scoped Logging (Admin User):")
	
	// Log with admin permissions (more fields visible)
	adminPermissions := []string{"admin"}
	logger.LogEventScoped("info", "User event (admin view)", userEvent, adminPermissions)
	
	fmt.Println("\n🔒 Scoped Logging (Security Team):")
	
	// Log with security permissions (security-specific fields)
	securityPermissions := []string{"security"}
	logger.LogEventScoped("info", "User event (security view)", userEvent, securityPermissions)
	
	fmt.Println("\n🔒 Scoped Logging (Full Access):")
	
	// Log with full permissions
	fullPermissions := []string{"admin", "security"}
	logger.LogEventScoped("info", "User event (full view)", userEvent, fullPermissions)
	
	fmt.Println("\n⚠️  Error Logging with Context:")
	
	// Error logging with structured data
	errorContext := map[string]interface{}{
		"error_code":    "AUTH_FAILED",
		"attempts":      3,
		"locked_until":  time.Now().Add(15 * time.Minute).Format(time.RFC3339),
		"user_agent":    "Mozilla/5.0...",
		"client_ip":     "192.168.1.100",
	}
	
	logger.Error("Authentication failed",
		String("user_id", "456"),
		Any("error_context", errorContext),
	)
	
	fmt.Println("\n🎉 Cereal-Zlog Integration Demo Complete!")
	fmt.Println("=========================================")
	fmt.Println("Key Benefits Demonstrated:")
	fmt.Println("✅ Unified serialization for complex log data")
	fmt.Println("✅ Field-level scoping for sensitive information")
	fmt.Println("✅ Structured logging with automatic serialization")
	fmt.Println("✅ Permission-based log field filtering")
	fmt.Println("✅ Type-safe logging operations")
}