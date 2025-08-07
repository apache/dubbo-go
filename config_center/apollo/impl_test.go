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

package apollo

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

import (
	"github.com/knadh/koanf"
	"github.com/knadh/koanf/parsers/json"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/rawbytes"

	"github.com/stretchr/testify/assert"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/config"
	"dubbo.apache.org/dubbo-go/v3/config_center"
	"dubbo.apache.org/dubbo-go/v3/remoting"
)

const (
	mockAppId         = "testApplication_yang"
	mockCluster       = "dev"
	mockNamespace     = "mockDubbogo.yaml"
	mockJsonNamespace = "mockDubbogo.json"
	mockNotifyRes     = `[{
	"namespaceName": "mockDubbogo.yaml",
	"notificationId": 53050,
	"messages": {
		"details": {
			"testApplication_yang+default+mockDubbogo": 53050
		}
	}
}]`
	mockServiceConfigRes = `[{
	"appName": "APOLLO-CONFIGSERVICE",
	"instanceId": "instance-300408ep:apollo-configservice:8080",
	"homepageUrl": "http://localhost:8080"
}]`

	mockConfigCacheRes = `{"content":"dubbo:\n  application:\n     name: \"demo-server\"\n     version: \"2.0\"\n"}`

	mockConfigCacheJsonRes = `{
    "content": "{\n    \"dubbo\": {\n        \"application\": {\n            \"name\": \"demo-server\",\n            \"version\": \"2.0\"\n        },\n        \"otel\": {\n            \"trace\": {\n                \"enable\": true,\n                \"sample-ratio\": 1.123\n            }\n        }\n    }\n}"
}`
)

var mockConfigRes = `{
	"appId": "testApplication_yang",
	"cluster": "default",
	"namespaceName": "mockDubbogo.yaml",
	"configurations":{
		"content":"dubbo:\n  application:\n     name: \"demo-server\"\n     version: \"2.0\"\n"
    },
	"releaseKey": "20191104105242-0f13805d89f834a4"
}`

func initApollo() *httptest.Server {
	return runMockConfigServer()
}

func configCacheResponse(rw http.ResponseWriter, _ *http.Request) {
	result := mockConfigCacheRes
	fmt.Fprintf(rw, "%s", result)
}

func configResponse(rw http.ResponseWriter, _ *http.Request) {
	result := mockConfigRes
	fmt.Fprintf(rw, "%s", result)
}

func configCacheJsonResponse(rw http.ResponseWriter, _ *http.Request) {
	result := mockConfigCacheJsonRes
	fmt.Fprintf(rw, "%s", result)
}

func notifyResponse(rw http.ResponseWriter, req *http.Request) {
	result := mockNotifyRes
	fmt.Fprintf(rw, "%s", result)
}

func serviceConfigResponse(rw http.ResponseWriter, _ *http.Request) {
	result := mockServiceConfigRes
	fmt.Fprintf(rw, "%s", result)
}

// run mock config server
func runMockConfigServer() *httptest.Server {
	uriHandlerMap := make(map[string]func(http.ResponseWriter, *http.Request))

	configCacheUri := fmt.Sprintf("/configfiles/json/%s/%s/%s", mockAppId, mockCluster, mockNamespace)
	uriHandlerMap[configCacheUri] = configCacheResponse
	configUri := fmt.Sprintf("/configs/%s/%s/%s", mockAppId, mockCluster, mockNamespace)
	uriHandlerMap[configUri] = configResponse

	configCacheJsonUri := fmt.Sprintf("/configfiles/json/%s/%s/%s", mockAppId, mockCluster, mockJsonNamespace)
	uriHandlerMap[configCacheJsonUri] = configCacheJsonResponse

	uriHandlerMap["/notifications/v2"] = notifyResponse
	uriHandlerMap["/services/config"] = serviceConfigResponse

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		uri := r.RequestURI
		for path, handler := range uriHandlerMap {
			if strings.HasPrefix(uri, path) {
				handler(w, r)
				break
			}
		}
	}))

	return ts
}

func TestGetConfig(t *testing.T) {
	configuration := initMockApollo(t)
	configs, err := configuration.GetProperties(mockNamespace, config_center.WithGroup("dubbo"))
	assert.NoError(t, err)
	koan := koanf.New(".")
	err = koan.Load(rawbytes.Provider([]byte(configs)), yaml.Parser())
	assert.NoError(t, err)
	rc := &config.RootConfig{}
	err = koan.UnmarshalWithConf(rc.Prefix(), rc, koanf.UnmarshalConf{Tag: "yaml"})
	assert.NoError(t, err)

	assert.Equal(t, "demo-server", rc.Application.Name)
}

func TestGetJsonConfig(t *testing.T) {
	configuration := initMockApollo(t)
	configs, err := configuration.GetProperties(mockJsonNamespace, config_center.WithGroup("dubbo"))
	assert.NoError(t, err)
	koan := koanf.New(":")
	err = koan.Load(rawbytes.Provider([]byte(configs)), json.Parser())
	assert.NoError(t, err)
	rc := &config.RootConfig{}
	err = koan.UnmarshalWithConf(rc.Prefix(), rc, koanf.UnmarshalConf{Tag: "json"})
	assert.NoError(t, err)

	assert.Equal(t, "demo-server", rc.Application.Name)
}

func TestGetConfigItem(t *testing.T) {
	configuration := initMockApollo(t)
	appName, err := configuration.GetInternalProperty("dubbo.application.name")
	assert.NoError(t, err)
	assert.Equal(t, "demo-server", appName)
}

func initMockApollo(t *testing.T) *apolloConfiguration {
	c := &config.RootConfig{ConfigCenter: &config.CenterConfig{
		Protocol:  "apollo",
		Address:   "localhost:8080",
		AppID:     "testApplication_yang",
		Cluster:   "dev",
		Namespace: "mockDubbogo.yaml",
	}}
	apollo := initApollo()
	apolloUrl := strings.ReplaceAll(apollo.URL, "http", "apollo")
	url, err := common.NewURL(apolloUrl, common.WithParams(c.ConfigCenter.GetUrlMap()))
	assert.NoError(t, err)
	configuration, err := newApolloConfiguration(url)
	assert.NoError(t, err)
	return configuration
}

func TestListener(t *testing.T) {
	listener := &apolloDataListener{}
	listener.wg.Add(2)
	apollo := initMockApollo(t)
	mockConfigRes = `{
	"appId": "testApplication_yang",
	"cluster": "default",
	"namespaceName": "mockDubbogo.yaml",
	"configurations": {
		"registries.hangzhouzk.username": "11111"
	},
	"releaseKey": "20191104105242-0f13805d89f834a4"
}`
	// test add
	apollo.AddListener(mockNamespace, listener)
	listener.wg.Wait()
	assert.Equal(t, "mockDubbogo.yaml", listener.event)
	assert.Greater(t, listener.count, 0)

	// test remove
	apollo.RemoveListener(mockNamespace, listener)
	listenerCount := 0
	apollo.listeners.Range(func(_, value interface{}) bool {
		apolloListener := value.(*apolloListener)
		for e := range apolloListener.listeners {
			t.Logf("listener:%v", e)
			listenerCount++
		}
		return true
	})
	assert.Equal(t, listenerCount, 0)
}

type apolloDataListener struct {
	wg    sync.WaitGroup
	count int
	event string
}

func (l *apolloDataListener) Process(configType *config_center.ConfigChangeEvent) {
	if configType.ConfigType != remoting.EventTypeUpdate {
		return
	}
	l.wg.Done()
	l.count++
	l.event = configType.Key
}
