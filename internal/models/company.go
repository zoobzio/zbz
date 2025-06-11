package models

import "zbz/lib"

type Company struct {
	zbz.Model
	Name   string `json:"name" validate:"required" desc:"Company name" edit:"Owner" ex:"Example Corp"`
	Domain string `json:"domain" validate:"required" desc:"Company domain" edit:"Owner" ex:"example.com"`
}
