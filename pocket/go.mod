module zbz/pocket

go 1.23.1

require (
	zbz/cereal v0.0.0-00010101000000-000000000000
	zbz/zlog v0.0.0-00010101000000-000000000000
)

require (
	google.golang.org/protobuf v1.36.6 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace (
	zbz/cereal => ../cereal
	zbz/depot => ../depot
	zbz/flux => ../flux
	zbz/zlog => ../zlog
)
