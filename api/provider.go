package zbz

// Provider interfaces define the resolution functions that contracts implement
// These are used by the engine as stand-ins for real contracts

// HTTPProvider defines how to resolve HTTP services
type HTTPProvider interface {
	HTTP() HTTP
}

// DatabaseProvider defines how to resolve Database services  
type DatabaseProvider interface {
	Database() Database
}


// AuthProvider defines how to resolve Auth services
type AuthProvider interface {
	Auth() Auth
}

// CoreProvider defines how to resolve Core services
type CoreProvider interface {
	Core() Core
}

// CacheProvider defines how to resolve Cache services
type CacheProvider interface {
	Cache() Cache
}