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

package input

import (
	"github.com/Jeffail/benthos/lib/input/reader"
	"github.com/Jeffail/benthos/lib/metrics"
	"github.com/Jeffail/benthos/lib/types"
	"github.com/Jeffail/benthos/lib/log"
)

//------------------------------------------------------------------------------

func init() {
	Constructors["nats"] = TypeSpec{
		constructor: NewNATS,
		description: `
Subscribe to a NATS subject. NATS is at-most-once, if you need at-least-once
behaviour then look at NATS Stream.

The urls can contain username/password semantics. e.g.
nats://derek:pass@localhost:4222`,
	}
}

//------------------------------------------------------------------------------

// NewNATS creates a new NATS input type.
func NewNATS(conf Config, mgr types.Manager, log log.Modular, stats metrics.Type) (Type, error) {
	n, err := reader.NewNATS(conf.NATS, log, stats)
	if err != nil {
		return nil, err
	}
	return NewReader("nats", reader.NewPreserver(n), log, stats)
}

//------------------------------------------------------------------------------
