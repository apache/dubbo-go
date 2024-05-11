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
	"sync"
	"sync/atomic"
)

const (
	jsScriptResultName = `__go_program_result`
	jsScriptPrefix     = "\n" + jsScriptResultName + ` = `
)

/*
The expect get js script is like

"

	(function route(invokers,invocation,context) {
		var result = [];
		for (var i = 0; i < invokers.length; i++) {
		    if ("127.0.0.1" === invokers[i].GetURL().Ip) {
				if (invokers[i].GetURL().Port !== "20000"){
					invokers[i].GetURL().Ip = "anotherIP"
					result.push(invokers[i]);
				}
		    }
		}
		return result;
	}(invokers,invocation,context));

"
---
  - Supports method calling.
    Parameter methods are mapped by the passed in type.
    The first letter is capitalized (does not comply with js specifications)

like `invokers[i].GetURL().SetParam("testKey","testValue")`

---
  - Like the Go language, it supports direct access to
    exportable variables within parameters.

like `invokers[i].GetURL().Port`

---
  - important! The parameters passed in will be references,
    Changing the parameters will change the value passed in the go language

---
  - The expected way to get the return value is
    like `var result = []; result.push(invokers[i]);`
*/
type jsInstances struct {
	insPool *sync.Pool                   // store *goja.runtime
	program atomic.Pointer[goja.Program] // applicationName to compiledProgram
}

func newJsInstances() *jsInstances {
	return &jsInstances{
		insPool: &sync.Pool{New: func() any {
			return newJsMather()
		}},
	}
}

type jsInstance struct {
	rt *goja.Runtime
}

func (i *jsInstances) RunScript(_ string, invokers []protocol.Invoker, invocation protocol.Invocation) ([]protocol.Invoker, error) {
	pg := i.program.Load()
	if pg == nil || len(invokers) == 0 {
		return invokers, nil
	}
	matcher := i.insPool.Get().(*jsInstance)

	packInvokers := make([]protocol.Invoker, 0, len(invokers))
	for _, invoker := range invokers {
		packInvokers = append(packInvokers, newScriptInvokerImpl(invoker))
	}

	matcher.initCallArgs(packInvokers, invocation)
	matcher.initReplyVar()
	scriptRes, err := matcher.runScript(i.program.Load())
	if err != nil {
		return nil, err
	}

	result := make([]protocol.Invoker, 0, len(scriptRes.(*goja.Object).Export().([]interface{})))
	for _, res := range scriptRes.(*goja.Object).Export().([]interface{}) {
		res.(*scriptInvokerPackImpl).setRanMode()
		result = append(result, res.(protocol.Invoker))
	}
	return result, nil
}

func (i *jsInstances) Compile(key, rawScript string) error {
	pg, err := goja.Compile(key+`_jsScriptRoute`, jsScriptPrefix+rawScript, true)
	if err != nil {
		return err
	}
	i.program.Store(pg)
	return nil
}

func (i *jsInstances) Destroy() {
	i.program.Store(nil)
}

func (j jsInstance) initCallArgs(invokers []protocol.Invoker, invocation protocol.Invocation) {
	j.rt.ClearInterrupt()
	err := j.rt.Set(`invokers`, invokers)
	if err != nil {
		panic(err)
	}
	err = j.rt.Set(`invocation`, invocation)
	if err != nil {
		panic(err)
	}
	err = j.rt.Set(`context`, invocation.GetAttachmentAsContext())
	if err != nil {
		panic(err)
	}
}

// must be set, or throw err like `var jsScriptResultName` not define
func (j jsInstance) initReplyVar() {
	err := j.rt.Set(jsScriptResultName, nil)
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
