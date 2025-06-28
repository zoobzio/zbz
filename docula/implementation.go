package docula

import (
	"context"
	"fmt"
	"time"

	"zbz/cereal"
	"zbz/universal"
)

// Universal interface implementation (universal.DataAccess[T])

// Get retrieves a content item by ResourceURI
func (z *zDocumentContract[T]) Get(ctx context.Context, resource universal.ResourceURI) (T, error) {
	var result T
	
	// Parse ResourceURI: "content://docs/getting-started" -> collection: "docs", path: "getting-started"
	if resource.Service() != "content" {
		return result, fmt.Errorf("invalid service '%s' for content operation", resource.Service())
	}
	
	collectionName := resource.ResourcePath()
	if collectionName == "" {
		collectionName = z.collectionName
	}
	
	path := resource.Identifier()
	if path == "" {
		return result, fmt.Errorf("resource URI must specify a path for Get operation")
	}
	
	// Build full URI for provider
	uri := BuildContentURI(collectionName, path)
	
	// Retrieve content from provider
	provider, err := z.service.getProvider(z.providerKey)
	if err != nil {
		return result, err
	}
	
	content, metadata, err := provider.Retrieve(ctx, uri)
	if err != nil {
		return result, err
	}
	
	// Process content using processor
	processed, err := z.processor.Process(ctx, content, metadata)
	if err != nil {
		return result, err
	}
	
	// Convert to T type
	return z.convertProcessedToT(processed)
}

// Set creates or updates a content item by ResourceURI
func (z *zDocumentContract[T]) Set(ctx context.Context, resource universal.ResourceURI, data T) error {
	// Parse ResourceURI to extract collection and path
	collectionName := resource.ResourcePath()
	if collectionName == "" {
		collectionName = z.collectionName
	}
	
	path := resource.Identifier()
	if path == "" {
		return fmt.Errorf("resource URI must specify a path for Set operation")
	}
	
	// Convert T to content and metadata
	content, metadata, err := z.convertTToContent(data)
	if err != nil {
		return err
	}
	
	// Build full URI for provider
	uri := BuildContentURI(collectionName, path)
	
	// Store content via provider
	provider, err := z.service.getProvider(z.providerKey)
	if err != nil {
		return err
	}
	
	// Check if content exists for update vs create
	exists, _ := provider.Exists(ctx, uri)
	
	err = provider.Store(ctx, uri, content, metadata)
	if err != nil {
		return err
	}
	
	// Emit capitan hook for content operation
	if exists {
		z.emitDocumentUpdatedHook(ctx, uri, data, metadata)
	} else {
		z.emitDocumentCreatedHook(ctx, uri, data, metadata)
	}
	
	return nil
}

// Delete removes a content item by ResourceURI
func (z *zDocumentContract[T]) Delete(ctx context.Context, resource universal.ResourceURI) error {
	collectionName := resource.ResourcePath()
	if collectionName == "" {
		collectionName = z.collectionName
	}
	
	path := resource.Identifier()
	if path == "" {
		return fmt.Errorf("resource URI must specify a path for Delete operation")
	}
	
	// Build full URI for provider
	uri := BuildContentURI(collectionName, path)
	
	// Get content before deletion for audit trail
	provider, err := z.service.getProvider(z.providerKey)
	if err != nil {
		return err
	}
	
	var deletedData T
	existingContent, err := z.Get(ctx, resource)
	if err == nil {
		deletedData = existingContent
	}
	
	// Delete from provider
	err = provider.Delete(ctx, uri)
	if err != nil {
		return err
	}
	
	// Emit capitan hook for content deletion
	z.emitDocumentDeletedHook(ctx, uri, deletedData)
	
	return nil
}

// List retrieves multiple content items matching a ResourceURI pattern
func (z *zDocumentContract[T]) List(ctx context.Context, pattern universal.ResourceURI) ([]T, error) {
	collectionName := pattern.ResourcePath()
	if collectionName == "" {
		collectionName = z.collectionName
	}
	
	// Build pattern for provider
	identifier := pattern.Identifier()
	if identifier == "" {
		identifier = "*"
	}
	
	searchPattern := BuildContentURI(collectionName, identifier)
	
	// List content from provider
	provider, err := z.service.getProvider(z.providerKey)
	if err != nil {
		return nil, err
	}
	
	items, err := provider.List(ctx, searchPattern)
	if err != nil {
		return nil, err
	}
	
	// Convert items to T type
	var results []T
	for _, item := range items {
		content, _, err := provider.Retrieve(ctx, item.URI)
		if err != nil {
			continue // Skip items that can't be retrieved
		}
		
		processed, err := z.processor.Process(ctx, content, item.Metadata)
		if err != nil {
			continue // Skip items that can't be processed
		}
		
		entity, err := z.convertProcessedToT(processed)
		if err != nil {
			continue // Skip items that can't be converted
		}
		
		results = append(results, entity)
	}
	
	return results, nil
}

