package docula

import (
	"context"
	"time"

	"zbz/capitan"
)

// Content hook types for capitan integration
type ContentHookType int

const (
	// Document lifecycle hooks
	DocumentCreated ContentHookType = iota + 1000 // Start at 1000 to avoid conflicts
	DocumentUpdated
	DocumentDeleted
	DocumentPublished
	DocumentUnpublished
	DocumentRendered
	
	// Collection hooks
	CollectionCreated
	CollectionDeleted
	
	// Template hooks
	TemplateApplied
	
	// Search hooks
	SearchExecuted
	
	// Provider hooks
	ProviderRegistered
	ProviderUnregistered
	
	// Contract hooks
	ContractCreated
	ContractDestroyed
)

// String implements capitan.HookType interface
func (h ContentHookType) String() string {
	switch h {
	case DocumentCreated:
		return "document.created"
	case DocumentUpdated:
		return "document.updated"
	case DocumentDeleted:
		return "document.deleted"
	case DocumentPublished:
		return "document.published"
	case DocumentUnpublished:
		return "document.unpublished"
	case DocumentRendered:
		return "document.rendered"
	case CollectionCreated:
		return "collection.created"
	case CollectionDeleted:
		return "collection.deleted"
	case TemplateApplied:
		return "template.applied"
	case SearchExecuted:
		return "search.executed"
	case ProviderRegistered:
		return "provider.registered"
	case ProviderUnregistered:
		return "provider.unregistered"
	case ContractCreated:
		return "contract.created"
	case ContractDestroyed:
		return "contract.destroyed"
	default:
		return "unknown"
	}
}

// Hook data structures

// DocumentCreatedData contains data for document creation events
type DocumentCreatedData struct {
	Collection  string          `json:"collection"`
	URI         string          `json:"uri"`
	ContentType string          `json:"content_type"`
	Author      string          `json:"author,omitempty"`
	Size        int64           `json:"size"`
	Metadata    ContentMetadata `json:"metadata"`
	Timestamp   time.Time       `json:"timestamp"`
}

// DocumentUpdatedData contains data for document update events
type DocumentUpdatedData struct {
	Collection  string          `json:"collection"`
	URI         string          `json:"uri"`
	ContentType string          `json:"content_type"`
	Author      string          `json:"author,omitempty"`
	OldSize     int64           `json:"old_size"`
	NewSize     int64           `json:"new_size"`
	Changes     []string        `json:"changes,omitempty"`
	Metadata    ContentMetadata `json:"metadata"`
	Timestamp   time.Time       `json:"timestamp"`
}

// DocumentDeletedData contains data for document deletion events
type DocumentDeletedData struct {
	Collection  string          `json:"collection"`
	URI         string          `json:"uri"`
	ContentType string          `json:"content_type"`
	Author      string          `json:"author,omitempty"`
	Size        int64           `json:"size"`
	Metadata    ContentMetadata `json:"metadata"`
	Timestamp   time.Time       `json:"timestamp"`
}

// DocumentPublishedData contains data for document publication events
type DocumentPublishedData struct {
	Collection  string          `json:"collection"`
	URI         string          `json:"uri"`
	ContentType string          `json:"content_type"`
	Author      string          `json:"author,omitempty"`
	Version     string          `json:"version,omitempty"`
	Metadata    ContentMetadata `json:"metadata"`
	Timestamp   time.Time       `json:"timestamp"`
}

// DocumentUnpublishedData contains data for document unpublication events
type DocumentUnpublishedData struct {
	Collection  string          `json:"collection"`
	URI         string          `json:"uri"`
	ContentType string          `json:"content_type"`
	Author      string          `json:"author,omitempty"`
	Reason      string          `json:"reason,omitempty"`
	Timestamp   time.Time       `json:"timestamp"`
}

// DocumentRenderedData contains data for document rendering events
type DocumentRenderedData struct {
	Collection  string        `json:"collection"`
	URI         string        `json:"uri"`
	Format      ContentFormat `json:"format"`
	Size        int64         `json:"size"`
	Duration    time.Duration `json:"duration"`
	CacheHit    bool          `json:"cache_hit"`
	Timestamp   time.Time     `json:"timestamp"`
}

// CollectionCreatedData contains data for collection creation events
type CollectionCreatedData struct {
	Name        string            `json:"name"`
	ContentType string            `json:"content_type"`
	Provider    string            `json:"provider"`
	Settings    map[string]any    `json:"settings,omitempty"`
	Timestamp   time.Time         `json:"timestamp"`
}

// CollectionDeletedData contains data for collection deletion events
type CollectionDeletedData struct {
	Name        string        `json:"name"`
	ItemCount   int64         `json:"item_count"`
	TotalSize   int64         `json:"total_size"`
	Timestamp   time.Time     `json:"timestamp"`
}

