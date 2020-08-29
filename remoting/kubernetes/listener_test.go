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
	"testing"
	"time"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"github.com/apache/dubbo-go/remoting"
)

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

type mockDataListener struct {
	eventList   []remoting.Event
	client      *Client
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

func TestListener(t *testing.T) {

	var tests = []struct {
		input struct {
			k string
			v string
		}
	}{
		{input: struct {
			k string
			v string
		}{k: "/dubbo", v: changedData}},
	}

	c := getTestClient(t)
	defer c.Close()

	listener := NewEventListener(c)
	dataListener := &mockDataListener{client: c, changedData: changedData, rc: make(chan remoting.Event)}
	listener.ListenServiceEvent("/dubbo", dataListener)
	time.Sleep(1e9)

	for _, tc := range tests {

		k := tc.input.k
		v := tc.input.v
		if err := c.Create(k, v); err != nil {
			t.Fatal(err)
		}

	}
	msg := <-dataListener.rc
	assert.Equal(t, changedData, msg.Content)
}
