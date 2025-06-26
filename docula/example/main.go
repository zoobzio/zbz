package main

import (
	"fmt"
	"log"
	"time"

	docula "zbz/docula/v2"
	"zbz/hodor"
)

func main() {
	fmt.Println("🚀 Docula V2 Example")
	fmt.Println("====================")

	// Create memory storage contract
	memoryContract := hodor.NewMemory(map[string]interface{}{})
	
	// Populate with sample markdown files
	sampleFiles := map[string]string{
		"index.md": `---
title: Welcome to Our API
nav_title: Home
nav_order: 1
category: Overview
tags: [welcome, intro]
---

# Welcome to Our API

This is the main documentation page for our awesome API.

## Getting Started

To get started with our API, you'll need to:

1. **Sign up** for an account
2. **Generate** an API key
3. **Make** your first request

## Features

Our API provides:

- **Fast responses** - Sub-100ms average
- **Reliable uptime** - 99.9% SLA
- **Great docs** - You're reading them!

### Code Example

` + "```go" + `
func main() {
    fmt.Println("Hello API!")
}
` + "```" + `

Ready to get started? Check out our [Authentication Guide](auth.md).`,

				"auth.md": `---
title: Authentication Guide
nav_title: Authentication
nav_order: 2
category: Getting Started
tags: [auth, security, api-keys]
---

# Authentication Guide

Learn how to authenticate with our API using API keys.

## API Key Authentication

All API requests must include an API key in the header:

` + "```bash" + `
curl -H "Authorization: Bearer YOUR_API_KEY" \
     https://api.example.com/users
` + "```" + `

## Security Best Practices

- **Never** commit API keys to version control
- **Rotate** keys regularly
- **Use** environment variables for storage

> **Important:** Keep your API keys secure and never share them publicly.`,

				"users.md": `---
title: User Management
nav_title: Users
nav_order: 3
category: API Reference
tags: [users, crud, endpoints]
---

# User Management

Manage users in your application with these endpoints.

## List Users

Get a paginated list of all users.

` + "```http" + `
GET /api/users
Authorization: Bearer YOUR_API_KEY
` + "```" + `

### Response

` + "```json" + `
{
  "users": [
    {
      "id": "usr_123",
      "email": "john@example.com",
      "name": "John Doe",
      "created_at": "2024-01-15T10:30:00Z"
    }
  ],
  "pagination": {
    "page": 1,
    "total": 42
  }
}
` + "```" + `

## Create User

Create a new user account.

` + "```http" + `
POST /api/users
Authorization: Bearer YOUR_API_KEY
Content-Type: application/json
` + "```" + `

### Request Body

` + "```json" + `
{
  "email": "jane@example.com",
  "name": "Jane Smith",
  "role": "user"
}
` + "```" + ``,

				"blog/announcing-v2.md": `---
title: Announcing Docula V2
slug: announcing-docula-v2
date: 2024-01-15
author: Development Team
tags: [announcement, release, features]
excerpt: We're excited to announce Docula V2 with living documentation and reactive updates!
---

# Announcing Docula V2

We're thrilled to announce the release of **Docula V2**, our most ambitious update yet!

## What's New

### 🔄 Living Documentation
Your docs now update automatically when you change markdown files in cloud storage. No more manual deploys!

### ☁️ Cloud-Native Storage
Store your documentation in S3, GCS, MinIO, or any Hodor-supported provider.

### ⚡ Reactive Updates
Thanks to our Flux integration, changes appear instantly across all documentation sites.

### 🎨 Multiple Site Templates
- **Documentation sites** for technical docs
- **Blog templates** for announcements
- **Knowledge bases** for support content

## Getting Started

` + "```yaml" + `
# docula.yaml
name: "my-docs"
storage:
  provider: "s3"
  config:
    bucket: "my-docs-bucket"
reactive_updates: true
` + "```" + `

Ready to upgrade? Check out our [migration guide](/docs/migration).

---

*Happy documenting!*  
The Docula Team`,
	}
	
	// Load sample files into memory storage
	for path, content := range sampleFiles {
		err := memoryContract.Set(path, []byte(content), time.Duration(0))
		if err != nil {
			log.Fatalf("Failed to load sample file %s: %v", path, err)
		}
	}

	// Create docula contract
	contract := docula.DoculaContract{
		Name:        "example-docs",
		Description: "Example documentation site",
		Storage:     memoryContract,
		ReactiveUpdates: false, // No flux for this simple example
		Sites: []docula.SiteConfig{
			{
				Template:    "docs",
				BasePath:    "/docs",
				Title:       "Example API Documentation",
				Description: "Comprehensive API documentation and guides",
				Features: map[string]bool{
					"search": true,
					"toc":    true,
				},
			},
			{
				Template:    "blog",
				BasePath:    "/blog",
				Title:       "Example Blog",
				Description: "Latest updates and announcements",
				Features: map[string]bool{
					"rss": true,
				},
			},
		},
	}

	// Create docula service
	service := contract.Docula()

	// Load content from storage
	fmt.Println("\n📖 Loading content...")
	if err := service.LoadContent(); err != nil {
		log.Fatal("Failed to load content:", err)
	}

	// Display loaded pages
	fmt.Println("\n📄 Loaded Pages:")
	fmt.Println("================")

	pages := service.ListPages()
	for path, page := range pages {
		fmt.Printf("\n🔸 %s\n", path)
		fmt.Printf("   Title: %s\n", page.Title)
		fmt.Printf("   Category: %s\n", page.Category)
		fmt.Printf("   Tags: %v\n", page.Tags)
		if len(page.TOC) > 0 {
			fmt.Printf("   TOC: %d headings\n", len(page.TOC))
		}
	}

	// Show which pages were loaded
	if len(pages) == 0 {
		fmt.Println("   No pages were loaded!")
		return
	}
	
	// Try to render the first available page
	var firstPage string
	for path := range pages {
		firstPage = path
		break
	}
	
	fmt.Printf("\n🎨 Rendering '%s':\n", firstPage)
	fmt.Println("=======================")

	html, err := service.RenderPageHTML(firstPage)
	if err != nil {
		log.Fatal("Failed to render page:", err)
	}

	fmt.Println(html)

	fmt.Println("\n✅ Docula V2 Example Complete!")
	fmt.Println("\nThis demonstrates:")
	fmt.Println("- ✅ Markdown processing with frontmatter")
	fmt.Println("- ✅ HTML sanitization (goldmark + bluemonday)")
	fmt.Println("- ✅ Memory storage integration")
	fmt.Println("- ✅ Table of contents extraction")
	fmt.Println("- ✅ Multi-site configuration")
	fmt.Println("\nNext steps: Add Flux for reactive updates!")
}