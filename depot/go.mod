module zbz/depot

go 1.23.1

// Local module replacements for development
replace zbz/zlog => ../zlog
replace zbz/cereal => ../cereal

require (
	github.com/fsnotify/fsnotify v1.7.0
	github.com/google/uuid v1.6.0
	gopkg.in/yaml.v3 v3.0.1
	zbz/cereal v0.0.0-00010101000000-000000000000
	zbz/zlog v0.0.0-00010101000000-000000000000
)

require golang.org/x/sys v0.4.0 // indirect
