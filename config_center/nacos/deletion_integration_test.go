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
	"os"
	"testing"
	"time"
)

import (
	"github.com/stretchr/testify/require"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/config_center"
	"dubbo.apache.org/dubbo-go/v3/remoting"
)

type deletionListener struct {
	events chan config_center.ConfigChangeEvent
}

func (l *deletionListener) Process(event *config_center.ConfigChangeEvent) {
	l.events <- *event
}

// This test owns a unique rule and requires a disposable Nacos server.
// Run with DUBBO_GO_NACOS_DELETE_ADDR=127.0.0.1:8848 go test ./config_center/nacos -run TestNacosRuleDeletion.
func TestNacosRuleDeletion(t *testing.T) {
	address := os.Getenv("DUBBO_GO_NACOS_DELETE_ADDR")
	if address == "" {
		t.Skip("set DUBBO_GO_NACOS_DELETE_ADDR to run against a real Nacos server")
	}
	key := fmt.Sprintf("delete-regression-%d.condition-router", time.Now().UnixNano())
	u, err := common.NewURL("nacos://" + address)
	require.NoError(t, err)
	u.SetParam(constant.ConfigGroupKey, "dubbo")
	u.SetParam(constant.ClientNameKey, key)
	u.SetParam(constant.NacosLogDirKey, t.TempDir())
	u.SetParam(constant.NacosCacheDirKey, t.TempDir())
	u.SetParam(constant.NacosLogLevelKey, "error")
	dc, err := newNacosDynamicConfiguration(u)
	require.NoError(t, err)
	t.Cleanup(dc.Destroy)
	t.Cleanup(func() { require.NoError(t, dc.RemoveConfig(key, "dubbo")) })

	initial := "enabled: true\n"
	require.NoError(t, dc.PublishConfig(key, "dubbo", initial))
	require.Eventually(t, func() bool {
		got, err := dc.GetRule(key)
		return err == nil && got == initial
	}, 20*time.Second, 50*time.Millisecond)

	l := &deletionListener{events: make(chan config_center.ConfigChangeEvent, 16)}
	dc.AddListener(key, l)
	t.Cleanup(func() { dc.RemoveListener(key, l) })
	wait := func(kind remoting.EventType, value string) {
		t.Helper()
		timer := time.NewTimer(20 * time.Second)
		defer timer.Stop()
		for {
			select {
			case event := <-l.events:
				if event.Key == key && event.ConfigType == kind && event.Value == value {
					return
				}
			case <-timer.C:
				t.Fatalf("no %v event for %s", kind, key)
			}
		}
	}

	// Receiving an update confirms the native listener is active before deletion.
	updated := "enabled: false\n"
	require.NoError(t, dc.PublishConfig(key, "dubbo", updated))
	wait(remoting.EventTypeUpdate, updated)
	require.NoError(t, dc.RemoveConfig(key, "dubbo"))
	wait(remoting.EventTypeDel, "")
}
