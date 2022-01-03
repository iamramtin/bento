package service

import (
	"fmt"
	"strings"

	"github.com/Jeffail/benthos/v3/internal/docs"
	"github.com/Jeffail/benthos/v3/lib/processor"
	"gopkg.in/yaml.v3"
)

// NewProcessorField defines a new processor field, it is then possible to
// extract an OwnedProcessor from the resulting parsed config with the method
// FieldProcessor.
func NewProcessorField(name string) *ConfigField {
	return &ConfigField{
		field: docs.FieldCommon(name, "").HasType(docs.FieldTypeProcessor),
	}
}

// FieldProcessor accesses a field from a parsed config that was defined with
// NewProcessorField and returns an OwnedProcessor, or an error if the
// configuration was invalid.
func (p *ParsedConfig) FieldProcessor(path ...string) (*OwnedProcessor, error) {
	proc, exists := p.field(path...)
	if !exists {
		return nil, fmt.Errorf("field '%v' was not found in the config", strings.Join(path, "."))
	}

	pNode, ok := proc.(*yaml.Node)
	if !ok {
		return nil, fmt.Errorf("unexpected value, expected object, got %T", proc)
	}

	var procConf processor.Config
	if err := pNode.Decode(&procConf); err != nil {
		return nil, err
	}

	iproc, err := p.mgr.NewProcessor(procConf)
	if err != nil {
		return nil, err
	}
	return &OwnedProcessor{iproc}, nil
}

// NewProcessorListField defines a new processor list field, it is then possible
// to extract a list of OwnedProcessor from the resulting parsed config with the
// method FieldProcessorList.
func NewProcessorListField(name string) *ConfigField {
	return &ConfigField{
		field: docs.FieldCommon(name, "").Array().HasType(docs.FieldTypeProcessor),
	}
}

// FieldProcessorList accesses a field from a parsed config that was defined
// with NewProcessorListField and returns a slice of OwnedProcessor, or an error
// if the configuration was invalid.
func (p *ParsedConfig) FieldProcessorList(path ...string) ([]*OwnedProcessor, error) {
	proc, exists := p.field(path...)
	if !exists {
		return nil, fmt.Errorf("field '%v' was not found in the config", strings.Join(path, "."))
	}

	procsArray, ok := proc.([]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected value, expected array, got %T", proc)
	}

	var procConfigs []processor.Config
	for i, iConf := range procsArray {
		node, ok := iConf.(*yaml.Node)
		if !ok {
			return nil, fmt.Errorf("value %v returned unexpected value, expected object, got %T", i, iConf)
		}

		var pconf processor.Config
		if err := node.Decode(&pconf); err != nil {
			return nil, fmt.Errorf("value %v: %w", i, err)
		}
		procConfigs = append(procConfigs, pconf)
	}

	procs := make([]*OwnedProcessor, len(procConfigs))
	for i, c := range procConfigs {
		iproc, err := p.mgr.NewProcessor(c)
		if err != nil {
			return nil, fmt.Errorf("processor %v: %w", i, err)
		}
		procs[i] = &OwnedProcessor{iproc}
	}

	return procs, nil
}
