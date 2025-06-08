package models

type Base struct {
	*Root
	OrganizationID string       `json:"-" validate:"required,uidv4"`
	Organization   Organization `json:"-"`
}
