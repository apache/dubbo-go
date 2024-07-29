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
	"context"
	"fmt"
	"reflect"
	"sync"
	"sync/atomic"
)

import (
	"github.com/dop251/goja"
)

import (
	"dubbo.apache.org/dubbo-go/v3/protocol"
)

const (
	jsScriptResultName = `__go_program_result`
	jsScriptPrefix     = "\n" + jsScriptResultName + ` = `
)

type jsInstances struct {
	insPool *sync.Pool // store *goja.runtime
	pgLock  sync.RWMutex
	program map[string]*program // rawScript to compiledProgram
}

type jsInstance struct {
	rt *goja.Runtime
}

type program struct {
	pg    *goja.Program
	count int32
}

func newProgram(pg *goja.Program) *program {
	return &program{
		pg:    pg,
		count: 1,
	}
}

func (p *program) addCount(i int) int {
	return int(atomic.AddInt32(&p.count, int32(i)))
}

func newJsInstances() *jsInstances {
	return &jsInstances{
		program: map[string]*program{},
		insPool: &sync.Pool{New: func() any {
			return newJsInstance()
		}},
	}
}

func (i *jsInstances) Run(rawScript string, invokers []protocol.Invoker, invocation protocol.Invocation) ([]protocol.Invoker, error) {
	i.pgLock.RLock()
	pg, ok := i.program[rawScript]
	i.pgLock.RUnlock()

	if !ok || len(invokers) == 0 {
		return invokers, nil
	}
	matcher := i.insPool.Get().(*jsInstance)

	packInvokers := make([]protocol.Invoker, 0, len(invokers))
	for _, invoker := range invokers {
		packInvokers = append(packInvokers, newScriptInvokerImpl(invoker))
	}

	ctx := invocation.GetAttachmentAsContext()
	matcher.initCallArgs(packInvokers, invocation, ctx)
	matcher.initReplyVar()
	scriptRes, err := matcher.runScript(pg.pg)
	if err != nil {
		return invokers, err
	}
	invocation.MergeAttachmentFromContext(ctx)

	rtInvokersArr, ok := scriptRes.(*goja.Object).Export().([]interface{})
	if !ok {
		return invokers, fmt.Errorf("script result is not array , script return type: %s", reflect.ValueOf(scriptRes.(*goja.Object).Export()).String())
	}

	result := make([]protocol.Invoker, 0, len(rtInvokersArr))
	for _, res := range rtInvokersArr {
		if i, ok := res.(*scriptInvokerWrapper); ok {
			i.setRanMode()
			result = append(result, res.(protocol.Invoker))
		} else {
			return invokers, fmt.Errorf("invalid element type , it should be (*scriptInvokerWrapper) , but type is : %s)", reflect.TypeOf(res).String())
		}
	}
	return result, nil
}

func (i *jsInstances) Compile(rawScript string) error {
	var (
		ok bool
		pg *program
	)

	i.pgLock.RLock()
	pg, ok = i.program[rawScript]
	i.pgLock.RUnlock()

	if ok {
		pg.addCount(1)
		return nil
	} else {

		i.pgLock.Lock()
		defer i.pgLock.Unlock()
		// double check to avoid race
		if pg, ok = i.program[rawScript]; ok {
			pg.addCount(1)
		} else {
			newPg, err := goja.Compile("", jsScriptPrefix+rawScript, true)
			if err != nil {
				return err
			}
			i.program[rawScript] = newProgram(newPg)
		}
		return nil
	}
}

func (i *jsInstances) Destroy(rawScript string) {
	i.pgLock.Lock()
	if pg, ok := i.program[rawScript]; ok {
		if pg.addCount(-1) == 0 {
			delete(i.program, rawScript)
		}
	}
	i.pgLock.Unlock()
}

func (j jsInstance) initCallArgs(invokers []protocol.Invoker, invocation protocol.Invocation, ctx context.Context) {
	j.rt.ClearInterrupt()
	err := j.rt.Set(`invokers`, invokers)
	if err != nil {
		panic(err)
	}
	err = j.rt.Set(`invocation`, invocation)
	if err != nil {
		panic(err)
	}
	err = j.rt.Set(`context`, ctx)
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

func newJsInstance() *jsInstance {
	return &jsInstance{
		rt: goja.New(),
	}
}

func (j jsInstance) runScript(pg *goja.Program) (res interface{}, err error) {
	defer func(res *interface{}, err *error) {
		if panicReason := recover(); panicReason != nil {
			*res = nil
			switch panicReason := panicReason.(type) {
			case string:
				*err = fmt.Errorf("panic: %s", panicReason)
			case error:
				*err = panicReason
			default:
				*err = fmt.Errorf("panic occurred: failed to retrieve panic information. Expected string or error, got %T", panicReason)
			}
		}
	}(&res, &err)
	res, err = j.rt.RunProgram(pg)
	return res, err
}
