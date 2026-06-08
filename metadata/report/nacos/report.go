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
	"crypto/md5"
	"encoding/json"
	"fmt"
	"strings"
)

import (
	gxset "github.com/dubbogo/gost/container/set"
	nacosClient "github.com/dubbogo/gost/database/kv/nacos"
	"github.com/dubbogo/gost/log/logger"

	"github.com/nacos-group/nacos-sdk-go/v2/vo"

	perrors "github.com/pkg/errors"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/metadata/info"
	"dubbo.apache.org/dubbo-go/v3/metadata/mapping"
	"dubbo.apache.org/dubbo-go/v3/metadata/report"
	"dubbo.apache.org/dubbo-go/v3/registry"
	"dubbo.apache.org/dubbo-go/v3/remoting/nacos"
)

func init() {
	mf := &nacosMetadataReportFactory{}
	extension.SetMetadataReportFactory("nacos", func() report.MetadataReportFactory {
		return mf
	})
}

// nacosMetadataReport is the implementation
// of MetadataReport based on nacos.
type nacosMetadataReport struct {
	client *nacosClient.NacosConfigClient
	group  string
}

// GetAppMetadata get metadata info from nacos
func (n *nacosMetadataReport) GetAppMetadata(application, revision string) (*info.MetadataInfo, error) {
	// compatible with java impl
	data, err := n.getConfig(vo.ConfigParam{
		DataId: application,
		Group:  revision,
	})
	if err != nil {
		return nil, err
	}
	if data == "" {
		// compatible with dubbo-go 3.1.x before
		data, err = n.getConfig(vo.ConfigParam{
			DataId: application + constant.KeySeparator + revision,
			Group:  n.group,
		})
		if err != nil {
			return nil, err
		}
	}
	var metadataInfo info.MetadataInfo
	err = json.Unmarshal([]byte(data), &metadataInfo)
	if err != nil {
		return nil, err
	}
	return &metadataInfo, nil
}

// PublishAppMetadata publish metadata info to nacos
func (n *nacosMetadataReport) PublishAppMetadata(application, revision string, meta *info.MetadataInfo) error {
	data, err := json.Marshal(meta)
	if err != nil {
		return err
	}
	// compatible with java impl
	err = n.storeMetadata(vo.ConfigParam{
		DataId:  application,
		Group:   revision,
		Content: string(data),
	})
	if err != nil {
		return err
	}
	// compatible with dubbo-go 3.1.x before
	return n.storeMetadata(vo.ConfigParam{
		DataId:  application + constant.KeySeparator + revision,
		Group:   n.group,
		Content: string(data),
	})
}

// storeMetadata will publish the metadata to Nacos
// if failed or error is not nil, error will be returned
func (n *nacosMetadataReport) storeMetadata(param vo.ConfigParam) error {
	res, err := n.client.Client().PublishConfig(param)
	if err != nil {
		return perrors.WithMessage(err, "Could not publish the metadata")
	}
	if !res {
		return perrors.New("Publish the metadata failed.")
	}
	return nil
}

// getConfig will read the config
func (n *nacosMetadataReport) getConfig(param vo.ConfigParam) (string, error) {
	cfg, err := n.client.Client().GetConfig(param)
	if err != nil {
		logger.Errorf("[Metadata][Nacos] finding the configuration failed, param=%v", param)
		return "", err
	}
	return cfg, nil
}

func (n *nacosMetadataReport) addListener(key string, group string, notify mapping.MappingListener) error {
	return n.client.Client().ListenConfig(vo.ConfigParam{
		DataId: key,
		Group:  group,
		OnChange: func(namespace, group, dataId, data string) {
			go callback(notify, dataId, data)
		},
	})
}

// isConfigNotExistErr reports whether err is Nacos's "config data not exist" signal, which it
// returns (rather than an empty value) for a key that has never been written. It must be treated
// as an empty old value so the first registration can create the mapping.
func isConfigNotExistErr(err error) bool {
	return err != nil && strings.Contains(strings.ToLower(err.Error()), "config data not exist")
}

