package models

type Field struct {
	Base
	Entity
	Type string `json:"type" validate:"oneof=string int float bool date time"`
}
