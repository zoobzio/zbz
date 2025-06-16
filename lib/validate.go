package zbz

import (
	"github.com/go-playground/validator/v10"
)

// Validate is an interface that defines methods for validating IDs and other values.
type Validate interface {
	IsValidID(v any) error
	IsValid(v any) error
}

// zValidate implements the Validate interface using the go-playground/validator package.
type zValidate struct {
	*validator.Validate
}

// IsValidID checks if the provided value is a valid UUID.
func (d *zValidate) IsValidID(v any) error {
	if err := d.Var(v, "uuid"); err != nil {
		return err
	}
	return nil
}

// IsValid checks if the provided value is valid according to the struct tags.
func (d *zValidate) IsValid(v any) error {
	if err := d.Struct(v); err != nil {
		return err
	}
	return nil
}

// validate is a global instance of the Validate interface.
var validate Validate

// NewValidate initializes a new Validate instance.
func init() {
	validate = &zValidate{validator.New()}
}
