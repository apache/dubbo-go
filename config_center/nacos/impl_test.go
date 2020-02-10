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
	"sync"
	"testing"
	"time"
)

import (
	"github.com/nacos-group/nacos-sdk-go/vo"
	"github.com/stretchr/testify/assert"
)

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/config_center"
	"github.com/apache/dubbo-go/config_center/parser"
)

func initNacosData(t *testing.T) (*nacosDynamicConfiguration, error) {
	regurl, _ := common.NewURL("registry://console.nacos.io:80")
	nacosConfiguration, err := newNacosDynamicConfiguration(&regurl)
	if err != nil {
		fmt.Println("error:newNacosDynamicConfiguration", err.Error())
		assert.NoError(t, err)
		return nil, err
	}
	nacosConfiguration.SetParser(&parser.DefaultConfigurationParser{})
	data := `
	dubbo.service.com.ikurento.user.UserProvider.cluster=failback
	dubbo.service.com.ikurento.user.UserProvider.protocol=myDubbo1
	dubbo.protocols.myDubbo.port=20000
	dubbo.protocols.myDubbo.name=dubbo
`
	sucess, err := (*nacosConfiguration.client.Client).PublishConfig(vo.ConfigParam{
		DataId:  "dubbo.properties",
		Group:   "dubbo",
		Content: data,
	})
	assert.NoError(t, err)
	if !sucess {
		fmt.Println("error: publishconfig error", data)
	}
	return nacosConfiguration, err
}

func Test_GetConfig(t *testing.T) {
	nacos, err := initNacosData(t)
	assert.NoError(t, err)
	configs, err := nacos.GetProperties("dubbo.properties", config_center.WithGroup("dubbo"))
	m, err := nacos.Parser().Parse(configs)
	assert.NoError(t, err)
	fmt.Println(m)
}

func Test_AddListener(t *testing.T) {
	nacos, err := initNacosData(t)
	assert.NoError(t, err)
	listener := &mockDataListener{}
	time.Sleep(time.Second * 2)
	nacos.AddListener("dubbo.properties", listener)
	listener.wg.Add(2)
	fmt.Println("begin to listen")
	data := `
	dubbo.service.com.ikurento.user.UserProvider.cluster=failback
	dubbo.service.com.ikurento.user.UserProvider.protocol=myDubbo
	dubbo.protocols.myDubbo.port=20000
	dubbo.protocols.myDubbo.name=dubbo
`
	sucess, err := (*nacos.client.Client).PublishConfig(vo.ConfigParam{
		DataId:  "dubbo.properties",
		Group:   "dubbo",
		Content: data,
	})
	assert.NoError(t, err)
	if !sucess {
		fmt.Println("error: publishconfig error", data)
	}
	listener.wg.Wait()
	fmt.Println("end", listener.event)

}

func Test_RemoveListener(t *testing.T) {
	//TODO not supported in current go_nacos_sdk version
}

type mockDataListener struct {
	wg    sync.WaitGroup
	event string
}

func (l *mockDataListener) Process(configType *config_center.ConfigChangeEvent) {
	fmt.Println("process!!!!!!!!!!")
	l.wg.Done()
	l.event = configType.Key
}
