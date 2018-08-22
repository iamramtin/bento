// Copyright (c) 2018 Ashley Jeffs
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, sub to the following conditions:
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

package metrics

//------------------------------------------------------------------------------

type fCounterVec struct {
	f func() StatCounter
}

func (f *fCounterVec) With(labels ...string) StatCounter {
	return f.f()
}

func fakeCounterVec(f func() StatCounter) StatCounterVec {
	return &fCounterVec{
		f: f,
	}
}

//------------------------------------------------------------------------------

type fTimerVec struct {
	f func() StatTimer
}

func (f *fTimerVec) With(labels ...string) StatTimer {
	return f.f()
}

func fakeTimerVec(f func() StatTimer) StatTimerVec {
	return &fTimerVec{
		f: f,
	}
}

//------------------------------------------------------------------------------

type fGaugeVec struct {
	f func() StatGauge
}

func (f *fGaugeVec) With(labels ...string) StatGauge {
	return f.f()
}

func fakeGaugeVec(f func() StatGauge) StatGaugeVec {
	return &fGaugeVec{
		f: f,
	}
}

//------------------------------------------------------------------------------
