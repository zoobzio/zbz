package zbz

import (
	"maps"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/docker/distribution/uuid"
)

type BaseModel interface {
	NewModelDefaults() *Model
}

// Meta defines the metadata for a core resource, including its name, description, and example.
type Meta struct {
	Name        string
	Description string
	Type        string
	Example     any
	Required    bool
	Validate    string
	Edit        string
	Fields      []*Meta
}

// Model is a base model that includes common fields such as ID, CreatedAt, and UpdatedAt.
type Model struct {
	ID        string    `json:"id" validate:"required,uuidv4" desc:"A unique identifier" ex:"123e4567-e89b-12d3-a456-426614174000"`
	CreatedAt time.Time `json:"createdAt" validate:"required" desc:"The time the user was created" ex:"2023-10-01T12:00:00Z"`
	UpdatedAt time.Time `json:"updatedAt" validate:"required" desc:"The time the user was last updated" ex:"2023-10-01T12:00:00Z"`
}

// New creates a new instance of Model with a unique ID and current timestamps.
func (m Model) NewModelDefaults() *Model {
	return &Model{
		ID:        uuid.Generate().String(),
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
}

func ExtractFields(t reflect.Type) ([]*Meta, map[string]any) {
	f := make([]*Meta, 0, t.NumField())
	ex := make(map[string]any)
	for i := range t.NumField() {
		field := t.Field(i)

		n := field.Tag.Get("json")
		s := field.Tag.Get("desc")
		v := field.Tag.Get("validate")
		e := field.Tag.Get("edit")
		x := field.Tag.Get("ex")
		t := field.Type.String()

		var exv any
		switch t {
		case "zbz.Model":
			// Skip the base model fields, these are handled separately
			continue
		case "int", "int32":
			i, _ := strconv.Atoi(x)
			exv = i
		case "int64":
			var i int64
			i, _ = strconv.ParseInt(x, 10, 64)
			exv = i
		case "float32":
			var f float64
			f, _ = strconv.ParseFloat(x, 32)
			exv = float32(f)
		case "float64":
			var f float64
			f, _ = strconv.ParseFloat(x, 64)
			exv = f
		case "string":
			exv = x
		case "bool":
			var b bool
			b, _ = strconv.ParseBool(x)
			exv = b
		case "time.Time":
			var t time.Time
			t, _ = time.Parse(time.RFC3339, x)
			exv = t
		case "[]byte":
			exv = []byte(x)
		}

		f = append(f, &Meta{
			Name:        n,
			Description: s,
			Type:        t,
			Required:    strings.Contains(v, "required"),
			Validate:    v,
			Edit:        e,
			Example:     exv,
		})
		ex[n] = exv
	}

	return f, ex
}

// ExtractMeta extracts metadata from a given model type T, which must implement BaseModel.
func ExtractMeta[T BaseModel]() *Meta {
	var model T
	t := reflect.TypeOf(model)

	var base Model
	bt := reflect.TypeOf(base)

	meta := &Meta{
		Name:        t.Name(),
		Description: "fix me", // c.description,
		Fields:      make([]*Meta, 0, t.NumField()+bt.NumField()),
	}

	tf, te := ExtractFields(t)
	btf, bte := ExtractFields(bt)

	meta.Fields = append(meta.Fields, tf...)
	meta.Fields = append(meta.Fields, btf...)

	maps.Copy(te, bte)
	meta.Example = te

	return meta
}
