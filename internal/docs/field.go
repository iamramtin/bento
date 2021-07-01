package docs

import (
	"fmt"
	"reflect"
)

// FieldType represents a field type.
type FieldType string

// ValueType variants.
var (
	FieldTypeString  FieldType = "string"
	FieldTypeInt     FieldType = "int"
	FieldTypeFloat   FieldType = "float"
	FieldTypeBool    FieldType = "bool"
	FieldTypeObject  FieldType = "object"
	FieldTypeUnknown FieldType = "unknown"

	// Core component types, only components that can be a child of another
	// component config are listed here.
	FieldTypeInput     FieldType = "input"
	FieldTypeBuffer    FieldType = "buffer"
	FieldTypeCache     FieldType = "cache"
	FieldTypeCondition FieldType = "condition"
	FieldTypeProcessor FieldType = "processor"
	FieldTypeRateLimit FieldType = "rate_limit"
	FieldTypeOutput    FieldType = "output"
	FieldTypeMetrics   FieldType = "metrics"
	FieldTypeTracer    FieldType = "tracer"
)

// IsCoreComponent returns the core component type of a field if applicable.
func (t FieldType) IsCoreComponent() (Type, bool) {
	switch t {
	case FieldTypeInput:
		return TypeInput, true
	case FieldTypeBuffer:
		return TypeBuffer, true
	case FieldTypeCache:
		return TypeCache, true
	case FieldTypeCondition:
		// TODO: V4 Remove this
		return "condition", true
	case FieldTypeProcessor:
		return TypeProcessor, true
	case FieldTypeRateLimit:
		return TypeRateLimit, true
	case FieldTypeOutput:
		return TypeOutput, true
	case FieldTypeTracer:
		return TypeTracer, true
	case FieldTypeMetrics:
		return TypeMetrics, true
	}
	return "", false
}

func getFieldTypeFromInterface(v interface{}) (FieldType, bool) {
	return getFieldTypeFromReflect(reflect.TypeOf(v))
}

func getFieldTypeFromReflect(t reflect.Type) (FieldType, bool) {
	switch t.Kind().String() {
	case "map":
		return FieldTypeObject, false
	case "slice":
		ft, _ := getFieldTypeFromReflect(t.Elem())
		return ft, true
	case "int", "int64":
		return FieldTypeInt, false
	case "float64":
		return FieldTypeFloat, false
	case "string":
		return FieldTypeString, false
	case "bool":
		return FieldTypeBool, false
	}
	return FieldTypeUnknown, false
}

// FieldKind represents a field kind.
type FieldKind string

// ValueType variants.
var (
	KindScalar  FieldKind = "scalar"
	KindArray   FieldKind = "array"
	Kind2DArray FieldKind = "2darray"
	KindMap     FieldKind = "map"
)

//------------------------------------------------------------------------------

// FieldSpec describes a component config field.
type FieldSpec struct {
	// Name of the field (as it appears in config).
	Name string `json:"name"`

	// Type of the field.
	//
	// TODO: Make this mandatory
	Type FieldType `json:"type"`

	// Kind of the field.
	Kind FieldKind `json:"kind"`

	// Description of the field purpose (in markdown).
	Description string `json:"description"`

	// IsAdvanced is true for optional fields that will not be present in most
	// configs.
	IsAdvanced bool `json:"is_advanced"`

	// IsDeprecated is true for fields that are deprecated and only exist
	// for backwards compatibility reasons.
	IsDeprecated bool `json:"is_deprecated"`

	// IsOptional is a boolean flag indicating that a field is optional, even
	// if there is no default. This prevents linting errors when the field
	// is missing.
	IsOptional bool `json:"is_optional"`

	// Default value of the field.
	Default *interface{} `json:"default,omitempty"`

	// Interpolation indicates that the field supports interpolation
	// functions.
	Interpolated bool `json:"interpolated"`

	// Examples is a slice of optional example values for a field.
	Examples []interface{} `json:"examples,omitempty"`

	// AnnotatedOptions for this field. Each option should have a summary.
	AnnotatedOptions [][2]string `json:"annotated_options,omitempty"`

	// Options for this field.
	Options []string `json:"options,omitempty"`

	// Children fields of this field (it must be an object).
	Children FieldSpecs `json:"children,omitempty"`

	// Version is an explicit version when this field was introduced.
	Version string `json:"version,omitempty"`

	omitWhenFn   func(field, parent interface{}) (why string, shouldOmit bool)
	customLintFn LintFunc
	skipLint     bool
}

