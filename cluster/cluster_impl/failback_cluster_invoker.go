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
	"context"
	"strconv"
	"sync"
	"time"
)

import (
	"github.com/Workiva/go-datastructures/queue"
)

import (
	"github.com/apache/dubbo-go/cluster"
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/common/extension"
	"github.com/apache/dubbo-go/common/logger"
	"github.com/apache/dubbo-go/protocol"
)

/**
 * When fails, record failure requests and schedule for retry on a regular interval.
 * Especially useful for services of notification.
 *
 * <a href="http://en.wikipedia.org/wiki/Failback">Failback</a>
 */
type failbackClusterInvoker struct {
	baseClusterInvoker

	once          sync.Once
	ticker        *time.Ticker
	maxRetries    int64
	failbackTasks int64
	taskList      *queue.Queue
}

func newFailbackClusterInvoker(directory cluster.Directory) protocol.Invoker {
	invoker := &failbackClusterInvoker{
		baseClusterInvoker: newBaseClusterInvoker(directory),
	}
	retriesConfig := invoker.GetUrl().GetParam(constant.RETRIES_KEY, constant.DEFAULT_FAILBACK_TIMES)
	retries, err := strconv.Atoi(retriesConfig)
	if err != nil || retries < 0 {
		logger.Error("Your retries config is invalid,pls do a check. And will use the default fail back times configuration instead.")
		retries = constant.DEFAULT_FAILBACK_TIMES_INT
	}

	failbackTasksConfig := invoker.GetUrl().GetParamInt(constant.FAIL_BACK_TASKS_KEY, constant.DEFAULT_FAILBACK_TASKS)
	if failbackTasksConfig <= 0 {
		failbackTasksConfig = constant.DEFAULT_FAILBACK_TASKS
	}
	invoker.maxRetries = int64(retries)
	invoker.failbackTasks = failbackTasksConfig
	return invoker
}

func (invoker *failbackClusterInvoker) tryTimerTaskProc(ctx context.Context, retryTask *retryTimerTask) {
	invoked := make([]protocol.Invoker, 0)
	invoked = append(invoked, retryTask.lastInvoker)

	retryInvoker := invoker.doSelect(retryTask.loadbalance, retryTask.invocation, retryTask.invokers, invoked)
	var result protocol.Result
	result = retryInvoker.Invoke(ctx, retryTask.invocation)
	if result.Error() != nil {
		retryTask.lastInvoker = retryInvoker
		invoker.checkRetry(retryTask, result.Error())
	}
}

func (invoker *failbackClusterInvoker) process(ctx context.Context) {
	invoker.ticker = time.NewTicker(time.Second * 1)
	for range invoker.ticker.C {
		// check each timeout task and re-run
		for {
			value, err := invoker.taskList.Peek()
			if err == queue.ErrDisposed {
				return
			}
			if err == queue.ErrEmptyQueue {
				break
			}

			retryTask := value.(*retryTimerTask)
			if time.Since(retryTask.lastT).Seconds() < 5 {
				break
			}

			// ignore return. the get must success.
			if _, err = invoker.taskList.Get(1); err != nil {
				logger.Warnf("get task found err: %v\n", err)
				break
			}
			go invoker.tryTimerTaskProc(ctx, retryTask)
		}
	}
}

func (invoker *failbackClusterInvoker) checkRetry(retryTask *retryTimerTask, err error) {
	logger.Errorf("Failed retry to invoke the method %v in the service %v, wait again. The exception: %v.\n",
		retryTask.invocation.MethodName(), invoker.GetUrl().Service(), err.Error())
	retryTask.retries++
	retryTask.lastT = time.Now()
	if retryTask.retries > invoker.maxRetries {
		logger.Errorf("Failed retry times exceed threshold (%v), We have to abandon, invocation-> %v.\n",
			retryTask.retries, retryTask.invocation)
	} else {
		invoker.taskList.Put(retryTask)
	}
}

// nolint
func (invoker *failbackClusterInvoker) Invoke(ctx context.Context, invocation protocol.Invocation) protocol.Result {
	invokers := invoker.directory.List(invocation)
	if err := invoker.checkInvokers(invokers, invocation); err != nil {
		logger.Errorf("Failed to invoke the method %v in the service %v, wait for retry in background. Ignored exception: %v.\n",
			invocation.MethodName(), invoker.GetUrl().Service(), err)
		return &protocol.RPCResult{}
	}

	//Get the service loadbalance config
	url := invokers[0].GetUrl()
	lb := url.GetParam(constant.LOADBALANCE_KEY, constant.DEFAULT_LOADBALANCE)
	//Get the service method loadbalance config if have
	methodName := invocation.MethodName()
	if v := url.GetMethodParam(methodName, constant.LOADBALANCE_KEY, ""); v != "" {
		lb = v
	}

	loadBalance := extension.GetLoadbalance(lb)
	invoked := make([]protocol.Invoker, 0, len(invokers))
	ivk := invoker.doSelect(loadBalance, invocation, invokers, invoked)
	//DO INVOKE
	result := ivk.Invoke(ctx, invocation)
	if result.Error() != nil {
		invoker.once.Do(func() {
			invoker.taskList = queue.New(invoker.failbackTasks)
			go invoker.process(ctx)
		})

		taskLen := invoker.taskList.Len()
		if taskLen >= invoker.failbackTasks {
			logger.Warnf("tasklist is too full > %d.\n", taskLen)
			return &protocol.RPCResult{}
		}

		timerTask := newRetryTimerTask(loadBalance, invocation, invokers, ivk)
		invoker.taskList.Put(timerTask)

		logger.Errorf("Failback to invoke the method %v in the service %v, wait for retry in background. Ignored exception: %v.\n",
			methodName, url.Service(), result.Error().Error())
		// ignore
		return &protocol.RPCResult{}
	}
	return result
}

func (invoker *failbackClusterInvoker) Destroy() {
	invoker.baseClusterInvoker.Destroy()

	// stop ticker
	if invoker.ticker != nil {
		invoker.ticker.Stop()
	}

	_ = invoker.taskList.Dispose()
}

type retryTimerTask struct {
	loadbalance cluster.LoadBalance
	invocation  protocol.Invocation
	invokers    []protocol.Invoker
	lastInvoker protocol.Invoker
	retries     int64
	lastT       time.Time
}

func newRetryTimerTask(loadbalance cluster.LoadBalance, invocation protocol.Invocation, invokers []protocol.Invoker,
	lastInvoker protocol.Invoker) *retryTimerTask {
	return &retryTimerTask{
		loadbalance: loadbalance,
		invocation:  invocation,
		invokers:    invokers,
		lastInvoker: lastInvoker,
		lastT:       time.Now(),
	}
}
