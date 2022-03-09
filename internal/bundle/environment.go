package bundle

import (
	"github.com/benthosdev/benthos/v4/internal/docs"
)

// Environment is a collection of Benthos component plugins that can be used in
// order to build and run streaming pipelines with access to different sets of
// plugins. This is useful for sandboxing, testing, etc.
type Environment struct {
	buffers    *BufferSet
	caches     *CacheSet
	inputs     *InputSet
	outputs    *OutputSet
	processors *ProcessorSet
	rateLimits *RateLimitSet
}

// NewEnvironment creates an empty environment.
func NewEnvironment() *Environment {
	return &Environment{
		buffers:    &BufferSet{},
		caches:     &CacheSet{},
		inputs:     &InputSet{},
		outputs:    &OutputSet{},
		processors: &ProcessorSet{},
		rateLimits: &RateLimitSet{},
	}
}

// Clone an existing environment to a new one that can be modified
// independently.
func (e *Environment) Clone() *Environment {
	newEnv := NewEnvironment()
	for _, v := range e.buffers.specs {
		_ = newEnv.buffers.Add(v.constructor, v.spec)
	}
	for _, v := range e.caches.specs {
		_ = newEnv.caches.Add(v.constructor, v.spec)
	}
	for _, v := range e.inputs.specs {
		_ = newEnv.inputs.Add(v.constructor, v.spec)
	}
	for _, v := range e.outputs.specs {
		_ = newEnv.outputs.Add(v.constructor, v.spec)
	}
	for _, v := range e.processors.specs {
		_ = newEnv.processors.Add(v.constructor, v.spec)
	}
	for _, v := range e.rateLimits.specs {
		_ = newEnv.rateLimits.Add(v.constructor, v.spec)
	}
	return newEnv
}

// GetDocs returns a documentation spec for an implementation of a component.
func (e *Environment) GetDocs(name string, ctype docs.Type) (docs.ComponentSpec, bool) {
	var spec docs.ComponentSpec
	var ok bool

	switch ctype {
	case docs.TypeBuffer:
		spec, ok = e.buffers.DocsFor(name)
	case docs.TypeCache:
		spec, ok = e.caches.DocsFor(name)
	case docs.TypeInput:
		spec, ok = e.inputs.DocsFor(name)
	case docs.TypeOutput:
		spec, ok = e.outputs.DocsFor(name)
	case docs.TypeProcessor:
		spec, ok = e.processors.DocsFor(name)
	case docs.TypeRateLimit:
		spec, ok = e.rateLimits.DocsFor(name)
	default:
		return docs.GetDocs(nil, name, ctype)
	}

	return spec, ok
}

// GlobalEnvironment contains service-wide singleton bundles.
var GlobalEnvironment = &Environment{
	buffers:    AllBuffers,
	caches:     AllCaches,
	inputs:     AllInputs,
	outputs:    AllOutputs,
	processors: AllProcessors,
	rateLimits: AllRateLimits,
}
