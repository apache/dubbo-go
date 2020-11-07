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

import "sync"

var urlTool URLTool
var lock sync.Mutex

// Define some func for URL, such as comparison of URL.
// Your can define your own implements, and invoke SetURLTool for high priority.
type URLTool interface {
	// The higher of the number, the higher of the priority.
	Priority() uint8
	// comparison of two URL, excluding some params
	CompareURLEqual(*URL, *URL, ...string) bool
}

func SetURLTool(tool URLTool) {
	lock.Lock()
	defer lock.Unlock()
	if urlTool == nil {
		urlTool = tool
		return
	}
	if urlTool.Priority() < tool.Priority() {
		urlTool = tool
	}
}
func GetURLTool() URLTool {
	return urlTool
}

// Config default urlTools.
func init() {
	SetURLTool(defaultURLTool{})
}

type defaultURLTool struct {
}

// default priority is 16.
func (defaultURLTool) Priority() uint8 {
	//default is 16.
	return 16
}

// default comparison implements
func (defaultURLTool) CompareURLEqual(l *URL, r *URL, execludeParam ...string) bool {
	return IsEquals(*l, *r, execludeParam...)
}
