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
package config

import (
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"path"
	"strconv"
	"strings"
	"sync"
)
import (
	perrors "github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)
import (
	"github.com/apache/dubbo-go/cluster/directory"
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/common/logger"
)

var (
	mutex sync.Mutex
)

/////////////////////////
// routerConfig
/////////////////////////
type ConditionRouterConfig struct {
	RawRule    string   `yaml:"rawRule"`
	Scope      string   `yaml:"scope"`
	Priority   int      `yaml:"priority"`
	Enabled    bool     `yaml:"enabled" default:"true"`
	Dynamic    bool     `yaml:"dynamic" default:"false"`
	Valid      bool     `yaml:"valid" default:"true"`
	Force      bool     `yaml:"force" default:"false"`
	Runtime    bool     `yaml:"runtime" default:"true"`
	Key        string   `yaml:"key"`
	Conditions []string `yaml:"conditions"`
}

func (*ConditionRouterConfig) Prefix() string {
	return constant.RouterConfigPrefix
}

func RouterInit(confRouterFile string) error {
	if len(confRouterFile) == 0 {
		return perrors.Errorf("application configure(provider) file name is nil")
	}

	if path.Ext(confRouterFile) != ".yml" {
		return perrors.Errorf("application configure file name{%v} suffix must be .yml", confRouterFile)
	}

	confFileStream, err := ioutil.ReadFile(confRouterFile)
	if err != nil {
		return perrors.Errorf("ioutil.ReadFile(file:%s) = error:%v", confRouterFile, perrors.WithStack(err))
	}
	routerConfig = &ConditionRouterConfig{}
	err = yaml.Unmarshal(confFileStream, routerConfig)
	if err != nil {
		return perrors.Errorf("yaml.Unmarshal() = error:%v", perrors.WithStack(err))
	}

	logger.Debugf("provider config{%#v}\n", routerConfig)
	directory.RouterUrlSet.Add(initRouterUrl())
	fmt.Println("=====", directory.RouterUrlSet.Size())
	return nil
}

func initRouterUrl() *common.URL {
	mutex.Lock()
	if routerConfig == nil {
		confRouterFile := os.Getenv(constant.CONF_ROUTER_FILE_PATH)
		err := RouterInit(confRouterFile)
		if err != nil {
			return nil
		}
	}
	mutex.Unlock()
	rule := parseCondition(routerConfig.Conditions)

	return common.NewURLWithOptions(
		common.WithProtocol(constant.ROUTE_PROTOCOL),
		common.WithIp(constant.ANYHOST_VALUE),
		common.WithParams(url.Values{}),
		common.WithParamsValue("enabled", strconv.FormatBool(routerConfig.Enabled)),
		common.WithParamsValue("dynamic", strconv.FormatBool(routerConfig.Dynamic)),
		common.WithParamsValue("force", strconv.FormatBool(routerConfig.Force)),
		common.WithParamsValue("runtime", strconv.FormatBool(routerConfig.Runtime)),
		common.WithParamsValue("priority", strconv.Itoa(routerConfig.Priority)),
		common.WithParamsValue("scope", routerConfig.Scope),
		common.WithParamsValue(constant.RULE_KEY, base64.URLEncoding.EncodeToString([]byte(rule))),
		common.WithParamsValue("router", "condition"),
		common.WithParamsValue(constant.CATEGORY_KEY, constant.ROUTERS_CATEGORY))
}

func parseCondition(conditions []string) string {
	var when, then string
	for _, condition := range conditions {
		condition = strings.Trim(condition, " ")
		if strings.Contains(condition, "=>") {
			array := strings.SplitN(condition, "=>", 2)
			consumer := strings.Trim(array[0], " ")
			provider := strings.Trim(array[1], " ")
			if len(consumer) != 0 {
				if len(when) != 0 {
					when = strings.Join([]string{when, consumer}, " & ")
				} else {
					when = consumer
				}
			}
			if len(provider) != 0 {
				if len(then) != 0 {
					then = strings.Join([]string{then, provider}, " & ")
				} else {
					then = provider
				}
			}

		}

	}

	return strings.Join([]string{when, then}, " => ")
}