// Exists checks if a content item exists by ResourceURI
func (z *zDocumentContract[T]) Exists(ctx context.Context, resource universal.ResourceURI) (bool, error) {
	collectionName := resource.ResourcePath()
	if collectionName == "" {
		collectionName = z.collectionName
	}
	
	path := resource.Identifier()
	if path == "" {
		return false, fmt.Errorf("resource URI must specify a path for Exists operation")
	}
	
	// Build full URI for provider
	uri := BuildContentURI(collectionName, path)
	
	// Check existence via provider
	provider, err := z.service.getProvider(z.providerKey)
	if err != nil {
		return false, err
	}
	
	return provider.Exists(ctx, uri)
}

// Count returns the number of content items matching a ResourceURI pattern
func (z *zDocumentContract[T]) Count(ctx context.Context, pattern universal.ResourceURI) (int64, error) {
	items, err := z.List(ctx, pattern)
	if err != nil {
		return 0, err
	}
	return int64(len(items)), nil
}

// Execute performs complex content operations via OperationURI
func (z *zDocumentContract[T]) Execute(ctx context.Context, operation universal.OperationURI, params any) (any, error) {
	if operation.Service() != "content" {
		return nil, fmt.Errorf("invalid service '%s' for content operation", operation.Service())
	}
	
	// Route based on operation category
	switch operation.Category() {
	case "render":
		return z.executeRenderOperation(ctx, operation.Operation(), params)
	case "search":
		return z.executeSearchOperation(ctx, operation.Operation(), params)
	case "template":
		return z.executeTemplateOperation(ctx, operation.Operation(), params)
	case "version":
		return z.executeVersionOperation(ctx, operation.Operation(), params)
	case "publish":
		return z.executePublishOperation(ctx, operation.Operation(), params)
	default:
		return nil, fmt.Errorf("unsupported content operation category: %s", operation.Category())
	}
}

// ExecuteMany performs multiple content operations
func (z *zDocumentContract[T]) ExecuteMany(ctx context.Context, operations []universal.Operation) ([]any, error) {
	var results []any
	for _, op := range operations {
		opURI, err := universal.ParseOperationURI(op.Type)
		if err != nil {
			return nil, fmt.Errorf("invalid operation URI '%s': %w", op.Type, err)
		}
		
		result, err := z.Execute(ctx, opURI, op.Params)
		if err != nil {
			return nil, err
		}
		results = append(results, result)
	}
	return results, nil
}

// Subscribe watches for content changes matching a ResourceURI pattern
func (z *zDocumentContract[T]) Subscribe(ctx context.Context, pattern universal.ResourceURI, callback universal.ChangeCallback[T]) (universal.SubscriptionID, error) {
	// Convert ResourceURI pattern to provider subscription format
	collectionName := pattern.ResourcePath()
	if collectionName == "" {
		collectionName = z.collectionName
	}
	
	identifier := pattern.Identifier()
	if identifier == "" {
		identifier = "*"
	}
	
	subscriptionPattern := BuildContentURI(collectionName, identifier)
	
	// Wrap universal callback to convert provider change events
	wrappedCallback := func(event ContentChangeEvent) {
		// Convert provider ContentChangeEvent to universal.ChangeEvent[T]
		universalEvent := universal.ChangeEvent[T]{
			Operation: string(event.Type),
			Resource:  pattern,
			Pattern:   pattern,
			Source:    "docula",
			Timestamp: event.Timestamp,
		}
		
		// Try to convert event content to typed entity
		if event.NewContent != nil {
			processed, err := z.processor.Process(ctx, event.NewContent, event.NewMetadata)
			if err == nil {
				entity, err := z.convertProcessedToT(processed)
				if err == nil {
					universalEvent.New = &entity
				}
			}
		}
		
		if event.OldContent != nil {
			processed, err := z.processor.Process(ctx, event.OldContent, event.OldMetadata)
			if err == nil {
				entity, err := z.convertProcessedToT(processed)
				if err == nil {
					universalEvent.Old = &entity
				}
			}
		}
		
		callback(universalEvent)
	}
	
	// Subscribe via provider
	provider, err := z.service.getProvider(z.providerKey)
	if err != nil {
		return "", err
	}
	
	providerID, err := provider.Subscribe(ctx, subscriptionPattern, wrappedCallback)
	return universal.SubscriptionID(providerID), err
}