// IsInterpolated indicates that the field supports interpolation functions.
func (f FieldSpec) IsInterpolated() FieldSpec {
	f.Interpolated = true
	f.customLintFn = LintBloblangField
	return f
}

// HasType returns a new FieldSpec that specifies a specific type.
func (f FieldSpec) HasType(t FieldType) FieldSpec {
	f.Type = t
	return f
}

// Optional marks this field as being optional, and therefore its absence in a
// config is not considered an error if when a default is not provided.
func (f FieldSpec) Optional() FieldSpec {
	f.IsOptional = true
	return f
}

// Advanced marks this field as being advanced, and therefore not commonly used.
func (f FieldSpec) Advanced() FieldSpec {
	f.IsAdvanced = true
	return f
}

// Array determines that this field is an array of the field type.
func (f FieldSpec) Array() FieldSpec {
	f.Kind = KindArray
	return f
}

// ArrayOfArrays determines that this is an array of arrays of the field type.
func (f FieldSpec) ArrayOfArrays() FieldSpec {
	f.Kind = Kind2DArray
	return f
}

// Map determines that this field is a map of arbitrary keys to a field type.
func (f FieldSpec) Map() FieldSpec {
	f.Kind = KindMap
	return f
}

// Scalar determines that this field is a scalar type (the default).
func (f FieldSpec) Scalar() FieldSpec {
	f.Kind = KindScalar
	return f
}

// HasDefault returns a new FieldSpec that specifies a default value.
func (f FieldSpec) HasDefault(v interface{}) FieldSpec {
	f.Default = &v
	return f
}

// AtVersion specifies the version at which this fields behaviour was last
// modified.
func (f FieldSpec) AtVersion(v string) FieldSpec {
	f.Version = v
	return f
}

// HasAnnotatedOptions returns a new FieldSpec that specifies a specific list of
// annotated options. Either
func (f FieldSpec) HasAnnotatedOptions(options ...string) FieldSpec {
	if len(f.Options) > 0 {
		panic("cannot combine annotated and non-annotated options for a field")
	}
	if len(options)%2 != 0 {
		panic("annotated field options must each have a summary")
	}
	for i := 0; i < len(options); i += 2 {
		f.AnnotatedOptions = append(f.AnnotatedOptions, [2]string{
			options[i], options[i+1],
		})
	}
	return f
}

// HasOptions returns a new FieldSpec that specifies a specific list of options.
func (f FieldSpec) HasOptions(options ...string) FieldSpec {
	if len(f.AnnotatedOptions) > 0 {
		panic("cannot combine annotated and non-annotated options for a field")
	}
	f.Options = options
	return f
}

// WithChildren returns a new FieldSpec that has child fields.
func (f FieldSpec) WithChildren(children ...FieldSpec) FieldSpec {
	if len(f.Type) == 0 {
		f.Type = FieldTypeObject
	}
	f.Children = append(f.Children, children...)
	return f
}

// OmitWhen specifies a custom func that, when provided a generic config struct,
// returns a boolean indicating when the field can be safely omitted from a
// config.
func (f FieldSpec) OmitWhen(fn func(field, parent interface{}) (why string, shouldOmit bool)) FieldSpec {
	f.omitWhenFn = fn
	return f
}

// Linter adds a linting function to a field. When linting is performed on a
// config the provided function will be called with a boxed variant of the field
// value, allowing it to perform linting on that value.
func (f FieldSpec) Linter(fn LintFunc) FieldSpec {
	f.customLintFn = fn
	return f
}

