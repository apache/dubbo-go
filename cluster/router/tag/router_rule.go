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
	tags                  []tag
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
	// TODO init tags
	return r, nil
}

func (t *RouterRule) getAddresses() []string {
	// TODO get all tag addresses
	return nil
}

func (t *RouterRule) getTagNames() []string {
	// TODO get all tag names
	return nil
}

func (t *RouterRule) getAddressToTagNames() map[string][]string {
	return t.addressToTagNames
}

func (t *RouterRule) getTagNameToAddresses() map[string][]string {
	return t.tagNameToAddresses
}

func (t *RouterRule) getTags() []tag {
	return t.tags
}

func (t *RouterRule) setTags(tags []tag) {
	t.tags = tags
}
