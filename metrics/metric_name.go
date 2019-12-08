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

var (
	emptyTags = make(map[string]string, 0)
)

type MetricName struct {
	key   string
	tags  map[string]string
	level MetricLevel
}

/*
 * It will return an instance of MetricName. You should know that the return value is not a pointer,
 * which means that if the key too long or the tags has too many key-value pair, the cost of memory will be expensive.
 */
func NewMetricName(key string, tags map[string]string, level MetricLevel) MetricName {
	if tags == nil {
		tags = emptyTags
	}
	return MetricName{
		key:   key,
		tags:  tags,
		level: level,
	}
}
