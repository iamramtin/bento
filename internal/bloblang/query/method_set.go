package query

import (
	"fmt"
	"sort"
)

// MethodSet contains an explicit set of methods to be available in a Bloblang
// query.
type MethodSet struct {
	constructors map[string]MethodCtor
	specs        map[string]MethodSpec
}

// NewMethodSet creates a method set without any methods in it.
func NewMethodSet() *MethodSet {
	return &MethodSet{
		constructors: map[string]MethodCtor{},
		specs:        map[string]MethodSpec{},
	}
}

// Add a new method to this set by providing a spec (name and documentation),
// a constructor to be called for each instantiation of the method, and
// information regarding the arguments of the method.
func (m *MethodSet) Add(spec MethodSpec, ctor MethodCtor) error {
	if !nameRegexp.MatchString(spec.Name) {
		return fmt.Errorf("method name '%v' does not match the required regular expression /%v/", spec.Name, nameRegexpRaw)
	}
	if _, exists := m.constructors[spec.Name]; exists {
		return fmt.Errorf("conflicting method name: %v", spec.Name)
	}
	if err := spec.Params.validate(); err != nil {
		return err
	}
	m.constructors[spec.Name] = ctor
	m.specs[spec.Name] = spec
	return nil
}

// Docs returns a slice of method specs, which document each method.
func (m *MethodSet) Docs() []MethodSpec {
	specSlice := make([]MethodSpec, 0, len(m.specs))
	for _, v := range m.specs {
		specSlice = append(specSlice, v)
	}
	sort.Slice(specSlice, func(i, j int) bool {
		return specSlice[i].Name < specSlice[j].Name
	})
	return specSlice
}

// List returns a slice of method names in alphabetical order.
func (m *MethodSet) List() []string {
	methodNames := make([]string, 0, len(m.constructors))
	for k := range m.constructors {
		methodNames = append(methodNames, k)
	}
	sort.Strings(methodNames)
	return methodNames
}

// Params attempts to obtain an argument specification for a given method type.
func (m *MethodSet) Params(name string) (Params, error) {
	spec, exists := m.specs[name]
	if !exists {
		return OldStyleParams(), badMethodErr(name)
	}
	return spec.Params, nil
}

// Init attempts to initialize a method of the set by name from a target
// function and zero or more arguments.
func (m *MethodSet) Init(name string, target Function, args *ParsedParams) (Function, error) {
	ctor, exists := m.constructors[name]
	if !exists {
		return nil, badMethodErr(name)
	}
	return wrapMethodCtorWithDynamicArgs(name, target, args, ctor)
}

// Without creates a clone of the method set that can be mutated in isolation,
// where a variadic list of methods will be excluded from the set.
func (m *MethodSet) Without(methods ...string) *MethodSet {
	excludeMap := make(map[string]struct{}, len(methods))
	for _, k := range methods {
		excludeMap[k] = struct{}{}
	}

	constructors := make(map[string]MethodCtor, len(m.constructors))
	for k, v := range m.constructors {
		if _, exists := excludeMap[k]; !exists {
			constructors[k] = v
		}
	}

	specs := map[string]MethodSpec{}
	for _, v := range m.specs {
		if _, exists := excludeMap[v.Name]; !exists {
			specs[v.Name] = v
		}
	}
	return &MethodSet{constructors, specs}
}

//------------------------------------------------------------------------------

// AllMethods is a set containing every single method declared by this package,
// and any globally declared plugin methods.
var AllMethods = NewMethodSet()

func registerMethod(spec MethodSpec, ctor MethodCtor) struct{} {
	if err := AllMethods.Add(spec, func(target Function, args *ParsedParams) (Function, error) {
		return ctor(target, args)
	}); err != nil {
		panic(err)
	}
	return struct{}{}
}

// InitMethodHelper attempts to initialise a method by its name, target function
// and arguments, this is convenient for writing tests.
func InitMethodHelper(name string, target Function, args ...interface{}) (Function, error) {
	spec, ok := AllMethods.specs[name]
	if !ok {
		return nil, badMethodErr(name)
	}
	parsedArgs, err := spec.Params.PopulateNameless(args...)
	if err != nil {
		return nil, err
	}
	return AllMethods.Init(name, target, parsedArgs)
}

// ListMethods returns a slice of method names, sorted alphabetically.
func ListMethods() []string {
	return AllMethods.List()
}

// MethodDocs returns a slice of specs, one for each method.
func MethodDocs() []MethodSpec {
	return AllMethods.Docs()
}

//------------------------------------------------------------------------------

func wrapMethodCtorWithDynamicArgs(name string, target Function, args *ParsedParams, fn MethodCtor) (Function, error) {
	fns := args.dynamic()
	if len(fns) == 0 {
		return fn(target, args)
	}
	return ClosureFunction("method "+name, func(ctx FunctionContext) (interface{}, error) {
		newArgs, err := args.ResolveDynamic(ctx)
		if err != nil {
			return nil, err
		}
		dynFunc, err := fn(target, newArgs)
		if err != nil {
			return nil, err
		}
		return dynFunc.Exec(ctx)
	}, aggregateTargetPaths(fns...)), nil
}
