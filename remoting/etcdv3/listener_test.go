package etcdv3

import (
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

func (suite *ClientTestSuite) TestListener() {

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

	c := suite.client
	t := suite.T()

	listener := NewEventListener(c)
	dataListener := &mockDataListener{client: c, changedData: changedData, rc: make(chan remoting.Event)}
	listener.ListenServiceEvent("/dubbo", dataListener)

	// NOTICE:  direct listen will lose create msg
	time.Sleep(time.Second)
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
