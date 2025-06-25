package http

import "context"

// RequestContext provides the minimal interface ZBZ needs for HTTP request handling
// Each HTTP implementation (Gin, Echo, etc.) should implement this interface
type RequestContext interface {
	Method() string
	Path() string
	PathParam(name string) string
	QueryParam(name string) string
	Header(name string) string
	BodyBytes() ([]byte, error)
	Cookie(name string) (string, error)
	SetCookie(name, value string, maxAge int, path, domain string, secure, httpOnly bool)
	Status(code int)
	SetHeader(name, value string)
	JSON(data any) error
	Data(contentType string, data []byte) error
	HTML(name string, data any) error
	Redirect(code int, url string)
	Set(key string, value any)
	Get(key string) (any, bool)
	Context() context.Context
	Unwrap() any
}