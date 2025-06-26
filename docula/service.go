package docula

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"zbz/flux"
	"zbz/hodor"
	"zbz/zlog"
)

// Service represents the main docula documentation service
type Service struct {
	contract  DoculaContract
	processor *MarkdownProcessor

	// Simple cache - just store the processed pages
	mu    sync.RWMutex
	pages map[string]*DocPage

	// Hodor storage
	storage *hodor.HodorContract
	
	// OpenAPI spec generator
	specGenerator *SpecGenerator
	
	// Flux collection watcher for reactive updates
	collectionWatcher *flux.CollectionWatcher
}

// NewService creates a new docula service from a contract
func NewService(contract DoculaContract) *Service {
	service := &Service{
		contract:      contract,
		processor:     NewMarkdownProcessor(),
		pages:         make(map[string]*DocPage),
		specGenerator: NewSpecGenerator(),
	}

	// Initialize storage if provided
	if contract.Storage != nil {
		service.storage = contract.Storage
		
		// Always try to enable reactive updates - this is the future!
		err := service.setupFluxWatcher()
		if err != nil {
			zlog.Warn("Live updates unavailable", zlog.Err(err))
		} else {
			zlog.Info("Live updates enabled automatically")
		}
	}

	return service
}

// LoadContent loads and processes markdown content from storage
func (s *Service) LoadContent() error {
	if s.storage == nil {
		return fmt.Errorf("no storage configured")
	}

	// List all markdown files
	files, err := s.storage.List("")
	if err != nil {
		return fmt.Errorf("failed to list markdown files: %w", err)
	}
	
	// Filter for .md files since we listed all
	var mdFiles []string
	for _, file := range files {
		if strings.HasSuffix(file, ".md") {
			mdFiles = append(mdFiles, file)
		}
	}
	files = mdFiles

	s.mu.Lock()
	defer s.mu.Unlock()

	// Process each file
	for _, path := range files {
		content, err := s.storage.Get(path)
		if err != nil {
			zlog.Warn("Failed to load file from storage", 
				zlog.String("path", path), 
				zlog.Err(err))
			continue
		}

		page, err := s.processor.ParseDocPage(path, content)
		if err != nil {
			zlog.Warn("Failed to process markdown file", 
				zlog.String("path", path), 
				zlog.Err(err))
			continue
		}

		s.pages[path] = page
		zlog.Debug("Loaded documentation page", 
			zlog.String("path", path), 
			zlog.String("title", page.Title))
	}

	return nil
}

// GetPage returns a processed page by path
func (s *Service) GetPage(path string) (*DocPage, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	page, exists := s.pages[path]
	return page, exists
}

// ListPages returns all loaded pages
func (s *Service) ListPages() map[string]*DocPage {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Return a copy to avoid race conditions
	result := make(map[string]*DocPage)
	for k, v := range s.pages {
		result[k] = v
	}
	return result
}

// RenderPageHTML returns the HTML content for a page
func (s *Service) RenderPageHTML(path string) (string, error) {
	page, exists := s.GetPage(path)
	if !exists {
		return "", fmt.Errorf("page not found: %s", path)
	}

	// For now, just return the content
	// In a real implementation, this would render with templates
	return string(page.Content), nil
}

// setupFluxWatcher configures reactive updates using Flux collection watching
func (s *Service) setupFluxWatcher() error {
	// Watch for changes to all markdown files using the new collection pattern
	watcher, err := flux.SyncCollection(
		s.storage,
		"*.md", // Watch all .md files
		s.handleContentChanges,
		flux.FluxOptions{
			SkipInitialCallback: true, // Don't trigger on initial load since we already loaded
		},
	)
	if err != nil {
		return fmt.Errorf("failed to create flux collection watcher: %w", err)
	}
	
	s.collectionWatcher = watcher
	zlog.Info("Reactive updates enabled", zlog.String("pattern", "*.md"))
	return nil
}

// handleContentChanges processes markdown file changes from Flux
func (s *Service) handleContentChanges(old, new map[string][]byte) {
	zlog.Info("Flux detected changes - updating documentation")
	
	s.mu.Lock()
	defer s.mu.Unlock()
	
	// Track what changed
	var changedFiles []string
	
	// Process new/updated files
	for path, content := range new {
		if strings.HasSuffix(path, ".md") {
			page, err := s.processor.ParseDocPage(path, content)
			if err != nil {
				zlog.Warn("Failed to process updated file", 
					zlog.String("path", path), 
					zlog.Err(err))
				continue
			}
			
			s.pages[path] = page
			changedFiles = append(changedFiles, path)
			zlog.Debug("Updated documentation page", 
				zlog.String("path", path), 
				zlog.String("title", page.Title))
		}
	}
	
	// Remove deleted files
	for path := range old {
		if _, exists := new[path]; !exists && strings.HasSuffix(path, ".md") {
			delete(s.pages, path)
			changedFiles = append(changedFiles, path)
			zlog.Debug("Removed documentation page", zlog.String("path", path))
		}
	}
	
	// Regenerate OpenAPI spec with updated content
	s.regenerateSpec()
	
	zlog.Info("Documentation updated", 
		zlog.Int("changed_files", len(changedFiles)),
		zlog.Strings("files", changedFiles))
}

// regenerateSpec updates the OpenAPI spec with current markdown content
func (s *Service) regenerateSpec() {
	// Add some sample endpoints for demonstration
	s.specGenerator.SetInfo(s.contract.Name, s.contract.Description, "1.0.0")
	s.specGenerator.RegisterEndpoint("GET", "/users", "ListUsers", "List all users", "Retrieve a paginated list of users")
	s.specGenerator.RegisterEndpoint("POST", "/users", "CreateUser", "Create user", "Create a new user account")
	s.specGenerator.RegisterEndpoint("GET", "/users/{id}", "GetUser", "Get user", "Retrieve a specific user by ID")
	s.specGenerator.RegisterModel("User", "User account information")
	
	// Enhance with markdown content
	s.specGenerator.EnhanceWithMarkdown(s.pages)
}

// GetSpecYAML returns the OpenAPI spec as YAML
func (s *Service) GetSpecYAML() ([]byte, error) {
	return s.specGenerator.GetYAML()
}

// GetSpecJSON returns the OpenAPI spec as JSON
func (s *Service) GetSpecJSON() ([]byte, error) {
	return s.specGenerator.GetJSON()
}

// TriggerUpdate simulates a content change for testing
func (s *Service) TriggerUpdate(path string, content string) error {
	if s.storage == nil {
		return fmt.Errorf("no storage configured")
	}
	
	// Update the content in storage - this will trigger Flux collection watcher if reactive updates are enabled
	err := s.storage.Set(path, []byte(content), time.Duration(0))
	if err != nil {
		return err
	}
	
	zlog.Debug("Updated content in storage", 
		zlog.String("path", path),
		zlog.String("note", "Flux will detect this change"))
	return nil
}

// Stop gracefully shuts down the service and cleans up resources
func (s *Service) Stop() error {
	if s.collectionWatcher != nil {
		return s.collectionWatcher.Stop()
	}
	return nil
}