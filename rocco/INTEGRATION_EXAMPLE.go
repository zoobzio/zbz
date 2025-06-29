package rocco

import (
	"net/http"

	"zbz/core"
)

// Example showing the beautifully simple rocco + core integration

type User struct {
	ID    string `json:"id" scope:"user:read,admin:read"`
	Email string `json:"email" scope:"user:read,admin:read"`
	Name  string `json:"name" scope:"user:read,admin:read"`
}

func ExampleRoccoCoreIntegration() {
	// 1. Register a core (this publishes API contracts automatically)
	userCore := core.NewCore[User]()
	core.RegisterCore(userCore) // Auto-publishes /api/users/{id}, /api/users, etc.

	// 2. Set up HTTP server with rocco middleware
	mux := http.NewServeMux()
	
	// Add rocco's API middleware - it handles ALL /api/* requests automatically
	handler := APIMiddleware()(mux)
	
	// 3. That's it! The following endpoints now work automatically:
	// GET /api/users/{id}    - Gets single user, applies security filtering
	// GET /api/users         - Lists users, applies security filtering  
	// POST /api/users        - Creates user (TODO: implement body parsing)
	// PUT /api/users/{id}    - Updates user (TODO: implement body parsing)
	// DELETE /api/users/{id} - Deletes user
	
	// How it works:
	// 1. Request comes in: GET /api/users/123
	// 2. Rocco looks up contract: core.GetAPIContract("GET", "/api/users/123")
	// 3. Rocco extracts params: {"id": "123"}
	// 4. Rocco builds ResourceURI: "db://users/123" 
	// 5. Rocco calls core: coreService.GetByURI(ctx, resourceURI)
	// 6. Rocco applies security filtering with cereal
	// 7. Rocco returns JSON response
	
	// Start server
	http.ListenAndServe(":8080", handler)
}

// What makes this integration beautiful:
// ✅ Zero configuration - just register a core, get full REST API
// ✅ Automatic security - rocco filters data based on user permissions
// ✅ Type safety - core publishes contracts, rocco uses them
// ✅ Clean separation - rocco handles HTTP, core handles business logic
// ✅ Contract-driven - core defines what endpoints exist and how they work