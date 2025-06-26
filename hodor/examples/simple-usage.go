package main

import (
	"fmt"
	"time"

	"zbz/hodor"
	
	// Import the providers you want to use
	hodors3 "zbz/providers/hodor-s3"
	hodormemory "zbz/providers/hodor-memory"
)

func main() {
	// Create configuration
	config := hodor.HodorConfig{
		BasePath:   "myapp/",
		BufferSize: 1024 * 1024, // 1MB
		Bucket:     "my-storage-bucket",
		Region:     "us-west-2",
		AccessKey:  "YOUR_ACCESS_KEY",
		SecretKey:  "YOUR_SECRET_KEY",
	}

	// Example 1: S3 storage with type-safe native access
	s3Contract, err := hodors3.NewS3Storage(config)
	if err != nil {
		fmt.Printf("Failed to create S3 storage: %v\n", err)
		// Fall back to memory for demo
		memoryContract, err := hodormemory.NewMemoryStorage(config)
		if err != nil {
			fmt.Printf("Failed to create memory storage: %v\n", err)
			return
		}
		
		// Register memory as singleton
		err = memoryContract.Register()
		if err != nil {
			fmt.Printf("Failed to register memory storage: %v\n", err)
			return
		}
		
		fmt.Println("✅ Using memory storage as fallback")
	} else {
		// Register S3 as global singleton
		err = s3Contract.Register()
		if err != nil {
			fmt.Printf("Failed to register S3 storage: %v\n", err)
			return
		}

		// Get type-safe native client (no casting!)
		s3Client := s3Contract.Native() // *s3.S3
		fmt.Printf("S3 client type: %T\n", s3Client)
		
		fmt.Println("✅ Using S3 storage")
	}

	// Use package-level functions (uses the singleton)
	err = hodor.Set("test-key", []byte("hello world"), 1*time.Hour)
	if err != nil {
		fmt.Printf("Failed to set: %v\n", err)
	} else {
		fmt.Println("✅ Data stored successfully")
	}

	// Retrieve data
	data, err := hodor.Get("test-key")
	if err != nil {
		fmt.Printf("Failed to get: %v\n", err)
	} else {
		fmt.Printf("✅ Retrieved: %s\n", string(data))
	}

	// List keys
	keys, err := hodor.List("test-")
	if err != nil {
		fmt.Printf("Failed to list: %v\n", err)
	} else {
		fmt.Printf("✅ Found keys: %v\n", keys)
	}

	fmt.Println("🎉 Hodor singleton pattern working with type safety!")
}