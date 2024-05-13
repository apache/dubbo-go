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

	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/protocol"
)

func init() {
	factory = make(map[string]ScriptInstances)
	setInstances(`javascript`, newJsInstances())
}

type ScriptInstances interface {
	Run(rawScript string, invokers []protocol.Invoker, invocation protocol.Invocation) ([]protocol.Invoker, error)
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

// scriptInvokerPack for security
// if script change input Invoker's url during Route() call ,
// it will influence call Route() next time ,
// there are no operation to recover .
type scriptInvokerPack struct {
	isRan     bool
	copiedURL *common.URL
	invoker   protocol.Invoker
}

func (f *scriptInvokerPack) GetURL() *common.URL {
	return f.copiedURL
}

func (f *scriptInvokerPack) IsAvailable() bool {
	if !f.isRan {
		return true
	} else {
		return f.invoker.IsAvailable()
	}
}

func (f *scriptInvokerPack) Destroy() {
	if !f.isRan {
		panic("Destroy should not be called")
	} else {
		f.invoker.Destroy()
	}
}

func (f *scriptInvokerPack) Invoke(ctx context.Context, inv protocol.Invocation) protocol.Result {
	if !f.isRan {
		panic("Invoke should not be called")
	} else {
		return f.invoker.Invoke(ctx, inv)
	}
}

func (f *scriptInvokerPack) setRanMode() {
	f.isRan = true
}

func newScriptInvokerImpl(invoker protocol.Invoker) *scriptInvokerPack {
	return &scriptInvokerPack{
		copiedURL: invoker.GetURL().Clone(),
		invoker:   invoker,
		isRan:     false,
	}
}
