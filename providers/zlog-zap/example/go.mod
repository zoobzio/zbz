module zbz/examples/zlog-zap-demo

go 1.23.1

require (
	go.uber.org/zap v1.27.0
	zbz/plugins/zlog-security v0.0.0
	zbz/providers/zlog-zap v0.0.0
	zbz/zlog v0.0.0
)

require go.uber.org/multierr v1.10.0 // indirect

replace zbz/zlog => ../../../zlog

replace zbz/providers/zlog-zap => ../

replace zbz/plugins/zlog-security => ../../../plugins/zlog-security
