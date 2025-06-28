package cereal

import (
	"fmt"
	"reflect"
)

// MergeOptions defines how merging should be performed
type MergeOptions struct {
	ArrayStrategy  ArrayMergeStrategy  // How to handle array fields
	StructStrategy StructMergeStrategy // How to handle nested structs
	MapStrategy    MapMergeStrategy    // How to handle map fields
	NilStrategy    NilMergeStrategy    // How to handle nil/zero values
	FieldRules     map[string]FieldMergeRule // Per-field merge rules
}

// Array merge strategies
type ArrayMergeStrategy int

const (
	ArrayReplace ArrayMergeStrategy = iota // Replace entire array
	ArrayAppend                            // Append to default array
	ArrayUnion                             // Union of both arrays (unique items)
)

// Struct merge strategies
type StructMergeStrategy int

const (
	StructDeepMerge StructMergeStrategy = iota // Recursively merge struct fields
	StructReplace                              // Replace entire struct
	StructSkipZero                             // Skip zero-value structs
)

// Map merge strategies
type MapMergeStrategy int

const (
	MapDeepMerge MapMergeStrategy = iota // Recursively merge map values
	MapReplace                           // Replace entire map
	MapShallow                           // Only merge top-level keys
)

// Nil handling strategies
type NilMergeStrategy int

const (
	NilKeepDefaults NilMergeStrategy = iota // Keep defaults when override is nil/zero
	NilAllowOverride                        // Allow nil/zero to override defaults
)

// Field-specific merge rules
type FieldMergeRule struct {
	Strategy string                           // "replace", "deep", "skip", "union", "append"
	Transform func(any, any) any            // Custom merge function for this field
}

// Default merge options
func DefaultMergeOptions() MergeOptions {
	return MergeOptions{
		ArrayStrategy:  ArrayReplace,
		StructStrategy: StructDeepMerge,
		MapStrategy:    MapDeepMerge,
		NilStrategy:    NilKeepDefaults,
		FieldRules:     make(map[string]FieldMergeRule),
	}
}

// MergeEngine handles the merging logic
type MergeEngine struct {
	options MergeOptions
}

// NewMergeEngine creates a new merge engine with the given options
func NewMergeEngine(options MergeOptions) *MergeEngine {
	return &MergeEngine{
		options: options,
	}
}

// Merge combines defaults and overrides using default options
func Merge[T any](defaults T, overrides T) T {
	engine := NewMergeEngine(DefaultMergeOptions())
	return engine.Merge(defaults, overrides).(T)
}

// MergeWithOptions combines defaults and overrides using custom options
func MergeWithOptions[T any](defaults T, overrides T, options MergeOptions) T {
	engine := NewMergeEngine(options)
	return engine.Merge(defaults, overrides).(T)
}

// Merge performs the actual merge operation
func (me *MergeEngine) Merge(defaults, overrides any) any {
	defaultVal := reflect.ValueOf(defaults)
	overrideVal := reflect.ValueOf(overrides)
	
	// Handle nil cases
	if !defaultVal.IsValid() {
		return overrides
	}
	if !overrideVal.IsValid() {
		return defaults
	}
	
	merged := me.mergeValues(defaultVal, overrideVal)
	return merged.Interface()
}

// mergeValues recursively merges two reflect.Values
func (me *MergeEngine) mergeValues(defaults, overrides reflect.Value) reflect.Value {
	// If types don't match, return override (or default if override is zero)
	if defaults.Type() != overrides.Type() {
		if me.isZeroValue(overrides) && me.options.NilStrategy == NilKeepDefaults {
			return defaults
		}
		return overrides
	}
	
	// Handle zero values based on strategy
	if me.isZeroValue(overrides) && me.options.NilStrategy == NilKeepDefaults {
		return defaults
	}
	
	// Handle different types
	switch defaults.Kind() {
	case reflect.Struct:
		return me.mergeStructs(defaults, overrides)
	case reflect.Map:
		return me.mergeMaps(defaults, overrides)
	case reflect.Slice, reflect.Array:
		return me.mergeSlices(defaults, overrides)
	case reflect.Ptr:
		return me.mergePointers(defaults, overrides)
	case reflect.Interface:
		return me.mergeInterfaces(defaults, overrides)
	default:
		// Primitives: override wins unless it's zero and we keep defaults
		if me.isZeroValue(overrides) && me.options.NilStrategy == NilKeepDefaults {
			return defaults
		}
		return overrides
	}
}

