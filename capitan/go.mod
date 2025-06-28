module zbz/capitan

go 1.23.1

require (
	zbz/cereal v0.0.0-00010101000000-000000000000
	zbz/zlog v0.0.0-00010101000000-000000000000
)

require (
	github.com/pelletier/go-toml/v2 v2.2.4 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace zbz/cereal => ../cereal

replace zbz/zlog => ../zlog
