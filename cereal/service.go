package cereal

// Private singleton for scoping logic
var cereal *zCereal

// Public singletons for each format
var (
	JSON *zJSON
	YAML *zYaml
	TOML *zTOML
)

func init() {
	// Initialize scoping service
	cereal = &zCereal{
		cache: newScopeCache(),
	}

	JSON = &zJSON{}
	YAML = &zYaml{}
	TOML = &zTOML{}
}
