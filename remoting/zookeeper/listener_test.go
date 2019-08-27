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

package zookeeper

import (
	"sync"
	"testing"
	"time"
)
import (
	"github.com/samuel/go-zookeeper/zk"
	"github.com/stretchr/testify/assert"
)
import (
	"github.com/apache/dubbo-go/common/logger"
	"github.com/apache/dubbo-go/remoting"
)

func initZkData(t *testing.T) (*zk.TestCluster, *ZookeeperClient, <-chan zk.Event) {
	ts, client, event, err := NewMockZookeeperClient("test", 15*time.Second)
	assert.NoError(t, err)

	data := `
	dubbo.consumer.request_timeout=5s
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

	err = client.Create("/dubbo/dubbo.properties")
	assert.NoError(t, err)

	_, err = client.Conn.Set("/dubbo/dubbo.properties", []byte(data), 0)
	assert.NoError(t, err)

	return ts, client, event
}
func TestListener(t *testing.T) {
	changedData := `
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
	var wait sync.WaitGroup
	ts, client, event := initZkData(t)
	defer ts.Stop()
	client.Wait.Add(1)
	wait.Add(1)
	go client.HandleZkEvent(event)
	listener := NewZkEventListener(client)
	dataListener := &mockDataListener{client: client, changedData: changedData, wait: &wait}
	listener.ListenServiceEvent("/dubbo", dataListener)

	_, err := client.Conn.Set("/dubbo/dubbo.properties", []byte(changedData), 1)
	assert.NoError(t, err)
	wait.Wait()
	assert.Equal(t, changedData, dataListener.eventList[1].Content)
	client.Close()

}

type mockDataListener struct {
	eventList   []remoting.Event
	client      *ZookeeperClient
	changedData string
	wait        *sync.WaitGroup
}

func (m *mockDataListener) DataChange(eventType remoting.Event) bool {
	logger.Info(eventType)
	m.eventList = append(m.eventList, eventType)
	if eventType.Content == m.changedData {
		m.wait.Done()
		m.client.Close()

	}
	return true
}
