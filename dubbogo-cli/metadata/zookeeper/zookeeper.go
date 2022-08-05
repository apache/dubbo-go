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
	"fmt"
	"log"
	"time"
)

import (
	gxzookeeper "github.com/dubbogo/gost/database/kv/zk"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/dubbogo-cli/metadata"
)

func init() {
	metadata.Register("zookeeper", NewZookeeperMetadataReport)
}

// ZookeeperMetadataReport is the implementation of
// MetadataReport based on zookeeper.
type ZookeeperMetadataReport struct {
	client  *gxzookeeper.ZookeeperClient
	rootDir string
}

// NewZookeeperMetadataReport create a zookeeper metadata reporter
func NewZookeeperMetadataReport(name string, zkAddrs []string) metadata.MetaData {
	client, err := gxzookeeper.NewZookeeperClient(
		name,
		zkAddrs,
		false,
		gxzookeeper.WithZkTimeOut(15*time.Second))
	if err != nil {
		panic(err)
	}
	return &ZookeeperMetadataReport{
		client:  client,
		rootDir: "/dubbo",
	}
}

// GetChildren get children node
func (z *ZookeeperMetadataReport) GetChildren(path string) ([]string, error) {
	delimiter := "/"
	if path == "" {
		delimiter = path
	}
	return z.client.GetChildren(z.rootDir + delimiter + path)
}

// ShowChildren shou children list
func (z *ZookeeperMetadataReport) ShowChildren() (map[string][]string, error) {
	methodsMap := map[string][]string{}
	inters, err := z.GetChildren("")
	if err != nil {
		return nil, err
	}
	for _, inter := range inters {
		if _, ok := methodsMap[inter]; !ok {
			methodsMap[inter] = make([]string, 0)
		}

		interChildren, err := z.GetChildren(inter)
		if err != nil {
			return nil, err
		}
		for _, interChild := range interChildren {
			interURLs, err := z.GetChildren(inter + "/" + interChild)
			if err != nil {
				log.Println(err)
			}
			for _, interURL := range interURLs {
				url, err := common.NewURL(interURL)
				if err != nil {
					return nil, err
				}
				methodsMap[inter] = append(methodsMap[inter], url.GetParam("methods", ""))
			}
		}
	}
	for k, v := range methodsMap {
		fmt.Printf("interface: %s\n", k)
		fmt.Printf("methods: %v\n", v)
	}
	return methodsMap, nil
}