// mergeStructs handles struct merging with tag support
func (me *MergeEngine) mergeStructs(defaults, overrides reflect.Value) reflect.Value {
	if me.options.StructStrategy == StructReplace {
		return overrides
	}
	
	if me.options.StructStrategy == StructSkipZero && me.isZeroValue(overrides) {
		return defaults
	}
	
	// Deep merge - create new struct and merge each field
	result := reflect.New(defaults.Type()).Elem()
	
	for i := 0; i < defaults.NumField(); i++ {
		field := defaults.Type().Field(i)
		defaultField := defaults.Field(i)
		overrideField := overrides.Field(i)
		
		// Skip unexported fields
		if !field.IsExported() {
			result.Field(i).Set(defaultField)
			continue
		}
		
		// Check for field-specific merge rules
		if rule, exists := me.getFieldRule(field); exists {
			mergedField := me.applyFieldRule(rule, defaultField, overrideField)
			result.Field(i).Set(mergedField)
			continue
		}
		
		// Check merge tag
		mergeTag := field.Tag.Get("merge")
		if mergeTag != "" {
			mergedField := me.applyMergeTag(mergeTag, defaultField, overrideField)
			result.Field(i).Set(mergedField)
			continue
		}
		
		// Default: recursive merge
		mergedField := me.mergeValues(defaultField, overrideField)
		result.Field(i).Set(mergedField)
	}
	
	return result
}

// mergeMaps handles map merging
func (me *MergeEngine) mergeMaps(defaults, overrides reflect.Value) reflect.Value {
	if me.options.MapStrategy == MapReplace || me.isZeroValue(defaults) {
		return overrides
	}
	
	if me.isZeroValue(overrides) {
		return defaults
	}
	
	// Create new map
	result := reflect.MakeMap(defaults.Type())
	
	// Copy defaults
	for _, key := range defaults.MapKeys() {
		result.SetMapIndex(key, defaults.MapIndex(key))
	}
	
	// Merge overrides
	for _, key := range overrides.MapKeys() {
		overrideVal := overrides.MapIndex(key)
		
		if defaultVal := defaults.MapIndex(key); defaultVal.IsValid() {
			// Key exists in both - merge values
			if me.options.MapStrategy == MapDeepMerge {
				mergedVal := me.mergeValues(defaultVal, overrideVal)
				result.SetMapIndex(key, mergedVal)
			} else {
				result.SetMapIndex(key, overrideVal)
			}
		} else {
			// Key only in override - add directly
			result.SetMapIndex(key, overrideVal)
		}
	}
	
	return result
}

// mergeSlices handles slice/array merging
func (me *MergeEngine) mergeSlices(defaults, overrides reflect.Value) reflect.Value {
	if me.isZeroValue(overrides) {
		return defaults
	}
	
	if me.isZeroValue(defaults) {
		return overrides
	}
	
	switch me.options.ArrayStrategy {
	case ArrayReplace:
		return overrides
		
	case ArrayAppend:
		result := reflect.MakeSlice(defaults.Type(), 0, defaults.Len()+overrides.Len())
		result = reflect.AppendSlice(result, defaults)
		result = reflect.AppendSlice(result, overrides)
		return result
		
	case ArrayUnion:
		return me.mergeSlicesUnion(defaults, overrides)
		
	default:
		return overrides
	}
}

// mergeSlicesUnion creates a union of two slices (unique elements)
func (me *MergeEngine) mergeSlicesUnion(defaults, overrides reflect.Value) reflect.Value {
	seen := make(map[any]bool)
	result := reflect.MakeSlice(defaults.Type(), 0, defaults.Len()+overrides.Len())
	
	// Add defaults
	for i := 0; i < defaults.Len(); i++ {
		item := defaults.Index(i)
		key := item.Interface()
		if !seen[key] {
			result = reflect.Append(result, item)
			seen[key] = true
		}
	}
	
	// Add overrides (skip duplicates)
	for i := 0; i < overrides.Len(); i++ {
		item := overrides.Index(i)
		key := item.Interface()
		if !seen[key] {
			result = reflect.Append(result, item)
			seen[key] = true
		}
	}
	
	return result
}

// mergePointers handles pointer merging
func (me *MergeEngine) mergePointers(defaults, overrides reflect.Value) reflect.Value {
	if overrides.IsNil() {
		if me.options.NilStrategy == NilAllowOverride {
			return overrides // Return nil
		}
		return defaults
	}
	
	if defaults.IsNil() {
		return overrides
	}
	
	// Both are non-nil, merge the pointed-to values
	mergedElem := me.mergeValues(defaults.Elem(), overrides.Elem())
	result := reflect.New(defaults.Type().Elem())
	result.Elem().Set(mergedElem)
	return result
}

