package rocco

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"zbz/cereal"
	"zbz/core"
	"zbz/zlog"
)

// APIMiddleware provides API request routing through core's contract system
func (a *zAuth) APIMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Only handle /api/* paths
			if !strings.HasPrefix(r.URL.Path, "/api/") {
				next.ServeHTTP(w, r)
				return
			}

			// Look up API contract in core
			contract, coreService, err := core.GetAPIContract(r.Method, r.URL.Path)
			if err != nil {
				// No contract found, let next handler deal with it
				next.ServeHTTP(w, r)
				return
			}

			zlog.Debug("API contract found",
				zlog.String("method", r.Method),
				zlog.String("path", r.URL.Path),
				zlog.String("operation", contract.Operation),
			)

			// Handle the API request
			a.handleAPIRequest(w, r, contract, coreService)
		})
	}
}

// handleAPIRequest processes the API request using the contract and core service
func (a *zAuth) handleAPIRequest(w http.ResponseWriter, r *http.Request, contract core.APIContract, coreService core.CoreService) {
	// 1. Authentication (if required)
	identity, err := a.extractIdentity(r)
	if err != nil {
		a.writeError(w, http.StatusUnauthorized, "Invalid authentication")
		return
	}
	
	// Check if authentication is required but missing
	if identity == nil && len(contract.Scopes) > 0 {
		a.writeError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	// 2. Authorization (check scopes)
	if len(contract.Scopes) > 0 && identity != nil {
		if !a.hasRequiredScopes(identity, contract.Scopes) {
			a.writeError(w, http.StatusForbidden, "Insufficient permissions")
			return
		}
	}

	// 3. Extract path parameters
	params := a.extractPathParams(r.URL.Path, contract.Endpoint)

	// 4. Build ResourceURI using contract's function
	resourceURI := contract.ResourceURI(params)

	zlog.Debug("API request routing",
		zlog.String("operation", contract.Operation),
		zlog.String("resource_uri", resourceURI.URI),
		zlog.Any("params", params),
	)

	// 5. Call core operation
	var result any
	switch contract.Operation {
	case "get":
		result, err = coreService.GetByURI(r.Context(), resourceURI)
	case "list":
		// For list operations, we'd use a pattern URI
		result, err = coreService.GetByURI(r.Context(), resourceURI)
	case "create", "update":
		// TODO: Parse request body and call SetByURI
		a.writeError(w, http.StatusNotImplemented, "Create/Update operations not yet implemented")
		return
	case "delete":
		err = coreService.DeleteByURI(r.Context(), resourceURI)
		if err == nil {
			a.writeJSON(w, map[string]string{"status": "deleted"})
			return
		}
	default:
		a.writeError(w, http.StatusNotImplemented, fmt.Sprintf("Operation '%s' not implemented", contract.Operation))
		return
	}

	if err != nil {
		zlog.Error("Core operation failed",
			zlog.String("operation", contract.Operation),
			zlog.String("error", err.Error()),
		)
		a.writeError(w, http.StatusInternalServerError, "Operation failed")
		return
	}

	// 6. Apply security filtering if we have an identity
	if identity != nil && result != nil {
		secCtx := NewSecurityContext(identity)
		filteredResult, err := secCtx.FilterData(result)
		if err != nil {
			zlog.Error("Security filtering failed",
				zlog.String("error", err.Error()),
			)
			a.writeError(w, http.StatusForbidden, "Access denied")
			return
		}
		result = filteredResult
	}

	// 7. Return JSON response
	a.writeJSON(w, result)
}

// extractIdentity gets identity from request (without requiring it)
func (a *zAuth) extractIdentity(r *http.Request) (*Identity, error) {
	identity, ok := GetIdentity(r.Context())
	if !ok {
		// Try to extract from token if middleware didn't run
		token := extractToken(r)
		if token == "" {
			return nil, nil
		}

		claims, err := a.parseAccessToken(token)
		if err != nil {
			return nil, err
		}

		identity = &Identity{
			ID:       claims["sub"].(string),
			Username: getStringClaim(claims, "username"),
			Email:    getStringClaim(claims, "email"),
			Provider: getStringClaim(claims, "provider"),
		}

		// Extract roles and permissions from claims
		if roles, ok := claims["roles"].([]interface{}); ok {
			for _, role := range roles {
				if roleStr, ok := role.(string); ok {
					identity.Roles = append(identity.Roles, roleStr)
				}
			}
		}

		if permissions, ok := claims["permissions"].([]interface{}); ok {
			for _, perm := range permissions {
				if permStr, ok := perm.(string); ok {
					identity.Permissions = append(identity.Permissions, permStr)
				}
			}
		}
	}

	return identity, nil
}

// hasRequiredScopes checks if identity has any of the required scopes
func (a *zAuth) hasRequiredScopes(identity *Identity, requiredScopes []string) bool {
	// Check roles
	for _, role := range identity.Roles {
		roleScope := "role:" + role
		for _, required := range requiredScopes {
			if required == roleScope {
				return true
			}
		}
	}

	// Check permissions
	for _, perm := range identity.Permissions {
		for _, required := range requiredScopes {
			if required == perm {
				return true
			}
		}
	}

	return false
}

// extractPathParams extracts parameters from URL path using endpoint pattern
func (a *zAuth) extractPathParams(path, pattern string) map[string]string {
	params := make(map[string]string)

	// Convert pattern to regex: /api/users/{id} -> /api/users/([^/]+)
	regexPattern := regexp.QuoteMeta(pattern)
	regexPattern = regexp.MustCompile(`\\\{([^}]+)\\\}`).ReplaceAllString(regexPattern, `([^/]+)`)
	regexPattern = "^" + regexPattern + "$"

	re := regexp.MustCompile(regexPattern)
	matches := re.FindStringSubmatch(path)

	if len(matches) > 1 {
		// Extract parameter names from pattern
		paramNames := regexp.MustCompile(`\{([^}]+)\}`).FindAllStringSubmatch(pattern, -1)
		
		for i, paramName := range paramNames {
			if i+1 < len(matches) {
				params[paramName[1]] = matches[i+1]
			}
		}
	}

	return params
}

// writeJSON writes a JSON response
func (a *zAuth) writeJSON(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	
	jsonBytes, err := cereal.JSON.Marshal(data)
	if err != nil {
		a.writeError(w, http.StatusInternalServerError, "JSON encoding failed")
		return
	}
	
	w.Write(jsonBytes)
}

// writeError writes an error response
func (a *zAuth) writeError(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	
	errorResponse := map[string]any{
		"error": map[string]any{
			"message": message,
			"code":    statusCode,
		},
	}
	
	jsonBytes, _ := cereal.JSON.Marshal(errorResponse)
	w.Write(jsonBytes)
}

// APIMiddleware convenience function for default instance
func APIMiddleware() func(http.Handler) http.Handler {
	return Default().(*zAuth).APIMiddleware()
}