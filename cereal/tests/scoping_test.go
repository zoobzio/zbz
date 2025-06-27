package cereal_tests

import (
	"encoding/json"
	"testing"

	"zbz/cereal"
)

// Test struct with various scoping levels
type User struct {
	ID           int    `json:"id" yaml:"id" toml:"id"`                                         // No scope = always allowed
	Name         string `json:"name" yaml:"name" toml:"name" scope:"public"`                        // Simple scope
	Email        string `json:"email" yaml:"email" toml:"email" scope:"read,write"`                   // OR logic: read OR write
	Phone        string `json:"phone" yaml:"phone" toml:"phone" scope:"self"`                         // Simple scope
	Password     string `json:"password" yaml:"password" toml:"password" scope:"admin"`                     // Admin only
	SSN          string `json:"ssn" yaml:"ssn" toml:"ssn" scope:"admin+pii"`                      // AND logic: admin AND pii
	AuditLog     string `json:"audit_log" yaml:"audit_log" toml:"audit_log" scope:"admin+security+executive"` // Triple AND: admin AND security AND executive
	ComplianceID string `json:"compliance_id" yaml:"compliance_id" toml:"compliance_id" scope:"compliance,admin+pii"` // Complex OR: compliance OR (admin AND pii)
}

func TestMarshalScoping(t *testing.T) {
	// Full user data (as it exists in database)
	user := User{
		ID:           123,
		Name:         "John Doe",
		Email:        "john@example.com",
		Phone:        "+1234567890",
		Password:     "secret123",
		SSN:          "123-45-6789",
		AuditLog:     "sensitive audit data",
		ComplianceID: "COMP-789",
	}

	tests := []struct {
		name            string
		permissions     []string
		expectedFields  []string // Fields that should be present
		forbiddenFields []string // Fields that should be absent
	}{
		{
			name:            "Public access only",
			permissions:     []string{"public"},
			expectedFields:  []string{"id", "name"},
			forbiddenFields: []string{"email", "phone", "password", "ssn", "audit_log", "compliance_id"},
		},
		{
			name:            "Read access",
			permissions:     []string{"public", "read"},
			expectedFields:  []string{"id", "name", "email"},
			forbiddenFields: []string{"phone", "password", "ssn", "audit_log", "compliance_id"},
		},
		{
			name:            "Self access",
			permissions:     []string{"public", "read", "self"},
			expectedFields:  []string{"id", "name", "email", "phone"},
			forbiddenFields: []string{"password", "ssn", "audit_log", "compliance_id"},
		},
		{
			name:            "Admin access (missing pii)",
			permissions:     []string{"public", "read", "admin"},
			expectedFields:  []string{"id", "name", "email", "password"},
			forbiddenFields: []string{"phone", "ssn", "audit_log", "compliance_id"}, // SSN needs admin+pii
		},
		{
			name:            "Admin + PII access",
			permissions:     []string{"public", "read", "admin", "pii"},
			expectedFields:  []string{"id", "name", "email", "password", "ssn", "compliance_id"}, // compliance_id: compliance OR (admin+pii)
			forbiddenFields: []string{"phone", "audit_log"},                                      // audit_log needs admin+security+executive
		},
		{
			name:            "Compliance access",
			permissions:     []string{"public", "compliance"},
			expectedFields:  []string{"id", "name", "compliance_id"}, // compliance_id: compliance OR (admin+pii)
			forbiddenFields: []string{"email", "phone", "password", "ssn", "audit_log"},
		},
		{
			name:            "Full executive access",
			permissions:     []string{"public", "read", "admin", "security", "executive"},
			expectedFields:  []string{"id", "name", "email", "password", "audit_log"},
			forbiddenFields: []string{"phone", "ssn", "compliance_id"}, // SSN needs pii, compliance_id needs pii
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Marshal with permissions
			data, err := cereal.JSON.Marshal(user, tt.permissions...)
			if err != nil {
				t.Fatalf("Marshal failed: %v", err)
			}

			// Parse result to check field presence
			var result map[string]any
			if err := json.Unmarshal(data, &result); err != nil {
				t.Fatalf("Failed to parse marshal result: %v", err)
			}

			// Check expected fields are present
			for _, field := range tt.expectedFields {
				if _, exists := result[field]; !exists {
					t.Errorf("Expected field '%s' to be present, but it was filtered out", field)
				}
			}

			// Check forbidden fields are absent
			for _, field := range tt.forbiddenFields {
				if _, exists := result[field]; exists {
					t.Errorf("Expected field '%s' to be filtered out, but it was present: %v", field, result[field])
				}
			}
		})
	}
}

