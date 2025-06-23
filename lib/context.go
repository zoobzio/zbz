package zbz

// Context provides a minimal framework-agnostic interface for ZBZ internal handlers
// Defines only the methods ZBZ internals actually need
// Note: This is now an alias for RequestContext for consistency
type Context = RequestContext