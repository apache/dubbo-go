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

package tag

import (
	"github.com/apache/dubbo-go/cluster/router"
	"github.com/apache/dubbo-go/common/yaml"
)

/**
 * %YAML1.2
 * ---
 * force: true
 * runtime: false
 * enabled: true
 * priority: 1
 * key: demo-provider
 * tags:
 * - name: tag1
 * addresses: [ip1, ip2]
 * - name: tag2
 * addresses: [ip3, ip4]
 * ...
 */
// RouterRule RouterRule config read from config file or config center
type RouterRule struct {
	router.BaseRouterRule `yaml:",inline""`
	Tags                  []Tag
	addressToTagNames     map[string][]string
	tagNameToAddresses    map[string][]string
}

func getRule(rawRule string) (*RouterRule, error) {
	r := &RouterRule{}
	err := yaml.UnmarshalYML([]byte(rawRule), r)
	if err != nil {
		return r, err
	}
	r.RawRule = rawRule
	r.init()
	return r, nil
}

func (t *RouterRule) init() {
	t.addressToTagNames = make(map[string][]string)
	t.tagNameToAddresses = make(map[string][]string)
	for _, tag := range t.Tags {
		for _, address := range tag.Addresses {
			t.addressToTagNames[address] = append(t.addressToTagNames[address], tag.Name)
		}
		t.tagNameToAddresses[tag.Name] = tag.Addresses
	}
}

func (t *RouterRule) getAddresses() []string {
	var result []string
	for _, tag := range t.Tags {
		result = append(result, tag.Addresses...)
	}
	return result
}

func (t *RouterRule) getTagNames() []string {
	var result []string
	for _, tag := range t.Tags {
		result = append(result, tag.Name)
	}
	return result
}

func (t *RouterRule) hasTag(tag string) bool {
	for _, t := range t.Tags {
		if tag == t.Name {
			return true
		}
	}
	return false
}

func (t *RouterRule) getAddressToTagNames() map[string][]string {
	return t.addressToTagNames
}

func (t *RouterRule) getTagNameToAddresses() map[string][]string {
	return t.tagNameToAddresses
}

func (t *RouterRule) getTags() []Tag {
	return t.Tags
}

func (t *RouterRule) setTags(tags []Tag) {
	t.Tags = tags
}
