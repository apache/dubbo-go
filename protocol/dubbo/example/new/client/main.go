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

package main

import (
	"context"
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/config"
	"dubbo.apache.org/dubbo-go/v3/config_center"
	"dubbo.apache.org/dubbo-go/v3/config_center/parser"
	gxset "github.com/dubbogo/gost/container/set"
)

import (
	"github.com/dubbogo/gost/log/logger"
)

import (
	"dubbo.apache.org/dubbo-go/v3/client"
	_ "dubbo.apache.org/dubbo-go/v3/imports"
)

type noOpConf struct {
}

func (n *noOpConf) Parser() parser.ConfigurationParser {
	return nil
}

func (n *noOpConf) SetParser(parser parser.ConfigurationParser) {
	return
}

func (n *noOpConf) AddListener(s string, listener config_center.ConfigurationListener, option ...config_center.Option) {
	return
}

func (n *noOpConf) RemoveListener(s string, listener config_center.ConfigurationListener, option ...config_center.Option) {
	return
}

func (n *noOpConf) GetProperties(s string, option ...config_center.Option) (string, error) {
	return "", nil
}

func (n *noOpConf) GetRule(s string, option ...config_center.Option) (string, error) {
	return "", nil
}

func (n *noOpConf) GetInternalProperty(s string, option ...config_center.Option) (string, error) {
	return "", nil
}

func (n *noOpConf) PublishConfig(s string, s2 string, s3 string) error {
	return nil
}

func (n *noOpConf) RemoveConfig(s string, s2 string) error {
	return nil
}

func (n *noOpConf) GetConfigKeysByGroup(group string) (*gxset.HashSet, error) {
	return nil, nil
}

type noOpConfFactory struct{}

// GetDynamicConfiguration Get Configuration with URL
func (f *noOpConfFactory) GetDynamicConfiguration(url *common.URL) (config_center.DynamicConfiguration, error) {
	return &noOpConf{}, nil
}

func main() {
	extension.SetConfigCenterFactory("no-op", func() config_center.DynamicConfigurationFactory {
		return &noOpConfFactory{}
	})
	if err := config.Load(); err != nil {
		panic(err)
	}
	cli, err := client.NewClient(
		client.WithClientProtocolDubbo(),
	)
	if err != nil {
		panic(err)
	}
	conn, err := cli.Dial("GreetProvider",
		client.WithURL("127.0.0.1:20000"),
	)
	if err != nil {
		panic(err)
	}
	var resp string
	if err := conn.CallUnary(context.Background(), []interface{}{"hello", "new", "dubbo"}, &resp, "Greet"); err != nil {
		logger.Errorf("GreetProvider.Greet err: %s", err)
		return
	}
	logger.Infof("Get Response: %s", resp)
}
