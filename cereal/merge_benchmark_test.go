package cereal

import (
	"testing"
	"time"
)

// Benchmark test for merge performance
func BenchmarkBasicMerge(b *testing.B) {
	defaults := UserConfig{
		Name:  "Default User",
		Email: "default@example.com",
		Settings: map[string]string{
			"theme": "dark",
			"lang":  "en",
		},
		Features: []string{"basic", "auth"},
		Debug:    false,
	}

	overrides := UserConfig{
		Name:  "John Doe",
		Settings: map[string]string{
			"theme":  "light",
			"locale": "us",
		},
		Features: []string{"basic", "premium"},
		Debug:    true,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Merge(defaults, overrides)
	}
}

func BenchmarkComplexMerge(b *testing.B) {
	type ComplexConfig struct {
		Database DatabaseConfig
		Cache    struct {
			TTL     time.Duration
			MaxSize int64
			Servers []string
		}
		Features map[string]bool
		Metadata *Metadata
	}

	defaults := ComplexConfig{
		Database: DatabaseConfig{
			Host:           "localhost",
			Port:           5432,
			MaxConnections: 10,
			Timeout:        30 * time.Second,
			Features:       []string{"ssl", "pooling"},
			ConnectionPool: map[string]int{"read": 5, "write": 2},
		},
		Cache: struct {
			TTL     time.Duration
			MaxSize int64
			Servers []string
		}{
			TTL:     time.Hour,
			MaxSize: 100 * 1024 * 1024,
			Servers: []string{"cache1", "cache2"},
		},
		Features: map[string]bool{
			"auth":    true,
			"logging": true,
		},
		Metadata: &Metadata{
			Version: "1.0",
			Tags:    []string{"default", "system"},
		},
	}

	overrides := ComplexConfig{
		Database: DatabaseConfig{
			MaxConnections: 50,
			Features:       []string{"ssl", "monitoring"},
		},
		Cache: struct {
			TTL     time.Duration
			MaxSize int64
			Servers []string
		}{
			TTL:     2 * time.Hour,
			Servers: []string{"cache3"},
		},
		Features: map[string]bool{
			"analytics": true,
		},
		Metadata: &Metadata{
			Version: "2.0",
			Tags:    []string{"user"},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Merge(defaults, overrides)
	}
}

func BenchmarkMergeMultiple(b *testing.B) {
	configs := make([]UserConfig, 10)
	for i := range configs {
		configs[i] = UserConfig{
			Name:     "Config" + string(rune('0'+i)),
			Features: []string{"feature" + string(rune('0'+i))},
			Settings: map[string]string{"key" + string(rune('0'+i)): "value" + string(rune('0'+i))},
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = MergeMultiple(configs...)
	}
}