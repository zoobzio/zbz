package models

type Company struct {
	Base
	Domain string `json:"domain" validate:"required"`
}
