package cereal

// ScopeProvider is the ONLY convention cereal cares about
// Other services handle their own conventions (ValidationProvider, AuditProvider, etc.)
// This keeps cereal focused on serialization + scoping
type ScopeProvider interface {
	GetRequiredScopes() []string
}

// checkScopeProvider checks if model implements ScopeProvider and validates permissions
func checkScopeProvider(model interface{}, permissions []string) bool {
	if scopeProvider, ok := model.(ScopeProvider); ok {
		requiredScopes := scopeProvider.GetRequiredScopes()
		return hasAllScopes(permissions, requiredScopes)
	}
	return true // No scope requirements = always allowed
}

// hasAllScopes checks if provided permissions contain all required scopes
func hasAllScopes(provided, required []string) bool {
	if len(required) == 0 {
		return true
	}
	
	providedSet := make(map[string]bool)
	for _, scope := range provided {
		providedSet[scope] = true
	}
	
	for _, required := range required {
		if !providedSet[required] {
			return false
		}
	}
	
	return true
}