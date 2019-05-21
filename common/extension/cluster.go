// Copyright 2016-2019 hxmhlt
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package extension

import (
	"github.com/dubbo/go-for-apache-dubbo/cluster"
)

var (
	clusters = make(map[string]func() cluster.Cluster)
)

func SetCluster(name string, fcn func() cluster.Cluster) {
	clusters[name] = fcn
}

func GetCluster(name string) cluster.Cluster {
	if clusters[name] == nil {
		panic("cluster for " + name + " is not existing, you must import corresponding package.")
	}
	return clusters[name]()
}
