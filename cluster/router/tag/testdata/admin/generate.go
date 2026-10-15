//go:build ignore

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

// Run from the pinned dubbo-admin checkout; see README.md.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

import (
	mesh "github.com/apache/dubbo-admin/api/mesh/v1alpha1"
	discovery "github.com/apache/dubbo-admin/pkg/config/discovery"
	"github.com/apache/dubbo-admin/pkg/core/clients"
	resources "github.com/apache/dubbo-admin/pkg/core/resource/apis/mesh/v1alpha1"
	model "github.com/apache/dubbo-admin/pkg/core/resource/model"
	"github.com/apache/dubbo-admin/pkg/core/store"
	governor "github.com/apache/dubbo-admin/pkg/governor/zk"

	"github.com/dubbogo/go-zookeeper/zk"
)

type noStore struct{}

func (noStore) ResourceRoute(model.Resource) (store.ResourceStore, error) {
	return nil, fmt.Errorf("fixture capture has no Admin cache")
}
func (noStore) ResourceKindRoute(model.ResourceKind) (store.ResourceStore, error) {
	return nil, fmt.Errorf("fixture capture has no Admin cache")
}
func main() {
	address := flag.String("zookeeper", "zookeeper://127.0.0.1:2181", "disposable ZooKeeper endpoint")
	out := flag.String("out", ".", "output directory")
	flag.Parse()
	conn, err := clients.NewZKConnection(*address)
	check(err)
	defer conn.Close()
	deadline := time.Now().Add(10 * time.Second)
	for conn.State() != zk.StateHasSession {
		if time.Now().After(deadline) {
			panic("ZooKeeper session timeout")
		}
		time.Sleep(20 * time.Millisecond)
	}
	for _, p := range []string{"/dubbo", "/dubbo/config"} {
		_, err = conn.Create(p, nil, 0, zk.WorldACL(zk.PermAll))
		if err != nil && err != zk.ErrNodeExists {
			panic(err)
		}
	}
	cfg := &discovery.Config{Address: discovery.AddressConfig{Registry: *address}}
	g, err := governor.NewZKRuleGovernor(cfg, noStore{}, nil)
	check(err)
	r := resources.NewTagRouteResourceWithAttributes("tag-compat-provider.tag-router", "tag-fixture")
	r.Spec = &mesh.TagRoute{Key: "tag-compat-provider", ConfigVersion: "v3.0", Enabled: true, Tags: []*mesh.Tag{{Name: "gray", Addresses: []string{"127.0.0.1:20890"}}}}
	path := "/dubbo/config/" + r.Name
	_ = conn.Delete(path, -1)
	defer conn.Delete(path, -1)
	check(g.CreateRule(r))
	capture(conn, path, filepath.Join(*out, "enabled.yaml"))
	r.Spec.Tags[0].Addresses = []string{"127.0.0.1:20891"}
	check(g.UpdateRule(r))
	capture(conn, path, filepath.Join(*out, "updated.yaml"))
	r.Spec.Enabled = false
	check(g.UpdateRule(r))
	capture(conn, path, filepath.Join(*out, "disabled.yaml"))
	r.Spec.Enabled = true
	r.Spec.Force = true
	r.Spec.Tags[0].Addresses = []string{"127.0.0.1:20892"}
	check(g.UpdateRule(r))
	capture(conn, path, filepath.Join(*out, "force.yaml"))
}
func capture(conn *zk.Conn, path, file string) {
	b, _, err := conn.Get(path)
	check(err)
	check(os.WriteFile(file, append([]byte(fixtureLicense), b...), 0644))
	fmt.Printf("%s -> %s\n", path, file)
}
func check(err error) {
	if err != nil {
		panic(err)
	}
}

const fixtureLicense = `# Licensed to the Apache Software Foundation (ASF) under one or more
# contributor license agreements.  See the NOTICE file distributed with
# this work for additional information regarding copyright ownership.
# The ASF licenses this file to You under the Apache License, Version 2.0
# (the "License"); you may not use this file except in compliance with
# the License.  You may obtain a copy of the License at
#
#   http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

`
