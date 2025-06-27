# ZBZ Framework CLI

Unified command-line interface for the ZBZ framework with discoverable demos and tools.

## Structure

```
cmd/
├── main.go              # Main zbz CLI with Cobra
├── demo.go              # Demo command definitions
├── demos/
│   ├── zlog/            # Logging framework demos
│   │   ├── json.go      # JSON structured logging
│   │   ├── security.go  # Security plugins & encryption
│   │   └── all.go       # All zlog demos
│   └── cereal/          # Serialization framework demos
│       ├── json.go      # JSON scoping with user data
│       ├── yaml.go      # YAML scoping with app config
│       ├── toml.go      # TOML scoping with server config
│       └── all.go       # All cereal demos
└── go.mod               # Dependencies
```

## Usage

### Build the CLI
```bash
cd cmd
go build -o zbz .
```

### Available Commands

#### Demo Commands
```bash
# Logging framework demos
./zbz demo zlog json       # JSON structured logging with zap
./zbz demo zlog security   # Security plugins, encryption, PII protection
./zbz demo zlog all        # All zlog demos

# Serialization framework demos  
./zbz demo cereal json     # JSON scoping with user data
./zbz demo cereal yaml     # YAML scoping with application config
./zbz demo cereal toml     # TOML scoping with server config
./zbz demo cereal all      # All cereal demos

# Help
./zbz demo                 # Show available demo categories
./zbz demo zlog           # Show zlog demo options
./zbz demo cereal         # Show cereal demo options
./zbz --help              # Full CLI help
```

## Features

### Extensible Architecture
- **Auto-discovery**: Commands are organized by category
- **Simple additions**: Add new demos by creating files in `demos/[category]/`
- **Cobra integration**: Full CLI framework with help, flags, etc.

### Current Demos

#### zlog (Logging)
- **JSON Demo**: Structured logging with zap provider, multiple log levels
- **Security Demo**: Field processors, encryption, PII protection, pattern detection
- Shows: Plugin system, output piping, custom processors

#### cereal (Serialization)  
- **JSON Demo**: User data with permission-based scoping
- **YAML Demo**: Application configuration with environment separation
- **TOML Demo**: Server configuration with security-sensitive data
- Shows: Multi-level access (public/private/admin), secure deserialization

## Future Extensions

Easy to add new command categories:

```bash
# Future tools category
./zbz tools generate-model User
./zbz tools migrate create AddUserTable

# Future benchmarks category  
./zbz bench zlog vs-logrus
./zbz bench cereal vs-json

# Future deployment category
./zbz deploy staging
./zbz deploy production --dry-run
```

## Architecture Benefits

1. **Single Binary**: One `zbz` command for everything
2. **Discoverable**: `zbz demo` shows what's available
3. **Extensible**: Add new categories without changing core CLI
4. **Organized**: Each demo is a focused, standalone example
5. **Professional**: Full Cobra CLI with help, completions, etc.

This replaces the scattered example directories with a unified, professional CLI experience.