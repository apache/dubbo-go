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

package cluster_impl

import (
	"container/list"
	perrors "github.com/pkg/errors"
	"sync"
	"time"
)

import (
	"github.com/apache/dubbo-go/cluster"
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/common/extension"
	"github.com/apache/dubbo-go/protocol"
)

type failbackClusterInvoker struct {
	baseClusterInvoker
}

var (
	retries       int64
	failbackTasks int64
	ticker        *time.Ticker
	once          sync.Once
	lock          sync.Mutex
	taskList      *Queue
)

func newFailbackClusterInvoker(directory cluster.Directory) protocol.Invoker {
	invoker := &failbackClusterInvoker{
		baseClusterInvoker: newBaseClusterInvoker(directory),
	}
	retriesConfig := invoker.GetUrl().GetParamInt(constant.RETRIES_KEY, constant.DEFAULT_FAILBACK_TIMES)
	if retriesConfig <= 0 {
		retriesConfig = constant.DEFAULT_FAILBACK_TIMES
	}
	failbackTasksConfig := invoker.GetUrl().GetParamInt(constant.FAIL_BACK_TASKS_KEY, constant.DEFAULT_FAILBACK_TASKS)
	if failbackTasksConfig <= 0 {
		failbackTasksConfig = constant.DEFAULT_FAILBACK_TASKS
	}
	retries = retriesConfig
	failbackTasks = failbackTasksConfig
	return invoker
}

func (invoker *failbackClusterInvoker) Invoke(invocation protocol.Invocation) protocol.Result {

	invokers := invoker.directory.List(invocation)
	err := invoker.checkInvokers(invokers, invocation)

	if err != nil {
		// add retry ticker task
		perrors.Errorf("Failed to invoke the method %v in the service %v, wait for retry in background. Ignored exception: %v.",
			invocation.MethodName(), invoker.GetUrl().Service(), err)
		return &protocol.RPCResult{}
	}
	url := invokers[0].GetUrl()

	methodName := invocation.MethodName()
	//Get the service loadbalance config
	lb := url.GetParam(constant.LOADBALANCE_KEY, constant.DEFAULT_LOADBALANCE)

	//Get the service method loadbalance config if have
	if v := url.GetMethodParam(methodName, constant.LOADBALANCE_KEY, ""); v != "" {
		lb = v
	}
	loadbalance := extension.GetLoadbalance(lb)

	invoked := []protocol.Invoker{}
	var result protocol.Result

	ivk := invoker.doSelect(loadbalance, invocation, invokers, invoked)
	invoked = append(invoked, ivk)
	//DO INVOKE
	result = ivk.Invoke(invocation)

	if result.Error() != nil {
		// add retry ticker task
		addFailed(loadbalance, invocation, invokers, invoker)
		perrors.Errorf("Failback to invoke the method %v in the service %v, wait for retry in background. Ignored exception: %v.",
			methodName, invoker.GetUrl().Service(), result.Error().Error())
		// ignore
		return &protocol.RPCResult{}
	}

	return result
}

func (invoker *failbackClusterInvoker) Destroy() {
	//this is must atom operation
	if invoker.destroyed.CAS(false, true) {
		invoker.directory.Destroy()
	}
	// stop ticker
	ticker.Stop()
}

func addFailed(balance cluster.LoadBalance, invocation protocol.Invocation, invokers []protocol.Invoker,
	invoker *failbackClusterInvoker) {
	initSingleTickerTaskInstance()
	// init one retryTimerTask
	timerTask := newRetryTimerTask(balance, invocation, invokers, invoker, retries, 5)
	taskList.push(timerTask)
	// process ticker task
	go func() {
		<-ticker.C
		value := taskList.pop()
		if value == nil {
			return
		}

		retryTask := value.(retryTimerTask)
		invoked := []protocol.Invoker{}
		invoked = append(invoked, retryTask.lastInvoker)
		retryInvoker := invoker.doSelect(retryTask.loadbalance, retryTask.invocation, retryTask.invokers,
			invoked)
		var result protocol.Result
		result = retryInvoker.Invoke(retryTask.invocation)
		if result.Error() != nil {
			perrors.Errorf("Failed retry to invoke the method %v in the service %v, wait again. The exception: %v.",
				invocation.MethodName(), invoker.GetUrl().Service(), result.Error().Error())
			retryTask.retries++
			if retryTask.retries > retries {
				perrors.Errorf("Failed retry times exceed threshold (%v), We have to abandon, invocation-> %v",
					retries, invocation)
			} else {
				taskList.push(retryTask)
			}
		}
	}()
}

func initSingleTickerTaskInstance() {
	once.Do(func() {
		newTickerTask()
	})
}

func newTickerTask() {
	ticker = time.NewTicker(time.Second * 1)
	taskList = newQueue()
}

type retryTimerTask struct {
	loadbalance cluster.LoadBalance
	invocation  protocol.Invocation
	invokers    []protocol.Invoker
	lastInvoker *failbackClusterInvoker
	retries     int64
	tick        int64
}

func newRetryTimerTask(loadbalance cluster.LoadBalance, invocation protocol.Invocation, invokers []protocol.Invoker,
	lastInvoker *failbackClusterInvoker, retries int64, tick int64) *retryTimerTask {
	return &retryTimerTask{
		loadbalance: loadbalance,
		invocation:  invocation,
		invokers:    invokers,
		lastInvoker: lastInvoker,
		retries:     retries,
		tick:        tick,
	}
}

type Queue struct {
	data *list.List
}

func newQueue() *Queue {
	q := new(Queue)
	q.data = list.New()
	return q
}

func (q *Queue) push(v interface{}) {
	defer lock.Unlock()
	lock.Lock()
	q.data.PushFront(v)
}

func (q *Queue) pop() interface{} {
	defer lock.Unlock()
	lock.Lock()
	iter := q.data.Back()
	v := iter.Value
	q.data.Remove(iter)
	return v
}
