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
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/Jeffail/benthos/lib/stream"
	"github.com/gorilla/mux"
	yaml "gopkg.in/yaml.v2"
)

//------------------------------------------------------------------------------

func (m *Type) registerEndpoints() {
	m.manager.RegisterEndpoint(
		"/streams",
		"GET: List all streams along with their status and uptimes."+
			" POST: Post an object of stream ids to stream configs, all"+
			" streams will be replaced by this new set.",
		m.HandleStreamsCRUD,
	)
	m.manager.RegisterEndpoint(
		"/streams/{id}",
		"Perform CRUD operations on streams, supporting POST (Create),"+
			" GET (Read), PUT (Update) and DELETE (Delete).",
		m.HandleStreamCRUD,
	)
}

// HandleStreamsCRUD is an http.HandleFunc for returning maps of active benthos
// streams by their id, status and uptime or overwriting the entire set of
// streams.
func (m *Type) HandleStreamsCRUD(w http.ResponseWriter, r *http.Request) {
	var serverErr, requestErr error
	defer func() {
		if r.Body != nil {
			r.Body.Close()
		}
		if serverErr != nil {
			m.logger.Errorf("Streams CRUD Error: %v\n", serverErr)
			http.Error(w, fmt.Sprintf("Error: %v", serverErr), http.StatusBadGateway)
		}
		if requestErr != nil {
			m.logger.Debugf("Streams request CRUD Error: %v\n", requestErr)
			http.Error(w, fmt.Sprintf("Error: %v", requestErr), http.StatusBadRequest)
		}
	}()

	type confInfo struct {
		Active    bool    `json:"active"`
		Uptime    float64 `json:"uptime"`
		UptimeStr string  `json:"uptime_str"`
	}
	infos := map[string]confInfo{}

	m.lock.Lock()
	for id, strInfo := range m.streams {
		infos[id] = confInfo{
			Active:    strInfo.IsRunning(),
			Uptime:    strInfo.Uptime().Seconds(),
			UptimeStr: strInfo.Uptime().String(),
		}
	}
	m.lock.Unlock()

	switch r.Method {
	case "GET":
		var resBytes []byte
		if resBytes, serverErr = json.Marshal(infos); serverErr == nil {
			w.Write(resBytes)
		}
		return
	case "POST":
	default:
		requestErr = errors.New("Method not supported")
		return
	}

	newSet := ConfigSet{}

	var setBytes []byte
	if setBytes, requestErr = ioutil.ReadAll(r.Body); requestErr != nil {
		return
	}
	if requestErr = yaml.Unmarshal(setBytes, &newSet); requestErr != nil {
		return
	}

	toDelete := []string{}
	toUpdate := map[string]stream.Config{}
	toCreate := map[string]stream.Config{}

	for id := range infos {
		if newConf, exists := newSet[id]; !exists {
			toDelete = append(toDelete, id)
		} else {
			toUpdate[id] = newConf
		}
	}
	for id, conf := range newSet {
		if _, exists := infos[id]; !exists {
			toCreate[id] = conf
		}
	}

	deadline, hasDeadline := r.Context().Deadline()
	if !hasDeadline {
		deadline = time.Now().Add(m.apiTimeout)
	}

	wg := sync.WaitGroup{}
	wg.Add(len(toDelete))
	wg.Add(len(toUpdate))
	wg.Add(len(toCreate))

	errDelete := make([]error, len(toDelete))
	errUpdate := make([]error, len(toUpdate))
	errCreate := make([]error, len(toCreate))

	for i, id := range toDelete {
		go func(sid string, j int) {
			errDelete[j] = m.Delete(sid, time.Until(deadline))
			wg.Done()
		}(id, i)
	}
	i := 0
	for id, conf := range toUpdate {
		newConf := conf
		go func(sid string, sconf *stream.Config, j int) {
			errUpdate[j] = m.Update(sid, *sconf, time.Until(deadline))
			wg.Done()
		}(id, &newConf, i)
		i++
	}
	i = 0
	for id, conf := range toCreate {
		newConf := conf
		go func(sid string, sconf *stream.Config, j int) {
			errCreate[j] = m.Create(sid, *sconf)
			wg.Done()
		}(id, &newConf, i)
		i++
	}

	wg.Wait()

	errs := []string{}
	for _, err := range errDelete {
		if err != nil {
			errs = append(errs, fmt.Sprintf("failed to delete stream: %v", err))
		}
	}
	for _, err := range errUpdate {
		if err != nil {
			errs = append(errs, fmt.Sprintf("failed to update stream: %v", err))
		}
	}
	for _, err := range errCreate {
		if err != nil {
			errs = append(errs, fmt.Sprintf("failed to create stream: %v", err))
		}
	}

	if len(errs) > 0 {
		requestErr = errors.New(strings.Join(errs, "\n"))
	}
}

