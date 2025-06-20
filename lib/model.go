package zbz

import (
	"reflect"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// BaseModel is a generic type that can be used to define models with common fields.
type BaseModel any

// Model is a base model that includes common fields such as ID, CreatedAt, and UpdatedAt.
type Model struct {
	ID        string    `db:"id" json:"id" validate:"required,uuid4" desc:"A unique identifier" ex:"123e4567-e89b-12d3-a456-426614174000"`
	CreatedAt time.Time `db:"created_at" json:"createdAt" validate:"required" desc:"The time the user was created" ex:"2023-10-01T12:00:00Z"`
	UpdatedAt time.Time `db:"updated_at" json:"updatedAt" validate:"required" desc:"The time the user was last updated" ex:"2023-10-01T12:00:00Z"`
}

// NewModel creates a new instance of a model with the provided context.
func NewModel[T BaseModel](ctx *gin.Context) (*T, error) {
	var obj T

	if err := ctx.ShouldBindJSON(&obj); err != nil {
		return nil, err
	}

	v := reflect.ValueOf(&obj).Elem()
	modelField := v.FieldByName("Model")
	if modelField.IsValid() && modelField.CanSet() {
		model := Model{
			ID:        uuid.NewString(),
			CreatedAt: time.Now().UTC(),
			UpdatedAt: time.Now().UTC(),
		}
		modelField.Set(reflect.ValueOf(model))
	}

	return &obj, nil
}

// PatchModel overlays fields from the JSON payload in ctx onto an existing model instance.
// Only fields in the JSON will be updated; others remain as in the original.
func PatchModel[T BaseModel](ctx *gin.Context, existing *T) (*T, error) {
	// Make a shallow copy so the original stays unchanged
	result := *existing

	// Bind JSON fields onto the copy (only fields present in the payload will change)
	if err := ctx.ShouldBindJSON(&result); err != nil {
		return nil, err
	}

	// Optionally, update the UpdatedAt field if your model embeds zbz.Model
	v := reflect.ValueOf(&result).Elem()
	modelField := v.FieldByName("Model")
	if modelField.IsValid() && modelField.CanSet() {
		// Only update UpdatedAt, preserve ID and CreatedAt
		model := modelField.Interface().(Model)
		model.UpdatedAt = time.Now().UTC()
		modelField.Set(reflect.ValueOf(model))
	}

	return &result, nil
}
