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

package common

var urlComparator URLComparator

// Define some func for URL, such as comparison of URL.
// Your can define your own implements, and invoke SetURLComparator.
type URLComparator interface {
	CompareURLEqual(*URL, *URL, ...string) bool
}

func SetURLComparator(comparator URLComparator) {
	urlComparator = comparator
}
func GetURLComparator() URLComparator {
	return urlComparator
}

// Config default defaultURLComparator.
func init() {
	SetURLComparator(defaultURLComparator{})
}

type defaultURLComparator struct {
}

// default comparison implements
func (defaultURLComparator) CompareURLEqual(l *URL, r *URL, excludeParam ...string) bool {
	return IsEquals(*l, *r, excludeParam...)
}
