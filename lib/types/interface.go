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

package types

import (
	"net/http"
	"time"
)

//------------------------------------------------------------------------------

// Cache is a key/value store that can be shared across components and executing
// threads of a Benthos service.
type Cache interface {
	// Get attempts to locate and return a cached value by its key, returns an
	// error if the key does not exist or if the command fails.
	Get(key string) ([]byte, error)

	// Set attempts to set the value of a key, returns an error if the command
	// fails.
	Set(key string, value []byte) error

	// Add attempts to set the value of a key only if the key does not already
	// exist, returns an error if the key already exists or if the command
	// fails.
	Add(key string, value []byte) error

	// Delete attempts to remove a key. Returns an error if a failure occurs.
	Delete(key string) error
}

//------------------------------------------------------------------------------

// Condition reads a message, calculates a condition and returns a boolean.
type Condition interface {
	// Check tests a message against a configured condition.
	Check(msg Message) bool
}

//------------------------------------------------------------------------------

// Manager is an interface expected by Benthos components that allows them to
// register their service wide behaviours such as HTTP endpoints and event
// listeners, and obtain service wide shared resources such as caches.
type Manager interface {
	// RegisterEndpoint registers a server wide HTTP endpoint.
	RegisterEndpoint(path, desc string, h http.HandlerFunc)

	// GetCache attempts to find a service wide cache by its name.
	GetCache(name string) (Cache, error)

	// GetCondition attempts to find a service wide condition by its name.
	GetCondition(name string) (Condition, error)
}

//------------------------------------------------------------------------------

// Closable defines a type that can be safely closed down and cleaned up.
type Closable interface {
	// CloseAsync triggers a closure of this object but does not block until
	// completion.
	CloseAsync()

	// WaitForClose is a blocking call to wait until the object has finished
	// closing down and cleaning up resources.
	WaitForClose(timeout time.Duration) error
}

//------------------------------------------------------------------------------

// Transactor is a type that sends messages and waits for a response back, the
// response indicates whether the message was successfully propagated to a new
// destination (and can be discarded from the source.)
type Transactor interface {
	// TransactionChan returns a channel used for consuming transactions from
	// this type. Every transaction received must be resolved before another
	// transaction will be sent.
	TransactionChan() <-chan Transaction
}

// TransactionReceiver is a type that receives transactions from a Transactor.
type TransactionReceiver interface {
	// StartReceiving starts the type receiving transactions from a Transactor.
	StartReceiving(<-chan Transaction) error
}

//------------------------------------------------------------------------------

// Producer is the higher level producer type.
type Producer interface {
	Transactor
}

// Consumer is the higher level consumer type.
type Consumer interface {
	TransactionReceiver
}

//------------------------------------------------------------------------------

// Output is a closable consumer.
type Output interface {
	Consumer
	Closable
}

// Input is a closable producer.
type Input interface {
	Producer
	Closable
}

//------------------------------------------------------------------------------
