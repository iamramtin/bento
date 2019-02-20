// Copyright (c) 2014 Ashley Jeffs
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package output

import (
	"bytes"
	"fmt"
	"io"
	"sync/atomic"
	"time"

	"github.com/Jeffail/benthos/lib/log"
	"github.com/Jeffail/benthos/lib/message"
	"github.com/Jeffail/benthos/lib/message/tracing"
	"github.com/Jeffail/benthos/lib/metrics"
	"github.com/Jeffail/benthos/lib/response"
	"github.com/Jeffail/benthos/lib/types"
)

//------------------------------------------------------------------------------

// LineWriter is an output type that writes messages to an io.WriterCloser type
// as lines.
type LineWriter struct {
	running int32

	typeStr string
	log     log.Modular
	stats   metrics.Type

	customDelim []byte

	transactions <-chan types.Transaction

	handle      io.WriteCloser
	closeOnExit bool

	closeChan  chan struct{}
	closedChan chan struct{}
}

// NewLineWriter creates a new LineWriter output type.
func NewLineWriter(
	handle io.WriteCloser,
	closeOnExit bool,
	customDelimiter []byte,
	typeStr string,
	log log.Modular,
	stats metrics.Type,
) (Type, error) {
	return &LineWriter{
		running:     1,
		typeStr:     typeStr,
		log:         log,
		stats:       stats,
		customDelim: customDelimiter,
		handle:      handle,
		closeOnExit: closeOnExit,
		closeChan:   make(chan struct{}),
		closedChan:  make(chan struct{}),
	}, nil
}

//------------------------------------------------------------------------------

// loop is an internal loop that brokers incoming messages to output pipe.
func (w *LineWriter) loop() {
	// Metrics paths
	var (
		mRunning      = w.stats.GetGauge("running")
		mCount        = w.stats.GetCounter("count")
		mPartsCount   = w.stats.GetCounter("parts.count")
		mSuccess      = w.stats.GetCounter("send.success")
		mPartsSuccess = w.stats.GetCounter("parts.send.success")
		mSent         = w.stats.GetCounter("batch.sent")
		mPartsSent    = w.stats.GetCounter("sent")
		mError        = w.stats.GetCounter("error")
	)

	defer func() {
		if w.closeOnExit {
			w.handle.Close()
		}
		mRunning.Decr(1)

		close(w.closedChan)
	}()
	mRunning.Incr(1)

	delim := []byte("\n")
	if len(w.customDelim) > 0 {
		delim = w.customDelim
	}

	for atomic.LoadInt32(&w.running) == 1 {
		var ts types.Transaction
		var open bool
		select {
		case ts, open = <-w.transactions:
			if !open {
				return
			}
			mCount.Incr(1)
			mPartsCount.Incr(int64(ts.Payload.Len()))
		case <-w.closeChan:
			return
		}

		spans := tracing.CreateChildSpans("output_"+w.typeStr, ts.Payload)

		var err error
		if ts.Payload.Len() == 1 {
			_, err = fmt.Fprintf(w.handle, "%s%s", ts.Payload.Get(0).Get(), delim)
		} else {
			_, err = fmt.Fprintf(w.handle, "%s%s%s", bytes.Join(message.GetAllBytes(ts.Payload), delim), delim, delim)
		}
		if err != nil {
			mError.Incr(1)
		} else {
			mSuccess.Incr(1)
			mPartsSuccess.Incr(int64(ts.Payload.Len()))
			mSent.Incr(1)
			mPartsSent.Incr(int64(ts.Payload.Len()))
		}

		for _, s := range spans {
			s.Finish()
		}

		select {
		case ts.ResponseChan <- response.NewError(err):
		case <-w.closeChan:
			return
		}
	}
}

// Connected returns a boolean indicating whether this output is currently
// connected to its target.
func (w *LineWriter) Connected() bool {
	return true
}

// Consume assigns a messages channel for the output to read.
func (w *LineWriter) Consume(ts <-chan types.Transaction) error {
	if w.transactions != nil {
		return types.ErrAlreadyStarted
	}
	w.transactions = ts
	go w.loop()
	return nil
}

// CloseAsync shuts down the File output and stops processing messages.
func (w *LineWriter) CloseAsync() {
	if atomic.CompareAndSwapInt32(&w.running, 1, 0) {
		close(w.closeChan)
	}
}

// WaitForClose blocks until the File output has closed down.
func (w *LineWriter) WaitForClose(timeout time.Duration) error {
	select {
	case <-w.closedChan:
	case <-time.After(timeout):
		return types.ErrTimeout
	}
	return nil
}

//------------------------------------------------------------------------------
