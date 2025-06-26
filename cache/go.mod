module zbz/cache

go 1.23.1

require (
	zbz/flux v0.0.0-00010101000000-000000000000
	zbz/hodor v0.0.0-00010101000000-000000000000
	zbz/zlog v0.0.0-00010101000000-000000000000
)

replace (
	zbz/flux => ../flux
	zbz/hodor => ../hodor
	zbz/zlog => ../zlog
)