// LintOptions enforces that a field value matches one of the provided options
// and returns a linting error if that is not the case. This is currently opt-in
// because some fields express options that are only a subset due to deprecated
// functionality.
//
// TODO: V4 Switch this to opt-out.
func (f FieldSpec) LintOptions() FieldSpec {
	f.customLintFn = func(ctx LintContext, line, col int, value interface{}) []Lint {
		str, ok := value.(string)
		if !ok {
			return []Lint{NewLintWarning(line, fmt.Sprintf("expected string value, got %T", value))}
		}
		if len(f.Options) > 0 {
			for _, optStr := range f.Options {
				if str == optStr {
					return nil
				}
			}
		} else {
			for _, optStr := range f.AnnotatedOptions {
				if str == optStr[0] {
					return nil
				}
			}
		}
		return []Lint{NewLintError(line, fmt.Sprintf("value %v is not a valid option for this field", str))}
	}
	return f
}

// Unlinted returns a field spec that will not be lint checked during a config
// parse.
func (f FieldSpec) Unlinted() FieldSpec {
	f.skipLint = true
	return f
}

func (f FieldSpec) shouldOmit(field, parent interface{}) (string, bool) {
	if f.omitWhenFn == nil {
		return "", false
	}
	return f.omitWhenFn(field, parent)
}

// FieldString returns a field spec for a common string typed field.
func FieldString(name, description string, examples ...interface{}) FieldSpec {
	return FieldCommon(name, description, examples...).HasType(FieldTypeString)
}

// FieldInt returns a field spec for a common int typed field.
func FieldInt(name, description string, examples ...interface{}) FieldSpec {
	return FieldCommon(name, description, examples...).HasType(FieldTypeInt)
}

// FieldFloat returns a field spec for a common float typed field.
func FieldFloat(name, description string, examples ...interface{}) FieldSpec {
	return FieldCommon(name, description, examples...).HasType(FieldTypeFloat)
}

// FieldBool returns a field spec for a common bool typed field.
func FieldBool(name, description string, examples ...interface{}) FieldSpec {
	return FieldCommon(name, description, examples...).HasType(FieldTypeBool)
}

// FieldAdvanced returns a field spec for an advanced field.
func FieldAdvanced(name, description string, examples ...interface{}) FieldSpec {
	return FieldSpec{
		Name:        name,
		Description: description,
		Kind:        KindScalar,
		IsAdvanced:  true,
		Examples:    examples,
	}
}

// FieldCommon returns a field spec for a common field.
func FieldCommon(name, description string, examples ...interface{}) FieldSpec {
	return FieldSpec{
		Name:        name,
		Description: description,
		Kind:        KindScalar,
		Examples:    examples,
	}
}

// FieldComponent returns a field spec for a component.
func FieldComponent() FieldSpec {
	return FieldSpec{
		Kind: KindScalar,
	}
}

// FieldDeprecated returns a field spec for a deprecated field.
func FieldDeprecated(name string, description ...string) FieldSpec {
	desc := "DEPRECATED: Do not use."
	if len(description) > 0 {
		desc = "DEPRECATED: " + description[0]
	}
	return FieldSpec{
		Name:         name,
		Description:  desc,
		Kind:         KindScalar,
		IsDeprecated: true,
	}
}

func (f FieldSpec) sanitise(s interface{}, filter FieldFilter) {
	if coreType, isCore := f.Type.IsCoreComponent(); isCore {
		switch f.Kind {
		case KindArray:
			if arr, ok := s.([]interface{}); ok {
				for _, ele := range arr {
					_ = SanitiseComponentConfig(coreType, ele, filter)
				}
			}
		case KindMap:
			if obj, ok := s.(map[string]interface{}); ok {
				for _, v := range obj {
					_ = SanitiseComponentConfig(coreType, v, filter)
				}
			}
		default:
			_ = SanitiseComponentConfig(coreType, s, filter)
		}
	} else if len(f.Children) > 0 {
		switch f.Kind {
		case KindArray:
			if arr, ok := s.([]interface{}); ok {
				for _, ele := range arr {
					f.Children.sanitise(ele, filter)
				}
			}
		case KindMap:
			if obj, ok := s.(map[string]interface{}); ok {
				for _, v := range obj {
					f.Children.sanitise(v, filter)
				}
			}
		default:
			f.Children.sanitise(s, filter)
		}
	}
}

