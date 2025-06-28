package docula

import (
	"context"
	"fmt"
	"sync"

	"zbz/capitan"
	"zbz/universal"
)

// Singleton service instance (z = self pattern)
var z *zDoculaService
var once sync.Once

// zDoculaService is the singleton docula service
type zDoculaService struct {
	providers map[string]ContentProvider
	contracts map[string]any // DocumentContract instances by collection name
	processors map[ContentType]ContentProcessor
	templates map[string]ContentProcessor
	mutex     sync.RWMutex
}

// Service returns the singleton docula service instance
func Service() *zDoculaService {
	once.Do(func() {
		z = &zDoculaService{
			providers:  make(map[string]ContentProvider),
			contracts:  make(map[string]any),
			processors: make(map[ContentType]ContentProcessor),
			templates:  make(map[string]ContentProcessor),
		}
	})
	return z
}

// Register registers a content provider with the docula service
func Register(providerKey string, provider ContentProvider) error {
	service := Service()
	service.mutex.Lock()
	defer service.mutex.Unlock()
	
	if provider == nil {
		return fmt.Errorf("provider cannot be nil")
	}
	
	service.providers[providerKey] = provider
	
	// Emit capitan hook for provider registration
	capitan.Emit(context.Background(), ProviderRegistered, "docula-service", ProviderRegisteredData{
		ProviderKey:  providerKey,
		ProviderType: provider.GetProvider(),
	}, nil)
	
	return nil
}

// RegisterWithConfig registers a content provider using configuration
func RegisterWithConfig(providerType string, config ContentProviderConfig) error {
	provider, err := NewProvider(providerType, config)
	if err != nil {
		return fmt.Errorf("failed to create provider '%s': %w", providerType, err)
	}
	
	providerKey := config.ProviderKey
	if providerKey == "" {
		providerKey = "default"
	}
	
	return Register(providerKey, provider)
}

// Documents creates a typed document contract for a collection
func Documents[T any](collectionName string) (DocumentContract[T], error) {
	return DocumentsWithProvider[T](collectionName, "default")
}

// DocumentsWithProvider creates a typed document contract with specific provider
func DocumentsWithProvider[T any](collectionName, providerKey string) (DocumentContract[T], error) {
	service := Service()
	
	// Check if provider exists
	service.mutex.RLock()
	_, exists := service.providers[providerKey]
	service.mutex.RUnlock()
	
	if !exists {
		return nil, fmt.Errorf("provider '%s' not registered", providerKey)
	}
	
	// Detect content type from T
	contentType := detectContentType[T]()
	
	// Get processor for content type
	processor := service.getProcessor(contentType)
	
	// Create contract instance
	contract := &zDocumentContract[T]{
		collectionName: collectionName,
		providerKey:    providerKey,
		service:        service,
		contentType:    contentType,
		processor:      processor,
	}
	
	// Store contract for later retrieval
	service.mutex.Lock()
	service.contracts[collectionName] = contract
	service.mutex.Unlock()
	
	// Emit capitan hook for contract creation
	capitan.Emit(context.Background(), ContractCreated, "docula-service", ContractCreatedData{
		CollectionName: collectionName,
		ContentType:    string(contentType),
		ProviderKey:    providerKey,
	}, nil)
	
	return contract, nil
}

// Markdown creates a markdown document contract
func Markdown(collectionName string) (DocumentContract[MarkdownDocument], error) {
	return Documents[MarkdownDocument](collectionName)
}

// Blog creates a blog post contract
func Blog(collectionName string) (DocumentContract[BlogPost], error) {
	return Documents[BlogPost](collectionName)
}

// Wiki creates a wiki page contract
func Wiki(collectionName string) (DocumentContract[WikiPage], error) {
	return Documents[WikiPage](collectionName)
}

// Knowledge creates a knowledge article contract
func Knowledge(collectionName string) (DocumentContract[KnowledgeArticle], error) {
	return Documents[KnowledgeArticle](collectionName)
}

// OpenAPI creates an OpenAPI document contract
func OpenAPI(collectionName string) (DocumentContract[OpenAPIDocument], error) {
	return Documents[OpenAPIDocument](collectionName)
}

// Provider returns a registered content provider
func Provider(providerKey string) (ContentProvider, error) {
	service := Service()
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	
	provider, exists := service.providers[providerKey]
	if !exists {
		return nil, fmt.Errorf("provider '%s' not registered", providerKey)
	}
	
	return provider, nil
}

// Providers returns all registered provider keys
func Providers() []string {
	service := Service()
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	
	keys := make([]string, 0, len(service.providers))
	for key := range service.providers {
		keys = append(keys, key)
	}
	
	return keys
}

// Collections returns all registered collection names
func Collections() []string {
	service := Service()
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	
	names := make([]string, 0, len(service.contracts))
	for name := range service.contracts {
		names = append(names, name)
	}
	
	return names
}

// RegisterProcessor registers a content processor for a content type
func RegisterProcessor(contentType ContentType, processor ContentProcessor) {
	service := Service()
	service.mutex.Lock()
	defer service.mutex.Unlock()
	
	service.processors[contentType] = processor
}

// RegisterTemplate registers a template processor
func RegisterTemplate(templateName string, processor ContentProcessor) {
	service := Service()
	service.mutex.Lock()
	defer service.mutex.Unlock()
	
	service.templates[templateName] = processor
}