func callback(notify mapping.MappingListener, dataId, data string) {
	set := report.DecodeServiceAppNames(data)
	if err := notify.OnEvent(registry.NewServiceMappingChangedEvent(dataId, set)); err != nil {
		logger.Errorf("[Metadata][Nacos] serviceMapping callback err=%v", err)
	}
}

func (n *nacosMetadataReport) removeServiceMappingListener(key string, group string) error {
	return n.client.Client().CancelListenConfig(vo.ConfigParam{
		DataId: key,
		Group:  group,
	})
}

// RegisterServiceAppMapping map the specified Dubbo service interface to current Dubbo app name
func (n *nacosMetadataReport) RegisterServiceAppMapping(key string, group string, value string) error {
	oldVal, err := n.getConfig(vo.ConfigParam{
		DataId: key,
		Group:  group,
	})
	if err != nil && !isConfigNotExistErr(err) {
		// A real read failure (network/auth/server): do not treat it as an empty value, or we
		// would publish only our app and overwrite an existing set. "config data not exist" is
		// Nacos's not-found signal, which is fine here and proceeds to the first write.
		return err
	}
	merged, changed := report.MergeServiceAppMapping(oldVal, value)
	if !changed {
		return nil
	}
	param := vo.ConfigParam{
		DataId:  key,
		Group:   group,
		Content: merged,
	}
	if oldVal != "" {
		// CasMd5 is an optimistic UPDATE: Nacos publishes only if the server content still
		// matches what we read, detecting concurrent appends. It cannot guard the first INSERT
		// (Nacos has no create-if-absent), so the initial concurrent registration of a
		// brand-new interface can still race. This is a known Nacos-only limitation; the
		// etcd and zookeeper reports do not have it.
		param.CasMd5 = fmt.Sprintf("%x", md5.Sum([]byte(oldVal)))
	}
	if err := n.storeMetadata(param); err != nil {
		if param.CasMd5 != "" {
			// Nacos surfaces a CAS rejection and a transport error the same way, so they
			// cannot be told apart here. Treat the failure as a retriable conflict rather
			// than risk dropping a real concurrent update; the underlying error is preserved
			// for diagnosis.
			return fmt.Errorf("publish mapping %s (%v): %w", key, err, report.ErrMappingCASConflict)
		}
		return err
	}
	return nil
}

// GetServiceAppMapping get the app names from the specified Dubbo service interface
func (n *nacosMetadataReport) GetServiceAppMapping(key string, group string, listener mapping.MappingListener) (*gxset.HashSet, error) {
	// add service mapping listener
	if listener != nil {
		if err := n.addListener(key, group, listener); err != nil {
			logger.Errorf("[Metadata][Nacos] add serviceMapping listener err=%v", err)
		}
	}
	v, err := n.getConfig(vo.ConfigParam{
		DataId: key,
		Group:  group,
	})
	if err != nil {
		return nil, err
	}
	if v == "" {
		return nil, perrors.New("There is no service app mapping data.")
	}
	return report.DecodeServiceAppNames(v), nil
}

// RemoveServiceAppMappingListener remove the serviceMapping listener from metadata center
func (n *nacosMetadataReport) RemoveServiceAppMappingListener(key string, group string) error {
	return n.removeServiceMappingListener(key, group)
}

type nacosMetadataReportFactory struct{}

// CreateMetadataReport creates the nacos-based metadata report implementation.
func (n *nacosMetadataReportFactory) CreateMetadataReport(url *common.URL) report.MetadataReport {
	url.SetParam(constant.NacosNamespaceID, url.GetParam(constant.MetadataReportNamespaceKey, ""))
	url.SetParam(constant.TimeoutKey, url.GetParam(constant.TimeoutKey, constant.DefaultRegTimeout))
	group := url.GetParam(constant.MetadataReportGroupKey, constant.ServiceDiscoveryDefaultGroup)
	url.SetParam(constant.NacosGroupKey, group)
	url.SetParam(constant.NacosUsername, url.Username)
	url.SetParam(constant.NacosPassword, url.Password)
	client, err := nacos.NewNacosConfigClientByUrl(url)
	if err != nil {
		logger.Errorf("[Metadata][Nacos] could not create nacos metadata report, url=%s", url.String())
		return nil
	}
	return &nacosMetadataReport{client: client, group: group}
}
