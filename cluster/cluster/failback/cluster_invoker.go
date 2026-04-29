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

package failback

import (
	"context"
	"strconv"
	"sync"
	"time"
)

import (
	"github.com/Workiva/go-datastructures/queue"

	"github.com/cenkalti/backoff/v4"

	"github.com/dubbogo/gost/log/logger"
)

import (
	"dubbo.apache.org/dubbo-go/v3/cluster/cluster/base"
	"dubbo.apache.org/dubbo-go/v3/cluster/directory"
	"dubbo.apache.org/dubbo-go/v3/cluster/loadbalance"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	protocolbase "dubbo.apache.org/dubbo-go/v3/protocol/base"
	"dubbo.apache.org/dubbo-go/v3/protocol/result"
)

/**
 * When fails, record failure requests and schedule for retry on a regular interval.
 * Especially useful for services of notification.
 *
 * <a href="http://en.wikipedia.org/wiki/Failback">Failback</a>
 */
type failbackClusterInvoker struct {
	base.BaseClusterInvoker

	once          sync.Once
	ticker        *time.Ticker
	maxRetries    int64
	failbackTasks int64
	taskList      *queue.Queue
}

func newFailbackClusterInvoker(directory directory.Directory) protocolbase.Invoker {
	invoker := &failbackClusterInvoker{
		BaseClusterInvoker: base.NewBaseClusterInvoker(directory),
	}
	retriesConfig := invoker.GetURL().GetParam(constant.RetriesKey, constant.DefaultFailbackTimes)
	retries, err := strconv.Atoi(retriesConfig)
	if err != nil || retries < 0 {
		logger.Error("[Failback] retries config invalid, using default")
		retries = constant.DefaultFailbackTimesInt
	}

	failbackTasksConfig := invoker.GetURL().GetParamInt(constant.FailBackTasksKey, constant.DefaultFailbackTasks)
	if failbackTasksConfig <= 0 {
		failbackTasksConfig = constant.DefaultFailbackTasks
	}
	invoker.maxRetries = int64(retries)
	invoker.failbackTasks = failbackTasksConfig
	return invoker
}

func (invoker *failbackClusterInvoker) tryTimerTaskProc(ctx context.Context, retryTask *retryTimerTask) {
	invoked := make([]protocolbase.Invoker, 0)
	invoked = append(invoked, retryTask.lastInvoker)

	retryInvoker := invoker.DoSelect(retryTask.loadbalance, retryTask.invocation, retryTask.invokers, invoked)
	res := retryInvoker.Invoke(ctx, retryTask.invocation)
	if res.Error() != nil {
		retryTask.lastInvoker = retryInvoker
		retryTask.lastErr = res.Error()
		retryTask.checkRetry()
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
			// use exponential backoff calculated wait time instead of fixed 5 seconds
			if time.Since(retryTask.lastT) < retryTask.nextBackoff {
				break
			}

			// ignore return. the get must success.
			if _, err = invoker.taskList.Get(1); err != nil {
				logger.Warnf("[Failback] get task failed, err=%v", err)
				break
			}
			go invoker.tryTimerTaskProc(ctx, retryTask)
		}
	}
}

