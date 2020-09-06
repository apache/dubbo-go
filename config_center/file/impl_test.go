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

package file

import (
	"fmt"
	"os"
	"sync"
	"testing"
	"time"
)

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/extension"
	"github.com/apache/dubbo-go/config_center"
)

import (
	"github.com/stretchr/testify/assert"
)

const (
	key = "com.dubbo.go"
)

func initFileData(t *testing.T) (*FileSystemDynamicConfiguration, error) {
	urlString := "registry://127.0.0.1:2181"
	regurl, err := common.NewURL(urlString)
	assert.NoError(t, err)
	dc, err := extension.GetConfigCenterFactory("file").GetDynamicConfiguration(&regurl)
	assert.NoError(t, err)

	return dc.(*FileSystemDynamicConfiguration), err
}

func TestPublishAndGetConfig(t *testing.T) {
	file, err := initFileData(t)
	assert.NoError(t, err)
	if err := file.PublishConfig(key, "", "A"); err != nil {
		t.Fatal(err)
	}

	if prop, err := file.GetProperties(key); err != nil {
		assert.Equal(t, "A", prop)
	}

	defer destroy(t, file.rootPath, file)
}

func TestAddListener(t *testing.T) {
	file, err := initFileData(t)
	group := "dubbogo"
	value := "Test Value"
	err = file.PublishConfig(key, group, value)
	assert.NoError(t, err)

	listener := &mockDataListener{}
	file.AddListener(key, listener, config_center.WithGroup(group))

	listener.wg.Add(1)
	value = "Test Value 2"
	err = file.PublishConfig(key, group, value)
	assert.NoError(t, err)

	listener.wg.Add(1)
	value = "Test Value 3"
	err = file.PublishConfig(key, group, value)
	assert.NoError(t, err)

	listener.wg.Wait()

	time.Sleep(time.Second)
	defer destroy(t, file.rootPath, file)
}

func TestAddAndRemoveListener(t *testing.T) {
	file, err := initFileData(t)
	group := "dubbogo"
	value := "Test Value"
	err = file.PublishConfig(key, group, value)
	assert.NoError(t, err)

	listener := &mockDataListener{}
	file.AddListener(key, listener, config_center.WithGroup(group))

	listener.wg.Add(1)
	value = "Test Value 2"
	err = file.PublishConfig(key, group, value)
	assert.NoError(t, err)

	// sleep, make sure callback run success, do `l.wg.Done()`
	time.Sleep(time.Second)
	file.RemoveListener(key, listener, config_center.WithGroup(group))

	listener.wg.Add(1)
	value = "Test Value 3"
	err = file.PublishConfig(key, group, value)
	assert.NoError(t, err)
	listener.wg.Done()

	listener.wg.Wait()

	time.Sleep(time.Second)
	defer destroy(t, file.rootPath, file)
}

func TestGetConfigKeysByGroup(t *testing.T) {
	file, err := initFileData(t)
	group := "dubbogo"
	value := "Test Value"
	err = file.PublishConfig(key, group, value)
	gs, err := file.GetConfigKeysByGroup(group)
	assert.NoError(t, err)
	assert.Equal(t, 1, gs.Size())
	defer destroy(t, file.rootPath, file)
}

func TestGetConfig(t *testing.T) {
	file, err := initFileData(t)
	assert.NoError(t, err)
	group := "dubbogo"
	value := "Test Value"
	err = file.PublishConfig(key, group, value)
	assert.NoError(t, err)
	prop, err := file.GetProperties(key, config_center.WithGroup(group))
	assert.NoError(t, err)
	assert.Equal(t, value, prop)
	defer destroy(t, file.rootPath, file)
}

func TestPublishConfig(t *testing.T) {
	file, err := initFileData(t)
	assert.NoError(t, err)
	group := "dubbogo"
	value := "Test Value"
	err = file.PublishConfig(key, group, value)
	assert.NoError(t, err)
	prop, err := file.GetInternalProperty(key, config_center.WithGroup(group))
	assert.NoError(t, err)
	assert.Equal(t, value, prop)
	defer destroy(t, file.rootPath, file)
}

func destroy(t *testing.T, path string, fdc *FileSystemDynamicConfiguration) {
	if err := os.RemoveAll(path); err != nil {
		t.Error(err)
	}
	fdc.Close()
}

type mockDataListener struct {
	wg    sync.WaitGroup
	event string
}

func (l *mockDataListener) Process(configType *config_center.ConfigChangeEvent) {
	fmt.Println("process!!!!!")
	l.wg.Done()
	l.event = configType.Key
}
