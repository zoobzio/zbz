module zbz/flux

go 1.23.1

// Local module replacements for development
replace zbz/zlog => ../zlog

replace zbz/universal => ../universal

replace zbz/capitan => ../capitan

require (
	gopkg.in/yaml.v3 v3.0.1
	zbz/universal v0.0.0-00010101000000-000000000000
	zbz/zlog v0.0.0-00010101000000-000000000000
)

require github.com/google/uuid v1.6.0 // indirect
