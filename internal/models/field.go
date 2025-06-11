package models

import zbz "zbz/lib"

type Field struct {
	zbz.Model
	Name string `json:"name" validate:"required" desc:"Field name" ex:"username"`
	Type string `json:"type" validate:"oneof=string int float bool date time" desc:"Field type" ex:"string"`
}
