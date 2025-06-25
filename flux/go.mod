module zbz/flux

go 1.23.1

// Local module replacements for development
replace zbz/zlog => ../zlog

replace zbz/hodor => ../hodor

require (
	gopkg.in/yaml.v3 v3.0.1
	zbz/hodor v0.0.0-00010101000000-000000000000
	zbz/zlog v0.0.0-00010101000000-000000000000
)

require github.com/google/uuid v1.6.0 // indirect
