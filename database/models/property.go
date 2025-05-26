package models

import (
	"encoding/json"
)

type Property struct {
	Base
	FieldID     string  `json:"-"`
	Field       Field   `json:"field"`
	StringValue string  `json:"-"`
	IntValue    int     `json:"-"`
	FloatValue  float64 `json:"-"`
	BoolValue   bool    `json:"-"`
	DateValue   string  `json:"-"`
	TimeValue   string  `json:"-"`
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
