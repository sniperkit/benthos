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

package manager

import (
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/Jeffail/benthos/lib/metrics"
	"github.com/Jeffail/benthos/lib/pipeline"
	"github.com/Jeffail/benthos/lib/processor"
	"github.com/Jeffail/benthos/lib/stream"
	"github.com/Jeffail/benthos/lib/types"
	"github.com/Jeffail/benthos/lib/log"
)

func harmlessConf() stream.Config {
	c := stream.NewConfig()
	c.Input.Type = "http_server"
	c.Output.Type = "http_server"
	return c
}

type mockProc struct {
	mChan chan struct{}
}

func (m *mockProc) ProcessMessage(msg types.Message) ([]types.Message, types.Response) {
	m.mChan <- struct{}{}
	return []types.Message{msg}, nil
}

func TestTypeProcsAndPipes(t *testing.T) {
	var mockProcs []*mockProc
	for i := 0; i < 6; i++ {
		mockProcs = append(mockProcs, &mockProc{
			mChan: make(chan struct{}),
		})
	}

	logger := log.New(os.Stdout, log.Config{LogLevel: "NONE"})
	stats := metrics.DudType{}

	mgr := New(
		OptSetLogger(logger),
		OptSetStats(stats),
		OptSetManager(types.DudMgr{}),
		OptAddInputPipelines(func(id string) (pipeline.Type, error) {
			if id != "foo" {
				t.Errorf("Wrong id: %v != %v", id, "foo")
			}
			return pipeline.NewProcessor(logger, stats, mockProcs[0]), nil
		}, func(id string) (pipeline.Type, error) {
			if id != "foo" {
				t.Errorf("Wrong id: %v != %v", id, "foo")
			}
			return pipeline.NewProcessor(logger, stats, mockProcs[1]), nil
		}),
		OptAddProcessors(func(id string) (processor.Type, error) {
			if id != "foo" {
				t.Errorf("Wrong id: %v != %v", id, "foo")
			}
			return mockProcs[2], nil
		}, func(id string) (processor.Type, error) {
			if id != "foo" {
				t.Errorf("Wrong id: %v != %v", id, "foo")
			}
			return mockProcs[3], nil
		}),
		OptAddOutputPipelines(func(id string) (pipeline.Type, error) {
			if id != "foo" {
				t.Errorf("Wrong id: %v != %v", id, "foo")
			}
			return pipeline.NewProcessor(logger, stats, mockProcs[4]), nil
		}, func(id string) (pipeline.Type, error) {
			if id != "foo" {
				t.Errorf("Wrong id: %v != %v", id, "foo")
			}
			return pipeline.NewProcessor(logger, stats, mockProcs[5]), nil
		}),
	)

	conf := harmlessConf()
	conf.Input.Type = "file"
	conf.Input.File.Path = "./package.go"

	if err := mgr.Create("foo", conf); err != nil {
		t.Fatal(err)
	}

	for i, proc := range mockProcs {
		select {
		case <-proc.mChan:
		case <-time.After(time.Second):
			t.Errorf("Timed out waiting for message to reach pipe: %v", i)
		}
	}

	if err := mgr.Stop(time.Second); err != nil {
		t.Error(err)
	}
}

func TestTypeBasicOperations(t *testing.T) {
	mgr := New(
		OptSetLogger(log.New(os.Stdout, log.Config{LogLevel: "NONE"})),
		OptSetStats(metrics.DudType{}),
		OptSetManager(types.DudMgr{}),
	)

	if err := mgr.Update("foo", harmlessConf(), time.Second); err == nil {
		t.Error("Expected error on empty update")
	}
	if _, err := mgr.Read("foo"); err == nil {
		t.Error("Expected error on empty read")
	}

	if err := mgr.Create("foo", harmlessConf()); err != nil {
		t.Fatal(err)
	}
	if err := mgr.Create("foo", harmlessConf()); err == nil {
		t.Error("Expected error on duplicate create")
	}

	if info, err := mgr.Read("foo"); err != nil {
		t.Error(err)
	} else if !info.Active {
		t.Error("Stream not active")
	} else if act, exp := info.Config, harmlessConf(); !reflect.DeepEqual(act, exp) {
		t.Errorf("Unexpected config: %v != %v", act, exp)
	}

	newConf := harmlessConf()
	newConf.Buffer.Type = "memory"

	if err := mgr.Update("foo", newConf, time.Second); err != nil {
		t.Error(err)
	}

	if info, err := mgr.Read("foo"); err != nil {
		t.Error(err)
	} else if !info.Active {
		t.Error("Stream not active")
	} else if act, exp := info.Config, newConf; !reflect.DeepEqual(act, exp) {
		t.Errorf("Unexpected config: %v != %v", act, exp)
	}

	if err := mgr.Delete("foo", time.Second); err != nil {
		t.Fatal(err)
	}
	if err := mgr.Delete("foo", time.Second); err == nil {
		t.Error("Expected error on duplicate delete")
	}

	if err := mgr.Stop(time.Second); err != nil {
		t.Error(err)
	}

	if exp, act := types.ErrTypeClosed, mgr.Create("foo", harmlessConf()); act != exp {
		t.Errorf("Unexpected error: %v != %v", act, exp)
	}
}

func TestTypeBasicClose(t *testing.T) {
	mgr := New(
		OptSetLogger(log.New(os.Stdout, log.Config{LogLevel: "NONE"})),
		OptSetStats(metrics.DudType{}),
		OptSetManager(types.DudMgr{}),
	)

	conf := harmlessConf()
	conf.Output.Type = "scalability_protocols"

	if err := mgr.Create("foo", conf); err != nil {
		t.Fatal(err)
	}

	if err := mgr.Stop(time.Second); err != nil {
		t.Error(err)
	}

	if exp, act := types.ErrTypeClosed, mgr.Create("foo", harmlessConf()); act != exp {
		t.Errorf("Unexpected error: %v != %v", act, exp)
	}
}
