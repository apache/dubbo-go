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
	"github.com/dop251/goja"
	"go.uber.org/atomic"
	"strings"
	"sync"
)

const (
	js_script_result_name = `__go_program_get_result`
	js_                   = "\n" + js_script_result_name + ` = route(invokers, invocation, context)`
)

type jsInstances struct {
	insPool *sync.Pool                   // store *goja.runtime
	program atomic.Pointer[goja.Program] // applicationName to compiledProgram
}

func newJsInstance() *jsInstances {
	return &jsInstances{
		insPool: &sync.Pool{New: func() any {
			return newJsMather()
		}},
	}
}

func (i *jsInstances) RunScript(_ string, invokers []protocol.Invoker, invocation protocol.Invocation) ([]protocol.Invoker, error) {
	matcher := i.insPool.Get().(*jsInstance)
	matcher.initCallArgs(invokers, invocation)
	matcher.initReplyVar()
	res, err := matcher.runScript(i.program.Load())
	if err != nil {
		return nil, err
	}
	result := make([]protocol.Invoker, 0)
	for _, ress := range res.([]interface{}) {
		result = append(result, ress.(protocol.Invoker))
	}
	return result, nil
}

func (i *jsInstances) Compile(rawScript string) error {
	pg, err := goja.Compile(`jsScriptRoute`, rawScript+js_, true)
	if err != nil {
		return err
	}
	i.program.Store(pg)
	return nil
}

type ScriptInstance interface {
	RunScript(rawScript string, invokers []protocol.Invoker, invocation protocol.Invocation) ([]protocol.Invoker, error)
	Compile(rawScript string) error
}

var factory map[string]ScriptInstance

func GetInstance(scriptType string) (ScriptInstance, error) {
	ins, ok := factory[strings.ToLower(scriptType)]
	if !ok {
		return nil, errors.New("script type not exist: " + scriptType)
	}
	return ins, nil
}

func setInstance(tpName string, instance ScriptInstance) {
	factory[tpName] = instance
}

func init() {
	setInstance(`javascript`, newJsInstance())
}
