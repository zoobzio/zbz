package main

import (
	"fmt"
	"log"
	"time"

	docula "zbz/docula/v2"
	"zbz/depot"
)

func main() {
	fmt.Println("🚀 Docula V2 - Living Documentation Demo")
	fmt.Println("========================================")

	// Create memory storage contract
	memoryContract := depot.NewMemory(map[string]interface{}{})
	
	// Populate with sample markdown files including API documentation
	sampleFiles := map[string]string{
		"api.md": `---
title: User Management API
nav_title: API Overview
category: API Reference
---

# User Management API

A comprehensive REST API for managing users in your application.

## Features

- **CRUD Operations** - Create, read, update, and delete users
- **Authentication** - Secure API key authentication
- **Validation** - Input validation and error handling
- **Pagination** - Efficient handling of large datasets

This API follows REST principles and returns JSON responses.`,

		"ListUsers.md": `---
title: List Users
category: Endpoints
tags: [users, list, pagination]
---

# List Users

Retrieve a paginated list of all users in the system.

## Query Parameters

- **limit** (integer): Number of users to return (default: 20, max: 100)
- **offset** (integer): Number of users to skip (default: 0)
- **filter** (string): Filter users by email or name

## Example Request

` + "```bash" + `
curl -H "Authorization: Bearer YOUR_API_KEY" \
     "https://api.example.com/users?limit=10&offset=0"
` + "```" + `

## Response Format

Returns an array of user objects with pagination metadata.`,

		"CreateUser.md": `---
title: Create User
category: Endpoints  
tags: [users, create, post]
---

# Create User

Create a new user account with the provided information.

## Required Fields

- **email** (string): Valid email address
- **name** (string): User's full name
- **role** (string): User role (user, admin)

## Validation Rules

- Email must be unique across all users
- Name must be at least 2 characters
- Role must be one of: user, admin

## Example Request

` + "```bash" + `
curl -X POST \
     -H "Authorization: Bearer YOUR_API_KEY" \
     -H "Content-Type: application/json" \
     -d '{"email":"john@example.com","name":"John Doe","role":"user"}' \
     https://api.example.com/users
` + "```" + `

Returns the created user object with generated ID and timestamps.`,

		"User.md": `---
title: User Model
category: Models
tags: [user, schema, model]
---

# User Model

Represents a user account in the system.

## Fields

- **id** (string): Unique identifier (UUID format)
- **email** (string): User's email address (unique)
- **name** (string): User's full name
- **role** (string): User role (user, admin)
- **created_at** (timestamp): Account creation time
- **updated_at** (timestamp): Last modification time
- **last_login** (timestamp): Last login time (nullable)

## Business Rules

- Users can only edit their own profile unless they are admins
- Admin users can manage all accounts
- Deleted users are soft-deleted (marked inactive)

## Example

` + "```json" + `
{
  "id": "usr_123456789",
  "email": "john@example.com",
  "name": "John Doe",
  "role": "user",
  "created_at": "2024-01-15T10:30:00Z",
  "updated_at": "2024-01-15T10:30:00Z",
  "last_login": "2024-01-20T14:22:00Z"
}
` + "```" + ``,
	}
	
	// Load sample files into memory storage
	for path, content := range sampleFiles {
		err := memoryContract.Set(path, []byte(content), time.Duration(0))
		if err != nil {
			log.Fatalf("Failed to load sample file %s: %v", path, err)
		}
	}

	// Create docula contract with reactive updates ENABLED
	contract := docula.DoculaContract{
		Name:            "user-api-docs",
		Description:     "User Management API Documentation",
		Storage:         memoryContract,
		ReactiveUpdates: true, // 🔄 Enable Flux integration!
		Sites: []docula.SiteConfig{
			{
				Template:    "docs",
				BasePath:    "/docs",
				Title:       "User API Documentation",
				Description: "Complete API documentation with examples",
				Features: map[string]bool{
					"search": true,
					"toc":    true,
				},
			},
		},
	}

	// Create docula service
	service := contract.Docula()

	// Load initial content
	fmt.Println("\n📖 Loading initial content...")
	if err := service.LoadContent(); err != nil {
		log.Fatal("Failed to load content:", err)
	}

	// Display loaded pages
	pages := service.ListPages()
	fmt.Printf("✅ Loaded %d documentation pages\n", len(pages))
	
	for path, page := range pages {
		fmt.Printf("   📄 %s (%s)\n", path, page.Title)
	}

	// 🎯 CONCEPT 1: Generate OpenAPI Spec with markdown enhancement
	fmt.Println("\n🔧 Generating OpenAPI Specification...")
	
	yamlSpec, err := service.GetSpecYAML()
	if err != nil {
		log.Fatal("Failed to generate YAML spec:", err)
	}
	
	fmt.Println("✅ Generated OpenAPI spec (enhanced with markdown):")
	fmt.Println("================================================")
	fmt.Println(string(yamlSpec))

	// 🎯 CONCEPT 2: Test reactive updates via Flux
	fmt.Println("\n🔄 Testing Reactive Updates...")
	fmt.Println("==============================")
	
	// Give Flux a moment to fully initialize
	time.Sleep(100 * time.Millisecond)
	
	// Simulate updating a markdown file
	updatedContent := `---
title: Create User (Updated!)
category: Endpoints  
tags: [users, create, post, updated]
---

# Create User (UPDATED VERSION!)

🎉 This content was updated via Flux reactive system!

Create a new user account with enhanced validation and security features.

## NEW: Enhanced Security

- **Rate limiting** - Prevent abuse with request throttling
- **Input sanitization** - XSS protection on all inputs
- **Audit logging** - Track all user creation events

## Required Fields

- **email** (string): Valid email address (now with enhanced validation)
- **name** (string): User's full name (supports Unicode)
- **role** (string): User role (user, admin, moderator)

This endpoint now supports additional roles and enhanced security measures!`

	fmt.Println("📝 Updating CreateUser.md content...")
	err = service.TriggerUpdate("CreateUser.md", updatedContent)
	if err != nil {
		log.Fatal("Failed to trigger update:", err)
	}

	// Give Flux time to process the change
	time.Sleep(200 * time.Millisecond)

	// 🎯 CONCEPT 3: Verify auto-regenerated spec
	fmt.Println("\n🔍 Verifying Auto-Regenerated Spec...")
	fmt.Println("=====================================")
	
	updatedYaml, err := service.GetSpecYAML()
	if err != nil {
		log.Fatal("Failed to get updated spec:", err)
	}
	
	fmt.Println("✅ Updated OpenAPI spec (with new markdown content):")
	fmt.Println("===================================================")
	fmt.Println(string(updatedYaml))

	// Test adding a completely new file
	fmt.Println("\n➕ Adding New Documentation File...")
	fmt.Println("===================================")
	
	newFileContent := `---
title: Delete User
category: Endpoints
tags: [users, delete, admin-only]
---

# Delete User

Permanently delete a user account (admin only).

## Security Note

⚠️ This is a destructive operation that cannot be undone.

## Authorization

Only admin users can delete accounts. Regular users receive a 403 Forbidden error.

## Soft vs Hard Delete

- **Soft delete** (default): Mark user as inactive
- **Hard delete** (query param): Permanently remove all data`

	err = service.TriggerUpdate("DeleteUser.md", newFileContent)
	if err != nil {
		log.Fatal("Failed to add new file:", err)
	}
	
	// Give Flux time to process
	time.Sleep(200 * time.Millisecond)
	
	// Show updated pages
	updatedPages := service.ListPages()
	fmt.Printf("✅ Now have %d documentation pages (was %d)\n", len(updatedPages), len(pages))
	
	fmt.Println("\n🎉 Living Documentation Demo Complete!")
	fmt.Println("=====================================")
	fmt.Println("\nThis demonstrated:")
	fmt.Println("✅ Markdown-enhanced OpenAPI spec generation")
	fmt.Println("✅ Flux reactive updates for live documentation")
	fmt.Println("✅ Automatic spec regeneration on content changes")
	fmt.Println("✅ Real-time content addition and modification")
	fmt.Println("\n🚀 Ready for production deployment!")
}