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
	"errors"
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

	maxRetries    int64
	failbackTasks int64

	lifecycleMu   sync.Mutex
	stopped       bool
	taskList      *queue.Queue
	retryCancel   context.CancelFunc
	processDone   chan struct{}
	retryDone     chan struct{}
	activeRetries int
	destroyOnce   sync.Once
}

var errFailbackInvokerStopped = errors.New("failback invoker is stopped")

func newFailbackClusterInvoker(directory directory.Directory) protocolbase.Invoker {
	invoker := &failbackClusterInvoker{
		BaseClusterInvoker: base.NewBaseClusterInvoker(directory),
	}
	retriesConfig := invoker.GetURL().GetParam(constant.RetriesKey, constant.DefaultFailbackTimes)
	retries, err := strconv.Atoi(retriesConfig)
	if err != nil || retries < 0 {
		logger.Error("[Cluster][Failback] retries config invalid, using default")
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
	if ctx.Err() != nil {
		return
	}

	invoked := make([]protocolbase.Invoker, 0)
	invoked = append(invoked, retryTask.lastInvoker)

	retryInvoker := invoker.DoSelect(retryTask.loadbalance, retryTask.invocation, retryTask.invokers, invoked)
	if retryInvoker == nil || ctx.Err() != nil {
		return
	}

	res := retryInvoker.Invoke(ctx, retryTask.invocation)
	if res.Error() != nil && ctx.Err() == nil {
		retryTask.lastInvoker = retryInvoker
		retryTask.lastErr = res.Error()
		retryTask.checkRetry()
	}
}

func (invoker *failbackClusterInvoker) process(ctx context.Context, taskList *queue.Queue, done chan struct{}) {
	defer close(done)

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if invoker.processRetryTasks(ctx, taskList) {
				return
			}
		}
	}
}

func (invoker *failbackClusterInvoker) processRetryTasks(ctx context.Context, taskList *queue.Queue) bool {
	for {
		select {
		case <-ctx.Done():
			return true
		default:
		}

		value, err := taskList.Peek()
		if err == queue.ErrDisposed {
			return true
		}
		if err == queue.ErrEmptyQueue {
			return false
		}
		if err != nil {
			logger.Warnf("[Cluster][Failback] peek task failed, err=%v", err)
			return false
		}

		retryTask := value.(*retryTimerTask)
		// use exponential backoff calculated wait time instead of fixed 5 seconds
		if time.Since(retryTask.lastT) < retryTask.nextBackoff {
			return false
		}

		// ignore return. the get must success.
		if _, err = taskList.Get(1); err != nil {
			logger.Warnf("[Cluster][Failback] get task failed, err=%v", err)
			return false
		}
		invoker.startRetry(ctx, retryTask)
	}
}