// Unsubscribe removes a content subscription
func (z *zDocumentContract[T]) Unsubscribe(ctx context.Context, id universal.SubscriptionID) error {
	provider, err := z.service.getProvider(z.providerKey)
	if err != nil {
		return err
	}
	
	return provider.Unsubscribe(ctx, SubscriptionID(id))
}

// Name returns the data access name
func (z *zDocumentContract[T]) Name() string {
	return fmt.Sprintf("docula-%s-%s", z.contentType, z.collectionName)
}

// Type returns the data access type
func (z *zDocumentContract[T]) Type() string {
	return "content"
}

// Content-specific methods (beyond universal interface)

// Render converts content to a specific format
func (z *zDocumentContract[T]) Render(ctx context.Context, document T, format ContentFormat) ([]byte, error) {
	// Convert T to ProcessedContent
	content, metadata, err := z.convertTToContent(document)
	if err != nil {
		return nil, err
	}
	
	processed := ProcessedContent{
		Raw:      content,
		Metadata: metadata,
	}
	
	// Process if needed
	if z.processor != nil {
		processed, err = z.processor.Process(ctx, content, metadata)
		if err != nil {
			return nil, err
		}
	}
	
	// Render to format
	return z.processor.Render(ctx, processed, format)
}

// Process converts raw content to typed document
func (z *zDocumentContract[T]) Process(ctx context.Context, raw []byte, processor ContentProcessor) (T, error) {
	var result T
	
	// Use provided processor or default
	proc := processor
	if proc == nil {
		proc = z.processor
	}
	
	metadata := ExtractContentMetadata(raw, proc.GetContentType())
	processed, err := proc.Process(ctx, raw, metadata)
	if err != nil {
		return result, err
	}
	
	return z.convertProcessedToT(processed)
}

// Template applies a template to content with data
func (z *zDocumentContract[T]) Template(ctx context.Context, document T, templateName string, data any) ([]byte, error) {
	// Get template processor
	templateProcessor, err := z.service.getTemplateProcessor(templateName)
	if err != nil {
		return nil, err
	}
	
	// Convert document to processed content
	content, metadata, err := z.convertTToContent(document)
	if err != nil {
		return nil, err
	}
	
	processed := ProcessedContent{
		Raw:      content,
		Metadata: metadata,
	}
	
	// Apply template
	return templateProcessor.Render(ctx, processed, FormatHTML)
}

// Search performs content search
func (z *zDocumentContract[T]) Search(ctx context.Context, query ContentQuery) ([]T, error) {
	// Set collection filter if not specified
	if len(query.Collections) == 0 {
		query.Collections = []string{z.collectionName}
	}
	
	// Execute search via provider
	provider, err := z.service.getProvider(z.providerKey)
	if err != nil {
		return nil, err
	}
	
	results, err := provider.Search(ctx, query)
	if err != nil {
		return nil, err
	}
	
	// Convert results to T type
	var documents []T
	for _, item := range results.Items {
		content, _, err := provider.Retrieve(ctx, item.URI)
		if err != nil {
			continue
		}
		
		processed, err := z.processor.Process(ctx, content, item.Metadata)
		if err != nil {
			continue
		}
		
		document, err := z.convertProcessedToT(processed)
		if err != nil {
			continue
		}
		
		documents = append(documents, document)
	}
	
	return documents, nil
}

// GetMetadata retrieves metadata for a content item
func (z *zDocumentContract[T]) GetMetadata(ctx context.Context, resource universal.ResourceURI) (ContentMetadata, error) {
	collectionName := resource.ResourcePath()
	if collectionName == "" {
		collectionName = z.collectionName
	}
	
	path := resource.Identifier()
	if path == "" {
		return ContentMetadata{}, fmt.Errorf("resource URI must specify a path")
	}
	
	uri := BuildContentURI(collectionName, path)
	
	provider, err := z.service.getProvider(z.providerKey)
	if err != nil {
		return ContentMetadata{}, err
	}
	
	_, metadata, err := provider.Retrieve(ctx, uri)
	return metadata, err
}

