package zbz

import (
	"fmt"
	"os"
)

// Config defines the interface for configuration management in the ZBZ application.
type Config interface {
	Host() string
	Port() string
	Title() string
	Version() string
	Description() string

	AuthDomain() string
	AuthClientID() string
	AuthClientSecret() string
	AuthCallback() string

	DSN() string
}

// ZbzConfig implements the Config interface and holds the configuration for the ZBZ application.
type ZbzConfig struct {
	logger Logger

	host        string
	port        string
	title       string
	version     string
	description string

	authDomain       string
	authClientID     string
	authClientSecret string
	authCallback     string

	pgHost     string
	pgPort     string
	pgDB       string
	pgUser     string
	pgPassword string
}

// NewConfig loads the configuration from environment variables or uses default values.
func NewConfig(l Logger) Config {
	return &ZbzConfig{
		logger:           l,
		host:             os.Getenv("API_HOST"),
		port:             os.Getenv("API_PORT"),
		title:            os.Getenv("API_TITLE"),
		version:          os.Getenv("API_VERSION"),
		description:      os.Getenv("API_DESCRIPTION"),
		authDomain:       os.Getenv("AUTH0_DOMAIN"),
		authClientID:     os.Getenv("AUTH0_CLIENT_ID"),
		authClientSecret: os.Getenv("AUTH0_CLIENT_SECRET"),
		authCallback:     os.Getenv("AUTH0_CALLBACK"),
		pgHost:           os.Getenv("POSTGRES_HOST"),
		pgPort:           os.Getenv("POSTGRES_PORT"),
		pgDB:             os.Getenv("POSTGRES_DB"),
		pgUser:           os.Getenv("POSTGRES_USER"),
		pgPassword:       os.Getenv("POSTGRES_PASSWORD"),
	}
}

// Host returns the host for the API server.
func (c *ZbzConfig) Host() string {
	return c.host
}

// Port returns the port for the API server.
func (c *ZbzConfig) Port() string {
	return c.port
}

// Title returns the title of the API.
func (c *ZbzConfig) Title() string {
	return c.title
}

// Version returns the version of the API.
func (c *ZbzConfig) Version() string {
	return c.version
}

// Description returns the description of the API.
func (c *ZbzConfig) Description() string {
	return c.description
}

// AuthDomain returns the Auth0 domain for authentication.
func (c *ZbzConfig) AuthDomain() string {
	return c.authDomain
}

// AuthClientID returns the Auth0 client ID for authentication.
func (c *ZbzConfig) AuthClientID() string {
	return c.authClientID
}

// AuthClientSecret returns the Auth0 client secret for authentication.
func (c *ZbzConfig) AuthClientSecret() string {
	return c.authClientSecret
}

// AuthCallback returns the Auth0 callback URL for authentication.
func (c *ZbzConfig) AuthCallback() string {
	return c.authCallback
}

// DSN returns the Data Source Name for connecting to the PostgreSQL database.
func (c *ZbzConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=disable",
		c.pgHost,
		c.pgUser,
		c.pgPassword,
		c.pgDB,
		c.pgPort,
	)
}