// HandleStreamCRUD is an http.HandleFunc for performing CRUD operations on
// individual streams.
func (m *Type) HandleStreamCRUD(w http.ResponseWriter, r *http.Request) {
	var serverErr, requestErr error
	defer func() {
		if r.Body != nil {
			r.Body.Close()
		}
		if serverErr != nil {
			m.logger.Errorf("Streams CRUD Error: %v\n", serverErr)
			http.Error(w, fmt.Sprintf("Error: %v", serverErr), http.StatusBadGateway)
		}
		if requestErr != nil {
			m.logger.Debugf("Streams request CRUD Error: %v\n", requestErr)
			http.Error(w, fmt.Sprintf("Error: %v", requestErr), http.StatusBadRequest)
		}
	}()

	id := mux.Vars(r)["id"]
	if len(id) == 0 {
		http.Error(w, "Var `id` must be set", http.StatusBadRequest)
		return
	}

	readConfig := func() (conf stream.Config, err error) {
		var confBytes []byte
		if confBytes, err = ioutil.ReadAll(r.Body); err != nil {
			return
		}

		conf = stream.NewConfig()
		err = yaml.Unmarshal(confBytes, &conf)
		return
	}

	deadline, hasDeadline := r.Context().Deadline()
	if !hasDeadline {
		deadline = time.Now().Add(m.apiTimeout)
	}

	var conf stream.Config
	switch r.Method {
	case "POST":
		if conf, requestErr = readConfig(); requestErr != nil {
			return
		}
		serverErr = m.Create(id, conf)
	case "GET":
		var info StreamStatus
		if info, serverErr = m.Read(id); serverErr == nil {
			sanit, _ := info.Config.Sanitised()

			var bodyBytes []byte
			if bodyBytes, serverErr = json.Marshal(struct {
				Active    bool        `json:"active"`
				Uptime    float64     `json:"uptime"`
				UptimeStr string      `json:"uptime_str"`
				Config    interface{} `json:"config"`
			}{
				Active:    info.Active,
				Uptime:    info.Uptime.Seconds(),
				UptimeStr: info.Uptime.String(),
				Config:    sanit,
			}); serverErr != nil {
				return
			}

			w.Write(bodyBytes)
		}
	case "PUT":
		if conf, requestErr = readConfig(); requestErr != nil {
			return
		}
		serverErr = m.Update(id, conf, time.Until(deadline))
	case "DELETE":
		serverErr = m.Delete(id, time.Until(deadline))
	default:
		requestErr = fmt.Errorf("verb not supported: %v", r.Method)
	}

	if serverErr == ErrStreamDoesNotExist {
		serverErr = nil
		http.Error(w, "Stream not found", http.StatusNotFound)
	}
	if serverErr == ErrStreamExists {
		serverErr = nil
		http.Error(w, "Stream already exists", http.StatusBadRequest)
	}
}

//------------------------------------------------------------------------------
