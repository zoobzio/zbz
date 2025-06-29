package rocco

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"
	
	"zbz/zlog"
	"zbz/capitan"
)

// Middleware provides basic authentication middleware
func (a *zAuth) Middleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract token from request
			token := extractToken(r)
			if token == "" {
				// No token, continue without identity
				next.ServeHTTP(w, r)
				return
			}
			
			// Parse and validate token
			claims, err := a.parseAccessToken(token)
			if err != nil {
				zlog.Debug("Invalid token provided",
					zlog.String("error", err.Error()),
					zlog.String("path", r.URL.Path),
				)
				// Invalid token, continue without identity
				next.ServeHTTP(w, r)
				return
			}
			
			// Build identity from claims
			identity := &Identity{
				ID:          claims["sub"].(string),
				Username:    getStringClaim(claims, "username"),
				Email:       getStringClaim(claims, "email"),
				Provider:    getStringClaim(claims, "provider"),
				SessionID:   getStringClaim(claims, "session_id"),
				AccessToken: token,
				TokenType:   "Bearer",
			}
			
			// Extract arrays from claims
			if roles, ok := claims["roles"].([]interface{}); ok {
				identity.Roles = interfaceToStringSlice(roles)
			}
			if permissions, ok := claims["permissions"].([]interface{}); ok {
				identity.Permissions = interfaceToStringSlice(permissions)
			}
			if scopes, ok := claims["scopes"].([]interface{}); ok {
				identity.Scopes = interfaceToStringSlice(scopes)
			}
			
			// Add identity to context
			ctx := WithIdentity(r.Context(), identity)
			
			zlog.Debug("Request authenticated",
				zlog.String("user_id", identity.ID),
				zlog.String("username", identity.Username),
				zlog.String("path", r.URL.Path),
				zlog.String("method", r.Method),
				zlog.Strings("roles", identity.Roles),
			)
			
			// Continue with identity in context
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// BouncerMiddleware provides content-aware authorization
func (a *zAuth) BouncerMiddleware(rules ...BouncerRule) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			
			// Find matching rule
			var matchedRule *BouncerRule
			for i := range rules {
				if a.matchesRule(&rules[i], r) {
					matchedRule = &rules[i]
					break
				}
			}
			
			// If no rule matches, allow through
			if matchedRule == nil {
				zlog.Debug("No authorization rule matched - allowing access",
					zlog.String("path", r.URL.Path),
					zlog.String("method", r.Method),
				)
				next.ServeHTTP(w, r)
				return
			}
			
			zlog.Debug("Authorization rule matched",
				zlog.String("rule_name", matchedRule.Name),
				zlog.String("path", r.URL.Path),
				zlog.String("method", r.Method),
			)
			
			// Apply the bouncer rule
			if err := a.applyBouncerRule(matchedRule, w, r); err != nil {
				zlog.Warn("Authorization denied",
					zlog.String("rule_name", matchedRule.Name),
					zlog.String("path", r.URL.Path),
					zlog.String("method", r.Method),
					zlog.String("error", err.Error()),
					zlog.Duration("duration", time.Since(start)),
				)
				
				// Emit authorization denied event
				capitan.EmitEvent("auth.denied", map[string]any{
					"rule_name": matchedRule.Name,
					"path":      r.URL.Path,
					"method":    r.Method,
					"error":     err.Error(),
					"timestamp": time.Now(),
					"duration":  time.Since(start),
				})
				// Rule failed, request has been handled by failure handler
				return
			}
			
			zlog.Debug("Authorization granted",
				zlog.String("rule_name", matchedRule.Name),
				zlog.String("path", r.URL.Path),
				zlog.String("method", r.Method),
				zlog.Duration("duration", time.Since(start)),
			)
			
			// Emit authorization granted event
			capitan.EmitEvent("auth.granted", map[string]any{
				"rule_name": matchedRule.Name,
				"path":      r.URL.Path,
				"method":    r.Method,
				"timestamp": time.Now(),
				"duration":  time.Since(start),
			})
			
			// Rule passed, continue
			next.ServeHTTP(w, r)
		})
	}
}

// matchesRule checks if a request matches a bouncer rule
func (a *zAuth) matchesRule(rule *BouncerRule, r *http.Request) bool {
	// Check path pattern
	if rule.PathPattern != "" {
		matched, err := regexp.MatchString(rule.PathPattern, r.URL.Path)
		if err != nil || !matched {
			return false
		}
	}
	
	// Check HTTP method
	if len(rule.Methods) > 0 {
		methodMatch := false
		for _, method := range rule.Methods {
			if strings.EqualFold(method, r.Method) {
				methodMatch = true
				break
			}
		}
		if !methodMatch {
			return false
		}
	}
	
	// Check required headers
	for key, value := range rule.Headers {
		if r.Header.Get(key) != value {
			return false
		}
	}
	
	return true
}

