package output

import (
	"github.com/benthosdev/benthos/v4/internal/component/metrics"
	"github.com/benthosdev/benthos/v4/internal/component/output"
	"github.com/benthosdev/benthos/v4/internal/docs"
	mqttconf "github.com/benthosdev/benthos/v4/internal/impl/mqtt/shared"
	"github.com/benthosdev/benthos/v4/internal/interop"
	"github.com/benthosdev/benthos/v4/internal/log"
	"github.com/benthosdev/benthos/v4/internal/old/output/writer"
	"github.com/benthosdev/benthos/v4/internal/tls"
)

//------------------------------------------------------------------------------

func init() {
	Constructors[TypeMQTT] = TypeSpec{
		constructor: fromSimpleConstructor(NewMQTT),
		Summary: `
Pushes messages to an MQTT broker.`,
		Description: `
The ` + "`topic`" + ` field can be dynamically set using function interpolations
described [here](/docs/configuration/interpolation#bloblang-queries). When sending batched
messages these interpolations are performed per message part.`,
		Async: true,
		Config: docs.FieldComponent().WithChildren(
			docs.FieldString("urls", "A list of URLs to connect to. If an item of the list contains commas it will be expanded into multiple URLs.", []string{"tcp://localhost:1883"}).Array(),
			docs.FieldString("topic", "The topic to publish messages to."),
			docs.FieldString("client_id", "An identifier for the client connection."),
			docs.FieldString("dynamic_client_id_suffix", "Append a dynamically generated suffix to the specified `client_id` on each run of the pipeline. This can be useful when clustering Benthos producers.").Optional().Advanced().HasAnnotatedOptions(
				"nanoid", "append a nanoid of length 21 characters",
			),
			docs.FieldInt("qos", "The QoS value to set for each message.").HasOptions("0", "1", "2"),
			docs.FieldString("connect_timeout", "The maximum amount of time to wait in order to establish a connection before the attempt is abandoned.", "1s", "500ms").HasDefault("30s").AtVersion("3.58.0"),
			docs.FieldString("write_timeout", "The maximum amount of time to wait to write data before the attempt is abandoned.", "1s", "500ms").HasDefault("3s").AtVersion("3.58.0"),
			docs.FieldBool("retained", "Set message as retained on the topic."),
			docs.FieldString("retained_interpolated", "Override the value of `retained` with an interpolable value, this allows it to be dynamically set based on message contents. The value must resolve to either `true` or `false`.").IsInterpolated().Advanced().AtVersion("3.59.0"),
			mqttconf.WillFieldSpec(),
			docs.FieldString("user", "A username to connect with.").Advanced(),
			docs.FieldString("password", "A password to connect with.").Advanced(),
			docs.FieldInt("keepalive", "Max seconds of inactivity before a keepalive message is sent.").Advanced(),
			tls.FieldSpec().AtVersion("3.45.0"),
			docs.FieldInt("max_in_flight", "The maximum number of messages to have in flight at a given time. Increase this to improve throughput."),
		),
		Categories: []string{
			"Services",
		},
	}
}

//------------------------------------------------------------------------------

// NewMQTT creates a new MQTT output type.
func NewMQTT(conf Config, mgr interop.Manager, log log.Modular, stats metrics.Type) (output.Streamed, error) {
	w, err := writer.NewMQTTV2(conf.MQTT, mgr, log, stats)
	if err != nil {
		return nil, err
	}
	a, err := NewAsyncWriter(TypeMQTT, conf.MQTT.MaxInFlight, w, log, stats)
	if err != nil {
		return nil, err
	}
	return OnlySinglePayloads(a), nil
}

//------------------------------------------------------------------------------
