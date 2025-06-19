package models

import zbz "zbz/lib"

type Field struct {
	zbz.Model
	Name string `db:"name" json:"name" validate:"required" desc:"Field name" ex:"username"`
	Type string `db:"type" json:"type" validate:"oneof=string int float bool date time" desc:"Field type" ex:"string"`
}