// TemplateAppliedData contains data for template application events
type TemplateAppliedData struct {
	TemplateName string        `json:"template_name"`
	URI          string        `json:"uri"`
	Collection   string        `json:"collection"`
	DataSize     int64         `json:"data_size"`
	OutputSize   int64         `json:"output_size"`
	Duration     time.Duration `json:"duration"`
	Timestamp    time.Time     `json:"timestamp"`
}

// SearchExecutedData contains data for search execution events
type SearchExecutedData struct {
	Query       ContentQuery  `json:"query"`
	ResultCount int64         `json:"result_count"`
	Duration    time.Duration `json:"duration"`
	Source      string        `json:"source,omitempty"`
	Timestamp   time.Time     `json:"timestamp"`
}

// ProviderRegisteredData contains data for provider registration events
type ProviderRegisteredData struct {
	ProviderKey  string    `json:"provider_key"`
	ProviderType string    `json:"provider_type"`
	Config       any       `json:"config,omitempty"`
	Timestamp    time.Time `json:"timestamp"`
}

// ProviderUnregisteredData contains data for provider unregistration events
type ProviderUnregisteredData struct {
	ProviderKey string    `json:"provider_key"`
	Reason      string    `json:"reason,omitempty"`
	Timestamp   time.Time `json:"timestamp"`
}

// ContractCreatedData contains data for contract creation events
type ContractCreatedData struct {
	CollectionName string    `json:"collection_name"`
	ContentType    string    `json:"content_type"`
	ProviderKey    string    `json:"provider_key"`
	Timestamp      time.Time `json:"timestamp"`
}

// ContractDestroyedData contains data for contract destruction events
type ContractDestroyedData struct {
	CollectionName string    `json:"collection_name"`
	Reason         string    `json:"reason,omitempty"`
	Timestamp      time.Time `json:"timestamp"`
}

// Hook emission helpers for the implementation

func (z *zDocumentContract[T]) emitDocumentCreatedHook(ctx context.Context, uri string, document T, metadata ContentMetadata) {
	data := DocumentCreatedData{
		Collection:  z.collectionName,
		URI:         uri,
		ContentType: string(z.contentType),
		Author:      extractAuthorFromContext(ctx),
		Size:        metadata.Size,
		Metadata:    metadata,
		Timestamp:   time.Now(),
	}
	
	capitan.Emit(ctx, DocumentCreated, "docula-service", data, nil)
}

func (z *zDocumentContract[T]) emitDocumentUpdatedHook(ctx context.Context, uri string, document T, metadata ContentMetadata) {
	data := DocumentUpdatedData{
		Collection:  z.collectionName,
		URI:         uri,
		ContentType: string(z.contentType),
		Author:      extractAuthorFromContext(ctx),
		NewSize:     metadata.Size,
		Metadata:    metadata,
		Timestamp:   time.Now(),
	}
	
	capitan.Emit(ctx, DocumentUpdated, "docula-service", data, nil)
}

func (z *zDocumentContract[T]) emitDocumentDeletedHook(ctx context.Context, uri string, document T) {
	data := DocumentDeletedData{
		Collection:  z.collectionName,
		URI:         uri,
		ContentType: string(z.contentType),
		Author:      extractAuthorFromContext(ctx),
		Timestamp:   time.Now(),
	}
	
	capitan.Emit(ctx, DocumentDeleted, "docula-service", data, nil)
}

func (z *zDocumentContract[T]) emitDocumentPublishedHook(ctx context.Context, uri string, document T, metadata ContentMetadata) {
	data := DocumentPublishedData{
		Collection:  z.collectionName,
		URI:         uri,
		ContentType: string(z.contentType),
		Author:      extractAuthorFromContext(ctx),
		Version:     metadata.Version,
		Metadata:    metadata,
		Timestamp:   time.Now(),
	}
	
	capitan.Emit(ctx, DocumentPublished, "docula-service", data, nil)
}

func (z *zDocumentContract[T]) emitDocumentUnpublishedHook(ctx context.Context, uri string, document T) {
	data := DocumentUnpublishedData{
		Collection:  z.collectionName,
		URI:         uri,
		ContentType: string(z.contentType),
		Author:      extractAuthorFromContext(ctx),
		Timestamp:   time.Now(),
	}
	
	capitan.Emit(ctx, DocumentUnpublished, "docula-service", data, nil)
}

func (z *zDocumentContract[T]) emitDocumentRenderedHook(ctx context.Context, uri string, format ContentFormat, size int64, duration time.Duration) {
	data := DocumentRenderedData{
		Collection: z.collectionName,
		URI:        uri,
		Format:     format,
		Size:       size,
		Duration:   duration,
		Timestamp:  time.Now(),
	}
	
	capitan.Emit(ctx, DocumentRendered, "docula-service", data, nil)
}