// SetMetadata updates metadata for a content item
func (z *zDocumentContract[T]) SetMetadata(ctx context.Context, resource universal.ResourceURI, metadata ContentMetadata) error {
	// For most providers, metadata is stored with content
	// This would require retrieving content, updating metadata, and storing back
	return fmt.Errorf("metadata-only updates not implemented yet")
}

// ListCollections returns all available collections
func (z *zDocumentContract[T]) ListCollections(ctx context.Context) ([]string, error) {
	provider, err := z.service.getProvider(z.providerKey)
	if err != nil {
		return nil, err
	}
	
	return provider.ListCollections(ctx)
}

// CreateCollection creates a new content collection
func (z *zDocumentContract[T]) CreateCollection(ctx context.Context, name string, config CollectionConfig) error {
	provider, err := z.service.getProvider(z.providerKey)
	if err != nil {
		return err
	}
	
	return provider.CreateCollection(ctx, name, config)
}

// Publish marks content as published
func (z *zDocumentContract[T]) Publish(ctx context.Context, document T) error {
	// Extract metadata and mark as published
	content, metadata, err := z.convertTToContent(document)
	if err != nil {
		return err
	}
	
	metadata.Status = StatusPublished
	now := time.Now()
	metadata.Published = &now
	
	// Store updated content
	uri := z.buildURIFromDocument(document)
	provider, err := z.service.getProvider(z.providerKey)
	if err != nil {
		return err
	}
	
	err = provider.Store(ctx, uri, content, metadata)
	if err != nil {
		return err
	}
	
	// Emit hook
	z.emitDocumentPublishedHook(ctx, uri, document, metadata)
	return nil
}

// Unpublish marks content as unpublished
func (z *zDocumentContract[T]) Unpublish(ctx context.Context, resource universal.ResourceURI) error {
	// Get current content
	document, err := z.Get(ctx, resource)
	if err != nil {
		return err
	}
	
	// Update status
	content, metadata, err := z.convertTToContent(document)
	if err != nil {
		return err
	}
	
	metadata.Status = StatusDraft
	metadata.Published = nil
	
	// Store updated content
	uri := z.buildURIFromResource(resource)
	provider, err := z.service.getProvider(z.providerKey)
	if err != nil {
		return err
	}
	
	err = provider.Store(ctx, uri, content, metadata)
	if err != nil {
		return err
	}
	
	// Emit hook
	z.emitDocumentUnpublishedHook(ctx, uri, document)
	return nil
}

// GetVersions retrieves version history for content
func (z *zDocumentContract[T]) GetVersions(ctx context.Context, resource universal.ResourceURI) ([]ContentVersion, error) {
	uri := z.buildURIFromResource(resource)
	
	provider, err := z.service.getProvider(z.providerKey)
	if err != nil {
		return nil, err
	}
	
	return provider.GetVersions(ctx, uri)
}

// CreateVersion creates a new version of content
func (z *zDocumentContract[T]) CreateVersion(ctx context.Context, resource universal.ResourceURI, document T) (string, error) {
	uri := z.buildURIFromResource(resource)
	content, _, err := z.convertTToContent(document)
	if err != nil {
		return "", err
	}
	
	provider, err := z.service.getProvider(z.providerKey)
	if err != nil {
		return "", err
	}
	
	return provider.CreateVersion(ctx, uri, content, "Updated via docula")
}

// CollectionName returns the collection name
func (z *zDocumentContract[T]) CollectionName() string {
	return z.collectionName
}

// ContentType returns the content type
func (z *zDocumentContract[T]) ContentType() string {
	return string(z.contentType)
}

// Provider returns the provider key
func (z *zDocumentContract[T]) Provider() string {
	return z.providerKey
}

// Helper methods for operation execution

func (z *zDocumentContract[T]) executeRenderOperation(ctx context.Context, operation string, params any) (any, error) {
	paramsMap, ok := params.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("params must be map[string]any for render operations")
	}
	
	switch operation {
	case "html":
		document := paramsMap["document"].(T)
		return z.Render(ctx, document, FormatHTML)
	case "markdown":
		document := paramsMap["document"].(T)
		return z.Render(ctx, document, FormatMarkdown)
	case "pdf":
		document := paramsMap["document"].(T)
		return z.Render(ctx, document, FormatPDF)
	default:
		return nil, fmt.Errorf("unsupported render operation: %s", operation)
	}
}

