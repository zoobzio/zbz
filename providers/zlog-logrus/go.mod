module zbz/providers/zlog-logrus

go 1.23.1

// Local module replacements for development
replace zbz/zlog => ../../zlog

require (
	zbz/zlog v0.0.0-00010101000000-000000000000
	github.com/sirupsen/logrus v1.9.3
	gopkg.in/natefinch/lumberjack.v2 v2.2.1
)