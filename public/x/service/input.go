package service

import (
	"context"
	"errors"
	"time"

	"github.com/Jeffail/benthos/v3/internal/shutdown"
	"github.com/Jeffail/benthos/v3/lib/input/reader"
	"github.com/Jeffail/benthos/v3/lib/message"
	"github.com/Jeffail/benthos/v3/lib/types"
)

// AckFunc is a common function returned by inputs that must be called once for
// each message consumed. This function ensures that the source of the message
// receives either an acknowledgement (err is nil) or an error that can either
// be propagated upstream as a nack, or trigger a reattempt at delivering the
// same message.
type AckFunc func(ctx context.Context, err error) error

// NoopAckFunc is a function with an AckFunc signature, but does nothing and
// always returns nil. It is convenient to return this from an input Read method
// when there is no formal mechanism (or desire) to acknowledge messages that
// are sent.
func NoopAckFunc(context.Context, error) error {
	return nil
}

// Input is an interface implemented by Benthos inputs. Calls to Read should
// block until either a message has been received, the connect is lose, or the
// provided context is cancelled.
type Input interface {
	// Establish a connection to the upstream service. Connect will always be
	// called first when a reader is instantiated, and will be continuously
	// called with back off until a nil error is returned.
	//
	// Once Connect returns a nil error the Read method will be called until
	// either ErrNotConnected is returned, or the reader is closed.
	Connect(context.Context) error

	// Read a single message from a source, along with a function to be called
	// once the message can be either acked (successfully sent or intentionally
	// filtered) or nacked (failed to be processed or dispatched to the output).
	//
	// The AckFunc will be called for every message at least once, but there are
	// no guarantees as to when this will occur.
	//
	// If this method returns ErrNotConnected then Read will not be called again
	// until Connect has returned a nil error. If ErrEndOfInput is returned then
	// Read will no longer be called and the pipeline will gracefully terminate.
	Read(context.Context) (*Message, AckFunc, error)

	Closer
}

//------------------------------------------------------------------------------

// Implements input.AsyncReader
type airGapReader struct {
	r Input

	sig *shutdown.Signaller
}

func newAirGapReader(r Input) reader.Async {
	return &airGapReader{r, shutdown.NewSignaller()}
}

func (a *airGapReader) ConnectWithContext(ctx context.Context) error {
	err := a.r.Connect(ctx)
	if err != nil && errors.Is(err, ErrEndOfInput) {
		err = types.ErrTypeClosed
	}
	return err
}

func (a *airGapReader) ReadWithContext(ctx context.Context) (types.Message, reader.AsyncAckFn, error) {
	msg, ackFn, err := a.r.Read(ctx)
	if err != nil {
		if errors.Is(err, ErrNotConnected) {
			err = types.ErrNotConnected
		} else if errors.Is(err, ErrEndOfInput) {
			err = types.ErrTypeClosed
		}
		return nil, nil, err
	}
	tMsg := message.New(nil)
	tMsg.Append(msg.part)
	return tMsg, func(c context.Context, r types.Response) error {
		return ackFn(c, r.Error())
	}, nil
}

func (a *airGapReader) CloseAsync() {
	go func() {
		if err := a.r.Close(context.Background()); err == nil {
			a.sig.ShutdownComplete()
		}
	}()
}

func (a *airGapReader) WaitForClose(tout time.Duration) error {
	select {
	case <-a.sig.HasClosedChan():
	case <-time.After(tout):
		return types.ErrTimeout
	}
	return nil
}
