package hodor

import (
	"fmt"
	"time"

	"zbz/cereal"
	"zbz/zlog"
)

// Types for reactive operations
type SubscriptionID string

type ChangeEvent struct {
	Key       string    `json:"key"`
	Operation string    `json:"operation"` // "create", "update", "delete"
	Timestamp time.Time `json:"timestamp"`
	ETag      string    `json:"etag,omitempty"`    // For change detection
	Size      int64     `json:"size,omitempty"`    // File size
}

type ChangeCallback func(event ChangeEvent)

// HodorProvider defines the interface that storage providers implement
// This is the standardized interface the service uses to interact with storage backends
type HodorProvider interface {
	// Core storage operations
	Get(key string) ([]byte, error)
	Set(key string, data []byte, ttl time.Duration) error
	Delete(key string) error
	Exists(key string) (bool, error)
	List(prefix string) ([]string, error)
	
	// Metadata operations
	Stat(key string) (FileInfo, error)
	
	// Reactive operations
	Subscribe(key string, callback ChangeCallback) (SubscriptionID, error)
	Unsubscribe(id SubscriptionID) error
	
	// Provider info
	GetProvider() string // Returns provider name for debugging
	Close() error       // Cleanup resources
}

// Private concrete hodor instance
var hodor *zHodor

// HodorConfig defines provider-agnostic hodor configuration
type HodorConfig struct {
	// Service configuration
	BasePath    string        `yaml:"base_path" json:"base_path"`
	BufferSize  int64         `yaml:"buffer_size" json:"buffer_size"`
	DefaultTTL  time.Duration `yaml:"default_ttl" json:"default_ttl"`
	
	// Cloud provider settings (S3/GCS/MinIO)
	Bucket    string `yaml:"bucket" json:"bucket"`
	Region    string `yaml:"region" json:"region"`
	AccessKey string `yaml:"access_key" json:"access_key"`
	SecretKey string `yaml:"secret_key" json:"secret_key"`
	Endpoint  string `yaml:"endpoint" json:"endpoint"` // For MinIO/custom S3
	
	// Filesystem settings
	BaseDir     string `yaml:"base_dir" json:"base_dir"`
	Permissions int    `yaml:"permissions" json:"permissions"`
	
	// Performance settings
	MaxRetries int           `yaml:"max_retries" json:"max_retries"`
	Timeout    time.Duration `yaml:"timeout" json:"timeout"`
	
	// Feature flags
	EnableWatching bool `yaml:"enable_watching" json:"enable_watching"`
	EnableSSL      bool `yaml:"enable_ssl" json:"enable_ssl"`
}

// DefaultConfig returns sensible defaults
func DefaultConfig() HodorConfig {
	return HodorConfig{
		BufferSize:     1024 * 1024, // 1MB
		DefaultTTL:     0,            // No expiration
		Permissions:    0644,
		MaxRetries:     3,
		Timeout:        30 * time.Second,
		EnableWatching: true,
	}
}

// zHodor is the singleton service layer that orchestrates storage operations
// Like zlog/cache singletons, this manages provider abstraction + cereal metadata serialization
type zHodor struct {
	provider     HodorProvider         // Backend provider wrapper
	serializer   cereal.CerealProvider // Cereal handles metadata serialization
	config       HodorConfig           // Service configuration
	contractName string                // Name of the contract that created this singleton
}



// configureFromContract initializes the singleton from a contract's registration
func configureFromContract(contractName string, provider HodorProvider, config HodorConfig) error {
	// Check if we need to replace existing singleton
	if hodor != nil && hodor.contractName != contractName {
		zlog.Info("Replacing hodor singleton",
			zlog.String("old_contract", hodor.contractName),
			zlog.String("new_contract", contractName))
		
		// Close old provider
		if err := hodor.provider.Close(); err != nil {
			zlog.Warn("Failed to close old provider", zlog.Err(err))
		}
	} else if hodor != nil && hodor.contractName == contractName {
		// Same contract, no need to replace
		return nil
	}

	// Set up cereal serialization for metadata handling
	cerealConfig := cereal.DefaultConfig()
	cerealConfig.Name = "hodor-serializer"
	cerealConfig.DefaultFormat = "json" // Use JSON for object metadata
	cerealConfig.EnableCaching = true   // Cache metadata serialization for performance
	cerealConfig.EnableScoping = true   // Enable scoped metadata for access control
	
	// Create JSON provider for metadata serialization
	cerealContract := cereal.NewJSONProvider(cerealConfig)
	cerealProvider := cerealContract.Provider()

	// Create service singleton
	hodor = &zHodor{
		provider:     provider,
		serializer:   cerealProvider, // Cereal handles metadata serialization
		config:       config,
		contractName: contractName,
	}

	zlog.Info("Hodor service configured from contract",
		zlog.String("contract", contractName),
		zlog.String("provider", provider.GetProvider()))

	return nil
}

