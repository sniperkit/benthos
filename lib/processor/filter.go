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

package processor

import (
	"fmt"

	"github.com/Jeffail/benthos/lib/metrics"
	"github.com/Jeffail/benthos/lib/processor/condition"
	"github.com/Jeffail/benthos/lib/types"
	"github.com/Jeffail/benthos/lib/log"
)

//------------------------------------------------------------------------------

func init() {
	Constructors["filter"] = TypeSpec{
		constructor: NewFilter,
		description: `
Tests each message against a condition, if the condition fails then the message
is dropped. You can read a [full list of conditions here](../conditions).`,
	}
}

//------------------------------------------------------------------------------

// FilterConfig contains configuration fields for the Filter processor.
type FilterConfig struct {
	condition.Config `json:",inline" yaml:",inline"`
}

// NewFilterConfig returns a FilterConfig with default values.
func NewFilterConfig() FilterConfig {
	return FilterConfig{
		Config: condition.NewConfig(),
	}
}

//------------------------------------------------------------------------------

// Filter is a processor that checks each message against a condition and
// rejects when the condition returns false.
type Filter struct {
	log   log.Modular
	stats metrics.Type

	condition condition.Type

	mCount   metrics.StatCounter
	mDropped metrics.StatCounter
	mSent    metrics.StatCounter
}

// NewFilter returns a Filter processor.
func NewFilter(
	conf Config, mgr types.Manager, log log.Modular, stats metrics.Type,
) (Type, error) {
	cond, err := condition.New(conf.Filter.Config, mgr, log, stats)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to construct condition '%v': %v",
			conf.Filter.Config.Type, err,
		)
	}
	return &Filter{
		log:       log.NewModule(".processor.filter"),
		stats:     stats,
		condition: cond,

		mCount:   stats.GetCounter("processor.filter.count"),
		mDropped: stats.GetCounter("processor.filter.dropped"),
		mSent:    stats.GetCounter("processor.filter.sent"),
	}, nil
}

//------------------------------------------------------------------------------

// ProcessMessage checks each message against a set of bounds.
func (c *Filter) ProcessMessage(msg types.Message) ([]types.Message, types.Response) {
	c.mCount.Incr(1)

	if !c.condition.Check(msg) {
		c.mDropped.Incr(1)
		return nil, types.NewSimpleResponse(nil)
	}

	c.mSent.Incr(1)
	msgs := [1]types.Message{msg}
	return msgs[:], nil
}

//------------------------------------------------------------------------------
