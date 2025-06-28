module zbz/cmd

go 1.23.1

require (
	github.com/pelletier/go-toml/v2 v2.2.4
	github.com/spf13/cobra v1.8.0
	go.uber.org/zap v1.27.0
	gopkg.in/yaml.v3 v3.0.1
	zbz/adapters/zlog/security v0.0.0
	zbz/cereal v0.0.0
	zbz/providers/zlog-zap v0.0.0
	zbz/universal v0.0.0
	zbz/zlog v0.0.0
)

require (
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	go.uber.org/multierr v1.10.0 // indirect
	zbz/capitan v0.0.0 // indirect
)

replace zbz/zlog => ../zlog

replace zbz/cereal => ../cereal

replace zbz/providers/zlog-zap => ../providers/zlog-zap

replace zbz/adapters/zlog/security => ../adapters/zlog/security

replace zbz/universal => ../universal

replace zbz/capitan => ../capitan
