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

package instance

import (
	"dubbo.apache.org/dubbo-go/v3/protocol"
	"errors"
	"strings"
)

type ScriptInstances interface {
	RunScript(rawScript string, invokers []protocol.Invoker, invocation protocol.Invocation) ([]protocol.Invoker, error)
	Compile(name, rawScript string) error
	Destroy()
}

var factory map[string]ScriptInstances

func GetInstances(scriptType string) (ScriptInstances, error) {
	ins, ok := factory[strings.ToLower(scriptType)]
	if !ok {
		return nil, errors.New("script type not be loaded: " + scriptType)
	}
	return ins, nil
}

func RangeInstances(f func(instance ScriptInstances) bool) {
	for _, instance := range factory {
		if !f(instance) {
			break
		}
	}
}

func setInstances(tpName string, instance ScriptInstances) {
	factory[tpName] = instance
}

func init() {
	factory = make(map[string]ScriptInstances)
	setInstances(`javascript`, newJsInstances())
}
