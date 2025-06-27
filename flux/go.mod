module zbz/flux

go 1.23.1

// Local module replacements for development
replace zbz/zlog => ../zlog

replace zbz/depot => ../depot

replace zbz/cereal => ../cereal

require (
	gopkg.in/yaml.v3 v3.0.1
	zbz/cereal v0.0.0-00010101000000-000000000000
	zbz/depot v0.0.0-00010101000000-000000000000
	zbz/zlog v0.0.0-00010101000000-000000000000
)

require (
	github.com/fsnotify/fsnotify v1.7.0 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/pelletier/go-toml/v2 v2.2.4 // indirect
	golang.org/x/sys v0.4.0 // indirect
)
