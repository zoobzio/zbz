module zbz/providers/hodor-s3

go 1.21

require (
	github.com/aws/aws-sdk-go v1.49.0
	zbz/hodor v0.0.0
)

replace zbz/hodor => ../../hodor

require (
	github.com/jmespath/go-jmespath v0.4.0 // indirect
)