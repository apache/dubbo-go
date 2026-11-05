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

package zookeeper

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"
)

import (
	gxzookeeper "github.com/dubbogo/gost/database/kv/zk"

	"github.com/go-zookeeper/zk"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

import (
	"dubbo.apache.org/dubbo-go/v3/metadata/definition"
)

func TestPublishServiceDefinition(t *testing.T) {
	tests := []struct{ name, root, iface, version, group, want string }{
		{"all segments", "/dubbo/", "com.example.Foo", "1.0", "my-group", "/dubbo/metadata/com.example.Foo/1.0/my-group/provider/my-app"},
		{"no version", "/dubbo/", "com.example.Foo", "", "my-group", "/dubbo/metadata/com.example.Foo/my-group/provider/my-app"},
		{"no group", "/dubbo/", "com.example.Foo", "1.0", "", "/dubbo/metadata/com.example.Foo/1.0/provider/my-app"},
		{"neither", "/dubbo/", "com.example.Foo", "", "", "/dubbo/metadata/com.example.Foo/provider/my-app"},
		{"Java URL encoding", "/dubbo/", "com.example.Foo$Inner /~*+é", "", "", "/dubbo/metadata/com.example.Foo%24Inner+%2F%7E*%2B%C3%A9/provider/my-app"},
		{"duplicate segments", "/dubbo/", "com.example.Foo", "same", "same", "/dubbo/metadata/com.example.Foo/same/same/provider/my-app"},
		{"provider group", "/dubbo/", "com.example.Foo", "", "provider", "/dubbo/metadata/com.example.Foo/provider/provider/my-app"},
		{"root slash", "/", "com.example.Foo", "", "", "/metadata/com.example.Foo/provider/my-app"},
		{"metadata root", "/dubbo/metadata/", "com.example.Foo", "", "", "/dubbo/metadata/com.example.Foo/provider/my-app"},
	}
	const payload = `{"canonicalName":"com.example.Foo","methods":[],"parameters":{},"types":[]}`
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, client := newTestReportWithMock()
			r.rootDir = tt.root
			require.NoError(t, r.PublishServiceDefinition(tt.iface, tt.version, tt.group, "my-app", payload))
			require.Len(t, client.data, 1)
			assert.Equal(t, []byte(payload), client.data[tt.want])
		})
	}
}

func TestPublishServiceDefinitionUpdateAndErrors(t *testing.T) {
	r, client := newTestReportWithMock()
	const path = "/dubbo/metadata/com.example.Foo/1/group/provider/app"
	require.NoError(t, r.PublishServiceDefinition("com.example.Foo", "1", "group", "app", `{"old":true}`))
	setErr := errors.New("set failed")
	client.errors["SetContent:"+path] = setErr
	require.ErrorIs(t, r.PublishServiceDefinition("com.example.Foo", "1", "group", "app", `{"new":true}`), setErr)
	delete(client.errors, "SetContent:"+path)
	require.NoError(t, r.PublishServiceDefinition("com.example.Foo", "1", "group", "app", `{"new":true}`))
	assert.Equal(t, []byte(`{"new":true}`), client.data[path])
	require.Len(t, client.data, 1)
	createErr := errors.New("create failed")
	client.errors["CreateWithValue:"+path] = createErr
	require.ErrorIs(t, r.PublishServiceDefinition("com.example.Foo", "1", "group", "app", `{}`), createErr)
}

// Set DUBBO_GO_METADATA_ZK to a ZooKeeper address to exercise real persistence.
func TestPublishServiceDefinitionLiveZooKeeper(t *testing.T) {
	address := os.Getenv("DUBBO_GO_METADATA_ZK")
	if address == "" {
		t.Skip("set DUBBO_GO_METADATA_ZK to run against ZooKeeper")
	}
	client, err := gxzookeeper.NewZookeeperClient("definition-test", strings.Split(address, ","), false, gxzookeeper.WithZkTimeOut(5*time.Second))
	require.NoError(t, err)
	t.Cleanup(client.Close)
	require.Eventually(t, func() bool { return client.Conn.State() == zk.StateHasSession }, 10*time.Second, 50*time.Millisecond)
	root := fmt.Sprintf("/dubbo-go-definition-test-%d", time.Now().UnixNano())
	path := root + "/metadata/com.example.Foo/1/group/provider/app"
	t.Cleanup(func() {
		for p := path; strings.HasPrefix(p, root); p = p[:strings.LastIndex(p, "/")] {
			if err := client.Delete(p); err != nil && !errors.Is(err, zk.ErrNoNode) {
				t.Errorf("cleanup %s: %v", p, err)
			}
		}
	})
	r := &zookeeperMetadataReport{client: zkClientWrapper{client}, rootDir: root + "/"}
	def := definition.ServiceDefinition{CanonicalName: "com.example.Foo", Methods: []definition.MethodDefinition{{Name: "greet", ReturnType: "java.lang.String"}}, Parameters: map[string]string{"application": "app"}, Types: []definition.TypeDefinition{}}
	payload, err := json.Marshal(def)
	require.NoError(t, err)
	require.NoError(t, r.PublishServiceDefinition(def.CanonicalName, "1", "group", "app", string(payload)))
	data, stat, err := client.GetContent(path)
	require.NoError(t, err)
	assert.Equal(t, payload, data)
	assert.Zero(t, stat.EphemeralOwner)
	var got definition.ServiceDefinition
	require.NoError(t, json.Unmarshal(data, &got))
	assert.Equal(t, def, got)
	def.Methods[0].Name = "hello"
	payload, err = json.Marshal(def)
	require.NoError(t, err)
	require.NoError(t, r.PublishServiceDefinition(def.CanonicalName, "1", "group", "app", string(payload)))
	data, _, err = client.GetContent(path)
	require.NoError(t, err)
	assert.Equal(t, payload, data)
}
