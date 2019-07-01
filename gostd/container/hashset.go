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

package container

import (
	"fmt"
	"strings"
)

var itemExists = struct{}{}

type HashSet struct {
	Items map[interface{}]struct{}
}

func NewSet(values ...interface{}) *HashSet {
	set := &HashSet{Items: make(map[interface{}]struct{})}
	if len(values) > 0 {
		set.Add(values...)
	}
	return set
}

func (set *HashSet) Add(items ...interface{}) {
	for _, item := range items {
		set.Items[item] = itemExists
	}
}

func (set *HashSet) Remove(items ...interface{}) {
	for _, item := range items {
		delete(set.Items, item)
	}
}

func (set *HashSet) Contains(items ...interface{}) bool {
	for _, item := range items {
		if _, contains := set.Items[item]; !contains {
			return false
		}
	}
	return true
}
func (set *HashSet) Empty() bool {
	return set.Size() == 0
}
func (set *HashSet) Size() int {
	return len(set.Items)
}

func (set *HashSet) Clear() {
	set.Items = make(map[interface{}]struct{})
}

func (set *HashSet) Values() []interface{} {
	values := make([]interface{}, set.Size())
	count := 0
	for item := range set.Items {
		values[count] = item
		count++
	}
	return values
}
func (set *HashSet) String() string {
	str := "HashSet\n"
	var items []string
	for k := range set.Items {
		items = append(items, fmt.Sprintf("%v", k))
	}
	str += strings.Join(items, ", ")
	return str
}
