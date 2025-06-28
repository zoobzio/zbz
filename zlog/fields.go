package zlog

// Field interface for structured logging
type Field interface {
	Key() string
	Value() any
}

// String creates a string field
func String(key, value string) Field {
	return &stringField{key: key, value: value}
}

// Int creates an int field
func Int(key string, value int) Field {
	return &intField{key: key, value: value}
}

// Bool creates a bool field
func Bool(key string, value bool) Field {
	return &boolField{key: key, value: value}
}

// Any creates a field with any value
func Any(key string, value any) Field {
	return &anyField{key: key, value: value}
}

// Error creates an error field
func Error(key string, err error) Field {
	return &errorField{key: key, err: err}
}

// Field implementations

type stringField struct {
	key   string
	value string
}

func (f *stringField) Key() string   { return f.key }
func (f *stringField) Value() any    { return f.value }

type intField struct {
	key   string
	value int
}

func (f *intField) Key() string   { return f.key }
func (f *intField) Value() any    { return f.value }

type boolField struct {
	key   string
	value bool
}

func (f *boolField) Key() string   { return f.key }
func (f *boolField) Value() any    { return f.value }

type anyField struct {
	key   string
	value any
}

func (f *anyField) Key() string   { return f.key }
func (f *anyField) Value() any    { return f.value }

type errorField struct {
	key string
	err error
}

func (f *errorField) Key() string   { return f.key }
func (f *errorField) Value() any    { 
	if f.err != nil {
		return f.err.Error()
	}
	return nil
}