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

package etcdv3

import (
	"net/url"
	"os"
	"testing"
	"time"
)

import (
	gxetcd "github.com/dubbogo/gost/database/kv/etcd/v3"
	"github.com/stretchr/testify/assert"
	"go.etcd.io/etcd/server/v3/embed"
)

import (
	"dubbo.apache.org/dubbo-go/v3/remoting"
)

const defaultEtcdV3WorkDir = "/tmp/default-dubbo-go-remote.etcd"

var changedData = `
	dubbo.consumer.request_timeout=3s
	dubbo.consumer.connect_timeout=5s
	dubbo.application.organization=ikurento.com
	dubbo.application.name=BDTService
	dubbo.application.module=dubbogo user-info server
	dubbo.application.version=0.0.1
	dubbo.application.owner=ZX
	dubbo.application.environment=dev
	dubbo.registries.hangzhouzk.protocol=zookeeper
	dubbo.registries.hangzhouzk.timeout=3s
	dubbo.registries.hangzhouzk.address=127.0.0.1:2181
	dubbo.registries.shanghaizk.protocol=zookeeper
	dubbo.registries.shanghaizk.timeout=3s
	dubbo.registries.shanghaizk.address=127.0.0.1:2182
	dubbo.service.com.ikurento.user.UserProvider.protocol=dubbo
	dubbo.service.com.ikurento.user.UserProvider.interface=com.ikurento.user.UserProvider
	dubbo.service.com.ikurento.user.UserProvider.loadbalance=random
	dubbo.service.com.ikurento.user.UserProvider.warmup=100
	dubbo.service.com.ikurento.user.UserProvider.cluster=failover
`

var etcd *embed.Etcd

func SetUpEtcdServer(t *testing.T) {
	var err error
	DefaultListenPeerURLs := "http://localhost:2382"
	DefaultListenClientURLs := "http://localhost:2381"
	lpurl, _ := url.Parse(DefaultListenPeerURLs)
	lcurl, _ := url.Parse(DefaultListenClientURLs)
	cfg := embed.NewConfig()
	cfg.LPUrls = []url.URL{*lpurl}
	cfg.LCUrls = []url.URL{*lcurl}
	cfg.Dir = defaultEtcdV3WorkDir
	etcd, err = embed.StartEtcd(cfg)
	if err != nil {
		t.Fatal(err)
	}
	select {
	case <-etcd.Server.ReadyNotify():
		t.Log("Server is ready!")
	case <-time.After(60 * time.Second):
		etcd.Server.Stop() // trigger a shutdown
		t.Logf("Server took too long to start!")
	}
}

func ClearEtcdServer(t *testing.T) {
	etcd.Close()
	if err := os.RemoveAll(defaultEtcdV3WorkDir); err != nil {
		t.Fail()
	}
}

func TestListener(t *testing.T) {

	tests := []struct {
		input struct {
			k string
			v string
		}
	}{
		{input: struct {
			k string
			v string
		}{k: "/dubbo/", v: changedData}},
	}
	SetUpEtcdServer(t)
	c, err := gxetcd.NewClient("test", []string{"localhost:2381"}, time.Second, 1)
	assert.NoError(t, err)

	listener := NewEventListener(c)
	dataListener := &mockDataListener{client: c, changedData: changedData, rc: make(chan remoting.Event)}
	listener.ListenServiceEvent("/dubbo/", dataListener)

	// NOTICE:  direct listen will lose create msg
	time.Sleep(time.Second)
	for _, tc := range tests {

		k := tc.input.k
		v := tc.input.v
		if err := c.Update(k, v); err != nil {
			t.Fatal(err)
		}

	}
	msg := <-dataListener.rc
	assert.Equal(t, changedData, msg.Content)
	ClearEtcdServer(t)
}

type mockDataListener struct {
	eventList   []remoting.Event
	client      *gxetcd.Client
	changedData string

	rc chan remoting.Event
}

func (m *mockDataListener) DataChange(eventType remoting.Event) bool {
	m.eventList = append(m.eventList, eventType)
	if eventType.Content == m.changedData {
		m.rc <- eventType
	}
	return true
}
