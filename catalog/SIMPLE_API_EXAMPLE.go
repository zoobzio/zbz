package catalog

// Example showing the dramatically simplified catalog API

type User struct {
	ID       string `json:"id" scope:"user:read,admin:read"`
	Email    string `json:"email" scope:"user:read,admin:read"`
	Password string `json:"-" scope:"admin:read" encrypt:"pii"`
	Profile  struct {
		Name string `json:"name" validate:"required"`
		Age  int    `json:"age" validate:"min=0,max=120"`
	} `json:"profile"`
}

func ExampleSimplifiedAPI() {
	// ✅ The ONLY way to get metadata - always works, handles everything internally
	metadata := Select[User]()
	
	// Everything you need is in the metadata object
	_ = metadata.TypeName        // "User"
	fields := metadata.Fields    // All field metadata
	_ = metadata.Functions       // Method metadata
	
	// Extract specific information from metadata
	var scopes []string
	for _, field := range fields {
		scopes = append(scopes, field.Scopes...)
	}
	
	// Type discovery
	allTypes := Browse() // []string{"User", "Order", ...}
	
	// Type name extraction
	userTypeName := GetTypeName[User]() // "User"
	
	// That's it! No complex APIs, no confusion about cache state
	// Select[T]() does everything you need
	_ = metadata
	_ = scopes
	_ = allTypes
	_ = userTypeName
}

// What we REMOVED:
// ❌ GetFields[T]()
// ❌ GetScopes[T]()
// ❌ GetEncryptionFields[T]()
// ❌ GetValidationFields[T]()
// ❌ GetRedactionRules[T]()
// ❌ Wrap[T]()
// ❌ HasConvention[T]()
// ❌ getByTypeName()
// ❌ listRegisteredTypes()
// ❌ ensureMetadata()
// ❌ Container[T]

// What remains:
// ✅ Select[T]() - gets complete metadata, handles caching
// ✅ Browse() - lists all registered types
// ✅ GetTypeName[T]() - extracts type name