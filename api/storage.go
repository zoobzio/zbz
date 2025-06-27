package zbz

import "zbz/depot"

// Export storage types and functions for public API

// BucketService is the main storage interface
type BucketService = depot.BucketService

// BucketDriver defines the interface for bucket driver implementations
type BucketDriver = depot.BucketDriver

// GetDriver returns a registered bucket driver by name
func GetDriver(name string) (BucketDriver, bool) {
	return depot.GetDriver(name)
}

// ListDrivers returns all registered driver names
func ListDrivers() []string {
	return depot.ListDrivers()
}