func (z *zDocumentContract[T]) executeSearchOperation(ctx context.Context, operation string, params any) (any, error) {
	paramsMap, ok := params.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("params must be map[string]any for search operations")
	}
	
	switch operation {
	case "text":
		query := ContentQuery{
			Text:        paramsMap["text"].(string),
			Collections: []string{z.collectionName},
		}
		if limit, ok := paramsMap["limit"].(int); ok {
			query.Limit = limit
		}
		return z.Search(ctx, query)
	default:
		return nil, fmt.Errorf("unsupported search operation: %s", operation)
	}
}

func (z *zDocumentContract[T]) executeTemplateOperation(ctx context.Context, operation string, params any) (any, error) {
	paramsMap, ok := params.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("params must be map[string]any for template operations")
	}
	
	switch operation {
	case "apply":
		document := paramsMap["document"].(T)
		templateName := paramsMap["template"].(string)
		data := paramsMap["data"]
		return z.Template(ctx, document, templateName, data)
	default:
		return nil, fmt.Errorf("unsupported template operation: %s", operation)
	}
}

func (z *zDocumentContract[T]) executeVersionOperation(ctx context.Context, operation string, params any) (any, error) {
	paramsMap, ok := params.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("params must be map[string]any for version operations")
	}
	
	switch operation {
	case "list":
		resourceURI := paramsMap["resource"].(universal.ResourceURI)
		return z.GetVersions(ctx, resourceURI)
	case "create":
		resourceURI := paramsMap["resource"].(universal.ResourceURI)
		document := paramsMap["document"].(T)
		return z.CreateVersion(ctx, resourceURI, document)
	default:
		return nil, fmt.Errorf("unsupported version operation: %s", operation)
	}
}

func (z *zDocumentContract[T]) executePublishOperation(ctx context.Context, operation string, params any) (any, error) {
	paramsMap, ok := params.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("params must be map[string]any for publish operations")
	}
	
	switch operation {
	case "publish":
		document := paramsMap["document"].(T)
		return nil, z.Publish(ctx, document)
	case "unpublish":
		resourceURI := paramsMap["resource"].(universal.ResourceURI)
		return nil, z.Unpublish(ctx, resourceURI)
	default:
		return nil, fmt.Errorf("unsupported publish operation: %s", operation)
	}
}

// Helper methods for type conversion

func (z *zDocumentContract[T]) convertProcessedToT(processed ProcessedContent) (T, error) {
	var result T
	
	// Use cereal to convert processed content to T
	// This assumes T has JSON tags that match ProcessedContent fields
	jsonBytes, err := cereal.JSON.Marshal(processed)
	if err != nil {
		return result, err
	}
	
	err = cereal.JSON.Unmarshal(jsonBytes, &result)
	return result, err
}

func (z *zDocumentContract[T]) convertTToContent(document T) ([]byte, ContentMetadata, error) {
	// Use cereal to convert T to raw content and metadata
	jsonBytes, err := cereal.JSON.Marshal(document)
	if err != nil {
		return nil, ContentMetadata{}, err
	}
	
	// Extract metadata if the document has it
	var metadata ContentMetadata
	documentMap := make(map[string]any)
	if err := cereal.JSON.Unmarshal(jsonBytes, &documentMap); err == nil {
		if metadataMap, ok := documentMap["metadata"].(map[string]any); ok {
			metadataBytes, _ := cereal.JSON.Marshal(metadataMap)
			cereal.JSON.Unmarshal(metadataBytes, &metadata)
		}
	}
	
	// For content, we might want to extract the "content" field specifically
	var content []byte
	if contentStr, ok := documentMap["content"].(string); ok {
		content = []byte(contentStr)
	} else {
		content = jsonBytes // Fallback to full JSON
	}
	
	return content, metadata, nil
}

func (z *zDocumentContract[T]) buildURIFromDocument(document T) string {
	// Try to extract path/URI from document
	// This is a simplified implementation
	return BuildContentURI(z.collectionName, "unknown")
}

func (z *zDocumentContract[T]) buildURIFromResource(resource universal.ResourceURI) string {
	collectionName := resource.ResourcePath()
	if collectionName == "" {
		collectionName = z.collectionName
	}
	
	path := resource.Identifier()
	if path == "" {
		path = "unknown"
	}
	
	return BuildContentURI(collectionName, path)
}