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
	"dubbo.apache.org/dubbo-go/v3/protocol/base"
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
	// baseGlobals is the set of global names present in a freshly created
	// runtime (built-ins). Any global outside this set is treated as
	// request/script state and removed on reset.
	baseGlobals map[string]struct{}
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

func (i *jsInstances) Run(rawScript string, invokers []base.Invoker, invocation base.Invocation) ([]base.Invoker, error) {
	i.pgLock.RLock()
	pg, ok := i.program[rawScript]
	i.pgLock.RUnlock()

	if !ok || len(invokers) == 0 {
		return invokers, nil
	}
	matcher := i.insPool.Get().(*jsInstance)
	defer func() {
		matcher.reset()
		i.insPool.Put(matcher)
	}()

	packInvokers := make([]base.Invoker, 0, len(invokers))
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

	rtInvokersArr, ok := scriptRes.(*goja.Object).Export().([]any)
	if !ok {
		return invokers, fmt.Errorf("script result is not array , script return type: %s", reflect.ValueOf(scriptRes.(*goja.Object).Export()).String())
	}

	result := make([]base.Invoker, 0, len(rtInvokersArr))
	for _, res := range rtInvokersArr {
		if i, ok := res.(*scriptInvokerWrapper); ok {
			i.setRanMode()
			result = append(result, res.(base.Invoker))
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

func (j jsInstance) initCallArgs(invokers []base.Invoker, invocation base.Invocation, ctx context.Context) {
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

// reset removes every global added since the runtime was created — the request
// bindings (invokers/invocation/context/result) as well as any globals defined
// by the executed user script — before returning the instance to the pool. This
// prevents cross-request state leakage via pooled goja.Runtime globals. Built-in
// globals captured at construction time are preserved.
func (j jsInstance) reset() {
	global := j.rt.GlobalObject()
	for _, key := range global.Keys() {
		if _, isBase := j.baseGlobals[key]; isBase {
			continue
		}
		_ = global.Delete(key)
	}
}

func newJsInstance() *jsInstance {
	rt := goja.New()
	base := make(map[string]struct{})
	for _, key := range rt.GlobalObject().Keys() {
		base[key] = struct{}{}
	}
	return &jsInstance{
		rt:          rt,
		baseGlobals: base,
	}
}

func (j jsInstance) runScript(pg *goja.Program) (res any, err error) {
	defer func(res *any, err *error) {
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