func (z *zDocumentContract[T]) emitSearchExecutedHook(ctx context.Context, query ContentQuery, resultCount int64, duration time.Duration) {
	data := SearchExecutedData{
		Query:       query,
		ResultCount: resultCount,
		Duration:    duration,
		Source:      z.collectionName,
		Timestamp:   time.Now(),
	}
	
	capitan.Emit(ctx, SearchExecuted, "docula-service", data, nil)
}

func (z *zDocumentContract[T]) emitTemplateAppliedHook(ctx context.Context, templateName, uri string, dataSize, outputSize int64, duration time.Duration) {
	data := TemplateAppliedData{
		TemplateName: templateName,
		URI:          uri,
		Collection:   z.collectionName,
		DataSize:     dataSize,
		OutputSize:   outputSize,
		Duration:     duration,
		Timestamp:    time.Now(),
	}
	
	capitan.Emit(ctx, TemplateApplied, "docula-service", data, nil)
}

// Helper function to extract author from context
func extractAuthorFromContext(ctx context.Context) string {
	// This would typically extract user information from context
	// For now, return a placeholder
	if author := ctx.Value("author"); author != nil {
		if authorStr, ok := author.(string); ok {
			return authorStr
		}
	}
	
	if userID := ctx.Value("user_id"); userID != nil {
		if userIDStr, ok := userID.(string); ok {
			return userIDStr
		}
	}
	
	return "system"
}

// Hook registration helpers for external services

// OnDocumentCreated registers a handler for document creation events
func OnDocumentCreated(handler capitan.InputHookFunc[DocumentCreatedData]) {
	capitan.RegisterInput(DocumentCreated, handler)
}

// OnDocumentUpdated registers a handler for document update events
func OnDocumentUpdated(handler capitan.InputHookFunc[DocumentUpdatedData]) {
	capitan.RegisterInput(DocumentUpdated, handler)
}

// OnDocumentPublished registers a handler for document publication events
func OnDocumentPublished(handler capitan.InputHookFunc[DocumentPublishedData]) {
	capitan.RegisterInput(DocumentPublished, handler)
}

// OnSearchExecuted registers a handler for search execution events
func OnSearchExecuted(handler capitan.InputHookFunc[SearchExecutedData]) {
	capitan.RegisterInput(SearchExecuted, handler)
}

// OnDocumentDeleted registers a handler for document deletion events
func OnDocumentDeleted(handler capitan.InputHookFunc[DocumentDeletedData]) {
	capitan.RegisterInput(DocumentDeleted, handler)
}

// OnProviderRegistered registers a handler for provider registration events
func OnProviderRegistered(handler capitan.InputHookFunc[ProviderRegisteredData]) {
	capitan.RegisterInput(ProviderRegistered, handler)
}

// EnableAutoTelemetry automatically emits telemetry events for all docula operations
func EnableAutoTelemetry() {
	// Document operation metrics
	OnDocumentCreated(func(data DocumentCreatedData) error {
		// Could emit to telemetry service automatically
		// telemetry.Counter("docula.documents.created").
		//     WithTag("collection", data.Collection).
		//     WithTag("content_type", data.ContentType).
		//     Increment()
		return nil
	})
	
	OnDocumentUpdated(func(data DocumentUpdatedData) error {
		// telemetry.Counter("docula.documents.updated").
		//     WithTag("collection", data.Collection).
		//     WithTag("content_type", data.ContentType).
		//     Increment()
		return nil
	})
	
	OnSearchExecuted(func(data SearchExecutedData) error {
		// telemetry.Histogram("docula.search.duration").
		//     WithTag("collection", data.Source).
		//     Observe(data.Duration.Seconds())
		//
		// telemetry.Counter("docula.search.executed").
		//     WithTag("result_count", fmt.Sprintf("%d", data.ResultCount)).
		//     Increment()
		return nil
	})
}

// EnableAutoIndexing automatically indexes content for search
func EnableAutoIndexing() {
	OnDocumentCreated(func(data DocumentCreatedData) error {
		// Could automatically index new documents in search engine
		// searchIndex.IndexDocument(data.URI, data.Metadata)
		return nil
	})
	
	OnDocumentUpdated(func(data DocumentUpdatedData) error {
		// searchIndex.UpdateDocument(data.URI, data.Metadata)
		return nil
	})
	
	OnDocumentDeleted(func(data DocumentDeletedData) error {
		// searchIndex.RemoveDocument(data.URI)
		return nil
	})
}

// EnableAutoBackup automatically backs up content changes
func EnableAutoBackup() {
	OnDocumentCreated(func(data DocumentCreatedData) error {
		// Could automatically backup new documents
		// backup.BackupDocument(data.URI, data.Metadata)
		return nil
	})
	
	OnDocumentUpdated(func(data DocumentUpdatedData) error {
		// backup.BackupDocument(data.URI, data.Metadata)
		return nil
	})
}