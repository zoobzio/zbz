package models

type Entity struct {
	Model string `json:"-" validate:"oneof=contact company"`
}

type EntityRecord struct {
	Entity
	Record string `json:"-" validate:"uuidv4"`
}