// applyBouncerRule applies a bouncer rule to a request
func (a *zAuth) applyBouncerRule(rule *BouncerRule, w http.ResponseWriter, r *http.Request) error {
	// Get identity from context
	identity, hasIdentity := GetIdentity(r.Context())
	
	// Check if authentication is required
	if rule.RequireAuth && !hasIdentity {
		err := ErrUnauthorized
		a.handleRuleFailure(rule, w, r, err)
		return err
	}
	
	// Check required roles
	if len(rule.RequiredRoles) > 0 && hasIdentity {
		if !hasAnyRole(identity.Roles, rule.RequiredRoles) {
			err := fmt.Errorf("missing required role: %v", rule.RequiredRoles)
			a.handleRuleFailure(rule, w, r, err)
			return err
		}
	}
	
	// Check required scopes
	if len(rule.RequiredScopes) > 0 && hasIdentity {
		if !hasAnyScope(identity.Scopes, rule.RequiredScopes) {
			err := fmt.Errorf("missing required scope: %v", rule.RequiredScopes)
			a.handleRuleFailure(rule, w, r, err)
			return err
		}
	}
	
	// Check required claims
	for claimKey, claimValue := range rule.RequiredClaims {
		if !hasIdentity {
			err := ErrUnauthorized
			a.handleRuleFailure(rule, w, r, err)
			return err
		}
		
		if actualValue, exists := identity.Claims[claimKey]; !exists || actualValue != claimValue {
			err := fmt.Errorf("missing or invalid claim: %s", claimKey)
			a.handleRuleFailure(rule, w, r, err)
			return err
		}
	}
	
	// Content-aware authorization
	if rule.ContentInspector != nil && rule.Authorizer != nil {
		// Extract resource from request content
		resource, err := rule.ContentInspector(r)
		if err != nil {
			zlog.Error("Content inspection failed",
				zlog.String("rule_name", rule.Name),
				zlog.String("path", r.URL.Path),
				zlog.String("error", err.Error()),
			)
			a.handleRuleFailure(rule, w, r, fmt.Errorf("content inspection failed: %w", err))
			return err
		}
		
		zlog.Debug("Resource extracted from request",
			zlog.String("rule_name", rule.Name),
			zlog.String("resource_type", resource.Type),
			zlog.String("resource_id", resource.ID),
			zlog.String("action", resource.Action),
		)
		
		// Authorize access to the resource
		if err := rule.Authorizer(identity, resource); err != nil {
			zlog.Warn("Resource access denied",
				zlog.String("rule_name", rule.Name),
				zlog.String("user_id", identity.ID),
				zlog.String("username", identity.Username),
				zlog.String("resource_type", resource.Type),
				zlog.String("resource_id", resource.ID),
				zlog.String("action", resource.Action),
				zlog.String("error", err.Error()),
			)
			
			// Emit resource access denied event
			capitan.EmitEvent("auth.resource_denied", map[string]any{
				"rule_name":     rule.Name,
				"user_id":       identity.ID,
				"username":      identity.Username,
				"resource_type": resource.Type,
				"resource_id":   resource.ID,
				"action":        resource.Action,
				"error":         err.Error(),
				"timestamp":     time.Now(),
			})
			
			a.handleRuleFailure(rule, w, r, fmt.Errorf("authorization failed: %w", err))
			return err
		}
		
		zlog.Info("Resource access granted",
			zlog.String("rule_name", rule.Name),
			zlog.String("user_id", identity.ID),
			zlog.String("username", identity.Username),
			zlog.String("resource_type", resource.Type),
			zlog.String("resource_id", resource.ID),
			zlog.String("action", resource.Action),
		)
		
		// Emit resource access granted event
		capitan.EmitEvent("auth.resource_granted", map[string]any{
			"rule_name":     rule.Name,
			"user_id":       identity.ID,
			"username":      identity.Username,
			"resource_type": resource.Type,
			"resource_id":   resource.ID,
			"action":        resource.Action,
			"timestamp":     time.Now(),
		})
	}
	
	// Rule passed, call success handler if defined
	if rule.OnSuccess != nil {
		rule.OnSuccess(w, r, identity)
	}
	
	return nil
}

