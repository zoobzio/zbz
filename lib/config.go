package zbz

import (
	"fmt"
	"os"

	"github.com/go-playground/validator/v10"
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

// zConfig implements the Config interface and holds the configuration for the ZBZ application.
type zConfig struct {
	host        string `validate:"required"`
	port        string `validate:"required"`
	title       string `validate:"required"`
	version     string `validate:"required"`
	description string `validate:"required"`

	authDomain       string `validate:"required"`
	authClientID     string `validate:"required"`
	authClientSecret string `validate:"required"`
	authCallback     string `validate:"required"`

	pgHost     string `validate:"required"`
	pgPort     string `validate:"required"`
	pgDB       string `validate:"required"`
	pgUser     string `validate:"required"`
	pgPassword string `validate:"required"`
}

// config is a global variable that holds the application configuration.
var config Config

// Host returns the host for the API server.
func (c *zConfig) Host() string {
	return c.host
}

// Port returns the port for the API server.
func (c *zConfig) Port() string {
	return c.port
}

// Title returns the title of the API.
func (c *zConfig) Title() string {
	return c.title
}

// Version returns the version of the API.
func (c *zConfig) Version() string {
	return c.version
}

// Description returns the description of the API.
func (c *zConfig) Description() string {
	return c.description
}

// AuthDomain returns the Auth0 domain for authentication.
func (c *zConfig) AuthDomain() string {
	return c.authDomain
}

// AuthClientID returns the Auth0 client ID for authentication.
func (c *zConfig) AuthClientID() string {
	return c.authClientID
}

// AuthClientSecret returns the Auth0 client secret for authentication.
func (c *zConfig) AuthClientSecret() string {
	return c.authClientSecret
}

// AuthCallback returns the Auth0 callback URL for authentication.
func (c *zConfig) AuthCallback() string {
	return c.authCallback
}

// DSN returns the Data Source Name for connecting to the PostgreSQL database.
func (c *zConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=disable",
		c.pgHost,
		c.pgUser,
		c.pgPassword,
		c.pgDB,
		c.pgPort,
	)
}

// init initializes the configuration by reading environment variables.
func init() {
	config = &zConfig{
		host:        os.Getenv("API_HOST"),
		port:        os.Getenv("API_PORT"),
		title:       os.Getenv("API_TITLE"),
		version:     os.Getenv("API_VERSION"),
		description: os.Getenv("API_DESCRIPTION"),

		authDomain:       os.Getenv("AUTH0_DOMAIN"),
		authClientID:     os.Getenv("AUTH0_CLIENT_ID"),
		authClientSecret: os.Getenv("AUTH0_CLIENT_SECRET"),
		authCallback:     os.Getenv("AUTH0_CALLBACK"),

		pgHost:     os.Getenv("POSTGRES_HOST"),
		pgPort:     os.Getenv("POSTGRES_PORT"),
		pgDB:       os.Getenv("POSTGRES_DB"),
		pgUser:     os.Getenv("POSTGRES_USER"),
		pgPassword: os.Getenv("POSTGRES_PASSWORD"),
	}

	v := validator.New()
	err := v.Struct(config)
	if err != nil {
		panic(fmt.Sprintf("Invalid configuration: %v", err))
	}
}
