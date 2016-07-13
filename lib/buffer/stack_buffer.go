/*
Copyright (c) 2014 Ashley Jeffs

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/

package buffer

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/jeffail/benthos/lib/buffer/ring"
	"github.com/jeffail/benthos/lib/types"
	"github.com/jeffail/util/metrics"
)

//--------------------------------------------------------------------------------------------------

// StackBuffer - An agent that wraps an output with a message buffer.
type StackBuffer struct {
	stats metrics.Aggregator

	buffer ring.MessageStack

	running int32

	messagesIn   <-chan types.Message
	messagesOut  chan types.Message
	responsesIn  <-chan types.Response
	responsesOut chan types.Response
	errorsChan   chan []error

	closedWG sync.WaitGroup
}

// NewStackBuffer - Create a new buffered agent type.
func NewStackBuffer(buffer ring.MessageStack, stats metrics.Aggregator) Type {
	m := StackBuffer{
		stats:        stats,
		buffer:       buffer,
		running:      1,
		messagesOut:  make(chan types.Message),
		responsesOut: make(chan types.Response),
		errorsChan:   make(chan []error),
	}

	return &m
}

//--------------------------------------------------------------------------------------------------

// inputLoop - Internal loop brokers incoming messages to output pipe.
func (m *StackBuffer) inputLoop() {
	defer close(m.responsesOut)
	defer m.closedWG.Done()
	defer m.buffer.Close()

	for atomic.LoadInt32(&m.running) == 1 {
		msg, open := <-m.messagesIn
		if !open {
			return
		}
		backlog, err := m.buffer.PushMessage(msg)
		if err == nil {
			m.stats.Gauge("buffer.backlog", backlog)
		}
		m.responsesOut <- types.NewSimpleResponse(err)
	}
}

// outputLoop - Internal loop brokers incoming messages to output pipe.
func (m *StackBuffer) outputLoop() {
	defer close(m.errorsChan)
	defer close(m.messagesOut)
	defer m.closedWG.Done()

	errs := []error{}
	errMap := map[error]struct{}{}

	var msg types.Message
	for atomic.LoadInt32(&m.running) == 1 {
		if msg.Parts == nil {
			var err error
			msg, err = m.buffer.NextMessage()
			if err != nil && err != types.ErrTypeClosed {
				// Unconventional errors here should always indicate some sort of corruption.
				// Hopefully the corruption was message specific and not the whole buffer, so we can
				// try shifting and reading again.
				m.buffer.ShiftMessage()
				if _, exists := errMap[err]; !exists {
					errMap[err] = struct{}{}
					errs = append(errs, err)
				}
			}
		}

		if msg.Parts != nil {
			m.messagesOut <- msg
			res, open := <-m.responsesIn
			if !open {
				return
			}
			if res.Error() == nil {
				msg = types.Message{}
				backlog, _ := m.buffer.ShiftMessage()
				m.stats.Gauge("buffer.backlog", backlog)
			} else {
				if _, exists := errMap[res.Error()]; !exists {
					errMap[res.Error()] = struct{}{}
					errs = append(errs, res.Error())
				}
			}
		}

		// If we have errors built up.
		if len(errs) > 0 {
			select {
			case m.errorsChan <- errs:
				errMap = map[error]struct{}{}
				errs = []error{}
			default:
				// Reader not ready, do not block here.
			}
		}
	}
}

// StartReceiving - Assigns a messages channel for the output to read.
func (m *StackBuffer) StartReceiving(msgs <-chan types.Message) error {
	if m.messagesIn != nil {
		return types.ErrAlreadyStarted
	}
	m.messagesIn = msgs

	if m.responsesIn != nil {
		m.closedWG.Add(2)
		go m.inputLoop()
		go m.outputLoop()
	}
	return nil
}

// MessageChan - Returns the channel used for consuming messages from this input.
func (m *StackBuffer) MessageChan() <-chan types.Message {
	return m.messagesOut
}

// StartListening - Sets the channel for reading responses.
func (m *StackBuffer) StartListening(responses <-chan types.Response) error {
	if m.responsesIn != nil {
		return types.ErrAlreadyStarted
	}
	m.responsesIn = responses

	if m.messagesIn != nil {
		m.closedWG.Add(2)
		go m.inputLoop()
		go m.outputLoop()
	}
	return nil
}

// ResponseChan - Returns the response channel.
func (m *StackBuffer) ResponseChan() <-chan types.Response {
	return m.responsesOut
}

// ErrorsChan - Returns the errors channel.
func (m *StackBuffer) ErrorsChan() <-chan []error {
	return m.errorsChan
}

// CloseAsync - Shuts down the StackBuffer output and stops processing messages.
func (m *StackBuffer) CloseAsync() {
	atomic.StoreInt32(&m.running, 0)
}

// WaitForClose - Blocks until the StackBuffer output has closed down.
func (m *StackBuffer) WaitForClose(timeout time.Duration) error {
	closed := make(chan struct{})
	go func() {
		m.closedWG.Wait()
		close(closed)
	}()

	select {
	case <-closed:
	case <-time.After(timeout):
		return types.ErrTimeout
	}
	return nil
}

//--------------------------------------------------------------------------------------------------