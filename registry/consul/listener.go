/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package consul

import (
	"sync"
)

import (
	consul "github.com/hashicorp/consul/api"
	"github.com/hashicorp/consul/api/watch"
	perrors "github.com/pkg/errors"
)

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/registry"
	"github.com/apache/dubbo-go/remoting"
)

// Consul listener wraps the consul watcher, to
// listen the service information change in consul
// registry.
type consulListener struct {
	// Registry url.
	registryUrl common.URL

	// Consumer url.
	consumerUrl common.URL

	// Consul watcher.
	plan *watch.Plan

	// Most recent service urls return by
	// watcher.
	urls []common.URL

	// All service information changes will
	// be wrapped into ServiceEvent, and be
	// sent into eventCh. Then listener's
	// Next method will get event from eventCh,
	// and return to upstream.
	eventCh chan *registry.ServiceEvent

	// All errors, happening in the listening
	// period, will be caught and send into
	// errCh. Then listener's Next method will
	// get error from errCh, and return to
	// upstream.
	errCh chan error

	// Running field represents whether consul
	// listener has been closed. When closing
	// listener, this field will be set to
	// false, and will notify consul watcher
	// to close.
	running bool

	// After listener notifies consul watcher
	// to close, listener will call wg.wait to
	// make sure that consul watcher is closed
	// before the listener closes.
	wg sync.WaitGroup
}

func newConsulListener(registryUrl common.URL, consumerUrl common.URL) (*consulListener, error) {
	params := make(map[string]interface{})
	params["type"] = "service"
	params["service"] = consumerUrl.Service()
	params["tag"] = "dubbo"
	params["passingonly"] = true
	plan, err := watch.Parse(params)
	if err != nil {
		return nil, err
	}

	listener := &consulListener{
		registryUrl: registryUrl,
		consumerUrl: consumerUrl,
		plan:        plan,
		urls:        make([]common.URL, 0),
		eventCh:     make(chan *registry.ServiceEvent, 1),
		errCh:       make(chan error, 1),
		running:     true,
	}

	// Set handler to consul watcher, and
	// make watcher begin to watch service
	// information change.
	listener.plan.Handler = listener.handler
	listener.wg.Add(1)
	go listener.run()
	return listener, nil
}

// Wrap the consul watcher run api. There are three
// conditions that will finish the run:
//   - set running to false
//   - call plan.Stop
//   - close eventCh and errCh
// If run meets first two conditions, it will close
// gracefully. However, if run meets the last condition,
// run will close with panic, so use recover to cover
// this case.
func (l *consulListener) run() {
	defer func() {
		recover()
	}()

	if l.running {
		err := l.plan.Run(l.registryUrl.Location)
		if err != nil {
			l.errCh <- err
		}
	}
}

func (l *consulListener) handler(idx uint64, raw interface{}) {
	var (
		service *consul.ServiceEntry
		event   *registry.ServiceEvent
		url     common.URL
		ok      bool
		err     error
	)

	services, ok := raw.([]*consul.ServiceEntry)
	if !ok {
		err = perrors.New("handler get non ServiceEntry type parameter")
		l.errCh <- err
		return
	}
	newUrls := make([]common.URL, 0)
	events := make([]*registry.ServiceEvent, 0)

	for _, service = range services {
		url, err = retrieveURL(service)
		if err != nil {
			l.errCh <- err
			return
		}
		newUrls = append(newUrls, url)
	}

	for _, url = range l.urls {
		ok = in(url, newUrls)
		if !ok {
			event := &registry.ServiceEvent{Action: remoting.EventTypeDel, Service: url}
			events = append(events, event)
		}
	}

	for _, url = range newUrls {
		ok = in(url, l.urls)
		if !ok {
			event := &registry.ServiceEvent{Action: remoting.EventTypeAdd, Service: url}
			events = append(events, event)
		}
	}

	l.urls = newUrls
	for _, event = range events {
		l.eventCh <- event
	}
}

func (l *consulListener) Next() (*registry.ServiceEvent, error) {
	select {
	case event := <-l.eventCh:
		return event, nil
	case err := <-l.errCh:
		return nil, err
	}
}

func (l *consulListener) Close() {
	l.running = false
	l.plan.Stop()
	close(l.eventCh)
	close(l.errCh)
	l.wg.Wait()
}
