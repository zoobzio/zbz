package cereal

import (
	"fmt"
	"strings"

	"zbz/cereal"
)

// JsonDemo demonstrates cereal JSON scoping with user data
func JsonDemo() {
	fmt.Println("📦 ZBZ Framework cereal JSON Scoping Demo")
	fmt.Println(strings.Repeat("=", 50))
	
	// Step 1: Define user data structure
	fmt.Println("\n👤 Step 1: User data structure with scoping")
	
	type User struct {
		ID       int    `json:"id" scope:"public"`
		Username string `json:"username" scope:"public"`
		Email    string `json:"email" scope:"private"`
		Phone    string `json:"phone" scope:"admin"`
		Password string `json:"-" scope:"never"`
		Role     string `json:"role" scope:"admin"`
		Profile  struct {
			Name     string `json:"name" scope:"public"`
			Bio      string `json:"bio" scope:"public"`
			Location string `json:"location" scope:"private"`
		} `json:"profile"`
	}
	
	user := User{
		ID:       12345,
		Username: "johndoe",
		Email:    "john.doe@example.com",
		Phone:    "555-123-4567",
		Password: "super-secret-password",
		Role:     "user",
	}
	user.Profile.Name = "John Doe"
	user.Profile.Bio = "Software developer and coffee enthusiast"
	user.Profile.Location = "San Francisco, CA"
	
	// Step 2: Public scope (guest users)
	fmt.Println("\n🌐 Step 2: Public scope - what guests see")
	publicJSON, err := cereal.JSON.Marshal(user, "public")
	if err != nil {
		fmt.Printf("❌ Error: %v\n", err)
		return
	}
	fmt.Printf("Public JSON:\n%s\n", string(publicJSON))
	
	// Step 3: Private scope (authenticated users)  
	fmt.Println("\n🔒 Step 3: Private scope - what authenticated users see")
	privateJSON, err := cereal.JSON.Marshal(user, "private")
	if err != nil {
		fmt.Printf("❌ Error: %v\n", err)
		return
	}
	fmt.Printf("Private JSON:\n%s\n", string(privateJSON))
	
	// Step 4: Admin scope (administrators)
	fmt.Println("\n👑 Step 4: Admin scope - what administrators see")
	adminJSON, err := cereal.JSON.Marshal(user, "admin")
	if err != nil {
		fmt.Printf("❌ Error: %v\n", err)
		return
	}
	fmt.Printf("Admin JSON:\n%s\n", string(adminJSON))
	
	// Step 5: Multiple users list
	fmt.Println("\n📋 Step 5: User list with scoping")
	users := []User{
		{
			ID: 1, Username: "alice", Email: "alice@example.com", 
			Phone: "555-111-1111", Role: "admin",
		},
		{
			ID: 2, Username: "bob", Email: "bob@example.com",
			Phone: "555-222-2222", Role: "user", 
		},
		{
			ID: 3, Username: "charlie", Email: "charlie@example.com",
			Phone: "555-333-3333", Role: "moderator",
		},
	}
	
	for i := range users {
		users[i].Profile.Name = fmt.Sprintf("User %d", users[i].ID)
		users[i].Profile.Bio = "Demo user account"
		users[i].Profile.Location = "Demo City"
	}
	
	fmt.Println("\n   Public view of user list:")
	userListJSON, err := cereal.JSON.Marshal(users, "public")
	if err != nil {
		fmt.Printf("❌ Error: %v\n", err)
		return
	}
	fmt.Printf("%s\n", string(userListJSON))
	
	// Step 6: Deserialization with scoping
	fmt.Println("\n⬅️  Step 6: Deserializing with scope validation")
	
	// Try to deserialize admin data with public scope
	var publicUser User
	err = cereal.JSON.Unmarshal(adminJSON, &publicUser, "public")
	if err != nil {
		fmt.Printf("✅ Correctly blocked admin data from public scope: %v\n", err)
	} else {
		fmt.Printf("❌ Security issue: admin data leaked to public scope\n")
	}
	
	// Deserialize with correct scope
	var adminUser User
	err = cereal.JSON.Unmarshal(adminJSON, &adminUser, "admin")
	if err != nil {
		fmt.Printf("❌ Error deserializing admin data: %v\n", err)
	} else {
		fmt.Printf("✅ Successfully deserialized with admin scope\n")
		fmt.Printf("   Admin user role: %s\n", adminUser.Role)
	}
	
	fmt.Println("\n✅ JSON scoping demo complete!")
	fmt.Println("🔐 Data exposed based on user permissions")
}