// handleRuleFailure handles bouncer rule failures
func (a *zAuth) handleRuleFailure(rule *BouncerRule, w http.ResponseWriter, r *http.Request, err error) {
	if rule.OnFailure != nil {
		rule.OnFailure(w, r, err)
		return
	}
	
	// Default failure handling
	status := http.StatusUnauthorized
	if err == ErrForbidden || strings.Contains(err.Error(), "forbidden") || 
	   strings.Contains(err.Error(), "missing required") {
		status = http.StatusForbidden
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	
	response := map[string]any{
		"error": map[string]any{
			"code":    status,
			"message": err.Error(),
			"rule":    rule.Name,
		},
	}
	
	json.NewEncoder(w).Encode(response)
}

// Predefined bouncer rules

// RequireAuth creates a rule that requires authentication
func RequireAuth(pathPattern string) BouncerRule {
	return BouncerRule{
		Name:        "require_auth",
		Description: "Requires valid authentication",
		PathPattern: pathPattern,
		RequireAuth: true,
	}
}

// RequireRole creates a rule that requires specific roles
func RequireRole(pathPattern string, roles ...string) BouncerRule {
	return BouncerRule{
		Name:         "require_role",
		Description:  fmt.Sprintf("Requires role: %v", roles),
		PathPattern:  pathPattern,
		RequireAuth:  true,
		RequiredRoles: roles,
	}
}

// RequireScope creates a rule that requires specific scopes
func RequireScope(pathPattern string, scopes ...string) BouncerRule {
	return BouncerRule{
		Name:          "require_scope",
		Description:   fmt.Sprintf("Requires scope: %v", scopes),
		PathPattern:   pathPattern,
		RequireAuth:   true,
		RequiredScopes: scopes,
	}
}

// ContentAwareRule creates a content-aware authorization rule
func ContentAwareRule(name, pathPattern string, inspector func(*http.Request) (Resource, error), authorizer func(*Identity, Resource) error) BouncerRule {
	return BouncerRule{
		Name:             name,
		Description:      "Content-aware authorization",
		PathPattern:      pathPattern,
		RequireAuth:      true,
		ContentInspector: inspector,
		Authorizer:       authorizer,
	}
}

// AdminOnly creates a rule that requires admin role
func AdminOnly(pathPattern string) BouncerRule {
	return RequireRole(pathPattern, "admin")
}

// APIKeyRequired creates a rule that requires API key authentication
func APIKeyRequired(pathPattern string) BouncerRule {
	return BouncerRule{
		Name:        "api_key_required",
		Description: "Requires API key authentication",
		PathPattern: pathPattern,
		RequireAuth: true,
		RequiredScopes: []string{"api"},
	}
}

// Content-aware examples

// UserDataRule creates a rule for user data access (users can only access their own data)
func UserDataRule(pathPattern string) BouncerRule {
	return ContentAwareRule(
		"user_data_access",
		pathPattern,
		func(r *http.Request) (Resource, error) {
			// Extract user ID from URL path (e.g., /api/users/123)
			parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/"), "/")
			if len(parts) >= 3 && parts[1] == "users" {
				return Resource{
					Type:   "user_data",
					ID:     parts[2],
					Action: strings.ToLower(r.Method),
				}, nil
			}
			return Resource{}, fmt.Errorf("cannot extract user ID from path")
		},
		func(identity *Identity, resource Resource) error {
			// Users can access their own data, admins can access any
			if hasRole(identity.Roles, "admin") {
				return nil
			}
			if identity.ID == resource.ID {
				return nil
			}
			return ErrForbidden
		},
	)
}

// DocumentAccessRule creates a rule for document access based on ownership
func DocumentAccessRule(pathPattern string) BouncerRule {
	return ContentAwareRule(
		"document_access",
		pathPattern,
		func(r *http.Request) (Resource, error) {
			// Extract document ID from URL
			parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/"), "/")
			if len(parts) >= 3 && parts[1] == "documents" {
				return Resource{
					Type:   "document",
					ID:     parts[2],
					Action: strings.ToLower(r.Method),
				}, nil
			}
			return Resource{}, fmt.Errorf("cannot extract document ID from path")
		},
		func(identity *Identity, resource Resource) error {
			// This would typically check document ownership in a database
			// For now, just check if user has document access permission
			requiredPerm := fmt.Sprintf("document:%s:%s", resource.ID, resource.Action)
			if hasPermission(identity.Permissions, requiredPerm) {
				return nil
			}
			
			// Or check for admin role
			if hasRole(identity.Roles, "admin") {
				return nil
			}
			
			return ErrForbidden
		},
	)
}

// Helper functions

func extractToken(r *http.Request) string {
	// Check Authorization header
	auth := r.Header.Get("Authorization")
	if auth != "" {
		parts := strings.SplitN(auth, " ", 2)
		if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
			return parts[1]
		}
	}
	
	// Check query parameter
	if token := r.URL.Query().Get("token"); token != "" {
		return token
	}
	
	// Check cookie
	if cookie, err := r.Cookie("zbz_auth"); err == nil {
		return cookie.Value
	}
	
	return ""
}

func getStringClaim(claims map[string]interface{}, key string) string {
	if value, ok := claims[key].(string); ok {
		return value
	}
	return ""
}

func interfaceToStringSlice(slice []interface{}) []string {
	result := make([]string, 0, len(slice))
	for _, item := range slice {
		if str, ok := item.(string); ok {
			result = append(result, str)
		}
	}
	return result
}

func hasAnyRole(userRoles, requiredRoles []string) bool {
	for _, required := range requiredRoles {
		for _, userRole := range userRoles {
			if userRole == required {
				return true
			}
		}
	}
	return false
}

func hasAnyScope(userScopes, requiredScopes []string) bool {
	for _, required := range requiredScopes {
		for _, userScope := range userScopes {
			if userScope == required {
				return true
			}
		}
	}
	return false
}