package buffer_test

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/Jeffail/benthos/v3/lib/buffer"
	"github.com/Jeffail/benthos/v3/lib/log"
	"github.com/Jeffail/benthos/v3/lib/metrics"
	yaml "gopkg.in/yaml.v3"

	_ "github.com/Jeffail/benthos/v3/public/components/all"
)

func TestConstructorDescription(t *testing.T) {
	if buffer.Descriptions() == "" {
		t.Error("package descriptions were empty")
	}
}

func TestConstructorBadType(t *testing.T) {
	conf := buffer.NewConfig()
	conf.Type = "not_exist"

	if _, err := buffer.New(conf, nil, log.Noop(), metrics.Noop()); err == nil {
		t.Error("Expected error, received nil for invalid type")
	}
}

func TestConstructorConfigYAMLInference(t *testing.T) {
	conf := []buffer.Config{}

	if err := yaml.Unmarshal([]byte(`[
		{
			"memory": {
				"value": "foo"
			},
			"none": {
				"query": "foo"
			}
		}
	]`), &conf); err == nil {
		t.Error("Expected error from multi candidates")
	}

	if err := yaml.Unmarshal([]byte(`[
		{
			"memory": {
				"limit": 10
			}
		}
	]`), &conf); err != nil {
		t.Error(err)
	}

	if exp, act := 1, len(conf); exp != act {
		t.Errorf("Wrong number of config parts: %v != %v", act, exp)
		return
	}
	if exp, act := buffer.TypeMemory, conf[0].Type; exp != act {
		t.Errorf("Wrong inferred type: %v != %v", act, exp)
	}
	if exp, act := 10, conf[0].Memory.Limit; exp != act {
		t.Errorf("Wrong default operator: %v != %v", act, exp)
	}
}

func TestSanitise(t *testing.T) {
	var actObj interface{}
	var act []byte
	var err error

	exp := `{` +
		`"type":"none",` +
		`"none":{}` +
		`}`

	conf := buffer.NewConfig()
	conf.Type = "none"
	conf.Memory.Limit = 10

	if actObj, err = buffer.SanitiseConfig(conf); err != nil {
		t.Fatal(err)
	}
	if act, err = json.Marshal(actObj); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(string(act), exp) {
		t.Errorf("Wrong sanitised output: %s != %v", act, exp)
	}

	exp = `{` +
		`"type":"memory",` +
		`"memory":{` +
		`"batch_policy":{"byte_size":0,"check":"","count":0,"enabled":false,"period":"","processors":[]},` +
		`"limit":20` +
		`}` +
		`}`

	conf = buffer.NewConfig()
	conf.Type = "memory"
	conf.Memory.Limit = 20

	if actObj, err = buffer.SanitiseConfig(conf); err != nil {
		t.Fatal(err)
	}
	if act, err = json.Marshal(actObj); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(string(act), exp) {
		t.Errorf("Wrong sanitised output: %s != %v", act, exp)
	}
}
