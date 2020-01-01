package output

import (
	"github.com/Jeffail/benthos/v3/lib/log"
	"github.com/Jeffail/benthos/v3/lib/metrics"
	"github.com/Jeffail/benthos/v3/lib/output/writer"
	"github.com/Jeffail/benthos/v3/lib/types"
)

//------------------------------------------------------------------------------

func init() {
	Constructors[TypeRedisHash] = TypeSpec{
		constructor: NewRedisHash,
		Description: `
Sets Redis hash objects using the HMSET command.

The field ` + "`key`" + ` supports
[interpolation functions](../config_interpolation.md#functions) evaluated per
message of a batch, allowing you to create a unique key for each message.

The field ` + "`fields`" + ` allows you to specify an explicit map of field
names to interpolated values, also evaluated per message of a batch:

` + "```yaml" + `
redis_hash:
  url: tcp://localhost:6379
  key: ${!json_field:id}
  fields:
    topic: ${!metadata:kafka_topic}
    partition: ${!metadata:kafka_partition}
    content: ${!json_field:document.text}
` + "```" + `

If the field ` + "`walk_metadata`" + ` is set to ` + "`true`" + ` then Benthos
will walk all metadata fields of messages and add them to the list of hash
fields to set.

If the field ` + "`walk_json_object`" + ` is set to ` + "`true`" + ` then
Benthos will walk each message as a JSON object, extracting keys and the string
representation of their value and adds them to the list of hash fields to set.

The order of hash field extraction is as follows:

1. Metadata (if enabled)
2. JSON object (if enabled)
3. Explicit fields

Where latter stages will overwrite matching field names of a former stage.`,
		Async: true,
	}
}

//------------------------------------------------------------------------------

// NewRedisHash creates a new RedisHash output type.
func NewRedisHash(conf Config, mgr types.Manager, log log.Modular, stats metrics.Type) (Type, error) {
	rhash, err := writer.NewRedisHash(conf.RedisHash, log, stats)
	if err != nil {
		return nil, err
	}
	if conf.RedisHash.MaxInFlight == 1 {
		return NewWriter(
			TypeRedisHash, rhash, log, stats,
		)
	}
	return NewAsyncWriter(
		TypeRedisHash, conf.RedisHash.MaxInFlight, rhash, log, stats,
	)
}

//------------------------------------------------------------------------------
