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
	"net/url"
	"strings"
	"sync"
	"testing"
)

import (
	apolloconstant "github.com/apolloconfig/agollo/v4/constant"
	"github.com/apolloconfig/agollo/v4/extension"

	"github.com/knadh/koanf"
	"github.com/knadh/koanf/parsers/json"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/rawbytes"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/config_center"
	"dubbo.apache.org/dubbo-go/v3/global"
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
	require.NoError(t, err)

	koan := koanf.New(".")
	err = koan.Load(rawbytes.Provider([]byte(configs)), yaml.Parser())
	require.NoError(t, err)

	appCfg := &global.ApplicationConfig{}
	err = koan.UnmarshalWithConf(constant.ApplicationConfigPrefix, appCfg, koanf.UnmarshalConf{Tag: "yaml"})
	require.NoError(t, err)

	assert.Equal(t, "demo-server", appCfg.Name)
}

func TestGetJsonConfig(t *testing.T) {
	configuration := initMockApollo(t)
	configs, err := configuration.GetProperties(mockJsonNamespace, config_center.WithGroup("dubbo"))
	require.NoError(t, err)

	koan := koanf.New(":")
	err = koan.Load(rawbytes.Provider([]byte(configs)), json.Parser())
	require.NoError(t, err)

	appCfg := &global.ApplicationConfig{}
	err = koan.UnmarshalWithConf("dubbo:application", appCfg, koanf.UnmarshalConf{Tag: "json"})
	require.NoError(t, err)

	assert.Equal(t, "demo-server", appCfg.Name)
}

func TestGetConfigItem(t *testing.T) {
	configuration := initMockApollo(t)
	appName, err := configuration.GetInternalProperty(constant.ApplicationConfigPrefix + ".name")
	require.NoError(t, err)
	assert.Equal(t, "demo-server", appName)
}

func initMockApollo(t *testing.T) *apolloConfiguration {
	// Register the YAML format parser with concurrent safety.
	extension.AddFormatParser(apolloconstant.YAML, &Parser{})
	extension.AddFormatParser(apolloconstant.YML, &Parser{})

	params := url.Values{}
	params.Set(constant.ConfigNamespaceKey, "mockDubbogo.yaml")
	params.Set(constant.ConfigAppIDKey, "testApplication_yang")
	params.Set(constant.ConfigClusterKey, "dev")
	params.Set(constant.ConfigBackupConfigKey, "false")

	apollo := initApollo()
	apolloUrl := strings.ReplaceAll(apollo.URL, "http", "apollo")
	apolloCfgURL, err := common.NewURL(apolloUrl, common.WithParams(params))
	require.NoError(t, err)
	configuration, err := newApolloConfiguration(apolloCfgURL)
	require.NoError(t, err)
	return configuration
}

func TestGetAddressWithProtocolPrefix(t *testing.T) {
	tests := []struct {
		name   string
		rawURL string
		want   string
	}{
		{
			name:   "with context path",
			rawURL: "apollo://127.0.0.1:8080/config",
			want:   "http://127.0.0.1:8080/config",
		},
		{
			name:   "without context path",
			rawURL: "apollo://127.0.0.1:8080",
			want:   "http://127.0.0.1:8080",
		},
		{
			name:   "multiple addresses with context path",
			rawURL: "apollo://127.0.0.1:8080,192.168.1.1:8080/config",
			want:   "http://127.0.0.1:8080/config,http://192.168.1.1:8080/config",
		},
		{
			name:   "trailing slash without context path",
			rawURL: "apollo://127.0.0.1:8080/",
			want:   "http://127.0.0.1:8080",
		},
		{
			name:   "all slashes path treated as empty",
			rawURL: "apollo://127.0.0.1:8080///",
			want:   "http://127.0.0.1:8080",
		},
		{
			name:   "context path with trailing slash",
			rawURL: "apollo://127.0.0.1:8080/config/",
			want:   "http://127.0.0.1:8080/config",
		},
		{
			name:   "multiple addresses with trailing slash and no context path",
			rawURL: "apollo://127.0.0.1:8080,192.168.1.1:8080/",
			want:   "http://127.0.0.1:8080,http://192.168.1.1:8080",
		},
	}

	config := &apolloConfiguration{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			apolloCfgURL, err := common.NewURL(tt.rawURL, common.WithProtocol("apollo"))
			require.NoError(t, err)
			assert.Equal(t, tt.want, config.getAddressWithProtocolPrefix(apolloCfgURL))
		})
	}
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
	assert.Positive(t, listener.count)

	// test remove
	apollo.RemoveListener(mockNamespace, listener)
	listenerCount := 0
	apollo.listeners.Range(func(_, value any) bool {
		apolloListener := value.(*apolloListener)
		for e := range apolloListener.listeners {
			t.Logf("listener:%v", e)
			listenerCount++
		}
		return true
	})
	assert.Equal(t, 0, listenerCount)
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

// custom parser with concurrent safety
type Parser struct {
}

func (d *Parser) Parse(configContent any) (map[string]any, error) {
	content, ok := configContent.(string)
	if !ok {
		return nil, nil
	}
	if content == "" {
		return nil, nil
	}
	koan := koanf.New(".")
	if err := koan.Load(rawbytes.Provider([]byte(content)), yaml.Parser()); err != nil {
		return nil, err
	}

	return koan.All(), nil
}
