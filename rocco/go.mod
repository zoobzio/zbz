module zbz/rocco

go 1.23.1

require (
	github.com/golang-jwt/jwt/v5 v5.2.2
	golang.org/x/crypto v0.39.0
	zbz/capitan v0.0.0-00010101000000-000000000000
	zbz/cereal v0.0.0-00010101000000-000000000000
	zbz/core v0.0.0-00010101000000-000000000000
	zbz/zlog v0.0.0-00010101000000-000000000000
)

require (
	github.com/gabriel-vasile/mimetype v1.4.2 // indirect
	github.com/go-playground/locales v0.14.1 // indirect
	github.com/go-playground/universal-translator v0.18.1 // indirect
	github.com/go-playground/validator/v10 v10.17.0 // indirect
	github.com/leodido/go-urn v1.2.4 // indirect
	github.com/pelletier/go-toml/v2 v2.2.4 // indirect
	golang.org/x/net v0.21.0 // indirect
	golang.org/x/sys v0.33.0 // indirect
	golang.org/x/text v0.26.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	zbz/catalog v0.0.0-00010101000000-000000000000 // indirect
	zbz/universal v0.0.0-00010101000000-000000000000 // indirect
)

replace zbz/capitan => ../capitan

replace zbz/catalog => ../catalog

replace zbz/cereal => ../cereal

replace zbz/core => ../core

replace zbz/universal => ../universal

replace zbz/zlog => ../zlog
