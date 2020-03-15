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

package kubernetes

import (
	"context"
	"strconv"
	"strings"
	"sync"
)

import (
	perrors "github.com/pkg/errors"
)

var (
	ErrWatcherSetAlreadyStopped = perrors.New("the watcher-set already be stopped")
	ErrKVPairNotFound           = perrors.New("k/v pair not found")
)

const (
	defaultWatcherChanSize = 100
)

type eventType int

const (
	Create eventType = iota
	Update
	Delete
)

func (e eventType) String() string {

	switch e {
	case Create:
		return "CREATE"
	case Update:
		return "UPDATE"
	case Delete:
		return "DELETE"
	default:
		return "UNKNOWN"
	}
}

// WatcherEvent
// watch event is element in watcherSet
type WatcherEvent struct {
	// event-type
	EventType eventType `json:"-"`
	// the dubbo-go should consume the key
	Key string `json:"k"`
	// the dubbo-go should consume the value
	Value string `json:"v"`
}

// Watchable WatcherSet
type WatcherSet interface {

	// put the watch event to the watch set
	Put(object *WatcherEvent) error
	// if prefix is false,
	// the len([]*WatcherEvent) == 1
	Get(key string, prefix bool) ([]*WatcherEvent, error)
	// watch the spec key or key prefix
	Watch(key string, prefix bool) (Watcher, error)
	// check the watcher set status
	Done() <-chan struct{}
}

// Watcher
type Watcher interface {
	// the watcher's id
	ID() string
	// result stream
	ResultChan() <-chan *WatcherEvent
	// Stop the watcher
	stop()
	// check the watcher status
	done() <-chan struct{}
}

// the watch set implement
type watcherSetImpl struct {

	// Client's ctx, client die, the watch set will die too
	ctx context.Context

	// protect watcher-set and watchers
	lock sync.RWMutex

	// the key is dubbo-go interest meta
	cache map[string]*WatcherEvent

	currentWatcherId uint64
	watchers         map[uint64]*watcher
}

// closeWatchers
// when the watcher-set was closed
func (s *watcherSetImpl) closeWatchers() {

	select {
	case <-s.ctx.Done():

		// parent ctx be canceled, close the watch-set's watchers
		s.lock.Lock()
		watchers := s.watchers
		s.lock.Unlock()

		for _, w := range watchers {
			// stop data stream
			// close(w.ch)
			// stop watcher
			w.stop()
		}
	}
}

// Watch
// watch on spec key, with or without prefix
func (s *watcherSetImpl) Watch(key string, prefix bool) (Watcher, error) {
	return s.addWatcher(key, prefix)
}

// Done
// get the watcher-set status
func (s *watcherSetImpl) Done() <-chan struct{} {
	return s.ctx.Done()
}

// Put
// put the watch event to watcher-set
func (s *watcherSetImpl) Put(watcherEvent *WatcherEvent) error {

	sendMsg := func(object *WatcherEvent, w *watcher) {

		select {
		case <-w.done():
			// the watcher already stop
		case w.ch <- object:
			// block send the msg
		}
	}

	s.lock.Lock()
	defer s.lock.Unlock()

	if err := s.valid(); err != nil {
		return err
	}

	// put to watcher-set
	if watcherEvent.EventType == Delete {
		delete(s.cache, watcherEvent.Key)
	} else {

		old, ok := s.cache[watcherEvent.Key]
		if ok {
			if old.Value == watcherEvent.Value {
				// already have this k/v pair
				return nil
			}
		}

		// refresh the watcherEvent
		s.cache[watcherEvent.Key] = watcherEvent
	}

	// notify watcher
	for _, w := range s.watchers {

		if !strings.Contains(watcherEvent.Key, w.interested.key) {
			//  this watcher no interest in this element
			continue
		}

		if !w.interested.prefix {
			if watcherEvent.Key == w.interested.key {
				go sendMsg(watcherEvent, w)
			}
			// not interest
			continue
		}
		go sendMsg(watcherEvent, w)
	}
	return nil
}

// valid
func (s *watcherSetImpl) valid() error {
	select {
	case <-s.ctx.Done():
		return ErrWatcherSetAlreadyStopped
	default:
		return nil
	}
}

// addWatcher
func (s *watcherSetImpl) addWatcher(key string, prefix bool) (Watcher, error) {

	if err := s.valid(); err != nil {
		return nil, err
	}

	s.lock.Lock()
	defer s.lock.Unlock()

	// increase the watcher-id
	s.currentWatcherId++

	w := &watcher{
		id:         s.currentWatcherId,
		watcherSet: s,
		interested: struct {
			key    string
			prefix bool
		}{key: key, prefix: prefix},
		ch:   make(chan *WatcherEvent, defaultWatcherChanSize),
		exit: make(chan struct{}),
	}
	s.watchers[s.currentWatcherId] = w
	return w, nil
}

// Get
// get elements from watcher-set
func (s *watcherSetImpl) Get(key string, prefix bool) ([]*WatcherEvent, error) {

	s.lock.RLock()
	defer s.lock.RUnlock()

	if err := s.valid(); err != nil {
		return nil, err
	}

	if !prefix {
		for k, v := range s.cache {
			if k == key {
				return []*WatcherEvent{v}, nil
			}
		}
		// object
		return nil, ErrKVPairNotFound
	}

	var out []*WatcherEvent

	for k, v := range s.cache {
		if strings.Contains(k, key) {
			out = append(out, v)
		}
	}

	if len(out) == 0 {
		return nil, ErrKVPairNotFound
	}

	return out, nil
}

// the watcher-set watcher
type watcher struct {
	id uint64

	// the underlay watcherSet
	watcherSet *watcherSetImpl

	// the interest topic
	interested struct {
		key    string
		prefix bool
	}
	ch chan *WatcherEvent

	closeOnce sync.Once
	exit      chan struct{}
}

// ResultChan
func (w *watcher) ResultChan() <-chan *WatcherEvent {
	return w.ch
}

// ID
// the watcher's id
func (w *watcher) ID() string {
	return strconv.FormatUint(w.id, 10)
}

// stop
// stop the watcher
func (w *watcher) stop() {

	// double close will panic
	w.closeOnce.Do(func() {
		close(w.exit)
	})
}

// done
// check watcher status
func (w *watcher) done() <-chan struct{} {
	return w.exit
}

// newWatcherSet
// new watcher set from parent context
func newWatcherSet(ctx context.Context) WatcherSet {
	s := &watcherSetImpl{
		ctx:      ctx,
		cache:    map[string]*WatcherEvent{},
		watchers: map[uint64]*watcher{},
	}
	go s.closeWatchers()
	return s
}
