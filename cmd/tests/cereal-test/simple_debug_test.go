package cereal_tests

import (
	"fmt"
	"testing"
	"zbz/cereal"
)

func TestSimpleDebug(t *testing.T) {
	user := SimpleUser{
		ID:       123,
		Name:     "John",
		Password: "secret",
	}

	// Test the scoping logic through marshaling
	data, _ := cereal.JSON.Marshal(user)
	fmt.Printf("No permissions JSON: %s\n", string(data))

	data, _ = cereal.JSON.Marshal(user, "public")
	fmt.Printf("Public permission JSON: %s\n", string(data))
}
