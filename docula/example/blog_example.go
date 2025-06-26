package main

import (
	"log"
	"os"
	"path/filepath"
	"time"

	docula "zbz/docula/v2"
	"zbz/hodor"
	"zbz/zlog"
)

func main() {
	zlog.Info("Starting Docula V2 Blog Example")

	// Create memory storage contract
	memoryContract := hodor.NewMemory(map[string]interface{}{})
	
	// Sample blog posts with proper frontmatter
	blogPosts := map[string]string{
		"blog/announcing-docula-v2.md": `---
title: Announcing Docula V2 - Living Documentation
slug: announcing-docula-v2
date: 2024-01-15
author: ZBZ Development Team
author_email: dev@zbz.dev
tags: [announcement, release, docula, v2]
excerpt: We're excited to announce Docula V2 with reactive updates, cloud storage, and beautiful templates!
featured_image: /images/docula-v2-hero.png
draft: false
---

# Announcing Docula V2 - Living Documentation

We're thrilled to announce the release of **Docula V2**, the most significant update to our documentation platform yet!

## 🚀 What's New

### ⚡ Reactive Updates
Your documentation now updates **automatically** when you change markdown files in cloud storage. No more manual builds or deployments!

### ☁️ Cloud-Native Storage
Store your documentation in **S3, GCS, MinIO**, or any Hodor-supported provider. Your docs live in the cloud and sync instantly.

### 🎨 Beautiful Templates
Choose from **multiple site templates**:
- **Documentation sites** for technical docs
- **Blog templates** for announcements  
- **Knowledge bases** for support content

### 🔧 Enhanced OpenAPI Integration
Your API documentation is now **automatically enhanced** with markdown content, creating rich, comprehensive specs.

## 💡 The Technology

Docula V2 is powered by our **Flux reactive system** and **Hodor cloud storage**:

` + "```go\n" + `// Enable reactive updates with one line
contract := docula.DoculaContract{
    Storage: cloudStorage,  // Reactive updates enabled automatically!
}
` + "```" + `

## 🎯 Perfect for Teams

- **Developers** write docs in markdown alongside code
- **Technical writers** can edit content directly in cloud storage
- **Product managers** can update specs without deployments
- **Support teams** can maintain knowledge bases effortlessly

## 📈 What This Means

Docula V2 transforms documentation from a **static artifact** into a **living, breathing part** of your development workflow.

**Ready to upgrade?** Check out our [migration guide](/docs/migration) and experience the future of documentation!

---

*Happy documenting!*  
**The ZBZ Team** 🚀`,

		"blog/flux-collection-watching.md": `---
title: Flux Collection Watching - A Game Changer
slug: flux-collection-watching
date: 2024-01-20
author: ZBZ Architecture Team
tags: [flux, architecture, reactive, collections]
excerpt: Discover how Flux collection watching enables powerful reactive patterns across the entire zbz ecosystem.
draft: false
---

# Flux Collection Watching - A Game Changer

Today we're excited to share a **foundational new capability** in Flux: **Collection Watching**.

## 🎯 The Problem

Traditional reactive systems watch **individual files**. But what about watching **groups of related files**?

- Documentation systems need to watch **all markdown files**
- Configuration systems need to watch **all config files**  
- Asset pipelines need to watch **all images and stylesheets**

## ✨ The Solution

**Flux Collection Watching** lets you reactively monitor entire file collections with simple patterns:

` + "```go\n" + `// Watch all markdown files
flux.SyncCollection(storage, "*.md", func(old, new map[string][]byte) {
    // Automatically called when ANY .md file changes
})

// Watch all config files
flux.SyncCollection(storage, "config/*.yaml", configHandler)

// Watch plugin directories
flux.SyncCollection(storage, "plugins/*/", pluginHandler)
` + "```" + `

## 🔥 Real-World Impact

### Documentation Systems
Docula V2 uses collection watching to **automatically regenerate** documentation when any content changes.

### Configuration Management
Watch entire config directories and **hot-reload** application settings.

### Asset Pipelines
Monitor asset directories and **automatically rebuild** CSS/JS bundles.

## 🏗️ How It Works

1. **Pattern Matching**: Supports glob patterns, extensions, and prefixes
2. **Change Aggregation**: Batches multiple rapid changes into single updates
3. **Smart Diffing**: Only triggers callbacks when content actually changes
4. **Resource Management**: Automatic subscription cleanup and memory management

## 📊 Performance Benefits

- **Throttled Updates**: Prevents excessive callbacks during bulk changes
- **Efficient Diffing**: Only processes files that actually changed
- **Concurrent Safety**: Thread-safe with proper mutex handling

## 🚀 The Future

Collection watching opens up **entirely new possibilities**:

- **Live-reloading development environments**
- **Hot-swappable plugin systems**
- **Real-time collaborative editing**
- **Instant deployment pipelines**

This pattern will become **foundational** across the zbz ecosystem, powering the next generation of reactive applications.

---

*Building the reactive future, one collection at a time.*  
**The ZBZ Architecture Team** ⚡`,

		"blog/go-htmx-revolution.md": `---
title: The Go + HTMX Revolution
slug: go-htmx-revolution  
date: 2024-01-25
author: ZBZ Frontend Team
tags: [go, htmx, frontend, ssr, performance]
excerpt: Why we chose Go templates + HTMX over React for our documentation platform, and why you should consider it too.
draft: false
---

# The Go + HTMX Revolution

At ZBZ, we made a **controversial decision**: we chose **Go + HTMX** over React for our documentation platform. Here's why.

## 🎯 The Problem with Modern Frontend

Modern frontend development has become **unnecessarily complex**:

- **Build tools**: Webpack, Vite, Rollup, Parcel...
- **State management**: Redux, Zustand, Jotai, MobX...
- **Component libraries**: Material-UI, Ant Design, Chakra...
- **Meta frameworks**: Next.js, Nuxt, SvelteKit...

For a **documentation site**, this is **massive overkill**.

## ✨ The Go + HTMX Approach

### **Server-Side Rendering (SSR)**
```go
// Generate HTML on the server
func (tr *TemplateRenderer) RenderBlogIndex() string {
    return tr.renderTemplate(blogTemplate, data)
}
```

### **Interactive Islands with HTMX**
```html
<!-- Live search without JavaScript -->
<input hx-get="/search" 
       hx-trigger="keyup changed delay:300ms" 
       hx-target="#results">
```

### **Zero Build Step**
- No `npm install`
- No build process
- No node_modules
- Just **go run main.go**

## 📊 Performance Comparison

| Metric | React SPA | Go + HTMX |
|--------|-----------|-----------|
| Initial Load | 2.1s | 0.3s |
| Bundle Size | 847kb | 12kb |
| Build Time | 45s | 0s |
| Memory Usage | 156MB | 23MB |

## 🔥 Developer Experience Benefits

### **Instant Feedback**
```bash
# Make change, refresh browser - done!
go run main.go
```

### **Simple Debugging**
- No source maps
- No transpilation
- Pure HTML in DevTools

### **Easy Deployment**
```bash
# Single binary deployment
go build -o docs-server
./docs-server
```

## 🎨 When Go + HTMX Shines

Perfect for:
- **Documentation sites**
- **Admin panels**
- **Content management systems**
- **Internal tools**
- **Marketing websites**

## 🚫 When to Avoid

Not ideal for:
- **Complex interactive applications**
- **Real-time collaborative tools**
- **Heavy client-side logic**
- **Offline-first applications**

## 🚀 The Future is Simple

We believe the frontend world is moving **back to simplicity**:

- **Server-side rendering** is making a comeback
- **Progressive enhancement** over client-side hydration
- **HTML-first** development

HTMX gives you **90% of the interactivity** with **10% of the complexity**.

## 📈 Results

Since adopting Go + HTMX for Docula:

- **5x faster** initial page loads
- **10x smaller** bundle sizes  
- **Zero build issues** in production
- **Happier developers** (seriously!)

Ready to simplify your frontend? **Try Go + HTMX today.**

---

*Simplicity is the ultimate sophistication.*  
**The ZBZ Frontend Team** 🎨`,
	}
	
	// Load blog posts into storage
	for path, content := range blogPosts {
		err := memoryContract.Set(path, []byte(content), time.Duration(0))
		if err != nil {
			log.Fatalf("Failed to load blog post %s: %v", path, err)
		}
	}

	// Create docula contract for blog
	contract := docula.DoculaContract{
		Name:        "zbz-blog",
		Description: "ZBZ Development Blog - Latest updates and insights",
		Storage:     memoryContract,
		Sites: []docula.SiteConfig{
			{
				Template:    "blog",
				BasePath:    "/blog",
				Title:       "ZBZ Development Blog",
				Description: "Latest updates, insights, and technical deep-dives from the ZBZ team",
				Features: map[string]bool{
					"rss":    true,
					"search": true,
					"tags":   true,
				},
				Theme: map[string]string{
					"primary_color":   "#0066cc",
					"secondary_color": "#333333",
					"font_family":     "-apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif",
				},
			},
		},
	}

	// Create docula service
	service := contract.Docula()

	// Load content
	zlog.Info("Loading blog content")
	if err := service.LoadContent(); err != nil {
		log.Fatal("Failed to load content:", err)
	}

	// Create template renderer
	renderer := docula.NewTemplateRenderer(service)

	// Create output directory
	outputDir := "./output"
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		log.Fatal("Failed to create output directory:", err)
	}

	// Generate blog index page
	zlog.Info("Generating blog index page")
	blogIndex, err := renderer.RenderBlogIndex(contract.Sites[0])
	if err != nil {
		log.Fatal("Failed to render blog index:", err)
	}

	// Write blog index
	indexPath := filepath.Join(outputDir, "index.html")
	if err := os.WriteFile(indexPath, []byte(blogIndex), 0644); err != nil {
		log.Fatal("Failed to write blog index:", err)
	}

	// Generate individual blog posts
	posts := renderer.GetBlogPosts()
	for _, post := range posts {
		if post.Draft {
			continue // Skip drafts
		}
		
		zlog.Info("Generating blog post", zlog.String("title", post.Title))
		
		postHTML, err := renderer.RenderBlogPost(post.Slug, contract.Sites[0])
		if err != nil {
			zlog.Warn("Failed to render blog post", 
				zlog.String("slug", post.Slug), 
				zlog.Err(err))
			continue
		}

		// Create posts directory
		postsDir := filepath.Join(outputDir, "posts")
		if err := os.MkdirAll(postsDir, 0755); err != nil {
			log.Fatal("Failed to create posts directory:", err)
		}

		// Write individual post
		postPath := filepath.Join(postsDir, post.Slug+".html")
		if err := os.WriteFile(postPath, []byte(postHTML), 0644); err != nil {
			zlog.Warn("Failed to write blog post", 
				zlog.String("path", postPath), 
				zlog.Err(err))
			continue
		}
	}

	// Test reactive updates
	zlog.Info("Testing reactive updates")
	
	// Add a new blog post dynamically
	newPost := `---
title: Real-Time Blog Updates!
slug: real-time-updates
date: 2024-01-26
author: ZBZ Demo
tags: [demo, reactive, live]
excerpt: This post was added via reactive updates!
draft: false
---

# Real-Time Blog Updates!

🎉 **This post was just added via Flux reactive updates!**

This demonstrates how Docula V2 can:

- Add new blog posts instantly
- Update existing content live
- Regenerate the entire site automatically

**No restarts. No rebuilds. Just pure reactive magic.**

## The Power of Living Documentation

When you change content in cloud storage:

1. **Flux detects the change** in milliseconds
2. **Docula processes** the new markdown
3. **Templates regenerate** automatically  
4. **Sites update** in real-time

This is the **future of content management**.

---

*Added live via reactive updates!* ⚡`

	err = service.TriggerUpdate("blog/real-time-updates.md", newPost)
	if err != nil {
		log.Fatal("Failed to add new post:", err)
	}

	// Give Flux time to process
	time.Sleep(300 * time.Millisecond)

	// Regenerate index with new post
	zlog.Info("Regenerating blog index with new post")
	updatedIndex, err := renderer.RenderBlogIndex(contract.Sites[0])
	if err != nil {
		log.Fatal("Failed to render updated blog index:", err)
	}

	// Write updated index
	if err := os.WriteFile(indexPath, []byte(updatedIndex), 0644); err != nil {
		log.Fatal("Failed to write updated blog index:", err)
	}

	// Generate the new post
	newPostHTML, err := renderer.RenderBlogPost("real-time-updates", contract.Sites[0])
	if err != nil {
		log.Fatal("Failed to render new blog post:", err)
	}

	newPostPath := filepath.Join(outputDir, "posts", "real-time-updates.html")
	if err := os.WriteFile(newPostPath, []byte(newPostHTML), 0644); err != nil {
		log.Fatal("Failed to write new blog post:", err)
	}

	// Get absolute path for display
	absPath, _ := filepath.Abs(outputDir)

	zlog.Info("Blog generation complete!", 
		zlog.String("output_directory", absPath),
		zlog.Int("total_posts", len(posts)+1))

	// Display results
	println("\n🎉 Blog Generation Complete!")
	println("===========================")
	println("Generated files:")
	println("📄 " + filepath.Join(absPath, "index.html") + " (Blog Index)")
	println("📄 " + filepath.Join(absPath, "posts/announcing-docula-v2.html"))
	println("📄 " + filepath.Join(absPath, "posts/flux-collection-watching.html"))
	println("📄 " + filepath.Join(absPath, "posts/go-htmx-revolution.html"))
	println("📄 " + filepath.Join(absPath, "posts/real-time-updates.html") + " (Added via Flux!)")
	println("\n🌐 Open in browser:")
	println("file://" + filepath.Join(absPath, "index.html"))

	// Cleanup
	service.Stop()
}