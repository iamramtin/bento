package mqttconf

import (
	"errors"

	"github.com/Jeffail/benthos/v3/internal/docs"
)

// Will holds configuration for the last will message that the broker emits,
// should benthos exit abnormally.
type Will struct {
	Enabled  bool   `json:"enabled" yaml:"enabled"`
	QoS      uint8  `json:"qos" yaml:"qos"`
	Retained bool   `json:"retained" yaml:"retained"`
	Topic    string `json:"topic" yaml:"topic"`
	Payload  string `json:"payload" yaml:"payload"`
}

// EmptyWill returns an empty will, meaning last will message should not be registered.
func EmptyWill() Will {
	return Will{}
}

// Validate the Will configuration and return nil or error accordingly.
func (w *Will) Validate() error {
	if !w.Enabled {
		return nil
	}
	if w.Topic == "" {
		return errors.New("include topic to register a last will")
	}
	return nil
}

// WillFieldSpec defines a last will message registration.
func WillFieldSpec() docs.FieldSpec {
	return docs.FieldAdvanced(
		"will", "Set last will message in case of Benthos failure",
	).WithChildren(
		docs.FieldCommon("enabled", "Whether to enable last will messages."),
		docs.FieldCommon("qos", "Set QoS for last will message.").HasOptions("0", "1", "2"),
		docs.FieldCommon("retained", "Set retained for last will message."),
		docs.FieldCommon("topic", "Set topic for last will message."),
		docs.FieldCommon("payload", "Set payload for last will message."),
	)
}
