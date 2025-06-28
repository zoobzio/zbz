module zbz/docula/v2

go 1.23.1

require (
	github.com/microcosm-cc/bluemonday v1.0.26
	github.com/yuin/goldmark v1.7.0
	github.com/yuin/goldmark-meta v1.1.0
	zbz/capitan v0.0.0-00010101000000-000000000000
	zbz/cereal v0.0.0-00010101000000-000000000000
	zbz/universal v0.0.0-00010101000000-000000000000
	zbz/zlog v0.0.0-00010101000000-000000000000
)

replace zbz/capitan => ../capitan

replace zbz/cereal => ../cereal

replace zbz/universal => ../universal

replace zbz/zlog => ../zlog

require (
	github.com/aymerick/douceur v0.2.0 // indirect
	github.com/gorilla/css v1.0.1 // indirect
	github.com/pelletier/go-toml/v2 v2.2.4 // indirect
	golang.org/x/net v0.17.0 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
