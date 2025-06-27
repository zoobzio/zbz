module zbz/cmd

go 1.23.1

require (
	github.com/spf13/cobra v1.8.0
	go.uber.org/zap v1.27.0
	zbz/cereal v0.0.0
	zbz/plugins/zlog-security v0.0.0
	zbz/providers/zlog-zap v0.0.0
	zbz/zlog v0.0.0
)

require (
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/pelletier/go-toml/v2 v2.2.4 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	go.uber.org/multierr v1.10.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace zbz/zlog => ../zlog

replace zbz/cereal => ../cereal

replace zbz/providers/zlog-zap => ../providers/zlog-zap

replace zbz/plugins/zlog-security => ../plugins/zlog-security
