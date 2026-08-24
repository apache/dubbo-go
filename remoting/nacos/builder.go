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
	"net"
	"strconv"
	"strings"
	"sync"
	"time"
)

import (
	nacosClient "github.com/dubbogo/gost/database/kv/nacos"
	"github.com/dubbogo/gost/log/logger"

	nacosConstant "github.com/nacos-group/nacos-sdk-go/v2/common/constant"

	perrors "github.com/pkg/errors"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
)

var (
	newNacosNamingClient = nacosClient.NewNacosNamingClient
	newNacosConfigClient = nacosClient.NewNacosConfigClient
)

// credentialIDs maps each distinct credential set to a small opaque id used
// in pool keys. The credentials themselves stay in this process-local map and
// never become part of the key, which may end up in logs.
var (
	credentialIDsMu sync.RWMutex
	credentialIDs   = make(map[string]string)
)

func credentialID(url *common.URL) string {
	tuple := strings.Join([]string{
		url.GetParam(constant.NacosUsername, ""),
		url.GetParam(constant.NacosPassword, ""),
		url.GetParam(constant.NacosAccessKey, ""),
		url.GetParam(constant.NacosSecretKey, ""),
	}, "\n")

	// Try read lock first for the common case (credential already exists)
	credentialIDsMu.RLock()
	id, ok := credentialIDs[tuple]
	credentialIDsMu.RUnlock()
	if ok {
		return id
	}

	// Need to create new credential ID, acquire write lock
	credentialIDsMu.Lock()
	defer credentialIDsMu.Unlock()
	// Double-check in case another goroutine created it while we waited
	id, ok = credentialIDs[tuple]
	if !ok {
		id = "cred" + strconv.Itoa(len(credentialIDs))
		credentialIDs[tuple] = id
	}
	return id
}

// nacosClientPoolKey derives the gost client-pool key from the fields that
// distinguish one nacos connection from another: server (endpoint/address),
// path, namespace and the full credential set. Components pointing at the same
// cluster (registry, config-center, metadata-report) resolve to the same key
// and share one SDK client session instead of each opening its own.
// Role-scoped client names must not be used as the key — they would defeat
// the sharing.
//
// Note on url.Location with multiple addresses: url.Location may contain
// multiple comma-separated addresses (e.g., "host1:8848,host2:8848") which
// are parsed into separate ServerConfig entries. The full Location string
// participates in the pool key, so different orderings or different server
// lists will create separate pool keys. This is intentional: the order and
// composition of servers affects client behavior, and configurations should
// be consistent across components that intend to share a client.
func nacosClientPoolKey(kind string, url *common.URL) string {
	// GetNacosConfig ignores url.Location when an endpoint is set; mirror
	// that here so URLs resolving to the same server set share one client.
	server := url.GetParam(constant.NacosEndpoint, "")
	if server == "" {
		server = url.Location
	}
	// Clients authenticated differently must never collapse into one pool
	// entry, so the full credential set participates via its opaque id.
	// Include url.Path so that nacos://host:port/pathA and nacos://host:port/pathB
	// create separate clients (path becomes ContextPath in ServerConfig).
	return strings.Join([]string{
		"dubbo-nacos", kind,
		server,
		url.Path,
		url.GetParam(constant.NacosNamespaceID, ""),
		credentialID(url),
	}, "|")
}

// NewNacosConfigClientByUrl read the config from url and build an instance
func NewNacosConfigClientByUrl(url *common.URL) (*nacosClient.NacosConfigClient, error) {
	sc, cc, err := GetNacosConfig(url)
	if err != nil {
		return nil, err
	}
	clientName := url.GetParam(constant.ClientNameKey, "")
	if len(clientName) <= 0 {
		return nil, perrors.New("nacos client name must set")
	}
	return newNacosConfigClient(nacosClientPoolKey("config", url), true, sc, cc)
}

