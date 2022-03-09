package metrics_test

import (
	"testing"

	yaml "gopkg.in/yaml.v3"

	"github.com/benthosdev/benthos/v4/internal/component/metrics"

	_ "github.com/benthosdev/benthos/v4/public/components/all"
)

func TestConstructorConfigYAMLInference(t *testing.T) {
	conf := []metrics.Config{}

	if err := yaml.Unmarshal([]byte(`[
		{
			"json_api": {
				"value": "foo"
			},
			"prometheus": {
				"query": "foo"
			}
		}
	]`), &conf); err == nil {
		t.Error("Expected error from multi candidates")
	}

	if err := yaml.Unmarshal([]byte(`[
		{
			"prometheus": {
				"push_interval": "foo"
			}
		}
	]`), &conf); err != nil {
		t.Error(err)
	}

	if exp, act := 1, len(conf); exp != act {
		t.Errorf("Wrong number of config parts: %v != %v", act, exp)
		return
	}
	if exp, act := "prometheus", conf[0].Type; exp != act {
		t.Errorf("Wrong inferred type: %v != %v", act, exp)
	}
	if exp, act := "benthos_push", conf[0].Prometheus.PushJobName; exp != act {
		t.Errorf("Wrong default operator: %v != %v", act, exp)
	}
	if exp, act := "foo", conf[0].Prometheus.PushInterval; exp != act {
		t.Errorf("Wrong value: %v != %v", act, exp)
	}
}