// Invoke executes with failback semantics: schedule retries on failure.
func (invoker *failbackClusterInvoker) Invoke(ctx context.Context, invocation protocolbase.Invocation) result.Result {
	invokers := invoker.Directory.List(invocation)
	if err := invoker.CheckInvokers(invokers, invocation); err != nil {
		logger.Errorf("[Failback] check invokers failed, method=%s service=%s err=%v",
			invocation.MethodName(), invoker.GetURL().Service(), err)
		return &result.RPCResult{}
	}

	// Get the service loadbalance config
	url := invokers[0].GetURL()
	lb := url.GetParam(constant.LoadbalanceKey, constant.DefaultLoadBalance)
	// Get the service method loadbalance config if have
	methodName := invocation.MethodName()
	if v := url.GetMethodParam(methodName, constant.LoadbalanceKey, ""); v != "" {
		lb = v
	}

	loadBalance := extension.GetLoadbalance(lb)
	invoked := make([]protocolbase.Invoker, 0, len(invokers))
	ivk := invoker.DoSelect(loadBalance, invocation, invokers, invoked)
	// DO INVOKE
	res := ivk.Invoke(ctx, invocation)
	if res.Error() != nil {
		invoker.once.Do(func() {
			invoker.taskList = queue.New(invoker.failbackTasks)
			go invoker.process(ctx)
		})

		taskLen := invoker.taskList.Len()
		if taskLen >= invoker.failbackTasks {
			logger.Warnf("[Failback] task list full, len=%d", taskLen)
			return &result.RPCResult{}
		}

		timerTask := newRetryTimerTask(loadBalance, invocation, invokers, ivk, invoker)
		invoker.taskList.Put(timerTask)

		logger.Errorf("[Failback] invoke failed, method=%s service=%s err=%v",
			methodName, url.Service(), res.Error().Error())
		// ignore
		return &result.RPCResult{}
	}
	return res
}

func (invoker *failbackClusterInvoker) Destroy() {
	invoker.BaseClusterInvoker.Destroy()

	// stop ticker
	if invoker.ticker != nil {
		invoker.ticker.Stop()
	}

	_ = invoker.taskList.Dispose()
}

type retryTimerTask struct {
	loadbalance    loadbalance.LoadBalance
	invocation     protocolbase.Invocation
	invokers       []protocolbase.Invoker
	lastInvoker    protocolbase.Invoker
	retries        int64
	maxRetries     int64
	lastT          time.Time
	nextBackoff    time.Duration               // next retry wait duration
	backoff        *backoff.ExponentialBackOff // exponential backoff calculator
	clusterInvoker *failbackClusterInvoker
	lastErr        error
}

func (t *retryTimerTask) checkRetry() {
	logger.Errorf("[Failback] retry failed, method=%s service=%s err=%v",
		t.invocation.MethodName(), t.clusterInvoker.GetURL().Service(), t.lastErr)
	t.retries++
	t.nextBackoff = t.backoff.NextBackOff() // calculate next exponential backoff wait time

	if t.retries > t.maxRetries || t.nextBackoff == backoff.Stop {
		logger.Errorf("[Failback] retry exceeded, retries=%d invocation=%v",
			t.retries, t.invocation)
		return
	}

	logger.Infof("[Failback] retry scheduled, backoff=%v method=%s", t.nextBackoff, t.invocation.MethodName())

	if err := t.clusterInvoker.taskList.Put(t); err != nil {
		logger.Errorf("[Failback] put task failed, task=%v err=%v", t, err)
		return
	}
	t.lastT = time.Now() // update lastT after successful Put
}

func newRetryTimerTask(loadbalance loadbalance.LoadBalance, invocation protocolbase.Invocation, invokers []protocolbase.Invoker,
	lastInvoker protocolbase.Invoker, cInvoker *failbackClusterInvoker) *retryTimerTask {
	bo := backoff.NewExponentialBackOff()
	bo.InitialInterval = 1 * time.Second
	bo.MaxInterval = 60 * time.Second
	bo.MaxElapsedTime = 0 // never timeout

	task := &retryTimerTask{
		loadbalance:    loadbalance,
		invocation:     invocation,
		invokers:       invokers,
		lastInvoker:    lastInvoker,
		lastT:          time.Now(),
		backoff:        bo,
		nextBackoff:    bo.NextBackOff(),
		clusterInvoker: cInvoker,
	}

	if retries, ok := invocation.GetAttachment(constant.RetriesKey); ok {
		rInt, _ := strconv.Atoi(retries)
		task.maxRetries = int64(rInt)
	} else {
		task.maxRetries = cInvoker.maxRetries
	}

	return task
}
