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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/config_center"
	"dubbo.apache.org/dubbo-go/v3/remoting"
)

type MockDataListener struct {
	events []*config_center.ConfigChangeEvent
}

func (m *MockDataListener) Process(configType *config_center.ConfigChangeEvent) {
	m.events = append(m.events, configType)
}

func TestDataListenerDataChangeDispatchesMatchingService(t *testing.T) {
	subscribeURL, err := common.NewURL("consumer://127.0.0.1:20000/org.apache.DemoService?group=g&version=1.0.0")
	require.NoError(t, err)
	providerURL, err := common.NewURL("dubbo://127.0.0.1:20001/org.apache.DemoService?group=g&version=1.0.0")
	require.NoError(t, err)
	listener := &MockDataListener{}
	dataListener := NewRegistryDataListener()
	dataListener.SubscribeURL(subscribeURL, listener)

	matched := dataListener.DataChange(remoting.Event{
		Path:   "/dubbo/org.apache.DemoService/providers/" + providerURL.String(),
		Action: remoting.EventTypeAdd,
	})

	require.True(t, matched)
	require.Len(t, listener.events, 1)
	assert.Equal(t, remoting.EventTypeAdd, listener.events[0].ConfigType)
	assert.Equal(t, providerURL.String(), listener.events[0].Value.(*common.URL).String())
}

func TestDataListenerDataChangeIgnoresUnmatchedService(t *testing.T) {
	subscribeURL, err := common.NewURL("consumer://127.0.0.1:20000/org.apache.OtherService")
	require.NoError(t, err)
	providerURL, err := common.NewURL("dubbo://127.0.0.1:20001/org.apache.DemoService")
	require.NoError(t, err)
	listener := &MockDataListener{}
	dataListener := NewRegistryDataListener()
	dataListener.SubscribeURL(subscribeURL, listener)

	matched := dataListener.DataChange(remoting.Event{
		Path:   "/dubbo/org.apache.DemoService/providers/" + providerURL.String(),
		Action: remoting.EventTypeAdd,
	})

	assert.False(t, matched)
	assert.Empty(t, listener.events)
}

func TestDataListenerUnSubscribeStopsDispatch(t *testing.T) {
	serviceURL, err := common.NewURL("consumer://127.0.0.1:20000/org.apache.DemoService")
	require.NoError(t, err)
	listener := &MockDataListener{}
	dataListener := NewRegistryDataListener()
	dataListener.SubscribeURL(serviceURL, listener)

	removed := dataListener.UnSubscribeURL(serviceURL)
	require.Same(t, listener, removed)
	assert.Empty(t, dataListener.subscribed)
}

func TestDataListenerCloseStopsDispatch(t *testing.T) {
	serviceURL, err := common.NewURL("consumer://127.0.0.1:20000/org.apache.DemoService")
	require.NoError(t, err)
	listener := &MockDataListener{}
	dataListener := NewRegistryDataListener()
	dataListener.SubscribeURL(serviceURL, listener)

	dataListener.Close()
	dataListener.SubscribeURL(serviceURL, listener)
	removed := dataListener.UnSubscribeURL(serviceURL)

	assert.Nil(t, removed)
	assert.False(t, dataListener.DataChange(remoting.Event{
		Path:   "/dubbo/org.apache.DemoService/providers/" + serviceURL.String(),
		Action: remoting.EventTypeAdd,
	}))
	assert.Empty(t, listener.events)
}

func TestConfigurationListenerProcessAndNext(t *testing.T) {
	reg := newTestRegistry(t)
	serviceURL, err := common.NewURL("dubbo://127.0.0.1:20000/org.apache.DemoService")
	require.NoError(t, err)
	listener := NewConfigurationListener(reg, serviceURL)
	defer listener.Close()

	listener.Process(&config_center.ConfigChangeEvent{
		Key:        "key",
		Value:      serviceURL,
		ConfigType: remoting.EventTypeAdd,
	})

	event, err := listener.Next()
	require.NoError(t, err)
	assert.Equal(t, remoting.EventTypeAdd, event.Action)
	assert.Same(t, serviceURL, event.Service)
}

/*
func Test_dataListener_DataChange(t *testing.T) {
	tests := []struct {
		name   string
		fields dataListenerFields
		args   args
		want   bool
	}{
		{
			name: "test",
			fields: dataListenerFields{
				interestedURL: nil,
				listener:      &MockDataListener{},
			},
			args: args{
				eventType: remoting.Event{
					Path: "com.ikurento.user.UserProvider/providers/jsonrpc%3A%2F%2F127.0.0.1%3A20001%2Fcom.ikurento.user.UserProvider%3Fanyhost%3Dtrue%26app.version%3D0.0.1%26application%3DBDTService%26category%3Dproviders%26cluster%3Dfailover%26dubbo%3Ddubbo-provider-golang-2.6.0%26environment%3Ddev%26group%3D%26interface%3Dcom.ikurento.user.UserProvider%26ip%3D10.32.20.124%26loadbalance%3Drandom%26methods.GetUser.loadbalance%3Drandom%26methods.GetUser.retries%3D1%26methods.GetUser.weight%3D0%26module%3Ddubbogo%2Buser-info%2Bserver%26name%3DBDTService%26organization%3Dikurento.com%26owner%3DZX%26pid%3D74500%26retries%3D0%26service.filter%3Decho%26side%3Dprovider%26timestamp%3D1560155407%26version%3D%26warmup%3D100",
				},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := newDataListener(tt.fields)
			if got := l.DataChange(tt.args.eventType); got != tt.want {
				t.Errorf("DataChange() = %v, want %v", got, tt.want)
			}
		})
	}
}

*/