// Convenience functions for common operations

// RenderHTML renders any document as HTML
func RenderHTML[T any](ctx context.Context, contract DocumentContract[T], document T) ([]byte, error) {
	return contract.Render(ctx, document, FormatHTML)
}

// RenderMarkdown renders any document as Markdown
func RenderMarkdown[T any](ctx context.Context, contract DocumentContract[T], document T) ([]byte, error) {
	return contract.Render(ctx, document, FormatMarkdown)
}

// RenderJSON renders any document as JSON
func RenderJSON[T any](ctx context.Context, contract DocumentContract[T], document T) ([]byte, error) {
	return contract.Render(ctx, document, FormatJSON)
}

// Search performs a text search across all collections
func Search(ctx context.Context, text string, limit int) (map[string]any, error) {
	service := Service()
	service.mutex.RLock()
	contracts := make(map[string]any)
	for name, contract := range service.contracts {
		contracts[name] = contract
	}
	service.mutex.RUnlock()
	
	results := make(map[string]any)
	
	for collectionName := range contracts {
		// This is a simplified search - real implementation would need type assertions
		// or a common search interface
		results[collectionName] = fmt.Sprintf("Search results for '%s' in collection '%s'", text, collectionName)
	}
	
	return results, nil
}

// PublishAll publishes all draft content in a collection
func PublishAll[T any](ctx context.Context, contract DocumentContract[T]) error {
	// List all documents in collection
	allPattern := universal.ResourceURI{}
	documents, err := contract.List(ctx, allPattern)
	if err != nil {
		return err
	}
	
	// Publish each document
	for _, document := range documents {
		err := contract.Publish(ctx, document)
		if err != nil {
			return fmt.Errorf("failed to publish document: %w", err)
		}
	}
	
	return nil
}

// Health checks the health of all registered providers
func Health(ctx context.Context) (map[string]ProviderHealth, error) {
	service := Service()
	service.mutex.RLock()
	defer service.mutex.RUnlock()
	
	health := make(map[string]ProviderHealth)
	
	for key, provider := range service.providers {
		providerHealth, err := provider.Health(ctx)
		if err != nil {
			providerHealth = ProviderHealth{
				Status:  "unhealthy",
				Message: err.Error(),
			}
		}
		health[key] = providerHealth
	}
	
	return health, nil
}

// Close closes all registered providers
func Close() error {
	service := Service()
	service.mutex.Lock()
	defer service.mutex.Unlock()
	
	var lastErr error
	for key, provider := range service.providers {
		if err := provider.Close(); err != nil {
			lastErr = fmt.Errorf("failed to close provider '%s': %w", key, err)
		}
	}
	
	// Clear all state
	service.providers = make(map[string]ContentProvider)
	service.contracts = make(map[string]any)
	
	return lastErr
}

// Internal helper methods

func (s *zDoculaService) getProvider(providerKey string) (ContentProvider, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	
	provider, exists := s.providers[providerKey]
	if !exists {
		return nil, fmt.Errorf("provider '%s' not registered", providerKey)
	}
	
	return provider, nil
}

func (s *zDoculaService) getProcessor(contentType ContentType) ContentProcessor {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	
	if processor, exists := s.processors[contentType]; exists {
		return processor
	}
	
	// Return default processor if none registered
	return &defaultProcessor{contentType: contentType}
}

func (s *zDoculaService) getTemplateProcessor(templateName string) (ContentProcessor, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	
	processor, exists := s.templates[templateName]
	if !exists {
		return nil, fmt.Errorf("template '%s' not registered", templateName)
	}
	
	return processor, nil
}

// detectContentType attempts to detect content type from Go type T
func detectContentType[T any]() ContentType {
	var zero T
	typeName := fmt.Sprintf("%T", zero)
	
	switch typeName {
	case "docula.MarkdownDocument":
		return ContentTypeMarkdown
	case "docula.BlogPost":
		return ContentTypeBlog
	case "docula.WikiPage":
		return ContentTypeWiki
	case "docula.KnowledgeArticle":
		return ContentTypeKnowledge
	case "docula.OpenAPIDocument":
		return ContentTypeOpenAPI
	default:
		return ContentTypeMarkdown // Default fallback
	}
}

// defaultProcessor provides basic content processing
type defaultProcessor struct {
	contentType ContentType
}

func (p *defaultProcessor) Process(ctx context.Context, raw []byte, metadata ContentMetadata) (ProcessedContent, error) {
	return ProcessedContent{
		Raw:      raw,
		Processed: raw,
		Metadata: metadata,
	}, nil
}

func (p *defaultProcessor) Render(ctx context.Context, content ProcessedContent, format ContentFormat) ([]byte, error) {
	switch format {
	case FormatHTML:
		return []byte(fmt.Sprintf("<pre>%s</pre>", content.Processed)), nil
	case FormatJSON:
		return content.Processed, nil
	default:
		return content.Processed, nil
	}
}

func (p *defaultProcessor) Validate(ctx context.Context, content ProcessedContent) error {
	return nil
}

func (p *defaultProcessor) GetSupportedFormats() []ContentFormat {
	return []ContentFormat{FormatHTML, FormatMarkdown, FormatJSON, FormatText}
}

func (p *defaultProcessor) GetContentType() ContentType {
	return p.contentType
}