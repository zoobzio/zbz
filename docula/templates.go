package docula

import (
	"fmt"
	"html/template"
	"sort"
	"strings"
	"time"
)

// BlogPost represents a blog post with metadata
type BlogPost struct {
	*DocPage
	Date          time.Time
	Author        string
	AuthorEmail   string
	Slug          string
	Excerpt       string
	FeaturedImage string
	Draft         bool
}

// TemplateRenderer handles rendering different site templates
type TemplateRenderer struct {
	service *Service
}

// NewTemplateRenderer creates a new template renderer
func NewTemplateRenderer(service *Service) *TemplateRenderer {
	return &TemplateRenderer{service: service}
}

// RenderBlogIndex renders the blog homepage with recent posts
func (tr *TemplateRenderer) RenderBlogIndex(siteConfig SiteConfig) (string, error) {
	posts := tr.getBlogPosts()
	
	// Sort by date (newest first)
	sort.Slice(posts, func(i, j int) bool {
		return posts[i].Date.After(posts[j].Date)
	})
	
	// Filter out drafts
	var publishedPosts []*BlogPost
	for _, post := range posts {
		if !post.Draft {
			publishedPosts = append(publishedPosts, post)
		}
	}
	
	data := struct {
		SiteConfig SiteConfig
		Posts      []*BlogPost
		Title      string
	}{
		SiteConfig: siteConfig,
		Posts:      publishedPosts,
		Title:      siteConfig.Title,
	}
	
	return tr.renderTemplate(blogIndexTemplate, data)
}

// RenderBlogPost renders an individual blog post
func (tr *TemplateRenderer) RenderBlogPost(slug string, siteConfig SiteConfig) (string, error) {
	posts := tr.getBlogPosts()
	
	var post *BlogPost
	for _, p := range posts {
		if p.Slug == slug {
			post = p
			break
		}
	}
	
	if post == nil {
		return "", fmt.Errorf("blog post not found: %s", slug)
	}
	
	data := struct {
		SiteConfig SiteConfig
		Post       *BlogPost
		Title      string
	}{
		SiteConfig: siteConfig,
		Post:       post,
		Title:      post.Title,
	}
	
	return tr.renderTemplate(blogPostTemplate, data)
}

// GetBlogPosts returns all blog posts (public method)
func (tr *TemplateRenderer) GetBlogPosts() []*BlogPost {
	return tr.getBlogPosts()
}

// getBlogPosts extracts blog posts from pages
func (tr *TemplateRenderer) getBlogPosts() []*BlogPost {
	var posts []*BlogPost
	
	pages := tr.service.ListPages()
	for path, page := range pages {
		// Look for blog posts (could be in blog/ directory or have date in frontmatter)
		if strings.HasPrefix(path, "blog/") || hasDateMetadata(page) {
			post := &BlogPost{
				DocPage: page,
				Date:    extractDate(page),
				Author:  extractStringMeta(page, "author"),
				AuthorEmail: extractStringMeta(page, "author_email"),
				Slug:    extractStringMeta(page, "slug"),
				Excerpt: extractStringMeta(page, "excerpt"),
				FeaturedImage: extractStringMeta(page, "featured_image"),
				Draft:   extractBoolMeta(page, "draft"),
			}
			
			// Generate slug if not provided
			if post.Slug == "" {
				post.Slug = generateSlug(page.Title)
			}
			
			posts = append(posts, post)
		}
	}
	
	return posts
}

// Helper functions for metadata extraction
func hasDateMetadata(page *DocPage) bool {
	// Check if any tags suggest this is a blog post
	for _, tag := range page.Tags {
		if tag == "announcement" || tag == "release" || tag == "blog" {
			return true
		}
	}
	return false
}

func extractDate(page *DocPage) time.Time {
	// Try to extract date from frontmatter - for now return current time
	// In a real implementation, this would parse the "date" field
	return time.Now()
}

func extractStringMeta(page *DocPage, key string) string {
	// In a real implementation, this would extract from parsed frontmatter
	// For now, return defaults
	switch key {
	case "author":
		return "ZBZ Team"
	case "slug":
		return generateSlug(page.Title)
	case "excerpt":
		// Extract first paragraph as excerpt
		content := string(page.Content)
		if idx := strings.Index(content, "</p>"); idx > 0 {
			start := strings.Index(content, "<p>")
			if start >= 0 {
				excerpt := content[start+3 : idx]
				// Strip HTML tags
				excerpt = strings.ReplaceAll(excerpt, "<strong>", "")
				excerpt = strings.ReplaceAll(excerpt, "</strong>", "")
				return excerpt
			}
		}
		return ""
	default:
		return ""
	}
}

func extractBoolMeta(page *DocPage, key string) bool {
	// For now, no posts are drafts
	return false
}

func generateSlug(title string) string {
	slug := strings.ToLower(title)
	slug = strings.ReplaceAll(slug, " ", "-")
	slug = strings.ReplaceAll(slug, "!", "")
	slug = strings.ReplaceAll(slug, "(", "")
	slug = strings.ReplaceAll(slug, ")", "")
	return slug
}

// renderTemplate renders a template with data
func (tr *TemplateRenderer) renderTemplate(tmplStr string, data interface{}) (string, error) {
	tmpl, err := template.New("page").Parse(tmplStr)
	if err != nil {
		return "", err
	}
	
	var buf strings.Builder
	err = tmpl.Execute(&buf, data)
	if err != nil {
		return "", err
	}
	
	return buf.String(), nil
}

