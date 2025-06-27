package depot

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

// DepotProvider defines the interface that storage providers implement
// This is the standardized interface the service uses to interact with storage backends
type DepotProvider interface {
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

// Private concrete depot instance
var depot *zDepot

// DepotConfig defines provider-agnostic depot configuration
type DepotConfig struct {
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
func DefaultConfig() DepotConfig {
	return DepotConfig{
		BufferSize:     1024 * 1024, // 1MB
		DefaultTTL:     0,            // No expiration
		Permissions:    0644,
		MaxRetries:     3,
		Timeout:        30 * time.Second,
		EnableWatching: true,
	}
}

// zDepot is the singleton service layer that orchestrates storage operations
// Like zlog/cache singletons, this manages provider abstraction + cereal metadata serialization
type zDepot struct {
	provider     DepotProvider         // Backend provider wrapper
	config       DepotConfig           // Service configuration
	contractName string                // Name of the contract that created this singleton
}



// configureFromContract initializes the singleton from a contract's registration
func configureFromContract(contractName string, provider DepotProvider, config DepotConfig) error {
	// Check if we need to replace existing singleton
	if depot != nil && depot.contractName != contractName {
		zlog.Info("Replacing depot singleton",
			zlog.String("old_contract", depot.contractName),
			zlog.String("new_contract", contractName))
		
		// Close old provider
		if err := depot.provider.Close(); err != nil {
			zlog.Warn("Failed to close old provider", zlog.Err(err))
		}
	} else if depot != nil && depot.contractName == contractName {
		// Same contract, no need to replace
		return nil
	}


	// Create service singleton
	depot = &zDepot{
		provider:     provider,
		config:       config,
		contractName: contractName,
	}

	zlog.Info("Depot service configured from contract",
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
func (h *zDepot) setWithMetadata(key string, data []byte, metadata interface{}) error {
	// Serialize metadata using cereal
	metadataBytes, err := cereal.JSON.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("failed to serialize metadata with cereal: %w", err)
	}
	
	// Create combined payload
	payload := CombinedPayload{
		Data:     data,
		Metadata: metadataBytes,
	}
	
	// Serialize the combined payload
	payloadBytes, err := cereal.JSON.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to serialize payload with cereal: %w", err)
	}
	
	// Store in provider
	return h.provider.Set(key, payloadBytes, 0)
}

// getWithMetadata retrieves data and deserializes metadata using cereal
func (h *zDepot) getWithMetadata(key string, metadata interface{}) ([]byte, error) {
	// Get from provider
	payloadBytes, err := h.provider.Get(key)
	if err != nil {
		return nil, err
	}
	
	// Deserialize combined payload
	var payload CombinedPayload
	err = cereal.JSON.Unmarshal(payloadBytes, &payload)
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize payload with cereal: %w", err)
	}
	
	// Deserialize metadata if provided
	if metadata != nil {
		err = cereal.JSON.Unmarshal(payload.Metadata, metadata)
		if err != nil {
			return nil, fmt.Errorf("failed to deserialize metadata with cereal: %w", err)
		}
	}
	
	return payload.Data, nil
}

// setJSON stores a JSON object using cereal serialization
func (h *zDepot) setJSON(key string, object interface{}) error {
	// Serialize object using cereal
	data, err := cereal.JSON.Marshal(object)
	if err != nil {
		return fmt.Errorf("failed to serialize object with cereal: %w", err)
	}
	
	// Store in provider
	return h.provider.Set(key, data, 0)
}

// getJSON retrieves and deserializes a JSON object using cereal
func (h *zDepot) getJSON(key string, target interface{}) error {
	// Get from provider
	data, err := h.provider.Get(key)
	if err != nil {
		return err
	}
	
	// Deserialize using cereal
	return cereal.JSON.Unmarshal(data, target)
}

// setJSONScoped stores a JSON object with field-level scoping
func (h *zDepot) setJSONScoped(key string, object interface{}, userPermissions []string) error {
	// Serialize object with scoping using cereal
	data, err := cereal.Marshal(object, userPermissions...)
	if err != nil {
		return fmt.Errorf("failed to serialize scoped object with cereal: %w", err)
	}
	
	// Store in provider
	return h.provider.Set(key, data, 0)
}

// getJSONScoped retrieves and deserializes a JSON object with scoping
func (h *zDepot) getJSONScoped(key string, target interface{}, userPermissions []string) error {
	// Get from provider
	data, err := h.provider.Get(key)
	if err != nil {
		return err
	}
	
	// Deserialize with scoping using cereal
	return cereal.Unmarshal(data, target, userPermissions...)
}





// Provider returns the current depot provider
func (h *zDepot) Provider() DepotProvider {
	if h == nil {
		return nil
	}
	return h.provider
}

// Config returns the current depot configuration
func (h *zDepot) Config() DepotConfig {
	if h == nil {
		return DepotConfig{}
	}
	return h.config
}

// Close shuts down the depot service
func (h *zDepot) Close() error {
	if h == nil {
		return nil
	}
	
	zlog.Info("Closing depot service")
	err := h.provider.Close()
	depot = nil // Clear singleton
	return err
}