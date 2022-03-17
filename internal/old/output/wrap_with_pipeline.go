package output

import (
	"time"

	"github.com/benthosdev/benthos/v4/internal/component/output"
	iprocessor "github.com/benthosdev/benthos/v4/internal/component/processor"
	"github.com/benthosdev/benthos/v4/internal/message"
	"github.com/benthosdev/benthos/v4/internal/shutdown"
)

//------------------------------------------------------------------------------

// WithPipeline is a type that wraps both an output type and a pipeline type
// by routing the pipeline through the output, and implements the output.Type
// interface in order to act like an ordinary output.
type WithPipeline struct {
	out  output.Streamed
	pipe iprocessor.Pipeline
}

// WrapWithPipeline routes a processing pipeline directly into an output and
// returns a type that manages both and acts like an ordinary output.
func WrapWithPipeline(out output.Streamed, pipeConstructor iprocessor.PipelineConstructorFunc) (*WithPipeline, error) {
	pipe, err := pipeConstructor()
	if err != nil {
		return nil, err
	}

	if err := out.Consume(pipe.TransactionChan()); err != nil {
		return nil, err
	}
	return &WithPipeline{
		out:  out,
		pipe: pipe,
	}, nil
}

// WrapWithPipelines wraps an output with a variadic number of pipelines.
func WrapWithPipelines(out output.Streamed, pipeConstructors ...iprocessor.PipelineConstructorFunc) (output.Streamed, error) {
	var err error
	for i := len(pipeConstructors) - 1; i >= 0; i-- {
		if out, err = WrapWithPipeline(out, pipeConstructors[i]); err != nil {
			return nil, err
		}
	}
	return out, nil
}

//------------------------------------------------------------------------------

// Consume starts the type listening to a message channel from a
// producer.
func (i *WithPipeline) Consume(tsChan <-chan message.Transaction) error {
	return i.pipe.Consume(tsChan)
}

// Connected returns a boolean indicating whether this output is currently
// connected to its target.
func (i *WithPipeline) Connected() bool {
	return i.out.Connected()
}

//------------------------------------------------------------------------------

// CloseAsync triggers a closure of this object but does not block.
func (i *WithPipeline) CloseAsync() {
	i.pipe.CloseAsync()
	go func() {
		_ = i.pipe.WaitForClose(shutdown.MaximumShutdownWait())
		i.out.CloseAsync()
	}()
}

// WaitForClose is a blocking call to wait until the object has finished closing
// down and cleaning up resources.
func (i *WithPipeline) WaitForClose(timeout time.Duration) error {
	return i.out.WaitForClose(timeout)
}

//------------------------------------------------------------------------------
