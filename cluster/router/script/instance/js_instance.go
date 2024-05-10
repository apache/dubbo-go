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
	"github.com/dop251/goja"
)

type jsInstance struct {
	rt *goja.Runtime
}

func (j jsInstance) initCallArgs(invokers []protocol.Invoker, invocation protocol.Invocation) {
	j.rt.ClearInterrupt()
	err := j.rt.Set(`invokers`, invokers)
	err = j.rt.Set(`invocation`, invocation)
	err = j.rt.Set(`context`, invocation.GetAttachmentAsContext())
	if err != nil {
		panic(err)
	}
}

// must be set, or throw err like `js_script_result_name` not define
func (j jsInstance) initReplyVar() {
	err := j.rt.Set(js_script_result_name, nil)
	if err != nil {
		panic(err)
	}
}

func newJsMather() *jsInstance {
	return &jsInstance{
		rt: goja.New(),
	}
}

func (j jsInstance) runScript(pg *goja.Program) (interface{}, error) {
	return j.rt.RunProgram(pg)
}
