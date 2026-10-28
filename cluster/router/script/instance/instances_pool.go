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
	"errors"
	"strings"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/protocol/base"
	"dubbo.apache.org/dubbo-go/v3/protocol/result"
)

func init() {
	factory = make(map[string]ScriptInstances)
	setInstances(`javascript`, newJsInstances())
}

// ScriptInstances is the engine bound to one script type. It owns the whole
// lifecycle of scripts of that type: compile, run and destroy.
// Compile compiles the script and increments its reference count; compiling
// the same script again only increments the count.
// Run executes a compiled script; Run returns the original invokers for a
// script that was never compiled or has been destroyed.
// Destroy decrements the reference count of the compiled script; the entry
// is removed from the cache once the count drops to zero.
type ScriptInstances interface {
	Run(rawScript string, invokers []base.Invoker, invocation base.Invocation) ([]base.Invoker, error)
	Compile(rawScript string) error
	Destroy(rawScript string)
}

var factory map[string]ScriptInstances

// GetInstances returns the engine instance for the given script type.
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

// setInstances sets the engine instance for the given script type.
func setInstances(tpName string, instance ScriptInstances) {
	factory[tpName] = instance
}

// scriptInvokerWrapper for security
// if script change input Invoker's url during Route() call ,
// it will influence call Route() next time ,
// there are no operation to recover .
type scriptInvokerWrapper struct {
	isRan     bool
	copiedURL *common.URL
	invoker   base.Invoker
}

func (f *scriptInvokerWrapper) GetURL() *common.URL {
	return f.copiedURL
}

func (f *scriptInvokerWrapper) IsAvailable() bool {
	if !f.isRan {
		return true
	} else {
		return f.invoker.IsAvailable()
	}
}

func (f *scriptInvokerWrapper) Destroy() {
	if !f.isRan {
		panic("Destroy should not be called")
	} else {
		f.invoker.Destroy()
	}
}

func (f *scriptInvokerWrapper) Invoke(ctx context.Context, inv base.Invocation) result.Result {
	if !f.isRan {
		panic("Invoke should not be called")
	} else {
		return f.invoker.Invoke(ctx, inv)
	}
}

func (f *scriptInvokerWrapper) setRanMode() {
	f.isRan = true
}

func newScriptInvokerImpl(invoker base.Invoker) *scriptInvokerWrapper {
	return &scriptInvokerWrapper{
		copiedURL: invoker.GetURL().Clone(),
		invoker:   invoker,
		isRan:     false,
	}
}
