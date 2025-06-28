package cereal_tests

import (
	"testing"
	"github.com/pelletier/go-toml/v2"
	"zbz/cereal"
)

func TestTOMLMarshalScoping(t *testing.T) {
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
		expectedFields  []string
		forbiddenFields []string
	}{
		{
			name:            "Public access only",
			permissions:     []string{"public"},
			expectedFields:  []string{"id", "name"},
			forbiddenFields: []string{"email", "phone", "password", "ssn", "audit_log", "compliance_id"},
		},
		{
			name:            "Admin + PII access",
			permissions:     []string{"public", "read", "admin", "pii"},
			expectedFields:  []string{"id", "name", "email", "password", "ssn", "compliance_id"},
			forbiddenFields: []string{"phone", "audit_log"},
		},
		{
			name:            "Compliance access",
			permissions:     []string{"public", "compliance"},
			expectedFields:  []string{"id", "name", "compliance_id"},
			forbiddenFields: []string{"email", "phone", "password", "ssn", "audit_log"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Marshal with permissions using TOML
			data, err := cereal.TOML.Marshal(user, tt.permissions...)
			if err != nil {
				t.Fatalf("TOML Marshal failed: %v", err)
			}

			// Parse result to check field presence
			var result map[string]interface{}
			if err := toml.Unmarshal(data, &result); err != nil {
				t.Fatalf("Failed to parse TOML marshal result: %v", err)
			}

			// Check expected fields are present
			for _, field := range tt.expectedFields {
				if _, exists := result[field]; !exists {
					t.Errorf("Expected field '%s' to be present in TOML, but it was filtered out", field)
				}
			}

			// Check forbidden fields are absent
			for _, field := range tt.forbiddenFields {
				if _, exists := result[field]; exists {
					t.Errorf("Expected field '%s' to be filtered out of TOML, but it was present: %v", field, result[field])
				}
			}
		})
	}
}

func TestTOMLUnmarshalScoping(t *testing.T) {
	tests := []struct {
		name               string
		inputTOML          string
		permissions        []string
		expectedSetFields  map[string]interface{}
		expectedZeroFields []string
	}{
		{
			name: "Public write access - malicious payload",
			inputTOML: `id = 999
name = "Hacker"
email = "hacker@evil.com"
password = "stolen"
ssn = "000-00-0000"
audit_log = "fake audit"`,
			permissions: []string{"public", "write"},
			expectedSetFields: map[string]interface{}{
				"id":    999,
				"name":  "Hacker",
				"email": "hacker@evil.com",
			},
			expectedZeroFields: []string{"phone", "password", "ssn", "audit_log", "compliance_id"},
		},
		{
			name: "Admin + PII access",
			inputTOML: `name = "PII Admin"
password = "secret"
ssn = "987-65-4321"
compliance_id = "COMP-123"`,
			permissions: []string{"admin", "pii"},
			expectedSetFields: map[string]interface{}{
				"password":      "secret",
				"ssn":           "987-65-4321",
				"compliance_id": "COMP-123",
			},
			expectedZeroFields: []string{"name", "email", "phone", "audit_log"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var user User

			// Unmarshal with scoping using TOML
			err := cereal.TOML.Unmarshal([]byte(tt.inputTOML), &user, tt.permissions...)
			if err != nil {
				t.Fatalf("TOML Unmarshal failed: %v", err)
			}

			// Check expected set fields
			userMap := map[string]interface{}{
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

func TestTOMLRoundTrip(t *testing.T) {
	original := User{
		ID:           456,
		Name:         "Alice",
		Email:        "alice@company.com",
		Password:     "supersecret",
		SSN:          "555-55-5555",
		ComplianceID: "COMP-999",
	}

	// Marshal with limited permissions using TOML
	data, err := cereal.TOML.Marshal(original, "public", "read")
	if err != nil {
		t.Fatalf("TOML Marshal failed: %v", err)
	}

	// Should only contain: id, name, email
	var filtered map[string]interface{}
	toml.Unmarshal(data, &filtered)

	if len(filtered) != 3 {
		t.Errorf("Expected 3 fields in filtered TOML data, got %d: %v", len(filtered), filtered)
	}

	// Unmarshal back with same permissions
	var result User
	err = cereal.TOML.Unmarshal(data, &result, "public", "read")
	if err != nil {
		t.Fatalf("TOML Unmarshal failed: %v", err)
	}

	// Should match the filtered fields
	if result.ID != original.ID || result.Name != original.Name || result.Email != original.Email {
		t.Errorf("TOML round trip failed: expected ID=%d, Name=%s, Email=%s; got ID=%d, Name=%s, Email=%s",
			original.ID, original.Name, original.Email,
			result.ID, result.Name, result.Email)
	}

	// Restricted fields should be empty
	if result.Password != "" || result.SSN != "" || result.ComplianceID != "" {
		t.Errorf("Restricted fields should be empty: Password=%s, SSN=%s, ComplianceID=%s",
			result.Password, result.SSN, result.ComplianceID)
	}
}