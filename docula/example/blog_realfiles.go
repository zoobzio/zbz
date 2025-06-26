package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	docula "zbz/docula/v2"
	"zbz/hodor"
	"zbz/zlog"
)

func main() {
	zlog.Info("Starting Docula V2 Blog Example with Real File Watching")

	// Create file system storage pointing to our blog content directory
	contentDir := "./blog-content"
	fileStorage, err := hodor.NewFileSystem(contentDir)
	if err != nil {
		log.Fatal("Failed to create file system storage:", err)
	}

	// Create docula contract for blog
	contract := docula.DoculaContract{
		Name:        "zbz-blog",
		Description: "ZBZ Development Blog with real file watching",
		Storage:     fileStorage, // Using file system storage!
		Sites: []docula.SiteConfig{
			{
				Template:    "blog",
				BasePath:    "/blog",
				Title:       "ZBZ Development Blog",
				Description: "Latest updates with live file watching",
				Features: map[string]bool{
					"rss":    true,
					"search": true,
					"tags":   true,
				},
			},
		},
	}

	// Create docula service - this will automatically enable reactive updates!
	service := contract.Docula()

	// Load content from files
	zlog.Info("Loading blog content from files")
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

	// Generate initial blog
	generateBlog := func(suffix string) {
		zlog.Info("Generating blog", zlog.String("suffix", suffix))
		
		// Generate blog index
		blogIndex, err := renderer.RenderBlogIndex(contract.Sites[0])
		if err != nil {
			log.Fatal("Failed to render blog index:", err)
		}

		// Write blog index
		indexPath := filepath.Join(outputDir, "index"+suffix+".html")
		if err := os.WriteFile(indexPath, []byte(blogIndex), 0644); err != nil {
			log.Fatal("Failed to write blog index:", err)
		}

		// Generate individual posts
		posts := renderer.GetBlogPosts()
		postsDir := filepath.Join(outputDir, "posts")
		if err := os.MkdirAll(postsDir, 0755); err != nil {
			log.Fatal("Failed to create posts directory:", err)
		}

		for _, post := range posts {
			if post.Draft {
				continue
			}
			
			postHTML, err := renderer.RenderBlogPost(post.Slug, contract.Sites[0])
			if err != nil {
				zlog.Warn("Failed to render blog post", 
					zlog.String("slug", post.Slug), 
					zlog.Err(err))
				continue
			}

			postPath := filepath.Join(postsDir, post.Slug+suffix+".html")
			if err := os.WriteFile(postPath, []byte(postHTML), 0644); err != nil {
				zlog.Warn("Failed to write blog post", 
					zlog.String("path", postPath), 
					zlog.Err(err))
			}
		}
		
		zlog.Info("Blog generation complete", 
			zlog.String("index", indexPath),
			zlog.Int("posts", len(posts)))
	}

	// Generate initial blog
	generateBlog("")

	// Display initial results
	absPath, _ := filepath.Abs(outputDir)
	println("\n🎉 Initial Blog Generated!")
	println("========================")
	println("📄 " + filepath.Join(absPath, "index.html"))
	println("📁 " + filepath.Join(absPath, "posts/"))
	println("\n🌐 Open in browser:")
	println("file://" + filepath.Join(absPath, "index.html"))

	// Now let's test REAL FILE WATCHING!
	println("\n🔥 Testing Real File Watching...")
	println("===============================")
	println("The system is now watching for file changes in: " + contentDir)
	println("Try editing one of the markdown files!")
	println("Press Ctrl+C to exit")

	// Wait a bit for user to see the message
	time.Sleep(2 * time.Second)

	// Demonstrate programmatic file changes
	go func() {
		for i := 0; i < 3; i++ {
			time.Sleep(5 * time.Second)
			
			// Create a new blog post by writing to the file system
			newPostPath := filepath.Join(contentDir, fmt.Sprintf("live-update-%d.md", i+1))
			newPostContent := fmt.Sprintf(`---
title: Live Update #%d - Real File Watching!
slug: live-update-%d
date: %s
author: File System Watcher
tags: [live, demo, file-watching]
excerpt: This post was created by writing to the file system - Flux detected it automatically!
draft: false
---

# Live Update #%d - Real File Watching!

🔥 **This post was created by writing directly to the file system!**

## How it works:

1. **File written** to %s
2. **fsnotify detects** the file change
3. **Hodor filesystem provider** triggers change event
4. **Flux collection watching** processes the update
5. **Docula regenerates** the blog automatically

## Time: %s

This demonstrates **true reactive file watching** - no manual triggers needed!

---

*Created by the file system watcher* 📁⚡`, 
				i+1, i+1, time.Now().Format("2006-01-02"), 
				i+1, newPostPath, time.Now().Format("15:04:05"))

			zlog.Info("Creating new blog post", 
				zlog.String("path", newPostPath),
				zlog.Int("update_number", i+1))

			if err := os.WriteFile(newPostPath, []byte(newPostContent), 0644); err != nil {
				zlog.Warn("Failed to write new post", zlog.Err(err))
				continue
			}

			// Give Flux time to detect and process
			time.Sleep(1 * time.Second)

			// Regenerate blog with updates
			suffix := fmt.Sprintf("-after-update-%d", i+1)
			generateBlog(suffix)

			println(fmt.Sprintf("\n✅ Update #%d Complete!", i+1))
			println("📄 New file: " + newPostPath)
			println("📄 Updated blog: " + filepath.Join(absPath, "index"+suffix+".html"))
		}

		println("\n🎉 File Watching Demo Complete!")
		println("===============================")
		println("Check the output directory to see all generated versions:")
		println("- index.html (initial)")
		println("- index-after-update-1.html")
		println("- index-after-update-2.html") 
		println("- index-after-update-3.html")
		println("\nEach version shows the blog growing as files were added!")
	}()

	// Keep the program running to demonstrate file watching
	// In a real application, this would be a web server
	select {}
}