// mergeInterfaces handles interface merging
func (me *MergeEngine) mergeInterfaces(defaults, overrides reflect.Value) reflect.Value {
	if overrides.IsNil() {
		return defaults
	}
	
	if defaults.IsNil() {
		return overrides
	}
	
	// If both contain the same concrete type, merge them
	if defaults.Elem().Type() == overrides.Elem().Type() {
		merged := me.mergeValues(defaults.Elem(), overrides.Elem())
		result := reflect.New(defaults.Type()).Elem()
		result.Set(merged)
		return result
	}
	
	// Different types - override wins
	return overrides
}

// getFieldRule gets the merge rule for a field
func (me *MergeEngine) getFieldRule(field reflect.StructField) (FieldMergeRule, bool) {
	rule, exists := me.options.FieldRules[field.Name]
	return rule, exists
}

// applyFieldRule applies a custom field rule
func (me *MergeEngine) applyFieldRule(rule FieldMergeRule, defaults, overrides reflect.Value) reflect.Value {
	if rule.Transform != nil {
		result := rule.Transform(defaults.Interface(), overrides.Interface())
		return reflect.ValueOf(result)
	}
	
	switch rule.Strategy {
	case "replace":
		if me.isZeroValue(overrides) && me.options.NilStrategy == NilKeepDefaults {
			return defaults
		}
		return overrides
	case "deep":
		return me.mergeValues(defaults, overrides)
	case "skip":
		return defaults
	case "union":
		if defaults.Kind() == reflect.Slice {
			savedStrategy := me.options.ArrayStrategy
			me.options.ArrayStrategy = ArrayUnion
			result := me.mergeSlices(defaults, overrides)
			me.options.ArrayStrategy = savedStrategy
			return result
		}
		return me.mergeValues(defaults, overrides)
	case "append":
		if defaults.Kind() == reflect.Slice {
			savedStrategy := me.options.ArrayStrategy
			me.options.ArrayStrategy = ArrayAppend
			result := me.mergeSlices(defaults, overrides)
			me.options.ArrayStrategy = savedStrategy
			return result
		}
		return me.mergeValues(defaults, overrides)
	default:
		return me.mergeValues(defaults, overrides)
	}
}

// applyMergeTag applies struct tag merge rules
func (me *MergeEngine) applyMergeTag(tag string, defaults, overrides reflect.Value) reflect.Value {
	switch tag {
	case "replace":
		if me.isZeroValue(overrides) && me.options.NilStrategy == NilKeepDefaults {
			return defaults
		}
		return overrides
	case "deep":
		return me.mergeValues(defaults, overrides)
	case "skip":
		return defaults
	case "union":
		if defaults.Kind() == reflect.Slice {
			savedStrategy := me.options.ArrayStrategy
			me.options.ArrayStrategy = ArrayUnion
			result := me.mergeSlices(defaults, overrides)
			me.options.ArrayStrategy = savedStrategy
			return result
		}
		return me.mergeValues(defaults, overrides)
	case "append":
		if defaults.Kind() == reflect.Slice {
			savedStrategy := me.options.ArrayStrategy
			me.options.ArrayStrategy = ArrayAppend
			result := me.mergeSlices(defaults, overrides)
			me.options.ArrayStrategy = savedStrategy
			return result
		}
		return me.mergeValues(defaults, overrides)
	default:
		return me.mergeValues(defaults, overrides)
	}
}

// isZeroValue checks if a value is the zero value for its type
func (me *MergeEngine) isZeroValue(v reflect.Value) bool {
	if !v.IsValid() {
		return true
	}
	
	switch v.Kind() {
	case reflect.Array, reflect.Map, reflect.Slice, reflect.String:
		return v.Len() == 0
	case reflect.Bool:
		return !v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Interface, reflect.Ptr:
		return v.IsNil()
	case reflect.Struct:
		return v.IsZero()
	default:
		return false
	}
}

// MergeMultiple merges multiple sources left-to-right (later sources override earlier ones)
func MergeMultiple[T any](sources ...T) T {
	if len(sources) == 0 {
		var zero T
		return zero
	}
	
	if len(sources) == 1 {
		return sources[0]
	}
	
	result := sources[0]
	for i := 1; i < len(sources); i++ {
		result = Merge(result, sources[i])
	}
	
	return result
}

// MergeWithValidation merges and validates the result
func MergeWithValidation[T any](defaults T, overrides T, validator func(T) error) (T, error) {
	merged := Merge(defaults, overrides)
	
	if validator != nil {
		if err := validator(merged); err != nil {
			return defaults, fmt.Errorf("merge validation failed: %w", err)
		}
	}
	
	return merged, nil
}