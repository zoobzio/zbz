package models

import (
	"encoding/json"
	zbz "zbz/lib"
)

type Property struct {
	zbz.Model
	FieldID     string  `db:"field_id" json:"-"`
	Field       Field   `db:"-" json:"field"`
	StringValue string  `db:"string_value" json:"-"`
	IntValue    int     `db:"int_value" json:"-"`
	FloatValue  float64 `db:"float_value" json:"-"`
	BoolValue   bool    `db:"bool_value" json:"-"`
	DateValue   string  `db:"date_value" json:"-"`
	TimeValue   string  `db:"time_value" json:"-"`
}

func (p Property) Value() any {
	switch p.Field.Type {
	case "string":
		return p.StringValue
	case "int":
		return p.IntValue
	case "float":
		return p.FloatValue
	case "bool":
		return p.BoolValue
	case "date":
		return p.DateValue
	case "time":
		return p.TimeValue
	}
	return nil
}

func (p Property) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		ID    string `json:"id"`
		Field Field  `json:"field"`
		Value any    `json:"value"`
	}{
		ID:    p.ID,
		Field: p.Field,
		Value: p.Value(),
	})
}
