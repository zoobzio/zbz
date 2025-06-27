package depot

import (
	"sync"
	"time"
)

// BucketService defines the interface for key-value storage with TTL support
type BucketService interface {
	Get(key string) ([]byte, error)
	Set(key string, data []byte, ttl time.Duration) error
	Delete(key string) error
	Exists(key string) (bool, error)
	List(prefix string) ([]string, error)
}

// BucketDriver defines the interface for bucket driver implementations
type BucketDriver interface {
	DriverName() string
	Connect(config map[string]any) (BucketService, error)
}

// Driver registry for pluggable bucket implementations
var drivers = make(map[string]BucketDriver)
var driverMutex sync.RWMutex

// RegisterDriver allows bucket drivers to register themselves
func RegisterDriver(name string, driver BucketDriver) {
	driverMutex.Lock()
	defer driverMutex.Unlock()
	drivers[name] = driver
}

// GetDriver returns a registered bucket driver by name
func GetDriver(name string) (BucketDriver, bool) {
	driverMutex.RLock()
	defer driverMutex.RUnlock()
	driver, exists := drivers[name]
	return driver, exists
}

// ListDrivers returns all registered driver names
func ListDrivers() []string {
	driverMutex.RLock()
	defer driverMutex.RUnlock()
	names := make([]string, 0, len(drivers))
	for name := range drivers {
		names = append(names, name)
	}
	return names
}
