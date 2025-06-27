package cereal

import (
	"reflect"
	"sync"
)

// zScopeCache caches reflection-based scope metadata for performance
type zScopeCache struct {
	cache map[reflect.Type]map[string][]string
	mu    sync.RWMutex
}

// newScopeCache creates a new scope cache
func newScopeCache() *zScopeCache {
	return &zScopeCache{
		cache: make(map[reflect.Type]map[string][]string),
	}
}

// getFieldScopes returns cached scope metadata for a type
func (sc *zScopeCache) getFieldScopes(t reflect.Type) (map[string][]string, bool) {
	sc.mu.RLock()
	defer sc.mu.RUnlock()
	scopes, exists := sc.cache[t]
	return scopes, exists
}

// setFieldScopes caches scope metadata for a type
func (sc *zScopeCache) setFieldScopes(t reflect.Type, scopes map[string][]string) {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	sc.cache[t] = scopes
}
