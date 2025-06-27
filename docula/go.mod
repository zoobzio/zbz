module zbz/docula/v2

go 1.23.1

require (
	github.com/microcosm-cc/bluemonday v1.0.26
	github.com/yuin/goldmark v1.7.0
	github.com/yuin/goldmark-meta v1.1.0
	gopkg.in/yaml.v2 v2.4.0
	zbz/flux v0.0.0-00010101000000-000000000000
	zbz/depot v0.0.0-00010101000000-000000000000
	zbz/zlog v0.0.0-00010101000000-000000000000
)

replace zbz/depot => ../../depot

replace zbz/flux => ../../flux

replace zbz/zlog => ../../zlog

require (
	github.com/aymerick/douceur v0.2.0 // indirect
	github.com/fsnotify/fsnotify v1.7.0 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/gorilla/css v1.0.1 // indirect
	golang.org/x/net v0.17.0 // indirect
	golang.org/x/sys v0.13.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
