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
	"fmt"
	"log"
	"strings"
	"time"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/metadata/definition"

	gxzookeeper "github.com/dubbogo/gost/database/kv/zk"
)

import (
	"dubbo.apache.org/dubbo-go/v3/tools/dubbogo-cli/metadata"
)

func init() {
	metadata.Register("zookeeper", NewZookeeperMetadataReport)
}

// ZookeeperMetadataReport is the implementation of MetadataReport based on ZooKeeper.
type ZookeeperMetadataReport struct {
	client  *gxzookeeper.ZookeeperClient
	rootDir string
	zkAddr  string // Store the ZooKeeper address for URL construction
}

// NewZookeeperMetadataReport creates a ZooKeeper metadata reporter
func NewZookeeperMetadataReport(name string, zkAddrs []string) metadata.MetaData {
	if len(zkAddrs) == 0 || zkAddrs[0] == "" {
		panic("No ZooKeeper address provided")
	}
	// Strip scheme if present (e.g., "zookeeper://127.0.0.1:2181" -> "127.0.0.1:2181")
	cleanAddrs := make([]string, len(zkAddrs))
	for i, addr := range zkAddrs {
		if strings.Contains(addr, "://") {
			parts := strings.SplitN(addr, "://", 2)
			if len(parts) == 2 {
				cleanAddrs[i] = parts[1]
			} else {
				cleanAddrs[i] = addr
			}
		} else {
			cleanAddrs[i] = addr
		}
	}

	client, err := gxzookeeper.NewZookeeperClient(
		name,
		cleanAddrs,
		false,
		gxzookeeper.WithZkTimeOut(15*time.Second))
	if err != nil {
		panic(err)
	}
	return &ZookeeperMetadataReport{
		client:  client,
		rootDir: "/dubbo",
		zkAddr:  cleanAddrs[0], // Store the cleaned address
	}
}

// GetChildren gets children nodes under a path
func (z *ZookeeperMetadataReport) GetChildren(path string) ([]string, error) {
	delimiter := "/"
	if path == "" {
		delimiter = ""
	}
	return z.client.GetChildren(z.rootDir + delimiter + path)
}

// ShowRegistryCenterChildren shows children list from the registry
func (z *ZookeeperMetadataReport) ShowRegistryCenterChildren() (map[string][]string, error) {
	methodsMap := map[string][]string{}
	inters, err := z.GetChildren("")
	if err != nil {
		return nil, fmt.Errorf("failed to get root children: %v", err)
	}
	for _, inter := range inters {
		if _, ok := methodsMap[inter]; !ok {
			methodsMap[inter] = make([]string, 0)
		}

		interChildren, err := z.GetChildren(inter)
		if err != nil {
			log.Printf("Failed to get children for %s: %v", inter, err)
			continue
		}
		for _, interChild := range interChildren {
			interURLs, err := z.GetChildren(inter + "/" + interChild)
			if err != nil {
				log.Printf("Failed to get URLs for %s/%s: %v", inter, interChild, err)
				continue
			}
			for _, interURL := range interURLs {
				// Fetch the content of the node instead of using the node name
				fullPath := z.rootDir + "/" + inter + "/" + interChild + "/" + interURL
				content, _, err := z.client.GetContent(fullPath)
				if err != nil {
					log.Printf("Failed to get content for %s: %v", fullPath, err)
					continue
				}
				log.Printf("Processing interURL content: %s", string(content))

				urlStr := string(content)
				if !strings.Contains(urlStr, "://") {
					urlStr = fmt.Sprintf("zookeeper://%s/%s", z.zkAddr, strings.TrimLeft(urlStr, "/"))
				}

				url, err := common.NewURL(urlStr)
				if err != nil {
					log.Printf("Failed to parse URL %s: %v", urlStr, err)
					continue
				}
				methods := url.GetParam("methods", "")
				if methods != "" {
					methodsMap[inter] = append(methodsMap[inter], strings.Split(methods, ",")...)
				}
			}
		}
	}
	for k, v := range methodsMap {
		fmt.Printf("interface: %s\n", k)
		fmt.Printf("methods: %v\n", v)
	}
	return methodsMap, nil
}

// ShowMetadataCenterChildren shows children list from the metadata center
func (z *ZookeeperMetadataReport) ShowMetadataCenterChildren() (map[string][]string, error) {
	methodsMap := map[string][]string{}
	inters, err := z.GetChildren("metadata")
	if err != nil {
		return nil, fmt.Errorf("failed to get metadata children: %v", err)
	}
	for _, inter := range inters {
		path := "metadata/" + inter
		var methods []string
		z.searchMetadataProvider(path, &methods)

		if _, ok := methodsMap[inter]; !ok && len(methods) != 0 {
			methodsMap[inter] = make([]string, 0)
		}
		for _, method := range methods {
			methodsMap[inter] = append(methodsMap[inter], method)
		}
	}
	return methodsMap, nil
}

// searchMetadataProvider recursively searches for provider metadata
func (z *ZookeeperMetadataReport) searchMetadataProvider(path string, methods *[]string) {
	interChildren, err := z.GetChildren(path)
	if err != nil {
		log.Printf("Failed to get children for %s: %v", path, err)
		return
	}
	for _, interChild := range interChildren {
		if interChild == "provider" {
			content, _, err := z.client.GetContent("/" + "dubbo" + "/" + path + "/" + interChild)
			if err != nil {
				fmt.Printf("Zookeeper Get Content Error: %v\n", err)
				return
			}

			var serviceDefinition definition.FullServiceDefinition
			err = json.Unmarshal(content, &serviceDefinition)
			if err != nil {
				fmt.Printf("Json Unmarshal fail: %v\n", err)
			}
			for _, method := range serviceDefinition.Methods {
				*methods = append(*methods, method.Name)
			}
		} else {
			z.searchMetadataProvider(path+"/"+interChild, methods)
		}
	}
}
