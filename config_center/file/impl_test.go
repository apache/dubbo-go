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
	"sync"
	"testing"
)

import (
	"github.com/apache/dubbo-go/config_center"
)

const (
	dubboPropertyFileName = "dubbo.properties"
)

func TestListener(t *testing.T) {
	fsdc := &fileSystemDynamicConfiguration{
		rootPath: "/Users/tc/Documents/workspace_2020/dubbo/dubbo-common/target/test-classes/config-center",
	}

	listener := &mockDataListener{}
	listener.wg.Add(1)
	fsdc.addListener("abc-def-ghi", listener)

	listener.wg.Wait()
	fsdc.close()
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
