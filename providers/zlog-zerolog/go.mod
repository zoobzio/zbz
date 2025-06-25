module zbz/providers/zlog-zerolog

go 1.23.1

// Local module replacements for development
replace zbz/zlog => ../../zlog

require (
	github.com/rs/zerolog v1.33.0
	zbz/zlog v0.0.0-00010101000000-000000000000
	gopkg.in/natefinch/lumberjack.v2 v2.2.1
)

require (
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.19 // indirect
	golang.org/x/sys v0.12.0 // indirect
)