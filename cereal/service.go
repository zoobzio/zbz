package cereal

// Public singletons for each format
var (
	JSON *zJSON
	YAML *zYaml
	TOML *zTOML
)

func init() {
	// Initialize format handlers
	JSON = &zJSON{}
	YAML = &zYaml{}
	TOML = &zTOML{}
}
