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

package nacos

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/config_center"
	"github.com/apache/dubbo-go/config_center/parser"
)

// run mock config server
func runMockConfigServer(configHandler func(http.ResponseWriter, *http.Request),
	configListenHandler func(http.ResponseWriter, *http.Request)) *httptest.Server {
	uriHandlerMap := make(map[string]func(http.ResponseWriter, *http.Request), 0)

	uriHandlerMap["/nacos/v1/cs/configs"] = configHandler
	uriHandlerMap["/nacos/v1/cs/configs/listener"] = configListenHandler

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		uri := r.RequestURI
		for path, handler := range uriHandlerMap {
			if uri == path {
				handler(w, r)
				break
			}
		}
	}))

	return ts
}

func mockCommonNacosServer() *httptest.Server {
	return runMockConfigServer(func(writer http.ResponseWriter, _ *http.Request) {
		data := "true"
		fmt.Fprintf(writer, "%s", data)
	}, func(writer http.ResponseWriter, _ *http.Request) {
		data := `dubbo.properties%02dubbo%02dubbo.service.com.ikurento.user.UserProvider.cluster=failback`
		fmt.Fprintf(writer, "%s", data)
	})
}

func initNacosData(t *testing.T) (*nacosDynamicConfiguration, error) {
	server := mockCommonNacosServer()
	nacosURL := strings.ReplaceAll(server.URL, "http", "registry")
	regurl, _ := common.NewURL(nacosURL)
	factory := &nacosDynamicConfigurationFactory{}
	nacosConfiguration, err := factory.GetDynamicConfiguration(regurl)
	assert.NoError(t, err)

	nacosConfiguration.SetParser(&parser.DefaultConfigurationParser{})

	return nacosConfiguration.(*nacosDynamicConfiguration), err
}

func TestGetConfig(t *testing.T) {
	nacos, err := initNacosData(t)
	assert.NoError(t, err)
	configs, err := nacos.GetProperties("dubbo.properties", config_center.WithGroup("dubbo"))
	_, err = nacos.Parser().Parse(configs)
	assert.NoError(t, err)
}

func TestNacosDynamicConfiguration_GetConfigKeysByGroup(t *testing.T) {
	data := `
{
    "PageItems": [
        {
            "dataId": "application"
        }
    ]
}
`
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(data))
	}))

	nacosURL := strings.ReplaceAll(ts.URL, "http", "registry")
	regurl, _ := common.NewURL(nacosURL)
	nacosConfiguration, err := newNacosDynamicConfiguration(regurl)
	assert.NoError(t, err)

	nacosConfiguration.SetParser(&parser.DefaultConfigurationParser{})

	configs, err := nacosConfiguration.GetConfigKeysByGroup("dubbo")
	assert.Nil(t, err)
	assert.Equal(t, 1, configs.Size())
	assert.True(t, configs.Contains("application"))

}

func TestNacosDynamicConfigurationPublishConfig(t *testing.T) {
	nacos, err := initNacosData(t)
	assert.Nil(t, err)
	key := "myKey"
	group := "/custom/a/b"
	value := "MyValue"
	err = nacos.PublishConfig(key, group, value)
	assert.Nil(t, err)
}

func TestAddListener(t *testing.T) {
	nacos, err := initNacosData(t)
	assert.NoError(t, err)
	listener := &mockDataListener{}
	time.Sleep(time.Second * 2)
	nacos.AddListener("dubbo.properties", listener)
}

func TestRemoveListener(_ *testing.T) {
	//TODO not supported in current go_nacos_sdk version
}

type mockDataListener struct {
	wg    sync.WaitGroup
	event string
}

func (l *mockDataListener) Process(configType *config_center.ConfigChangeEvent) {
	l.wg.Done()
	l.event = configType.Key
}
