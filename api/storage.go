package zbz

import "zbz/hodor"

// Export storage types and functions for public API

// BucketService is the main storage interface
type BucketService = hodor.BucketService

// BucketDriver defines the interface for bucket driver implementations
type BucketDriver = hodor.BucketDriver

// GetDriver returns a registered bucket driver by name
func GetDriver(name string) (BucketDriver, bool) {
	return hodor.GetDriver(name)
}

// ListDrivers returns all registered driver names
func ListDrivers() []string {
	return hodor.ListDrivers()
}