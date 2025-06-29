package rocco

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	
	"zbz/cereal"
	"zbz/zlog"
	"zbz/capitan"
)

// SecurityContext wraps an identity with cereal-aware data filtering
type SecurityContext struct {
	identity *Identity
}

// NewSecurityContext creates a security context from an identity
func NewSecurityContext(identity *Identity) *SecurityContext {
	return &SecurityContext{identity: identity}
}

// FilterData applies permission-based filtering to any data structure
// This is where rocco + cereal magic happens
func (sc *SecurityContext) FilterData(data any) (any, error) {
	if sc.identity == nil {
		zlog.Warn("No identity in security context - returning unfiltered data")
		return data, nil
	}
	
	// Build permission list from identity
	permissions := sc.buildPermissions()
	
	zlog.Debug("Applying security filtering",
		zlog.String("user_id", sc.identity.ID),
		zlog.String("username", sc.identity.Username),
		zlog.Strings("permissions", permissions),
		zlog.String("data_type", fmt.Sprintf("%T", data)),
	)
	
	// Use cereal's security scoping
	filtered, err := cereal.FilterByPermissions(data, permissions)
	if err != nil {
		zlog.Error("Security filtering failed",
			zlog.String("user_id", sc.identity.ID),
			zlog.String("error", err.Error()),
		)
		return nil, fmt.Errorf("security filtering failed: %w", err)
	}
	
	// Emit security event
	capitan.EmitEvent("security.data_filtered", map[string]any{
		"user_id":     sc.identity.ID,
		"username":    sc.identity.Username,
		"data_type":   fmt.Sprintf("%T", data),
		"permissions": permissions,
		"timestamp":   sc.identity.LastActive,
	})
	
	return filtered, nil
}

// ValidateInput validates input data against user permissions
// Prevents privilege escalation by filtering input payloads
func (sc *SecurityContext) ValidateInput(input any) error {
	if sc.identity == nil {
		return fmt.Errorf("no identity in security context")
	}
	
	permissions := sc.buildPermissions()
	
	zlog.Debug("Validating input permissions",
		zlog.String("user_id", sc.identity.ID),
		zlog.String("username", sc.identity.Username),
		zlog.Strings("permissions", permissions),
	)
	
	// Use cereal's permission validation
	if err := cereal.ValidatePermissions(input, permissions); err != nil {
		zlog.Warn("Input validation failed - potential privilege escalation attempt",
			zlog.String("user_id", sc.identity.ID),
			zlog.String("username", sc.identity.Username),
			zlog.String("error", err.Error()),
		)
		
		// Emit security alert
		capitan.EmitEvent("security.privilege_escalation_attempt", map[string]any{
			"user_id":   sc.identity.ID,
			"username":  sc.identity.Username,
			"error":     err.Error(),
			"timestamp": sc.identity.LastActive,
		})
		
		return fmt.Errorf("permission validation failed: %w", err)
	}
	
	return nil
}

// buildPermissions creates a permission list from identity
func (sc *SecurityContext) buildPermissions() []string {
	var permissions []string
	
	// Add role-based permissions
	for _, role := range sc.identity.Roles {
		permissions = append(permissions, "role:"+role)
	}
	
	// Add explicit permissions
	permissions = append(permissions, sc.identity.Permissions...)
	
	// Add scope-based permissions
	for _, scope := range sc.identity.Scopes {
		permissions = append(permissions, "scope:"+scope)
	}
	
	// Add user-specific permission
	permissions = append(permissions, "user:"+sc.identity.ID)
	
	return permissions
}

// GetSecurityContext extracts security context from HTTP request context
func GetSecurityContext(ctx context.Context) (*SecurityContext, bool) {
	identity, ok := GetIdentity(ctx)
	if !ok {
		return nil, false
	}
	return NewSecurityContext(identity), true
}

// Enhanced bouncer rules that leverage cereal scoping

// SecureDataRule creates a rule that automatically applies security filtering
func SecureDataRule(pathPattern string, dataExtractor func(ctx context.Context) (any, error)) BouncerRule {
	return ContentAwareRule(
		"secure_data_access",
		pathPattern,
		func(r *http.Request) (Resource, error) {
			// Extract the data that will be returned
			data, err := dataExtractor(r.Context())
			if err != nil {
				return Resource{}, fmt.Errorf("data extraction failed: %w", err)
			}
			
			return Resource{
				Type: "secure_data",
				ID:   "data_access",
				Action: strings.ToLower(r.Method),
				Attributes: map[string]any{
					"data": data,
				},
			}, nil
		},
		func(identity *Identity, resource Resource) error {
			// Apply cereal security filtering
			sc := NewSecurityContext(identity)
			data := resource.Attributes["data"]
			
			filtered, err := sc.FilterData(data)
			if err != nil {
				return fmt.Errorf("security filtering failed: %w", err)
			}
			
			// Replace the resource data with filtered version
			resource.Attributes["filtered_data"] = filtered
			
			zlog.Info("Secure data access granted with filtering",
				zlog.String("user_id", identity.ID),
				zlog.String("username", identity.Username),
				zlog.String("action", resource.Action),
			)
			
			return nil
		},
	)
}

// UserScopedDataRule ensures users can only access data they own or have permission to see
func UserScopedDataRule(pathPattern string) BouncerRule {
	return ContentAwareRule(
		"user_scoped_data",
		pathPattern,
		func(r *http.Request) (Resource, error) {
			// Extract resource ID from URL (e.g., /api/users/123/profile)
			parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/"), "/")
			if len(parts) < 3 {
				return Resource{}, fmt.Errorf("invalid URL pattern for user scoped data")
			}
			
			return Resource{
				Type:   "user_data",
				ID:     parts[2], // User ID from URL
				Action: strings.ToLower(r.Method),
			}, nil
		},
		func(identity *Identity, resource Resource) error {
			// Check if user is accessing their own data
			if identity.ID == resource.ID {
				return nil // Users can always access their own data
			}
			
			// Check if user has admin privileges
			for _, role := range identity.Roles {
				if role == "admin" || role == "super_admin" {
					zlog.Info("Admin access to user data granted",
						zlog.String("admin_user", identity.Username),
						zlog.String("target_user", resource.ID),
					)
					return nil
				}
			}
			
			// Check for specific permissions (e.g., "user:123:read")
			requiredPerm := fmt.Sprintf("user:%s:%s", resource.ID, resource.Action)
			for _, perm := range identity.Permissions {
				if perm == requiredPerm {
					return nil
				}
			}
			
			return ErrForbidden
		},
	)
}