func TestUnmarshalScoping(t *testing.T) {
	tests := []struct {
		name               string
		inputJSON          string
		permissions        []string
		expectedSetFields  map[string]any // Fields that should be set
		expectedZeroFields []string       // Fields that should remain zero
	}{
		{
			name:        "Public write access - malicious payload",
			inputJSON:   `{"id":999,"name":"Hacker","email":"hacker@evil.com","password":"stolen","ssn":"000-00-0000","audit_log":"fake audit"}`,
			permissions: []string{"public", "write"},
			expectedSetFields: map[string]any{
				"id":    999,               // No scope = allowed
				"name":  "Hacker",          // scope:"public" = allowed
				"email": "hacker@evil.com", // scope:"read,write" = write allowed
			},
			expectedZeroFields: []string{"phone", "password", "ssn", "audit_log", "compliance_id"}, // All restricted
		},
		{
			name:        "Admin access - partial payload",
			inputJSON:   `{"name":"Admin User","password":"newpass","ssn":"123-45-6789","audit_log":"admin tried to set audit"}`,
			permissions: []string{"public", "admin"},
			expectedSetFields: map[string]any{
				"name":     "Admin User", // scope:"public" = allowed
				"password": "newpass",    // scope:"admin" = allowed
			},
			expectedZeroFields: []string{"phone", "email", "ssn", "audit_log", "compliance_id"}, // SSN needs admin+pii, audit needs admin+security+executive
		},
		{
			name:        "Admin + PII access",
			inputJSON:   `{"name":"PII Admin","password":"secret","ssn":"987-65-4321","compliance_id":"COMP-123"}`,
			permissions: []string{"admin", "pii"},
			expectedSetFields: map[string]any{
				"password":      "secret",      // scope:"admin" = allowed
				"ssn":           "987-65-4321", // scope:"admin+pii" = both allowed
				"compliance_id": "COMP-123",    // scope:"compliance,admin+pii" = admin+pii satisfied
			},
			expectedZeroFields: []string{"name", "email", "phone", "audit_log"}, // name needs public, email needs read/write, audit needs admin+security+executive
		},
		{
			name:        "Compliance only access",
			inputJSON:   `{"compliance_id":"COMP-456","ssn":"111-11-1111","password":"hack"}`,
			permissions: []string{"compliance"},
			expectedSetFields: map[string]any{
				"compliance_id": "COMP-456", // scope:"compliance,admin+pii" = compliance satisfied
			},
			expectedZeroFields: []string{"name", "email", "phone", "password", "ssn", "audit_log"}, // All others restricted
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var user User

			// Unmarshal with scoping
			err := cereal.JSON.Unmarshal([]byte(tt.inputJSON), &user, tt.permissions...)
			if err != nil {
				t.Fatalf("Unmarshal failed: %v", err)
			}

			// Check expected set fields
			userMap := map[string]any{
				"id":            user.ID,
				"name":          user.Name,
				"email":         user.Email,
				"phone":         user.Phone,
				"password":      user.Password,
				"ssn":           user.SSN,
				"audit_log":     user.AuditLog,
				"compliance_id": user.ComplianceID,
			}

			for field, expectedValue := range tt.expectedSetFields {
				actualValue := userMap[field]
				if actualValue != expectedValue {
					t.Errorf("Expected field '%s' to be set to %v, but got %v", field, expectedValue, actualValue)
				}
			}

			// Check zero fields remain zero
			for _, field := range tt.expectedZeroFields {
				actualValue := userMap[field]
				if !isZeroValue(actualValue) {
					t.Errorf("Expected field '%s' to be zero/empty, but got %v", field, actualValue)
				}
			}
		})
	}
}

// Helper function to check if value is zero
func isZeroValue(v any) bool {
	switch val := v.(type) {
	case string:
		return val == ""
	case int:
		return val == 0
	default:
		return v == nil
	}
}

func TestComplexScopingScenarios(t *testing.T) {
	t.Run("Round trip test", func(t *testing.T) {
		// Original data
		original := User{
			ID:           456,
			Name:         "Alice",
			Email:        "alice@company.com",
			Password:     "supersecret",
			SSN:          "555-55-5555",
			ComplianceID: "COMP-999",
		}

		// Marshal with limited permissions
		data, err := cereal.JSON.Marshal(original, "public", "read")
		if err != nil {
			t.Fatalf("Marshal failed: %v", err)
		}

		// Should only contain: id, name, email
		var filtered map[string]any
		json.Unmarshal(data, &filtered)

		if len(filtered) != 3 {
			t.Errorf("Expected 3 fields in filtered data, got %d: %v", len(filtered), filtered)
		}

		// Unmarshal back with same permissions
		var result User
		err = cereal.JSON.Unmarshal(data, &result, "public", "read")
		if err != nil {
			t.Fatalf("Unmarshal failed: %v", err)
		}

		// Should match the filtered fields
		if result.ID != original.ID || result.Name != original.Name || result.Email != original.Email {
			t.Errorf("Round trip failed: expected ID=%d, Name=%s, Email=%s; got ID=%d, Name=%s, Email=%s",
				original.ID, original.Name, original.Email,
				result.ID, result.Name, result.Email)
		}

		// Restricted fields should be empty
		if result.Password != "" || result.SSN != "" || result.ComplianceID != "" {
			t.Errorf("Restricted fields should be empty: Password=%s, SSN=%s, ComplianceID=%s",
				result.Password, result.SSN, result.ComplianceID)
		}
	})
}
