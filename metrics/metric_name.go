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

package metrics

import (
	"sync"
)

const (
	keyTagsSeparator  = "-"
	keyValueSeparator = ":"
)

var (
	emptyTags = make(map[string]string, 0)
)

type MetricName struct {
	Key         string
	Tags        map[string]string
	Level       MetricLevel
	hashKeyOnce sync.Once
	hashKey     string
}

/**
 * sometimes we try to use the MetricName as the map's Key.
 * However the Tags in MetricName is slice so that we can't use the MetricName like: map[MetricName]XXX
 * So I define this method, it will generate a string to be the Key.
 * It's similar to com.alibaba.metrics.MetricName#hashCode in Java dubbo
 * It means that, the HashKey consist of Key and Tags, but Level will be ignored.
 */
func (mn *MetricName) HashKey() string {
	mn.hashKeyOnce.Do(func() {
		mn.hashKey = mn.Key
		if len(mn.Tags) <= 0 {
			return
		}

		mn.hashKey += keyTagsSeparator
		for key, value := range mn.Tags {
			mn.hashKey += key + keyValueSeparator + value
		}
	})
	return mn.hashKey
}

/*
 * It will return an instance of MetricName.
 */
func NewMetricName(key string, tags map[string]string, level MetricLevel) *MetricName {
	if tags == nil {
		tags = emptyTags
	}
	return &MetricName{
		Key:   key,
		Tags:  tags,
		Level: level,
	}
}
