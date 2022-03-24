package output

import (
	"github.com/benthosdev/benthos/v4/internal/batch/policy"
	"github.com/benthosdev/benthos/v4/internal/component/metrics"
	"github.com/benthosdev/benthos/v4/internal/component/output"
	"github.com/benthosdev/benthos/v4/internal/docs"
	"github.com/benthosdev/benthos/v4/internal/impl/aws/session"
	"github.com/benthosdev/benthos/v4/internal/interop"
	"github.com/benthosdev/benthos/v4/internal/log"
	"github.com/benthosdev/benthos/v4/internal/old/output/writer"
	"github.com/benthosdev/benthos/v4/internal/old/util/retries"
)

func init() {
	Constructors[TypeAWSKinesis] = TypeSpec{
		constructor: fromSimpleConstructor(func(conf Config, mgr interop.Manager, log log.Modular, stats metrics.Type) (output.Streamed, error) {
			return newKinesis(TypeAWSKinesis, conf.AWSKinesis, mgr, log, stats)
		}),
		Version: "3.36.0",
		Summary: `
Sends messages to a Kinesis stream.`,
		Description: `
Both the ` + "`partition_key`" + `(required) and ` + "`hash_key`" + ` (optional)
fields can be dynamically set using function interpolations described
[here](/docs/configuration/interpolation#bloblang-queries). When sending batched messages the
interpolations are performed per message part.

### Credentials

By default Benthos will use a shared credentials file when connecting to AWS
services. It's also possible to set them explicitly at the component level,
allowing you to transfer data across accounts. You can find out more
[in this document](/docs/guides/cloud/aws).`,
		Async:   true,
		Batches: true,
		Config: docs.FieldComponent().WithChildren(
			docs.FieldString("stream", "The stream to publish messages to."),
			docs.FieldString("partition_key", "A required key for partitioning messages.").IsInterpolated(),
			docs.FieldString("hash_key", "A optional hash key for partitioning messages.").IsInterpolated().Advanced(),
			docs.FieldInt("max_in_flight", "The maximum number of messages to have in flight at a given time. Increase this to improve throughput."),
			policy.FieldSpec(),
		).WithChildren(session.FieldSpecs()...).WithChildren(retries.FieldSpecs()...),
		Categories: []string{
			"Services",
			"AWS",
		},
	}
}

//------------------------------------------------------------------------------

func newKinesis(name string, conf writer.KinesisConfig, mgr interop.Manager, log log.Modular, stats metrics.Type) (output.Streamed, error) {
	kin, err := writer.NewKinesisV2(conf, mgr, log)
	if err != nil {
		return nil, err
	}
	w, err := NewAsyncWriter(name, conf.MaxInFlight, kin, log, stats)
	if err != nil {
		return w, err
	}
	return NewBatcherFromConfig(conf.Batching, w, mgr, log, stats)
}
