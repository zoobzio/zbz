package main

import (
	"fmt"
	"log"
	"zbz/cereal"
)

// User represents a user with different permission-scoped fields
type User struct {
	ID           int    `json:"id"`                                         // No scope = always visible
	Name         string `json:"name" scope:"public"`                        // Public info
	Email        string `json:"email" scope:"read,write"`                   // read OR write permission
	Phone        string `json:"phone" scope:"self"`                         // Only the user themselves
	Password     string `json:"password" scope:"admin"`                     // Admin only
	SSN          string `json:"ssn" scope:"admin+pii"`                      // Admin AND pii permission
	AuditLog     string `json:"audit_log" scope:"admin+security+executive"` // All three required
	ComplianceID string `json:"compliance_id" scope:"compliance,admin+pii"` // compliance OR (admin+pii)
}

func json() {
	// Create a user with all fields populated
	user := User{
		ID:           123,
		Name:         "John Doe",
		Email:        "john@example.com",
		Phone:        "+1-555-0123",
		Password:     "secret123",
		SSN:          "123-45-6789",
		AuditLog:     "User created on 2024-01-01",
		ComplianceID: "COMP-2024-001",
	}

	fmt.Println("=== JSON Serialization with Scoping ===\n")

	// Example 1: No permissions - only unscoped fields visible
	fmt.Println("1. No permissions (anonymous user):")
	data, _ := cereal.JSON.Marshal(user)
	fmt.Printf("   %s\n\n", string(data))

	// Example 2: Public permission
	fmt.Println("2. Public permission:")
	data, _ = cereal.JSON.Marshal(user, "public")
	fmt.Printf("   %s\n\n", string(data))

	// Example 3: Multiple permissions
	fmt.Println("3. Public + read permissions:")
	data, _ = cereal.JSON.Marshal(user, "public", "read")
	fmt.Printf("   %s\n\n", string(data))

	// Example 4: Admin without PII
	fmt.Println("4. Admin permission (but no PII access):")
	data, _ = cereal.JSON.Marshal(user, "public", "admin")
	fmt.Printf("   %s\n\n", string(data))

	// Example 5: Admin with PII
	fmt.Println("5. Admin + PII permissions:")
	data, _ = cereal.JSON.Marshal(user, "public", "admin", "pii")
	fmt.Printf("   %s\n\n", string(data))

	// Example 6: Full executive access
	fmt.Println("6. Full executive access:")
	data, _ = cereal.JSON.Marshal(user, "public", "admin", "security", "executive")
	fmt.Printf("   %s\n\n", string(data))

	// Example 7: Compliance access
	fmt.Println("7. Compliance officer access:")
	data, _ = cereal.JSON.Marshal(user, "public", "compliance")
	fmt.Printf("   %s\n\n", string(data))

	fmt.Println("=== JSON Deserialization with Scoping ===\n")

	// Example 8: Unmarshaling with limited permissions
	maliciousJSON := `{
		"id": 999,
		"name": "Hacker",
		"email": "hacker@evil.com",
		"password": "stolen",
		"ssn": "000-00-0000",
		"audit_log": "Fake audit entry"
	}`

	var newUser User

	fmt.Println("8. Unmarshaling malicious JSON with only 'public' permission:")
	fmt.Printf("   Input: %s\n", maliciousJSON)

	err := cereal.JSON.Unmarshal([]byte(maliciousJSON), &newUser, "public")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("   Result: ID=%d, Name=%s, Email=%s, Password=%s, SSN=%s\n",
		newUser.ID, newUser.Name, newUser.Email, newUser.Password, newUser.SSN)
	fmt.Println("   Note: Only 'id' and 'name' were set; restricted fields remain empty")

	// Example 9: Round-trip with scoping
	fmt.Println("\n9. Round-trip example:")

	// Marshal with limited permissions
	limitedData, _ := cereal.JSON.Marshal(user, "public", "read")
	fmt.Printf("   Marshaled with 'public,read': %s\n", string(limitedData))

	// Unmarshal back with same permissions
	var roundTripUser User
	cereal.JSON.Unmarshal(limitedData, &roundTripUser, "public", "read")

	fmt.Printf("   After round-trip: ID=%d, Name=%s, Email=%s, Password=%s\n",
		roundTripUser.ID, roundTripUser.Name, roundTripUser.Email, roundTripUser.Password)
}
