module zbz/adapters/zlog-loki

go 1.23.1

replace zbz/zlog => ../../zlog
replace zbz/capitan => ../../capitan
replace zbz/cereal => ../../cereal

require (
	zbz/zlog v0.0.0-00010101000000-000000000000
	zbz/capitan v0.0.0-00010101000000-000000000000
)