// Invoke executes with failback semantics: schedule retries on failure.
func (invoker *failbackClusterInvoker) Invoke(ctx context.Context, invocation protocolbase.Invocation) result.Result {
	if invoker.isStopped() {
		return &result.RPCResult{Err: errFailbackInvokerStopped}
	}
	if err := invoker.CheckWhetherDestroyed(); err != nil {
		return &result.RPCResult{Err: err}
	}

	invokers := invoker.Directory.List(invocation)
	if err := invoker.CheckInvokers(invokers, invocation); err != nil {
		logger.Errorf("[Cluster][Failback] check invokers failed, method=%s service=%s err=%v",
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
	if ivk == nil {
		return &result.RPCResult{Err: errors.New("invoker is nil")}
	}
	res := ivk.Invoke(ctx, invocation)
	if res.Error() != nil {
		timerTask := newRetryTimerTask(loadBalance, invocation, invokers, ivk, invoker)
		invoker.enqueueInitialRetry(ctx, timerTask)

		logger.Errorf("[Cluster][Failback] invoke failed, method=%s service=%s err=%v",
			methodName, url.Service(), res.Error().Error())
		// ignore
		return &result.RPCResult{}
	}
	return res
}

func (invoker *failbackClusterInvoker) isStopped() bool {
	invoker.lifecycleMu.Lock()
	defer invoker.lifecycleMu.Unlock()
	return invoker.stopped
}

func (invoker *failbackClusterInvoker) Destroy() {
	invoker.destroyOnce.Do(func() {
		invoker.lifecycleMu.Lock()
		invoker.stopped = true
		if invoker.retryCancel != nil {
			invoker.retryCancel()
		}
		taskList := invoker.taskList
		processDone := invoker.processDone
		retryDone := invoker.retryDone
		if taskList != nil {
			_ = taskList.Dispose()
		}
		invoker.lifecycleMu.Unlock()

		invoker.waitForShutdown(processDone, retryDone)
		invoker.BaseClusterInvoker.Destroy()
	})
}

func (invoker *failbackClusterInvoker) enqueueInitialRetry(ctx context.Context, retryTask *retryTimerTask) {
	invoker.lifecycleMu.Lock()
	defer invoker.lifecycleMu.Unlock()

	if invoker.stopped || invoker.Destroyed.Load() {
		return
	}

	if invoker.taskList == nil {
		if ctx == nil {
			ctx = context.Background()
		}
		retryCtx, retryCancel := context.WithCancel(context.WithoutCancel(ctx))
		invoker.retryCancel = retryCancel
		invoker.taskList = queue.New(invoker.failbackTasks)
		invoker.processDone = make(chan struct{})
		go invoker.process(retryCtx, invoker.taskList, invoker.processDone)
	}

	if invoker.taskList.Len() >= invoker.failbackTasks {
		logger.Warnf("[Cluster][Failback] task list full, len=%d", invoker.taskList.Len())
		return
	}

	if err := invoker.taskList.Put(retryTask); err != nil {
		logger.Warnf("[Cluster][Failback] put initial task failed, err=%v", err)
	}
}

func (invoker *failbackClusterInvoker) startRetry(ctx context.Context, retryTask *retryTimerTask) {
	invoker.lifecycleMu.Lock()
	defer invoker.lifecycleMu.Unlock()

	if invoker.stopped || ctx.Err() != nil {
		return
	}
	if invoker.activeRetries == 0 {
		invoker.retryDone = make(chan struct{})
	}
	invoker.activeRetries++
	retryDone := invoker.retryDone
	go func() {
		defer invoker.finishRetry(retryDone)
		invoker.tryTimerTaskProc(ctx, retryTask)
	}()
}

func (invoker *failbackClusterInvoker) finishRetry(retryDone chan struct{}) {
	invoker.lifecycleMu.Lock()
	defer invoker.lifecycleMu.Unlock()

	invoker.activeRetries--
	if invoker.activeRetries == 0 && invoker.retryDone == retryDone {
		close(retryDone)
	}
}

func (invoker *failbackClusterInvoker) enqueueRetry(retryTask *retryTimerTask) bool {
	invoker.lifecycleMu.Lock()
	defer invoker.lifecycleMu.Unlock()

	if invoker.stopped || invoker.taskList == nil {
		return false
	}

	retryTask.lastT = time.Now()
	if err := invoker.taskList.Put(retryTask); err != nil {
		logger.Warnf("[Cluster][Failback] put retry task failed, err=%v", err)
		return false
	}
	return true
}

func (invoker *failbackClusterInvoker) waitForShutdown(processDone, retryDone <-chan struct{}) {
	if processDone == nil && retryDone == nil {
		return
	}

	wait := func(done <-chan struct{}, name string) bool {
		if done == nil {
			return true
		}

		timer := time.NewTimer(constant.DefaultShutdownConfigStepTimeout)
		defer timer.Stop()

		select {
		case <-done:
			return true
		case <-timer.C:
			logger.Warnf("[Cluster][Failback] timed out waiting for %s shutdown", name)
			return false
		}
	}

	if !wait(processDone, "retry processor") {
		return
	}
	_ = wait(retryDone, "retry tasks")
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
	logger.Errorf("[Cluster][Failback] retry failed, method=%s service=%s err=%v",
		t.invocation.MethodName(), t.clusterInvoker.GetURL().Service(), t.lastErr)
	t.retries++
	t.nextBackoff = t.backoff.NextBackOff() // calculate next exponential backoff wait time

	if t.retries > t.maxRetries || t.nextBackoff == backoff.Stop {
		logger.Errorf("[Cluster][Failback] retry exceeded, retries=%d invocation=%v",
			t.retries, t.invocation)
		return
	}

	if !t.clusterInvoker.enqueueRetry(t) {
		return
	}
	logger.Infof("[Cluster][Failback] retry scheduled, backoff=%v method=%s", t.nextBackoff, t.invocation.MethodName())
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