// Metadata operations using cereal

// CombinedPayload wraps data with metadata for storage
type CombinedPayload struct {
	Data     []byte `json:"data"`
	Metadata []byte `json:"metadata"`
}

// setWithMetadata stores data with structured metadata using cereal
func (h *zHodor) setWithMetadata(key string, data []byte, metadata interface{}) error {
	// Serialize metadata using cereal
	metadataBytes, err := h.serializer.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("failed to serialize metadata with cereal: %w", err)
	}
	
	// Create combined payload
	payload := CombinedPayload{
		Data:     data,
		Metadata: metadataBytes,
	}
	
	// Serialize the combined payload
	payloadBytes, err := h.serializer.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to serialize payload with cereal: %w", err)
	}
	
	// Store in provider
	return h.provider.Set(key, payloadBytes, 0)
}

// getWithMetadata retrieves data and deserializes metadata using cereal
func (h *zHodor) getWithMetadata(key string, metadata interface{}) ([]byte, error) {
	// Get from provider
	payloadBytes, err := h.provider.Get(key)
	if err != nil {
		return nil, err
	}
	
	// Deserialize combined payload
	var payload CombinedPayload
	err = h.serializer.Unmarshal(payloadBytes, &payload)
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize payload with cereal: %w", err)
	}
	
	// Deserialize metadata if provided
	if metadata != nil {
		err = h.serializer.Unmarshal(payload.Metadata, metadata)
		if err != nil {
			return nil, fmt.Errorf("failed to deserialize metadata with cereal: %w", err)
		}
	}
	
	return payload.Data, nil
}

// setJSON stores a JSON object using cereal serialization
func (h *zHodor) setJSON(key string, object interface{}) error {
	// Serialize object using cereal
	data, err := h.serializer.Marshal(object)
	if err != nil {
		return fmt.Errorf("failed to serialize object with cereal: %w", err)
	}
	
	// Store in provider
	return h.provider.Set(key, data, 0)
}

// getJSON retrieves and deserializes a JSON object using cereal
func (h *zHodor) getJSON(key string, target interface{}) error {
	// Get from provider
	data, err := h.provider.Get(key)
	if err != nil {
		return err
	}
	
	// Deserialize using cereal
	return h.serializer.Unmarshal(data, target)
}

// setJSONScoped stores a JSON object with field-level scoping
func (h *zHodor) setJSONScoped(key string, object interface{}, userPermissions []string) error {
	// Serialize object with scoping using cereal
	data, err := h.serializer.MarshalScoped(object, userPermissions)
	if err != nil {
		return fmt.Errorf("failed to serialize scoped object with cereal: %w", err)
	}
	
	// Store in provider
	return h.provider.Set(key, data, 0)
}

// getJSONScoped retrieves and deserializes a JSON object with scoping
func (h *zHodor) getJSONScoped(key string, target interface{}, userPermissions []string) error {
	// Get from provider
	data, err := h.provider.Get(key)
	if err != nil {
		return err
	}
	
	// Deserialize with scoping using cereal
	return h.serializer.UnmarshalScoped(data, target, userPermissions, "read")
}





// Provider returns the current hodor provider
func (h *zHodor) Provider() HodorProvider {
	if h == nil {
		return nil
	}
	return h.provider
}

// Config returns the current hodor configuration
func (h *zHodor) Config() HodorConfig {
	if h == nil {
		return HodorConfig{}
	}
	return h.config
}

// Close shuts down the hodor service
func (h *zHodor) Close() error {
	if h == nil {
		return nil
	}
	
	zlog.Info("Closing hodor service")
	err := h.provider.Close()
	hodor = nil // Clear singleton
	return err
}