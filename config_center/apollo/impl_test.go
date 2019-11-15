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
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/config"
	"github.com/apache/dubbo-go/config_center"
	"github.com/apache/dubbo-go/config_center/parser"
	"github.com/apache/dubbo-go/remoting"
)

const (
	mockAppId     = "testApplication_yang"
	mockCluster   = "dev"
	mockNamespace = "mockDubbog.properties"
	mockNotifyRes = `[{
	"namespaceName": "mockDubbog.properties",
	"notificationId": 53050,
	"messages": {
		"details": {
			"testApplication_yang+default+mockDubbog": 53050
		}
	}
}]`
	mockServiceConfigRes = `[{
	"appName": "APOLLO-CONFIGSERVICE",
	"instanceId": "instance-300408ep:apollo-configservice:8080",
	"homepageUrl": "http://localhost:8080"
}]`
)

var (
	mockConfigRes = `{
	"appId": "testApplication_yang",
	"cluster": "default",
	"namespaceName": "mockDubbog.properties",
	"configurations": {
		"registries.hangzhouzk.username": "",
		"application.owner": "ZX",
		"registries.shanghaizk.username": "",
		"protocols.dubbo.ip": "127.0.0.1",
		"protocol_conf.dubbo.getty_session_param.tcp_write_timeout": "5s",
		"services.UserProvider.cluster": "failover",
		"application.module": "dubbogo user-info server",
		"services.UserProvider.interface": "com.ikurento.user.UserProvider",
		"protocol_conf.dubbo.getty_session_param.compress_encoding": "false",
		"registries.shanghaizk.address": "127.0.0.1:2182",
		"protocol_conf.dubbo.session_timeout": "20s",
		"registries.shanghaizk.timeout": "3s",
		"protocol_conf.dubbo.getty_session_param.keep_alive_period": "120s",
		"services.UserProvider.warmup": "100",
		"application.version": "0.0.1",
		"registries.hangzhouzk.protocol": "zookeeper",
		"registries.hangzhouzk.password": "",
		"protocols.dubbo.name": "dubbo",
		"protocol_conf.dubbo.getty_session_param.wait_timeout": "1s",
		"protocols.dubbo.port": "20000",
		"application_config.owner": "demo",
		"application_config.name": "demo",
		"application_config.version": "0.0.1",
		"application_config.environment": "dev",
		"protocol_conf.dubbo.getty_session_param.session_name": "server",
		"application.name": "BDTService",
		"registries.hangzhouzk.timeout": "3s",
		"protocol_conf.dubbo.getty_session_param.tcp_read_timeout": "1s",
		"services.UserProvider.loadbalance": "random",
		"protocol_conf.dubbo.session_number": "700",
		"protocol_conf.dubbo.getty_session_param.max_msg_len": "1024",
		"services.UserProvider.registry": "hangzhouzk",
		"application_config.module": "demo",
		"services.UserProvider.methods[0].name": "GetUser",
		"protocol_conf.dubbo.getty_session_param.tcp_no_delay": "true",
		"services.UserProvider.methods[0].retries": "1",
		"protocol_conf.dubbo.getty_session_param.tcp_w_buf_size": "65536",
		"protocol_conf.dubbo.getty_session_param.tcp_r_buf_size": "262144",
		"registries.shanghaizk.password": "",
		"application_config.organization": "demo",
		"registries.shanghaizk.protocol": "zookeeper",
		"protocol_conf.dubbo.getty_session_param.tcp_keep_alive": "true",
		"registries.hangzhouzk.address": "127.0.0.1:2181",
		"application.environment": "dev",
		"services.UserProvider.protocol": "dubbo",
		"application.organization": "ikurento.com",
		"protocol_conf.dubbo.getty_session_param.pkg_wq_size": "512",
		"services.UserProvider.methods[0].loadbalance": "random"
	},
	"releaseKey": "20191104105242-0f13805d89f834a4"
}`
)

func initApollo() *httptest.Server {
	handlerMap := make(map[string]func(http.ResponseWriter, *http.Request), 1)
	handlerMap[mockNamespace] = configResponse

	return runMockConfigServer(handlerMap, notifyResponse)
}

