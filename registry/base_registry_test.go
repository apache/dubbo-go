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

package registry

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

import (
	"github.com/stretchr/testify/require"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
)

type baseRegistryTestFacade struct {
	BaseRegistry

	subscribeCalls atomic.Int32
	firstSubscribe chan struct{}
	listener       Listener
	subscribeErr   error
}

func newBaseRegistryTestFacade(listener Listener, subscribeErr error) *baseRegistryTestFacade {
	facade := &baseRegistryTestFacade{
		firstSubscribe: make(chan struct{}),
		listener:       listener,
	}
	facade.InitBaseRegistry(common.NewURLWithOptions(), facade)
	facade.subscribeErr = subscribeErr
	return facade
}

func (f *baseRegistryTestFacade) DoSubscribe(*common.URL) (Listener, error) {
	if f.subscribeCalls.Add(1) == 1 {
		close(f.firstSubscribe)
	}
	return f.listener, f.subscribeErr
}

func (f *baseRegistryTestFacade) DoUnsubscribe(*common.URL) (Listener, error) {
	return nil, nil
}

func (f *baseRegistryTestFacade) CreatePath(string) error {
	return nil
}

func (f *baseRegistryTestFacade) DoRegister(string, string) error {
	return nil
}

func (f *baseRegistryTestFacade) DoUnregister(string, string) error {
	return nil
}

func (f *baseRegistryTestFacade) CloseAndNilClient() {}

func (f *baseRegistryTestFacade) CloseListener() {
	if f.listener != nil {
		f.listener.Close()
	}
}

func (f *baseRegistryTestFacade) InitListeners() {}

type baseRegistryTestListener struct {
	closed    chan struct{}
	closeOnce sync.Once
}

func (l *baseRegistryTestListener) Next() (*ServiceEvent, error) {
	<-l.closed
	return nil, errors.New("listener closed")
}

func (l *baseRegistryTestListener) Close() {
	l.closeOnce.Do(func() { close(l.closed) })
}

type gatedRegistryTestListener struct {
	release   chan struct{}
	nextReady chan struct{}
	closeOnce sync.Once
}

func (l *gatedRegistryTestListener) Next() (*ServiceEvent, error) {
	close(l.nextReady)
	<-l.release
	return &ServiceEvent{}, nil
}

func (l *gatedRegistryTestListener) Close() {
	l.closeOnce.Do(func() {})
}

type baseRegistryTestNotify struct {
	notified atomic.Int32
}

func (n *baseRegistryTestNotify) Notify(*ServiceEvent) {
	n.notified.Add(1)
}

func (*baseRegistryTestNotify) NotifyAll([]*ServiceEvent, func()) {}

func TestBaseRegistrySubscribeDestroyInterruptsRetryDelay(t *testing.T) {
	listener := &baseRegistryTestListener{closed: make(chan struct{})}
	facade := newBaseRegistryTestFacade(listener, errors.New("subscribe failed"))

	subscribeDone := make(chan error, 1)
	go func() {
		subscribeDone <- facade.Subscribe(common.NewURLWithOptions(), &baseRegistryTestNotify{})
	}()

	select {
	case <-facade.firstSubscribe:
	case <-time.After(time.Second):
		t.Fatal("Subscribe did not attempt the initial subscription")
	}

	start := time.Now()
	facade.Destroy()

	select {
	case err := <-subscribeDone:
		require.Error(t, err)
	case <-time.After(time.Second):
		t.Fatal("Subscribe remained blocked after Destroy")
	}
	require.Less(t, time.Since(start), time.Second)
}

func TestBaseRegistrySubscribeDoesNotNotifyAfterDestroy(t *testing.T) {
	listener := &gatedRegistryTestListener{
		release:   make(chan struct{}),
		nextReady: make(chan struct{}),
	}
	facade := newBaseRegistryTestFacade(listener, nil)
	notify := &baseRegistryTestNotify{}
	subscribeDone := make(chan error, 1)
	go func() {
		subscribeDone <- facade.Subscribe(common.NewURLWithOptions(), notify)
	}()

	select {
	case <-listener.nextReady:
	case <-time.After(time.Second):
		t.Fatal("Subscribe did not enter listener.Next")
	}

	facade.Destroy()
	close(listener.release)

	select {
	case err := <-subscribeDone:
		require.Error(t, err)
	case <-time.After(time.Second):
		t.Fatal("Subscribe did not exit after Destroy")
	}
	require.Zero(t, notify.notified.Load())
}
