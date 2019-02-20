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

package input

import (
	"sync/atomic"
	"time"

	"github.com/Jeffail/benthos/lib/input/reader"
	"github.com/Jeffail/benthos/lib/log"
	"github.com/Jeffail/benthos/lib/message/tracing"
	"github.com/Jeffail/benthos/lib/metrics"
	"github.com/Jeffail/benthos/lib/types"
	"github.com/Jeffail/benthos/lib/util/throttle"
)

//------------------------------------------------------------------------------

// Reader is an input implementation that reads messages from a reader.Type.
type Reader struct {
	running   int32
	connected int32

	typeStr string
	reader  reader.Type

	stats metrics.Type
	log   log.Modular

	connThrot *throttle.Type

	transactions chan types.Transaction
	responses    chan types.Response

	closeChan  chan struct{}
	closedChan chan struct{}
}

// NewReader creates a new Reader input type.
func NewReader(
	typeStr string,
	r reader.Type,
	log log.Modular,
	stats metrics.Type,
) (Type, error) {
	rdr := &Reader{
		running:      1,
		typeStr:      typeStr,
		reader:       r,
		log:          log,
		stats:        stats,
		transactions: make(chan types.Transaction),
		responses:    make(chan types.Response),
		closeChan:    make(chan struct{}),
		closedChan:   make(chan struct{}),
	}

	rdr.connThrot = throttle.New(throttle.OptCloseChan(rdr.closeChan))

	go rdr.loop()
	return rdr, nil
}

//------------------------------------------------------------------------------

func (r *Reader) loop() {
	// Metrics paths
	var (
		mRunning     = r.stats.GetGauge("running")
		mCount       = r.stats.GetCounter("count")
		mPartsCount  = r.stats.GetCounter("parts.count")
		mRcvd        = r.stats.GetCounter("batch.received")
		mPartsRcvd   = r.stats.GetCounter("received")
		mReadSuccess = r.stats.GetCounter("read.success")
		mReadError   = r.stats.GetCounter("read.error")
		mSendSuccess = r.stats.GetCounter("send.success")
		mSendError   = r.stats.GetCounter("send.error")
		mAckSuccess  = r.stats.GetCounter("ack.success")
		mAckError    = r.stats.GetCounter("ack.error")
		mConn        = r.stats.GetCounter("connection.up")
		mFailedConn  = r.stats.GetCounter("connection.failed")
		mLostConn    = r.stats.GetCounter("connection.lost")
		mLatency     = r.stats.GetTimer("latency")
	)

	defer func() {
		err := r.reader.WaitForClose(time.Second)
		for ; err != nil; err = r.reader.WaitForClose(time.Second) {
		}
		mRunning.Decr(1)
		atomic.StoreInt32(&r.connected, 0)

		close(r.transactions)
		close(r.closedChan)
	}()
	mRunning.Incr(1)

	for {
		if err := r.reader.Connect(); err != nil {
			if err == types.ErrTypeClosed {
				return
			}
			r.log.Errorf("Failed to connect to %v: %v\n", r.typeStr, err)
			mFailedConn.Incr(1)
			if !r.connThrot.Retry() {
				return
			}
		} else {
			r.connThrot.Reset()
			break
		}
	}
	mConn.Incr(1)
	atomic.StoreInt32(&r.connected, 1)

	for atomic.LoadInt32(&r.running) == 1 {
		msg, err := r.reader.Read()

		// If our reader says it is not connected.
		if err == types.ErrNotConnected {
			mLostConn.Incr(1)
			atomic.StoreInt32(&r.connected, 0)

			// Continue to try to reconnect while still active.
			for atomic.LoadInt32(&r.running) == 1 {
				if err = r.reader.Connect(); err != nil {
					// Close immediately if our reader is closed.
					if err == types.ErrTypeClosed {
						return
					}

					r.log.Errorf("Failed to reconnect to %v: %v\n", r.typeStr, err)
					mFailedConn.Incr(1)

					if !r.connThrot.Retry() {
						return
					}
				} else if msg, err = r.reader.Read(); err != types.ErrNotConnected {
					mConn.Incr(1)
					atomic.StoreInt32(&r.connected, 1)
					r.connThrot.Reset()
					break
				}
			}
		}

		// Close immediately if our reader is closed.
		if err == types.ErrTypeClosed {
			return
		}

		if err != nil || msg == nil {
			if err != types.ErrTimeout && err != types.ErrNotConnected {
				mReadError.Incr(1)
				r.log.Errorf("Failed to read message: %v\n", err)
			}
			if !r.connThrot.Retry() {
				return
			}
			continue
		} else {
			r.connThrot.Reset()
			mCount.Incr(1)
			mPartsCount.Incr(int64(msg.Len()))
			mReadSuccess.Incr(1)
			mPartsRcvd.Incr(int64(msg.Len()))
			mRcvd.Incr(1)
		}

		tracing.InitSpans("input_"+r.typeStr, msg)
		select {
		case r.transactions <- types.NewTransaction(msg, r.responses):
		case <-r.closeChan:
			return
		}

		select {
		case res, open := <-r.responses:
			if !open {
				return
			}
			if res.Error() != nil {
				mSendError.Incr(1)
			} else {
				mSendSuccess.Incr(1)
			}
			if res.Error() != nil || !res.SkipAck() {
				if err = r.reader.Acknowledge(res.Error()); err != nil {
					mAckError.Incr(1)
				} else {
					tTaken := time.Since(msg.CreatedAt()).Nanoseconds()
					mLatency.Timing(tTaken)
					mAckSuccess.Incr(1)
				}
			}
		case <-r.closeChan:
			return
		}
		tracing.FinishSpans(msg)
	}
}

// TransactionChan returns a transactions channel for consuming messages from
// this input type.
func (r *Reader) TransactionChan() <-chan types.Transaction {
	return r.transactions
}

// Connected returns a boolean indicating whether this input is currently
// connected to its target.
func (r *Reader) Connected() bool {
	return atomic.LoadInt32(&r.connected) == 1
}

// CloseAsync shuts down the Reader input and stops processing requests.
func (r *Reader) CloseAsync() {
	if atomic.CompareAndSwapInt32(&r.running, 1, 0) {
		r.reader.CloseAsync()
		close(r.closeChan)
	}
}

// WaitForClose blocks until the Reader input has closed down.
func (r *Reader) WaitForClose(timeout time.Duration) error {
	select {
	case <-r.closedChan:
	case <-time.After(timeout):
		return types.ErrTimeout
	}
	return nil
}

//------------------------------------------------------------------------------