func configResponse(rw http.ResponseWriter, req *http.Request) {
	result := fmt.Sprintf(mockConfigRes)
	fmt.Fprintf(rw, "%s", result)
}

func notifyResponse(rw http.ResponseWriter, req *http.Request) {
	result := fmt.Sprintf(mockNotifyRes)
	fmt.Fprintf(rw, "%s", result)
}

func serviceConfigResponse(rw http.ResponseWriter, req *http.Request) {
	result := fmt.Sprintf(mockServiceConfigRes)
	fmt.Fprintf(rw, "%s", result)
}

//run mock config server
func runMockConfigServer(handlerMap map[string]func(http.ResponseWriter, *http.Request),
	notifyHandler func(http.ResponseWriter, *http.Request)) *httptest.Server {
	uriHandlerMap := make(map[string]func(http.ResponseWriter, *http.Request), 0)
	for namespace, handler := range handlerMap {
		uri := fmt.Sprintf("/configs/%s/%s/%s", mockAppId, mockCluster, namespace)
		uriHandlerMap[uri] = handler
	}
	uriHandlerMap["/notifications/v2"] = notifyHandler
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

func Test_GetConfig(t *testing.T) {
	configuration := initMockApollo(t)
	configs, err := configuration.GetConfig(mockNamespace, config_center.WithGroup("dubbo"))
	assert.NoError(t, err)
	configuration.SetParser(&parser.DefaultConfigurationParser{})
	mapContent, err := configuration.Parser().Parse(configs)
	assert.NoError(t, err)
	assert.Equal(t, "ikurento.com", mapContent["application.organization"])
}

func Test_GetConfigItem(t *testing.T) {
	configuration := initMockApollo(t)
	configs, err := configuration.GetConfig("application.organization")
	assert.NoError(t, err)
	configuration.SetParser(&parser.DefaultConfigurationParser{})
	assert.NoError(t, err)
	assert.Equal(t, "ikurento.com", configs)
}

func initMockApollo(t *testing.T) *apolloConfiguration {
	c := &config.BaseConfig{ConfigCenterConfig: &config.ConfigCenterConfig{
		Protocol:  "apollo",
		Address:   "106.12.25.204:8080",
		Group:     "testApplication_yang",
		Cluster:   "dev",
		Namespace: "mockDubbog.properties",
	}}
	apollo := initApollo()
	apolloUrl := strings.ReplaceAll(apollo.URL, "http", "apollo")
	url, err := common.NewURL(context.TODO(), apolloUrl, common.WithParams(c.ConfigCenterConfig.GetUrlMap()))
	assert.NoError(t, err)
	configuration, err := newApolloConfiguration(&url)
	assert.NoError(t, err)
	return configuration
}

func TestAddListener(t *testing.T) {
	listener := &apolloDataListener{}
	listener.wg.Add(1)
	apollo := initMockApollo(t)
	mockConfigRes = `{
	"appId": "testApplication_yang",
	"cluster": "default",
	"namespaceName": "mockDubbog.properties",
	"configurations": {
		"registries.hangzhouzk.username": "11111"
	},
	"releaseKey": "20191104105242-0f13805d89f834a4"
}`
	apollo.AddListener(mockNamespace, listener)
	listener.wg.Wait()
	assert.Equal(t, "registries.hangzhouzk.username", listener.event)
	assert.Greater(t, listener.count, 0)
}

func TestRemoveListener(t *testing.T) {
	listener := &apolloDataListener{}
	apollo := initMockApollo(t)
	mockConfigRes = `{
	"appId": "testApplication_yang",
	"cluster": "default",
	"namespaceName": "mockDubbog.properties",
	"configurations": {
		"registries.hangzhouzk.username": "11111"
	},
	"releaseKey": "20191104105242-0f13805d89f834a4"
}`
	apollo.AddListener(mockNamespace, listener)
	apollo.RemoveListener(mockNamespace, listener)
	assert.Equal(t, "", listener.event)
	listenerCount := 0
	apollo.listeners.Range(func(key, value interface{}) bool {
		apolloListener := value.(*apolloListener)
		for e := range apolloListener.listeners {
			fmt.Println(e)
			listenerCount++
		}
		return true
	})
	assert.Equal(t, listenerCount, 0)
	assert.Equal(t, listener.count, 0)
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