// Blog templates
const blogIndexTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{.Title}}</title>
    <script src="https://unpkg.com/htmx.org@1.9.10"></script>
    <style>
        body { 
            max-width: 800px; 
            margin: 0 auto; 
            padding: 2rem; 
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif;
            line-height: 1.6;
        }
        .header { 
            border-bottom: 2px solid #eee; 
            margin-bottom: 2rem; 
            padding-bottom: 1rem;
        }
        .post-card { 
            border: 1px solid #ddd; 
            border-radius: 8px; 
            padding: 1.5rem; 
            margin-bottom: 1.5rem;
            transition: shadow 0.2s;
        }
        .post-card:hover {
            box-shadow: 0 4px 12px rgba(0,0,0,0.1);
        }
        .post-title { 
            margin: 0 0 0.5rem 0; 
            color: #333;
        }
        .post-meta { 
            color: #666; 
            font-size: 0.9rem; 
            margin-bottom: 1rem;
        }
        .post-excerpt { 
            color: #555; 
            margin-bottom: 1rem;
        }
        .read-more { 
            color: #0066cc; 
            text-decoration: none; 
            font-weight: 500;
        }
        .read-more:hover { 
            text-decoration: underline; 
        }
        .search-box {
            margin-bottom: 2rem;
            padding: 0.5rem;
            border: 1px solid #ccc;
            border-radius: 4px;
            width: 100%;
            font-size: 1rem;
        }
    </style>
</head>
<body>
    <div class="header">
        <h1>{{.SiteConfig.Title}}</h1>
        <p>{{.SiteConfig.Description}}</p>
        
        <input type="search" 
               class="search-box"
               placeholder="Search posts..." 
               hx-get="/blog/search" 
               hx-trigger="keyup changed delay:300ms" 
               hx-target="#search-results"
               hx-include="[name='q']"
               name="q">
        <div id="search-results"></div>
    </div>

    <main>
        {{range .Posts}}
        <article class="post-card">
            <h2 class="post-title">{{.Title}}</h2>
            <div class="post-meta">
                By {{.Author}} • {{.Date.Format "January 2, 2006"}}
                {{if .Tags}} • Tags: {{range $i, $tag := .Tags}}{{if $i}}, {{end}}{{$tag}}{{end}}{{end}}
            </div>
            {{if .Excerpt}}
            <div class="post-excerpt">{{.Excerpt}}</div>
            {{end}}
            <a href="/blog/posts/{{.Slug}}" class="read-more">Read More →</a>
        </article>
        {{else}}
        <p>No blog posts yet. Check back soon!</p>
        {{end}}
    </main>

    <footer style="margin-top: 3rem; padding-top: 2rem; border-top: 1px solid #eee; color: #666; text-align: center;">
        <p>Powered by <strong>Docula V2</strong> 🚀 Living Documentation</p>
    </footer>
</body>
</html>`

const blogPostTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{.Post.Title}} - {{.SiteConfig.Title}}</title>
    <style>
        body { 
            max-width: 800px; 
            margin: 0 auto; 
            padding: 2rem; 
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif;
            line-height: 1.6;
        }
        .header { 
            margin-bottom: 2rem;
        }
        .back-link { 
            color: #0066cc; 
            text-decoration: none;
            margin-bottom: 1rem;
            display: inline-block;
        }
        .post-title { 
            margin: 0 0 1rem 0; 
            color: #333;
            font-size: 2.5rem;
        }
        .post-meta { 
            color: #666; 
            margin-bottom: 2rem;
            padding-bottom: 1rem;
            border-bottom: 1px solid #eee;
        }
        .post-content {
            font-size: 1.1rem;
        }
        .post-content h1, .post-content h2, .post-content h3 {
            margin-top: 2rem;
            margin-bottom: 1rem;
        }
        .post-content code {
            background: #f5f5f5;
            padding: 0.2rem 0.4rem;
            border-radius: 3px;
            font-family: 'Monaco', 'Consolas', monospace;
        }
        .post-content pre {
            background: #f8f8f8;
            padding: 1rem;
            border-radius: 5px;
            overflow-x: auto;
        }
        .post-content blockquote {
            border-left: 4px solid #0066cc;
            margin-left: 0;
            padding-left: 1rem;
            color: #555;
        }
    </style>
</head>
<body>
    <div class="header">
        <a href="/blog" class="back-link">← Back to Blog</a>
        <h1 class="post-title">{{.Post.Title}}</h1>
        <div class="post-meta">
            By {{.Post.Author}} • {{.Post.Date.Format "January 2, 2006"}}
            {{if .Post.Tags}} • Tags: {{range $i, $tag := .Post.Tags}}{{if $i}}, {{end}}{{$tag}}{{end}}{{end}}
        </div>
    </div>

    <article class="post-content">
        {{.Post.Content}}
    </article>

    <footer style="margin-top: 3rem; padding-top: 2rem; border-top: 1px solid #eee; color: #666; text-align: center;">
        <p><a href="/blog" style="color: #0066cc;">← Back to Blog</a> | Powered by <strong>Docula V2</strong> 🚀</p>
    </footer>
</body>
</html>`