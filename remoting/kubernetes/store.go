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
	ErrStoreAlreadyStopped = perrors.New("the store already be stopped")
	ErrKVPairNotFound      = perrors.New("k/v pair not found")
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

// Object
// object is element in store
type Object struct {
	// event-type
	EventType eventType `json:"-"`
	// the dubbo-go should consume the key
	Key string `json:"k"`
	// the dubbo-go should consume the value
	Value string `json:"v"`
}

// Watchable Store
type Store interface {

	// put the object to the store
	Put(object *Object) error
	// if prefix is false,
	// the len([]*Object) == 1
	Get(key string, prefix bool) ([]*Object, error)
	// watch the spec key or key prefix
	Watch(key string, prefix bool) (Watcher, error)
	// check the store status
	Done() <-chan struct{}
}

// Stopped Watcher
type Watcher interface {
	// the watcher's id
	ID() string
	// result stream
	ResultChan() <-chan *Object
	// Stop the watcher
	stop()
	// check the watcher status
	done() <-chan struct{}
}

// the store
type storeImpl struct {

	// Client's ctx, client die, the store will die too
	ctx context.Context

	// protect store and watchers
	lock sync.RWMutex

	// the key is dubbo-go interest meta
	cache map[string]*Object

	currentWatcherId uint64
	watchers         map[uint64]*watcher
}

func (s *storeImpl) loop() {

	select {
	case <-s.ctx.Done():
		// parent ctx be canceled, close the store
		s.lock.Lock()
		defer s.lock.Unlock()
		for _, w := range s.watchers {
			// stop data stream
			close(w.ch)
			// stop watcher
			w.stop()
		}
	}
}

// Watch
// watch on spec key, with or without prefix
func (s *storeImpl) Watch(key string, prefix bool) (Watcher, error) {
	return s.addWatcher(key, prefix)
}

// Done
// get the store status
func (s *storeImpl) Done() <-chan struct{} {
	return s.ctx.Done()
}

// Put
// put the object to store
func (s *storeImpl) Put(object *Object) error {

	sendMsg := func(object *Object, w *watcher) {
		s.lock.Lock()
		defer s.lock.Unlock()
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

	// put to store
	if object.EventType == Delete {
		delete(s.cache, object.Key)
	} else {

		old, ok := s.cache[object.Key]
		if ok {
			if old.Value == object.Value {
				// already have this k/v pair
				return nil
			}
		}

		// refresh the object
		s.cache[object.Key] = object
	}

	// notify watcher
	for _, w := range s.watchers {

		if !strings.Contains(object.Key, w.interested.key) {
			//  this watcher no interest in this element
			continue
		}

		if !w.interested.prefix {
			if object.Key == w.interested.key {
				go sendMsg(object, w)
			}
			// not interest
			continue
		}
		go sendMsg(object, w)
	}
	return nil
}

// valid
// valid the client status
// NOTICE:
// should protected by lock
func (s *storeImpl) valid() error {
	select {
	case <-s.ctx.Done():
		return ErrStoreAlreadyStopped
	default:
		return nil
	}
}

// addWatcher
func (s *storeImpl) addWatcher(key string, prefix bool) (Watcher, error) {

	w := &watcher{
		store: s,
		interested: struct {
			key    string
			prefix bool
		}{key: key, prefix: prefix},
		ch:   make(chan *Object, defaultWatcherChanSize),
		exit: make(chan struct{}),
	}

	s.lock.Lock()
	defer s.lock.Unlock()

	if err := s.valid(); err != nil {
		return nil, err
	}

	s.watchers[s.currentWatcherId] = w
	w.id = s.currentWatcherId
	s.currentWatcherId = s.currentWatcherId + 1
	return w, nil
}

// Get
// get elements from cache
func (s *storeImpl) Get(key string, prefix bool) ([]*Object, error) {

	s.lock.RLock()
	defer s.lock.RUnlock()

	if err := s.valid(); err != nil {
		return nil, err
	}

	if !prefix {
		for k, v := range s.cache {
			if k == key {
				return []*Object{v}, nil
			}
		}
		// object
		return nil, ErrKVPairNotFound
	}

	var out []*Object

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

// the store watcher
type watcher struct {
	id uint64

	// the underlay store
	store *storeImpl

	// the interest topic
	interested struct {
		key    string
		prefix bool
	}
	ch chan *Object

	closeOnce sync.Once
	exit      chan struct{}
}

// ResultChan
func (w *watcher) ResultChan() <-chan *Object {
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

// newStore
// new store from parent context
func newStore(ctx context.Context) Store {
	s := &storeImpl{
		ctx:      ctx,
		cache:    map[string]*Object{},
		watchers: map[uint64]*watcher{},
	}
	go s.loop()
	return s
}