// GetNacosConfig will return the nacos config
func GetNacosConfig(url *common.URL) ([]nacosConstant.ServerConfig, nacosConstant.ClientConfig, error) {
	if url == nil {
		return []nacosConstant.ServerConfig{}, nacosConstant.ClientConfig{}, perrors.New("url is empty!")
	}

	if len(url.Location) == 0 {
		return []nacosConstant.ServerConfig{}, nacosConstant.ClientConfig{},
			perrors.New("url.location is empty!")
	}

	var serverConfigs []nacosConstant.ServerConfig
	// if the endpoint is set, the location will be ignored
	if len(url.GetParam(constant.NacosEndpoint, "")) == 0 {
		addresses := strings.Split(url.Location, ",")
		serverConfigs = make([]nacosConstant.ServerConfig, 0, len(addresses))
		for _, addr := range addresses {
			ip, portStr, err := net.SplitHostPort(addr)
			if err != nil {
				return []nacosConstant.ServerConfig{}, nacosConstant.ClientConfig{},
					perrors.WithMessagef(err, "split [%s] ", addr)
			}
			portContextPath := strings.Split(portStr, constant.PathSeparator)
			port, err := strconv.Atoi(portContextPath[0])
			if err != nil {
				return []nacosConstant.ServerConfig{}, nacosConstant.ClientConfig{},
					perrors.WithMessagef(err, "port [%s] ", portContextPath[0])
			}
			var contextPath string
			if len(portContextPath) > 1 {
				contextPath = constant.PathSeparator + strings.Join(portContextPath[1:], constant.PathSeparator)
			}
			if contextPath == "" && len(url.Path) > 0 {
				contextPath = url.Path
			}
			serverConfigs = append(serverConfigs, nacosConstant.ServerConfig{IpAddr: ip, Port: uint64(port), ContextPath: contextPath})
		}
	}

	timeout := url.GetParamDuration(constant.NacosTimeout, constant.DefaultRegTimeout)

	clientConfig := nacosConstant.ClientConfig{
		TimeoutMs:            uint64(int32(timeout / time.Millisecond)),
		NamespaceId:          url.GetParam(constant.NacosNamespaceID, ""),
		Username:             url.GetParam(constant.NacosUsername, ""),
		Password:             url.GetParam(constant.NacosPassword, ""),
		BeatInterval:         url.GetParamInt(constant.NacosBeatIntervalKey, 5000),
		AppName:              url.GetParam(constant.NacosAppNameKey, ""),
		Endpoint:             url.GetParam(constant.NacosEndpoint, ""),
		RegionId:             url.GetParam(constant.NacosRegionIDKey, ""),
		AccessKey:            url.GetParam(constant.NacosAccessKey, ""),
		SecretKey:            url.GetParam(constant.NacosSecretKey, ""),
		OpenKMS:              url.GetParamBool(constant.NacosOpenKmsKey, false),
		CacheDir:             url.GetParam(constant.NacosCacheDirKey, ""),
		UpdateThreadNum:      url.GetParamByIntValue(constant.NacosUpdateThreadNumKey, 20),
		NotLoadCacheAtStart:  url.GetParamBool(constant.NacosNotLoadLocalCache, true),
		LogDir:               url.GetParam(constant.NacosLogDirKey, ""),
		LogLevel:             url.GetParam(constant.NacosLogLevelKey, "info"),
		UpdateCacheWhenEmpty: url.GetParamBool(constant.NacosUpdateCacheWhenEmpty, true),
	}
	return serverConfigs, clientConfig, nil
}

// NewNacosClientByURL created
func NewNacosClientByURL(url *common.URL) (*nacosClient.NacosNamingClient, error) {
	scs, cc, err := GetNacosConfig(url)
	if err != nil {
		return nil, err
	}
	clientName := url.GetParam(constant.ClientNameKey, "")
	if len(clientName) <= 0 {
		return nil, perrors.New("nacos client name must set")
	}
	logger.Infof("[Remoting][Nacos] new nacos client, config=%+v", scs)
	return newNacosNamingClient(nacosClientPoolKey("naming", url), true, scs, cc)
}
