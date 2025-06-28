package cereal_tests

import (
	"fmt"
	"testing"
	"zbz/cereal"
)

type SimpleUser struct {
	ID       int    `json:"id"`                     // No scope
	Name     string `json:"name" scope:"public"`    // Simple scope
	Password string `json:"password" scope:"admin"` // Admin only
}

func TestDebugScoping(t *testing.T) {
	user := SimpleUser{
		ID:       123,
		Name:     "John",
		Password: "secret",
	}

	fmt.Printf("Original user: %+v\n", user)

	// Test with no permissions - should only include ID (no scope tag)
	data, err := cereal.JSON.Marshal(user)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}
	fmt.Printf("No permissions JSON: %s\n", string(data))

	// Test with public permission - should include ID and Name
	data, err = cereal.JSON.Marshal(user, "public")
	if err != nil {
		t.Fatalf("Marshal with public failed: %v", err)
	}
	fmt.Printf("Public permission JSON: %s\n", string(data))

	// Test admin permission only - should include ID and Password (but NOT Name since Name requires "public")
	data, err = cereal.JSON.Marshal(user, "admin")
	if err != nil {
		t.Fatalf("Marshal with admin failed: %v", err)
	}
	fmt.Printf("Admin only permission JSON: %s\n", string(data))

	// Test admin + public permissions - should include all fields
	data, err = cereal.JSON.Marshal(user, "admin", "public")
	if err != nil {
		t.Fatalf("Marshal with admin+public failed: %v", err)
	}
	fmt.Printf("Admin + public permission JSON: %s\n", string(data))
}
