package input

import (
	"github.com/benthosdev/benthos/v4/internal/component/input"
	"github.com/benthosdev/benthos/v4/internal/component/metrics"
	"github.com/benthosdev/benthos/v4/internal/docs"
	"github.com/benthosdev/benthos/v4/internal/impl/redis/old"
	"github.com/benthosdev/benthos/v4/internal/interop"
	"github.com/benthosdev/benthos/v4/internal/log"
	"github.com/benthosdev/benthos/v4/internal/old/input/reader"
)

//------------------------------------------------------------------------------

func init() {
	Constructors[TypeRedisList] = TypeSpec{
		constructor: fromSimpleConstructor(NewRedisList),
		Summary: `
Pops messages from the beginning of a Redis list using the BLPop command.`,
		Config: docs.FieldComponent().WithChildren(old.ConfigDocs()...).WithChildren(
			docs.FieldString("key", "The key of a list to read from."),
			docs.FieldString("timeout", "The length of time to poll for new messages before reattempting.").Advanced(),
		),
		Categories: []string{
			"Services",
		},
	}
}

//------------------------------------------------------------------------------

// NewRedisList creates a new Redis List input type.
func NewRedisList(conf Config, mgr interop.Manager, log log.Modular, stats metrics.Type) (input.Streamed, error) {
	r, err := reader.NewRedisList(conf.RedisList, log, stats)
	if err != nil {
		return nil, err
	}
	return NewAsyncReader(TypeRedisList, true, reader.NewAsyncPreserver(r), log, stats)
}

//------------------------------------------------------------------------------
