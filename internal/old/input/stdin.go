package input

import (
	"context"
	"errors"
	"io"
	"os"
	"time"

	"github.com/benthosdev/benthos/v4/internal/codec"
	"github.com/benthosdev/benthos/v4/internal/component"
	"github.com/benthosdev/benthos/v4/internal/component/input"
	"github.com/benthosdev/benthos/v4/internal/component/metrics"
	"github.com/benthosdev/benthos/v4/internal/docs"
	"github.com/benthosdev/benthos/v4/internal/interop"
	"github.com/benthosdev/benthos/v4/internal/log"
	"github.com/benthosdev/benthos/v4/internal/message"
	"github.com/benthosdev/benthos/v4/internal/old/input/reader"
)

//------------------------------------------------------------------------------

func init() {
	Constructors[TypeSTDIN] = TypeSpec{
		constructor: fromSimpleConstructor(NewSTDIN),
		Summary: `
Consumes data piped to stdin as line delimited messages.`,
		Description: `
If the multipart option is set to true then lines are interpretted as message
parts, and an empty line indicates the end of the message.

If the delimiter field is left empty then line feed (\n) is used.`,
		Config: docs.FieldComponent().WithChildren(
			codec.ReaderDocs.AtVersion("3.42.0"),
			docs.FieldInt("max_buffer", "The maximum message buffer size. Must exceed the largest message to be consumed.").Advanced(),
		),
		Categories: []string{
			"Local",
		},
	}
}

//------------------------------------------------------------------------------

// STDINConfig contains config fields for the STDIN input type.
type STDINConfig struct {
	Codec     string `json:"codec" yaml:"codec"`
	MaxBuffer int    `json:"max_buffer" yaml:"max_buffer"`
}

// NewSTDINConfig creates a STDINConfig populated with default values.
func NewSTDINConfig() STDINConfig {
	return STDINConfig{
		Codec:     "lines",
		MaxBuffer: 1000000,
	}
}

//------------------------------------------------------------------------------

// NewSTDIN creates a new STDIN input type.
func NewSTDIN(conf Config, mgr interop.Manager, log log.Modular, stats metrics.Type) (input.Streamed, error) {
	rdr, err := newStdinConsumer(conf.STDIN)
	if err != nil {
		return nil, err
	}
	return NewAsyncReader(
		TypeSTDIN, true,
		reader.NewAsyncCutOff(reader.NewAsyncPreserver(rdr)),
		log, stats,
	)
}

//------------------------------------------------------------------------------

type stdinConsumer struct {
	scanner codec.Reader
}

func newStdinConsumer(conf STDINConfig) (*stdinConsumer, error) {
	codecConf := codec.NewReaderConfig()
	codecConf.MaxScanTokenSize = conf.MaxBuffer
	ctor, err := codec.GetReader(conf.Codec, codecConf)
	if err != nil {
		return nil, err
	}

	scanner, err := ctor("", os.Stdin, func(_ context.Context, err error) error {
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &stdinConsumer{scanner}, nil
}

// ConnectWithContext attempts to establish a connection to the target S3 bucket
// and any relevant queues used to traverse the objects (SQS, etc).
func (s *stdinConsumer) ConnectWithContext(ctx context.Context) error {
	return nil
}

// ReadWithContext attempts to read a new message from the target S3 bucket.
func (s *stdinConsumer) ReadWithContext(ctx context.Context) (*message.Batch, reader.AsyncAckFn, error) {
	parts, codecAckFn, err := s.scanner.Next(ctx)
	if err != nil {
		if errors.Is(err, context.Canceled) ||
			errors.Is(err, context.DeadlineExceeded) {
			err = component.ErrTimeout
		}
		if err != component.ErrTimeout {
			s.scanner.Close(ctx)
		}
		if errors.Is(err, io.EOF) {
			return nil, nil, component.ErrTypeClosed
		}
		return nil, nil, err
	}
	_ = codecAckFn(ctx, nil)

	msg := message.QuickBatch(nil)
	msg.Append(parts...)

	if msg.Len() == 0 {
		return nil, nil, component.ErrTimeout
	}

	return msg, func(rctx context.Context, res error) error {
		return nil
	}, nil
}

// CloseAsync begins cleaning up resources used by this reader asynchronously.
func (s *stdinConsumer) CloseAsync() {
	go func() {
		if s.scanner != nil {
			s.scanner.Close(context.Background())
		}
	}()
}

// WaitForClose will block until either the reader is closed or a specified
// timeout occurs.
func (s *stdinConsumer) WaitForClose(time.Duration) error {
	return nil
}
