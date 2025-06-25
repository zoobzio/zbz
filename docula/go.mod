module zbz/docula

go 1.23.1

// Local module replacements for development
replace zbz/zlog => ../zlog
replace zbz/remark => ../remark

require (
	zbz/zlog v0.0.0-00010101000000-000000000000
	zbz/remark v0.0.0-00010101000000-000000000000
	gopkg.in/yaml.v3 v3.0.1
)