//------------------------------------------------------------------------------

// FieldSpecs is a slice of field specs for a component.
type FieldSpecs []FieldSpec

// Merge with another set of FieldSpecs.
func (f FieldSpecs) Merge(specs FieldSpecs) FieldSpecs {
	return append(f, specs...)
}

// Add more field specs.
func (f FieldSpecs) Add(specs ...FieldSpec) FieldSpecs {
	return append(f, specs...)
}

// FieldFilter defines a filter closure that returns a boolean for a component
// field indicating whether the field should be kept within a generated config.
type FieldFilter func(spec FieldSpec) bool

func (f FieldFilter) shouldDrop(spec FieldSpec) bool {
	if f == nil {
		return false
	}
	return !f(spec)
}

// ShouldDropDeprecated returns a field filter that removes all deprecated
// fields when the boolean argument is true.
func ShouldDropDeprecated(b bool) FieldFilter {
	if !b {
		return nil
	}
	return func(spec FieldSpec) bool {
		return !spec.IsDeprecated
	}
}

func (f FieldSpecs) sanitise(s interface{}, filter FieldFilter) {
	m, ok := s.(map[string]interface{})
	if !ok {
		return
	}
	for _, spec := range f {
		if filter.shouldDrop(spec) {
			delete(m, spec.Name)
			continue
		}
		v := m[spec.Name]
		if _, omit := spec.shouldOmit(v, m); omit {
			delete(m, spec.Name)
		} else {
			spec.sanitise(v, filter)
		}
	}
}

//------------------------------------------------------------------------------

// LintContext is provided to linting functions, and provides context about the
// wider configuration.
type LintContext struct {
	// A map of label names to the line they were defined at.
	Labels map[string]int

	// DocsProvider provides documentation for component implementations.
	DocsProvider Provider
}

// NewLintContext creates a new linting context.
func NewLintContext() LintContext {
	return LintContext{
		Labels:       map[string]int{},
		DocsProvider: globalProvider,
	}
}

// LintFunc is a common linting function for field values.
type LintFunc func(ctx LintContext, line, col int, value interface{}) []Lint

// LintLevel describes the severity level of a linting error.
type LintLevel int

// Lint levels
const (
	LintError   LintLevel = iota
	LintWarning LintLevel = iota
)

// Lint describes a single linting issue found with a Benthos config.
type Lint struct {
	Line   int
	Column int // Optional, omitted from lint report unless >= 1
	Level  LintLevel
	What   string
}

// NewLintError returns an error lint.
func NewLintError(line int, msg string) Lint {
	return Lint{Line: line, Level: LintError, What: msg}
}

// NewLintWarning returns a warning lint.
func NewLintWarning(line int, msg string) Lint {
	return Lint{Line: line, Level: LintWarning, What: msg}
}

//------------------------------------------------------------------------------

func getDefault(pathName string, field FieldSpec) (interface{}, error) {
	if field.Default != nil {
		// TODO: Should be deep copy here?
		return *field.Default, nil
	} else if field.Kind == KindArray {
		return []interface{}{}, nil
	} else if field.Kind == Kind2DArray {
		return []interface{}{}, nil
	} else if field.Kind == KindMap {
		return map[string]interface{}{}, nil
	} else if len(field.Children) > 0 {
		m := map[string]interface{}{}
		for _, v := range field.Children {
			var err error
			if m[v.Name], err = getDefault(pathName+"."+v.Name, v); err != nil {
				return nil, err
			}
		}
		return m, nil
	}
	return nil, fmt.Errorf("field '%v' is required and was not present in the config", pathName)
}
