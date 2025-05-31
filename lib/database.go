package zbz

import (
	"github.com/go-playground/validator/v10"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// Database provides methods to interact with the database.
type Database interface {
	Create(value any) *gorm.DB
	First(dest any, conds ...any) *gorm.DB
	Model(value any) *gorm.DB
	Delete(value any, conds ...any) *gorm.DB

	IsValidID(v any) error
	IsValid(v any) error
}

// ZbzDatabase holds the configuration for the database connection.
type ZbzDatabase struct {
	*gorm.DB
	*validator.Validate
	config Config
	log    Logger
}

// NewDatabase initializes a new Database instance with the provided configuration.
func NewDatabase(l Logger, c Config) Database {
	dsn := c.DSN()

	cx, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		l.Fatal("Failed to connect to database:", err)
	}

	v := validator.New()

	return &ZbzDatabase{
		DB:       cx,
		Validate: v,
		config:   c,
		log:      l,
	}
}

// IsValidID checks if the provided value is a valid UUID.
func (d *ZbzDatabase) IsValidID(v any) error {
	if err := d.Var(v, "uuid"); err != nil {
		return err
	}
	return nil
}

// IsValid checks if the provided value is valid according to the struct tags.
func (d *ZbzDatabase) IsValid(v any) error {
	if err := d.Struct(v); err != nil {
		return err
	}
	return nil
}
