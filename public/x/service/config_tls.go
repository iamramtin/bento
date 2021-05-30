package service

import (
	"crypto/tls"
	"fmt"
	"strings"

	"github.com/Jeffail/benthos/v3/internal/docs"
	btls "github.com/Jeffail/benthos/v3/lib/util/tls"
	"gopkg.in/yaml.v3"
)

// NewTLSField describes a new object type config field that describes TLS
// settings for networked components. It is then possible to extract a
// *tls.Config from the resulting parsed config with the method FieldTLS.
func NewTLSField(name string) *ConfigField {
	tf := btls.FieldSpec()
	tf.Name = name
	// Remove the explicit "enabled" flag.
	newChildren := make([]docs.FieldSpec, 0, len(tf.Children)-1)
	for _, f := range tf.Children {
		if f.Name != "enabled" {
			newChildren = append(newChildren, f)
		}
	}
	tf.Children = newChildren
	return &ConfigField{field: tf}
}

// FieldTLS accesses an object field that was parsed from a config field created
// using NewTLSField and returns a *tls.Config, or an error if the configuration
// was invalid.
func (p *ParsedConfig) FieldTLS(path ...string) (*tls.Config, error) {
	v, exists := p.field(path...)
	if !exists {
		return nil, fmt.Errorf("field '%v' was not found in the config", strings.Join(path, "."))
	}

	var node yaml.Node
	if err := node.Encode(v); err != nil {
		return nil, err
	}

	conf := btls.NewConfig()
	if err := node.Decode(&conf); err != nil {
		return nil, err
	}

	return conf.Get()
}
