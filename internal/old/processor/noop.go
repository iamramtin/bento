package processor

import (
	"time"

	"github.com/benthosdev/benthos/v4/internal/component/metrics"
	"github.com/benthosdev/benthos/v4/internal/component/processor"
	"github.com/benthosdev/benthos/v4/internal/docs"
	"github.com/benthosdev/benthos/v4/internal/interop"
	"github.com/benthosdev/benthos/v4/internal/log"
	"github.com/benthosdev/benthos/v4/internal/message"
)

//------------------------------------------------------------------------------

func init() {
	Constructors[TypeNoop] = TypeSpec{
		constructor: NewNoop,
		Summary:     "Noop is a processor that does nothing, the message passes through unchanged. Why? Sometimes doing nothing is the braver option.",
		Config:      docs.FieldObject("", ""),
	}
}

//------------------------------------------------------------------------------

// NoopConfig configures the no-op processor.
type NoopConfig struct{}

// NewNoopConfig creates a new default no-op processor config.
func NewNoopConfig() NoopConfig {
	return NoopConfig{}
}

// Noop is a no-op processor that does nothing.
type Noop struct{}

// NewNoop returns a Noop processor.
func NewNoop(
	conf Config, mgr interop.Manager, log log.Modular, stats metrics.Type,
) (processor.V1, error) {
	return &Noop{}, nil
}

//------------------------------------------------------------------------------

// ProcessMessage does nothing and returns the message unchanged.
func (c *Noop) ProcessMessage(msg *message.Batch) ([]*message.Batch, error) {
	msgs := [1]*message.Batch{msg}
	return msgs[:], nil
}

// CloseAsync shuts down the processor and stops processing requests.
func (c *Noop) CloseAsync() {
}

// WaitForClose blocks until the processor has closed down.
func (c *Noop) WaitForClose(timeout time.Duration) error {
	return nil
}

//------------------------------------------------------------------------------
