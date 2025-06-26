module zbz/providers/cache-redis

go 1.23.1

require (
	github.com/redis/go-redis/v9 v9.11.0
	zbz/cache v0.0.0-00010101000000-000000000000
)

replace zbz/cache => ../../cache