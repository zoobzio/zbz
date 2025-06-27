# ZLog Zap Provider

A zap implementation for the ZBZ logging framework that provides both native zap logger access and the common zlog provider interface.

## Features

- **Self-registering**: Automatically becomes the active logger when created
- **Dual access**: Get both `*zap.Logger` (native) and `zlog.ZlogProvider` (common interface)
- **Type-safe**: Full type safety with no casting required
- **Configuration-driven**: Support for development and production presets
- **Field conversion**: Automatic conversion between zlog and zap field types

## Usage

### Quick Start

```go
import "zbz/providers/zlog-zap"

// Development logging (console output, debug level)
contract := zap.NewDevelopment()

// Production logging (JSON output, info level)  
contract := zap.NewProduction()
```

### Custom Configuration

```go
contract := zap.New(zlog.Config{
    Name:    "my-app",
    Level:   zlog.DEBUG,
    Format:  "console", 
    Console: true,
    Depot:   nil,
})
```

### Using the Contract

```go
// Get native zap logger for direct zap API usage
zapLogger := contract.Logger() // *zap.Logger

// Get common provider interface for service integration
provider := contract.Provider() // zlog.ZlogProvider

// Services that accept zlog can use either
someService.SetLogger(provider)
```

### Global Usage

Once created, the zap logger automatically becomes the global logger for all zlog usage:

```go
import "zbz/zlog"

// This will use the zap logger
zlog.Info("Hello world", zlog.String("key", "value"))
```

## Configuration

### Config Options

- `Name`: Logger name/identifier
- `Level`: Log level (zlog.DEBUG, zlog.INFO, zlog.WARN, zlog.ERROR, zlog.FATAL)
- `Format`: Output format ("json" or "console")
- `Console`: Enable console output (true for development, false for production)
- `Depot`: Depot storage configuration for persistent logging (optional)

### Example Production Config

```go
contract := zap.New(zlog.Config{
    Name:    "production-app",
    Level:   zlog.INFO,
    Format:  "json", 
    Console: false,
    Depot: &zlog.DepotConfig{
        Contract:     depotContract,
        KeyPrefix:    "logs/app",
        BufferSize:   1024,
        FlushInterval: 30 * time.Second,
    },
})
```

## Field Type Support

All zlog field types are automatically converted to appropriate zap fields:

- `String`, `Int`, `Float64`, `Bool`, `Error`, `Duration`, `Time`
- `ByteString`, `Strings`, `Any`
- Automatic fallback for unknown types

## Integration Example

```go
package main

import (
    "zbz/providers/zlog-zap"
    "zbz/zlog"
)

func main() {
    // Set up zap as the global logger
    zapContract := zap.NewDevelopment()
    
    // All zlog usage now uses zap
    zlog.Info("Application starting",
        zlog.String("version", "1.0.0"),
        zlog.Int("port", 8080))
    
    // Services can use the native logger directly
    zapLogger := zapContract.Logger()
    zapLogger.Info("Direct zap usage")
    
    // Or use the common provider interface
    provider := zapContract.Provider()
    provider.Info("Common interface usage", []zlog.Field{
        zlog.String("component", "auth"),
    })
}
```