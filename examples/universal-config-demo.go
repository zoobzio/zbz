package main

import (
	"context"
	"fmt"
	"time"
	
	// Import services
	"zbz/cache"
	"zbz/hodor"
	"zbz/zlog"
	
	// Import providers for each service
	cacheredis "zbz/providers/cache-redis"
	hodors3 "zbz/providers/hodor-s3"
	zlogzap "zbz/providers/zlog-zap"
	zlogzerolog "zbz/providers/zlog-zerolog"
)

func main() {
	fmt.Println("🚀 ZBZ Universal Contract Pattern Demo")
	fmt.Println("====================================")

	// Universal Configurations - provider-agnostic!
	cacheConfig := cache.CacheConfig{
		DefaultTTL:       1 * time.Hour,
		KeyPrefix:        "demo:",
		Serialization:    "json",
		URL:              "redis://localhost:6379",
		PoolSize:         10,
		MaxRetries:       3,
		ReadTimeout:      5 * time.Second,
		WriteTimeout:     3 * time.Second,
		EnablePipelining: true,
		MaxSize:          100 * 1024 * 1024, // Used by memory provider
		BaseDir:          "/tmp/demo-cache",  // Used by filesystem provider
	}

	hodorConfig := hodor.HodorConfig{
		BasePath:   "demo/",
		BufferSize: 1024 * 1024,
		Bucket:     "my-demo-bucket",
		Region:     "us-west-2",
		AccessKey:  "demo-key",
		SecretKey:  "demo-secret",
		BaseDir:    "/tmp/demo-storage", // Used by filesystem provider
		MaxRetries: 3,
		Timeout:    30 * time.Second,
	}

	zlogConfig := zlog.ZlogConfig{
		Name:        "demo-app",
		Level:       "info",
		Format:      "json",
		Development: false,
		OutputFile:  "/tmp/demo.log", // Used by file-based providers
		BufferSize:  1024,
		Sampling: &zlog.SamplingConfig{
			Initial:    10,
			Thereafter: 100,
		},
	}

	// 1. Cache Service with Redis
	fmt.Println("\n📦 Setting up Cache with Redis...")
	cacheContract, err := cacheredis.NewRedisCache(cacheConfig)
	if err != nil {
		fmt.Printf("❌ Failed to create Redis cache: %v\n", err)
		return
	}

	// Register as singleton
	err = cacheContract.Register()
	if err != nil {
		fmt.Printf("❌ Failed to register cache: %v\n", err)
		return
	}

	// Use package functions (singleton)
	err = cache.Set(context.Background(), "test", []byte("hello cache"))
	if err != nil {
		fmt.Printf("❌ Cache set failed: %v\n", err)
	} else {
		fmt.Println("✅ Cache set successful")
	}

	// Type-safe native access
	redisClient := cacheContract.Native() // redis.Cmdable - no casting!
	fmt.Printf("✅ Redis client type: %T\n", redisClient)

	// 2. Storage Service with S3
	fmt.Println("\n📁 Setting up Storage with S3...")
	storageContract, err := hodors3.NewS3Storage(hodorConfig)
	if err != nil {
		fmt.Printf("❌ Failed to create S3 storage: %v\n", err)
		return
	}

	err = storageContract.Register()
	if err != nil {
		fmt.Printf("❌ Failed to register storage: %v\n", err)
		return
	}

	// Use package functions (singleton)
	err = hodor.Set("demo.txt", []byte("hello storage"), 0)
	if err != nil {
		fmt.Printf("❌ Storage set failed: %v\n", err)
	} else {
		fmt.Println("✅ Storage set successful")
	}

	// Type-safe native access
	s3Client := storageContract.Native() // *s3.S3 - no casting!
	fmt.Printf("✅ S3 client type: %T\n", s3Client)

	// 3. Logging Service with Zap
	fmt.Println("\n📝 Setting up Logging with Zap...")
	logContract, err := zlogzap.NewZapLogger(zlogConfig)
	if err != nil {
		fmt.Printf("❌ Failed to create Zap logger: %v\n", err)
		return
	}

	err = logContract.Register()
	if err != nil {
		fmt.Printf("❌ Failed to register logger: %v\n", err)
		return
	}

	// Use package functions (singleton)
	zlog.Info("Demo logging working", zlog.String("status", "success"))
	fmt.Println("✅ Logging successful")

	// Type-safe native access
	zapLogger := logContract.Native() // *zap.Logger - no casting!
	fmt.Printf("✅ Zap logger type: %T\n", zapLogger)

	// 4. Alternative: Zerolog Provider Example
	fmt.Println("\n📝 Alternative: Zerolog Provider...")
	zerologContract, err := zlogzerolog.NewZerologLogger(zlogConfig)
	if err != nil {
		fmt.Printf("❌ Failed to create Zerolog logger: %v\n", err)
	} else {
		zerologLogger := zerologContract.Native() // *zerolog.Logger - no casting!
		fmt.Printf("✅ Zerolog logger type: %T\n", zerologLogger)
	}

	fmt.Println("\n🎉 Universal Contract Pattern Demo Complete!")
	fmt.Println("==========================================")
	fmt.Println("✅ Same config structure works across all providers")
	fmt.Println("✅ Type-safe native access without casting")
	fmt.Println("✅ Package-level functions use singletons")
	fmt.Println("✅ Provider selection happens in Go code, not config")
	fmt.Println("✅ Hot-reload ready with flux integration")
}