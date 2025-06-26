package main

import (
	"context"
	"fmt"
	"time"

	"zbz/cache"
	
	// Import the providers you want to use
	cacheredis "zbz/providers/cache-redis"
	cachememory "zbz/providers/cache-memory"
	cachefilesystem "zbz/providers/cache-filesystem"
)

type User struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

func main() {
	// Create configuration
	config := cache.CacheConfig{
		DefaultTTL:    1 * time.Hour,
		KeyPrefix:     "myapp:",
		Serialization: "json",
		URL:           "redis://localhost:6379",
		MaxSize:       100 * 1024 * 1024,
	}

	// Example 1: Redis cache with type-safe native access
	redisContract, err := cacheredis.NewRedisCache(config)
	if err != nil {
		fmt.Printf("Failed to create Redis cache: %v\n", err)
		return
	}

	// Register as global singleton
	err = redisContract.Register()
	if err != nil {
		fmt.Printf("Failed to register Redis cache: %v\n", err)
		return
	}

	// Use package-level functions (uses the singleton)
	ctx := context.Background()
	err = cache.Set(ctx, "test", []byte("hello"))
	if err != nil {
		fmt.Printf("Failed to set: %v\n", err)
	}

	// Get type-safe native client (no casting!)
	redisClient := redisContract.Native() // redis.Cmdable
	fmt.Printf("Redis client type: %T\n", redisClient)

	// Example 2: Switch to memory cache
	memoryContract, err := cachememory.NewMemoryCache(config)
	if err != nil {
		fmt.Printf("Failed to create memory cache: %v\n", err)
		return
	}

	// Register as singleton (replaces Redis)
	err = memoryContract.Register()
	if err != nil {
		fmt.Printf("Failed to register memory cache: %v\n", err)
		return
	}

	// Package functions now use memory cache
	err = cache.Set(ctx, "test2", []byte("world"))
	if err != nil {
		fmt.Printf("Failed to set: %v\n", err)
	}

	// Get memory provider for direct access
	memoryProvider := memoryContract.Native() // *MemoryProvider
	fmt.Printf("Memory provider type: %T\n", memoryProvider)

	// Example 3: Use contract directly (no singleton)
	fsContract, err := cachefilesystem.NewFilesystemCache(config)
	if err != nil {
		fmt.Printf("Failed to create filesystem cache: %v\n", err)
		return
	}

	// Use contract directly without registering
	fsProvider := fsContract.Provider()
	err = fsProvider.Set(ctx, "direct", []byte("test"), time.Hour)
	if err != nil {
		fmt.Printf("Failed to set directly: %v\n", err)
	}

	fmt.Println("✅ All cache types working with type safety!")
}