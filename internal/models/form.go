package models

import zbz "zbz/lib"

type Form struct {
	zbz.Model
	Name string `json:"name" validate:"required" desc:"Form name" edit:"Owner" ex:"User Registration"`
}
