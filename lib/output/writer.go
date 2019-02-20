// Copyright (c) 2018 Ashley Jeffs
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
	"sync/atomic"
	"time"

	"github.com/Jeffail/benthos/lib/log"
	"github.com/Jeffail/benthos/lib/message/tracing"
	"github.com/Jeffail/benthos/lib/metrics"
	"github.com/Jeffail/benthos/lib/output/writer"
	"github.com/Jeffail/benthos/lib/response"
	"github.com/Jeffail/benthos/lib/types"
	"github.com/Jeffail/benthos/lib/util/throttle"
)

//------------------------------------------------------------------------------

// Writer is an output type that writes messages to a writer.Type.
type Writer struct {
	running     int32
	isConnected int32

	typeStr string
	writer  writer.Type

	log   log.Modular
	stats metrics.Type

	transactions <-chan types.Transaction

	closeChan  chan struct{}
	closedChan chan struct{}
}

// NewWriter creates a new Writer output type.
func NewWriter(
	typeStr string,
	w writer.Type,
	log log.Modular,
	stats metrics.Type,
) (Type, error) {
	return &Writer{
		running:      1,
		typeStr:      typeStr,
		writer:       w,
		log:          log,
		stats:        stats,
		transactions: nil,
		closeChan:    make(chan struct{}),
		closedChan:   make(chan struct{}),
	}, nil
}

//------------------------------------------------------------------------------

// loop is an internal loop that brokers incoming messages to output pipe.
func (w *Writer) loop() {
	// Metrics paths
	var (
		mRunning      = w.stats.GetGauge("running")
		mCount        = w.stats.GetCounter("count")
		mPartsCount   = w.stats.GetCounter("parts.count")
		mSuccess      = w.stats.GetCounter("send.success")
		mPartsSuccess = w.stats.GetCounter("parts.send.success")
		mError        = w.stats.GetCounter("send.error")
		mSent         = w.stats.GetCounter("batch.sent")
		mPartsSent    = w.stats.GetCounter("sent")
		mConn         = w.stats.GetCounter("connection.up")
		mFailedConn   = w.stats.GetCounter("connection.failed")
		mLostConn     = w.stats.GetCounter("connection.lost")
	)

	defer func() {
		err := w.writer.WaitForClose(time.Second)
		for ; err != nil; err = w.writer.WaitForClose(time.Second) {
		}
		mRunning.Decr(1)
		atomic.StoreInt32(&w.isConnected, 0)
		close(w.closedChan)
	}()
	mRunning.Incr(1)

	throt := throttle.New(throttle.OptCloseChan(w.closeChan))

	for {
		if err := w.writer.Connect(); err != nil {
			// Close immediately if our writer is closed.
			if err == types.ErrTypeClosed {
				return
			}

			w.log.Errorf("Failed to connect to %v: %v\n", w.typeStr, err)
			mFailedConn.Incr(1)
			if !throt.Retry() {
				return
			}
		} else {
			break
		}
	}
	mConn.Incr(1)
	atomic.StoreInt32(&w.isConnected, 1)

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
		err := w.writer.Write(ts.Payload)

		// If our writer says it is not connected.
		if err == types.ErrNotConnected {
			mLostConn.Incr(1)
			atomic.StoreInt32(&w.isConnected, 0)

			// Continue to try to reconnect while still active.
			for atomic.LoadInt32(&w.running) == 1 {
				if err = w.writer.Connect(); err != nil {
					// Close immediately if our writer is closed.
					if err == types.ErrTypeClosed {
						return
					}

					w.log.Errorf("Failed to reconnect to %v: %v\n", w.typeStr, err)
					mFailedConn.Incr(1)
					if !throt.Retry() {
						return
					}
				} else if err = w.writer.Write(ts.Payload); err != types.ErrNotConnected {
					atomic.StoreInt32(&w.isConnected, 1)
					mConn.Incr(1)
					break
				} else if !throt.Retry() {
					return
				}
			}
		}

		// Close immediately if our writer is closed.
		if err == types.ErrTypeClosed {
			return
		}

		if err != nil {
			w.log.Errorf("Failed to send message to %v: %v\n", w.typeStr, err)
			mError.Incr(1)
			if !throt.Retry() {
				return
			}
		} else {
			mSuccess.Incr(1)
			mPartsSuccess.Incr(int64(ts.Payload.Len()))
			mSent.Incr(1)
			mPartsSent.Incr(int64(ts.Payload.Len()))
			throt.Reset()
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

// Consume assigns a messages channel for the output to read.
func (w *Writer) Consume(ts <-chan types.Transaction) error {
	if w.transactions != nil {
		return types.ErrAlreadyStarted
	}
	w.transactions = ts
	go w.loop()
	return nil
}

// Connected returns a boolean indicating whether this output is currently
// connected to its target.
func (w *Writer) Connected() bool {
	return atomic.LoadInt32(&w.isConnected) == 1
}

// CloseAsync shuts down the File output and stops processing messages.
func (w *Writer) CloseAsync() {
	if atomic.CompareAndSwapInt32(&w.running, 1, 0) {
		w.writer.CloseAsync()
		close(w.closeChan)
	}
}

// WaitForClose blocks until the File output has closed down.
func (w *Writer) WaitForClose(timeout time.Duration) error {
	select {
	case <-w.closedChan:
	case <-time.After(timeout):
		return types.ErrTimeout
	}
	return nil
}

//------------------------------------------------------------------------------
