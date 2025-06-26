# Cache Filesystem Provider

Persistent file-based cache provider for the ZBZ BYOC cache system.

## Features

- **Persistent Storage**: Data survives application restarts
- **TTL Support**: File-based expiration with metadata headers
- **Atomic Writes**: Uses temp files for crash-safe operations
- **Subdirectory Organization**: Avoids filesystem limitations with too many files
- **Safe Filenames**: MD5 hashing for key-to-filename conversion
- **Configurable Permissions**: Set file permissions for security

## Installation

```bash
go get zbz/providers/cache-filesystem
```

## Usage

```go
import "zbz/providers/cache-filesystem"

// Provider will auto-register as "filesystem"
provider, err := cache.NewProvider("filesystem", map[string]interface{}{
    "base_dir":    "/var/cache/myapp",
    "permissions": 0644,
})

// Use with cache service
userCache := cache.NewContract[User]("users", provider)
service := userCache.Cache()
```

## Configuration

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `base_dir` | string | /tmp/zbz-cache | Base directory for cache files |
| `permissions` | int | 0644 | File permissions (octal) |

## File Structure

The provider organizes files to avoid filesystem limitations:

```
/var/cache/myapp/
├── 01/
│   ├── 0123456789abcdef0123456789abcdef.cache
│   └── 0987654321fedcba0987654321fedcba.cache
├── 02/
│   └── 02468ace13579bdf02468ace13579bdf.cache
└── .../
```

Each cache file contains:
- **Header (8 bytes)**: Expiration timestamp or "00000000" for no expiration
- **Data**: The actual cached value

## API Examples

```go
// Basic operations
service.SetStruct(ctx, "user:123", user, 1*time.Hour)
service.GetStruct(ctx, "user:123", &user)
service.Delete(ctx, "user:123")

// Batch operations
users := map[string]User{"user1": user1, "user2": user2}
service.SetStructMulti(ctx, users, 2*time.Hour)
retrievedUsers, _ := service.GetStructMulti(ctx, []string{"user1", "user2"})

// Native filesystem access
native := service.NativeClient().(struct {
    BaseDir     string
    Permissions os.FileMode
})
fmt.Println("Cache directory:", native.BaseDir)

// Direct file operations (if needed)
cachePath := filepath.Join(native.BaseDir, "custom.data")
os.WriteFile(cachePath, data, native.Permissions)
```

## Performance

- **Write Speed**: ~10k ops/sec (SSD), ~1k ops/sec (HDD)
- **Read Speed**: ~50k ops/sec (SSD), ~5k ops/sec (HDD)
- **File Overhead**: ~8 bytes per cache entry (TTL header)
- **Directory Limit**: 256 subdirectories (prevents filesystem slowdown)

## Use Cases

### Development & Testing
```go
// Temporary cache for development
provider, _ := cache.NewProvider("filesystem", map[string]interface{}{
    "base_dir": "/tmp/dev-cache",
})
```

### Production Persistent Cache
```go
// Production cache with proper permissions
provider, _ := cache.NewProvider("filesystem", map[string]interface{}{
    "base_dir":    "/var/cache/myapp",
    "permissions": 0600, // Owner read/write only
})
```

### Backup Cache Layer
```go
// Use as fallback when Redis is unavailable
primaryCache := cache.NewContract[User]("users", redisProvider)
backupCache := cache.NewContract[User]("users-backup", filesystemProvider)
```

## Considerations

### Advantages
- **Persistence**: Data survives restarts and crashes  
- **No Dependencies**: Works without external services
- **Simple**: Easy to inspect and debug cache contents
- **Atomic**: Crash-safe writes with temp files

### Limitations  
- **Performance**: Slower than memory/Redis for high-throughput
- **Concurrency**: Limited by filesystem performance
- **Key Reversal**: Cannot easily list original keys (uses MD5 hashes)
- **Cleanup**: Manual cleanup of expired files needed in some cases

### Best Practices
- Use SSDs for better performance
- Set appropriate file permissions for security
- Monitor disk space usage
- Consider cleanup scripts for very large caches
- Use as backup layer rather than primary cache for high-traffic