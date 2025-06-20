package zbz

import (
	"maps"
	"reflect"
	"strconv"
	"strings"
	"time"
)

// Meta defines the metadata for a core resource, including its name, description, and example.
type Meta struct {
	// Name is the name of the model which is derived from the type name.
	Name string

	// SourceName is the database column name
	SourceName string

	// DstName is the name of the field after it has been serialized to JSON
	DstName string

	// Description provides a human-readable description of the field
	Description string

	// Type is the Go type of the field, such as int, string, time.Time, etc.
	Type string

	// SourceType is the data type of the field as it related to the database
	SourceType string

	// Example provides an example value for the field, which can be of any type.
	Example any

	// Required indicates whether the field is required or not.
	Required bool

	// Validate is a string that contains validation rules for the field, such as "required", "email", etc.
	Validate string

	// Edit is a string that indicates how the field should be edited, such as "text", "select", etc.
	Edit string

	// Columns is a list of database column names that correspond to the fields in the model.
	Columns []string

	// Fields is a slice of Meta that contains metadata for each field in the model.
	Fields []*Meta
}

// extractFields extracts fields from a given type and returns metadata about them.
func extractFields(t reflect.Type) ([]*Meta, []string, map[string]any) {
	f := make([]*Meta, 0, t.NumField())
	c := []string{}
	ex := make(map[string]any)
	for i := range t.NumField() {
		field := t.Field(i)

		n := field.Name
		d := field.Tag.Get("db")
		j := field.Tag.Get("json")
		s := field.Tag.Get("desc")
		v := field.Tag.Get("validate")
		e := field.Tag.Get("edit")
		x := field.Tag.Get("ex")
		t := field.Type.String()
		st := "text"

		var exv any
		switch t {
		case "zbz.Model":
			// Skip the base model fields, these are handled separately
			continue
		case "int", "int32":
			i, _ := strconv.Atoi(x)
			exv = i
			st = "integer"
		case "int64":
			var i int64
			i, _ = strconv.ParseInt(x, 10, 64)
			exv = i
			st = "bigint"
		case "float32":
			var f float64
			f, _ = strconv.ParseFloat(x, 32)
			exv = float32(f)
			st = "real"
		case "float64":
			var f float64
			f, _ = strconv.ParseFloat(x, 64)
			exv = f
			st = "double precision"
		case "string":
			exv = x
		case "bool":
			var b bool
			b, _ = strconv.ParseBool(x)
			exv = b
			st = "boolean"
		case "time.Time":
			var t time.Time
			t, _ = time.Parse(time.RFC3339, x)
			exv = t
			st = "timestamp with time zone"
		case "[]byte":
			exv = []byte(x)
			st = "bytea"
		}

		f = append(f, &Meta{
			Name:        n,
			SourceName:  d,
			DstName:     j,
			Description: s,
			Type:        t,
			SourceType:  st,
			Required:    strings.Contains(v, "required"),
			Validate:    v,
			Edit:        e,
			Example:     exv,
		})
		if d != "-" {
			c = append(c, d)
		}
		ex[d] = exv
	}

	return f, c, ex
}

// ExtractMeta extracts metadata from a given model type T, which must implement BaseModel.
func extractMeta[T BaseModel]() *Meta {
	var model T
	t := reflect.TypeOf(model)

	var base Model
	bt := reflect.TypeOf(base)

	meta := &Meta{
		Name:        t.Name(),
		Description: "fix me", // c.description,
		Fields:      make([]*Meta, 0, t.NumField()+bt.NumField()),
		Columns:     make([]string, 0, t.NumField()+bt.NumField()),
	}

	tf, tc, te := extractFields(t)
	btf, btc, bte := extractFields(bt)

	meta.Fields = append(meta.Fields, tf...)
	meta.Fields = append(meta.Fields, btf...)

	meta.Columns = append(meta.Columns, tc...)
	meta.Columns = append(meta.Columns, btc...)

	maps.Copy(te, bte)
	meta.Example = te

	return meta
}
