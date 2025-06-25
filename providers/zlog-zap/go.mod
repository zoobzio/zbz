module zbz/providers/zlog-zap

go 1.23.1

// Local module replacements for development
replace zbz/zlog => ../../zlog

require (
	go.uber.org/zap v1.27.0
	gopkg.in/natefinch/lumberjack.v2 v2.2.1
	zbz/zlog v0.0.0-00010101000000-000000000000
)

require go.uber.org/multierr v1.10.0 // indirect
