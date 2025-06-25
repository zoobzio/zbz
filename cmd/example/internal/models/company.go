package models

import "zbz/api"

type Company struct {
	zbz.Model
	Name   string `db:"name" json:"name" validate:"required" desc:"Company name" edit:"Owner" ex:"Example Corp"`
	Domain string `db:"domain" json:"domain" validate:"required" desc:"Company domain" edit:"Owner" ex:"example.com"`
}
