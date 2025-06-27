# Depot Usage Examples

## New Architecture Usage

### Basic Contract Creation and Mounting

```go
// Create a memory storage contract
contract := depot.NewMemory(map[string]interface{}{})

// Mount it with an alias
err := contract.Mount("my-storage")
if err != nil {
    log.Fatal(err)
}

// Use the contract for storage operations
err = contract.Set("hello.txt", []byte("Hello World"), 0)
data, err := contract.Get("hello.txt")
files, err := contract.List("")

// Get the underlying client for advanced operations
client := contract.Client() // Returns map[string][]byte for memory provider

// Unmount when done
err = contract.Unmount()
```

### Package-Level Mount Management

```go
// Quick memory mount
err := depot.QuickMemory("temp-storage")

// List all mounts
mounts := depot.List()
for _, mount := range mounts {
    fmt.Printf("Mount: %s at %s\n", mount.Alias, mount.MountPath)
}

// Check mount status
status, err := depot.Status("temp-storage")

// Unmount
err = depot.Unmount("temp-storage")
```

### Future S3 Provider Usage

```go
// This will be available once S3 provider is implemented
s3Contract := s3.New(s3.Config{
    Bucket: "my-bucket",
    Region: "us-east-1",
})

err := s3Contract.Mount("s3-data")

// Use S3 operations through contract
data, err := s3Contract.Get("file.txt")

// Or get S3 client directly
s3Client := s3Contract.Client() // Returns *s3.Client
```

## Architecture Benefits

- **Independent Contracts**: Each contract is self-contained
- **Self-Mounting**: Contracts handle their own mount lifecycle  
- **Dual Interface**: Both standardized operations and direct client access
- **Service Singleton**: Mount registry kept separate from contracts
- **Provider Flexibility**: Easy to add new storage backends