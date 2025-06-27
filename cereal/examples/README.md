# Cereal Examples

This directory contains examples demonstrating the scoping functionality of the cereal serialization library.

## Running the Examples

### Command Line Interface
```bash
# Run all examples (default)
go run .

# Run individual examples
go run . json    # JSON example only
go run . yaml    # YAML example only  
go run . toml    # TOML example only

# Show help
go run . help
```

### Individual Examples
Each example demonstrates different aspects of permission-based field filtering:

**JSON Example** - User data with privacy controls:
- Marshal filtering based on permissions  
- Unmarshal protection against privilege escalation
- Round-trip serialization with scoping
- Common permission patterns (public, admin, PII, etc.)

**YAML Example** - Configuration file scoping:
- Configuration file scoping for different audiences
- Protecting sensitive config values
- Multi-level permission requirements (admin+security)
- Configuration migration between permission levels

**TOML Example** - Server configuration management:
- Infrastructure configuration with role-based access
- Deployment-specific config filtering
- Security credential protection
- Multi-environment configuration scenarios

## Key Concepts

### Permission Syntax
- Single permission: `scope:"admin"`
- OR logic (any permission): `scope:"read,write"`
- AND logic (all required): `scope:"admin+pii"`
- Complex combinations: `scope:"compliance,admin+security"`

### Explicit Serializer Selection
```go
// Always use explicit serializer selection
data, _ := cereal.JSON.Marshal(obj, permissions...)
data, _ := cereal.YAML.Marshal(obj, permissions...)  
data, _ := cereal.TOML.Marshal(obj, permissions...)
```

### Bidirectional Protection
- **Marshal**: Completely omits fields the user can't see
- **Unmarshal**: Zeros out fields the user can't set

This ensures data security in both directions - users can't see what they shouldn't, and they can't set values they shouldn't have access to.