package models

import zbz "zbz/api"

type Form struct {
	zbz.Model
	Name string `db:"name" json:"name" validate:"required" desc:"Form name" edit:"Owner" ex:"User Registration"`
}
