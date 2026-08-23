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
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

import (
	"github.com/golang/mock/gomock"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

import (
	clusterpkg "dubbo.apache.org/dubbo-go/v3/cluster/cluster"
	"dubbo.apache.org/dubbo-go/v3/cluster/directory/static"
	"dubbo.apache.org/dubbo-go/v3/cluster/loadbalance/random"
	_ "dubbo.apache.org/dubbo-go/v3/cluster/router/tag"
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/protocol/base"
	"dubbo.apache.org/dubbo-go/v3/protocol/invocation"
	"dubbo.apache.org/dubbo-go/v3/protocol/mock"
	"dubbo.apache.org/dubbo-go/v3/protocol/result"
)

var failbackUrl, _ = common.NewURL(
	fmt.Sprintf("dubbo://%s:%d/com.ikurento.user.UserProvider", constant.LocalHostValue, constant.DefaultPort))

// registerFailback register failbackCluster to failbackCluster extension.
func registerFailback(invoker *mock.MockInvoker) base.Invoker {
	extension.SetLoadbalance("random", random.NewRandomLoadBalance)
	failbackCluster := newFailbackCluster()

	var invokers []base.Invoker
	invokers = append(invokers, invoker)

	invoker.EXPECT().GetURL().Return(failbackUrl).AnyTimes()

	staticDir := static.NewDirectory(invokers)
	clusterInvoker := failbackCluster.Join(staticDir)
	return clusterInvoker
}

// success firstly, failback should return origin invoke result.
func TestFailbackSuceess(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	invoker := mock.NewMockInvoker(ctrl)
	clusterInvoker := registerFailback(invoker).(*failbackClusterInvoker)

	invoker.EXPECT().GetURL().Return(failbackUrl).AnyTimes()

	invoker.EXPECT().IsAvailable().Return(true).AnyTimes()

	mockResult := &result.RPCResult{Rest: clusterpkg.Rest{Tried: 0, Success: true}}
	invoker.EXPECT().Invoke(gomock.Any(), gomock.Any()).Return(mockResult).AnyTimes()

	result := clusterInvoker.Invoke(context.Background(), &invocation.RPCInvocation{})
	assert.Equal(t, mockResult, result)

	invoker.EXPECT().Destroy().Return()
	clusterInvoker.Destroy()
	clusterInvoker.Destroy()
}

func TestFailbackDestroyWithoutFailure(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	invoker := mock.NewMockInvoker(ctrl)
	clusterInvoker := registerFailback(invoker).(*failbackClusterInvoker)

	invoker.EXPECT().Destroy().Return()
	require.NotPanics(t, clusterInvoker.Destroy)
	require.NotPanics(t, clusterInvoker.Destroy)
}

func TestFailbackInvokeAfterDestroy(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	invoker := mock.NewMockInvoker(ctrl)
	clusterInvoker := registerFailback(invoker).(*failbackClusterInvoker)

	invoker.EXPECT().Destroy().Return()
	clusterInvoker.Destroy()

	result := clusterInvoker.Invoke(context.Background(), &invocation.RPCInvocation{})
	require.ErrorIs(t, result.Error(), errFailbackInvokerStopped)
}

func TestFailbackRetryUsesIndependentContext(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	invoker := mock.NewMockInvoker(ctrl)
	clusterInvoker := registerFailback(invoker).(*failbackClusterInvoker)

	invoker.EXPECT().GetURL().Return(failbackUrl).AnyTimes()
	invoker.EXPECT().IsAvailable().Return(true).AnyTimes()

	failedResult := &result.RPCResult{Err: errors.New("error")}
	successResult := &result.RPCResult{Rest: clusterpkg.Rest{Tried: 0, Success: true}}
	retryStarted := make(chan struct{})
	retryContextErr := make(chan error, 1)
	var callCount atomic.Int32

	invoker.EXPECT().Invoke(gomock.Any(), gomock.Any()).Times(2).DoAndReturn(
		func(ctx context.Context, _ base.Invocation) result.Result {
			if callCount.Add(1) == 1 {
				return failedResult
			}
			retryContextErr <- ctx.Err()
			close(retryStarted)
			return successResult
		},
	)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	result := clusterInvoker.Invoke(ctx, &invocation.RPCInvocation{})
	require.NoError(t, result.Error())
	<-ctx.Done()

	select {
	case <-retryStarted:
	case <-time.After(5 * time.Second):
		t.Fatal("failback retry did not start after caller context cancellation")
	}
	require.NoError(t, <-retryContextErr)

	invoker.EXPECT().Destroy().Return()
	clusterInvoker.Destroy()
}

func TestFailbackDestroyCancelsRetry(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	invoker := mock.NewMockInvoker(ctrl)
	clusterInvoker := registerFailback(invoker).(*failbackClusterInvoker)

	invoker.EXPECT().GetURL().Return(failbackUrl).AnyTimes()
	invoker.EXPECT().IsAvailable().Return(true).AnyTimes()

	failedResult := &result.RPCResult{Err: errors.New("error")}
	retryStarted := make(chan struct{})
	retryReturned := make(chan struct{})
	var callCount atomic.Int32

	invoker.EXPECT().Invoke(gomock.Any(), gomock.Any()).Times(2).DoAndReturn(
		func(ctx context.Context, _ base.Invocation) result.Result {
			if callCount.Add(1) == 1 {
				return failedResult
			}
			close(retryStarted)
			<-ctx.Done()
			close(retryReturned)
			return &result.RPCResult{Err: ctx.Err()}
		},
	)

	result := clusterInvoker.Invoke(context.Background(), &invocation.RPCInvocation{})
	require.NoError(t, result.Error())

	select {
	case <-retryStarted:
	case <-time.After(5 * time.Second):
		t.Fatal("failback retry did not start")
	}

	invoker.EXPECT().Destroy().Return()
	destroyed := make(chan struct{})
	go func() {
		clusterInvoker.Destroy()
		close(destroyed)
	}()

	select {
	case <-retryReturned:
	case <-time.After(2 * time.Second):
		t.Fatal("retry invocation did not observe shutdown cancellation")
	}
	select {
	case <-destroyed:
	case <-time.After(2 * time.Second):
		t.Fatal("Destroy did not return after retry cancellation")
	}

	clusterInvoker.Destroy()
}

func TestFailbackDestroyHasBoundedRetryWait(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	invoker := mock.NewMockInvoker(ctrl)
	clusterInvoker := registerFailback(invoker).(*failbackClusterInvoker)

	invoker.EXPECT().GetURL().Return(failbackUrl).AnyTimes()
	invoker.EXPECT().IsAvailable().Return(true).AnyTimes()

	failedResult := &result.RPCResult{Err: errors.New("error")}
	retryStarted := make(chan struct{})
	retryReturned := make(chan struct{})
	releaseRetry := make(chan struct{})
	var releaseOnce sync.Once
	release := func() {
		releaseOnce.Do(func() {
			close(releaseRetry)
		})
	}
	defer release()
	var callCount atomic.Int32

	invoker.EXPECT().Invoke(gomock.Any(), gomock.Any()).Times(2).DoAndReturn(
		func(_ context.Context, _ base.Invocation) result.Result {
			if callCount.Add(1) == 1 {
				return failedResult
			}
			close(retryStarted)
			<-releaseRetry
			close(retryReturned)
			return &result.RPCResult{Rest: clusterpkg.Rest{Tried: 0, Success: true}}
		},
	)

	result := clusterInvoker.Invoke(context.Background(), &invocation.RPCInvocation{})
	require.NoError(t, result.Error())

	select {
	case <-retryStarted:
	case <-time.After(5 * time.Second):
		t.Fatal("failback retry did not start")
	}

	invoker.EXPECT().Destroy().Return()
	destroyed := make(chan struct{})
	go func() {
		clusterInvoker.Destroy()
		close(destroyed)
	}()

	select {
	case <-destroyed:
	case <-time.After(5 * time.Second):
		t.Fatal("Destroy blocked past the bounded retry wait")
	}
	release()

	select {
	case <-retryReturned:
	case <-time.After(2 * time.Second):
		t.Fatal("retry goroutine did not finish after release")
	}
}

func TestFailbackWaitForShutdownUsesPerStepTimeout(t *testing.T) {
	processDone := make(chan struct{})
	retryDone := make(chan struct{})
	waitDone := make(chan struct{})

	go func() {
		time.Sleep(constant.DefaultShutdownConfigStepTimeout - time.Second)
		close(processDone)
		time.Sleep(1500 * time.Millisecond)
		close(retryDone)
	}()
	go func() {
		(&failbackClusterInvoker{}).waitForShutdown(processDone, retryDone)
		close(waitDone)
	}()

	select {
	case <-waitDone:
	case <-time.After(constant.DefaultShutdownConfigStepTimeout + time.Second):
		t.Fatal("waitForShutdown did not return")
	}

	select {
	case <-retryDone:
	default:
		t.Fatal("waitForShutdown returned before the retry tasks completed")
	}
}

func TestFailbackDoesNotEnqueueAfterDestroy(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	invoker := mock.NewMockInvoker(ctrl)
	clusterInvoker := registerFailback(invoker).(*failbackClusterInvoker)

	invoker.EXPECT().GetURL().Return(failbackUrl).AnyTimes()
	invoker.EXPECT().IsAvailable().Return(true).AnyTimes()
	invoker.EXPECT().Invoke(gomock.Any(), gomock.Any()).Return(
		&result.RPCResult{Err: errors.New("error")},
	)

	result := clusterInvoker.Invoke(context.Background(), &invocation.RPCInvocation{})
	require.NoError(t, result.Error())

	value, err := clusterInvoker.taskList.Peek()
	require.NoError(t, err)
	retryTask := value.(*retryTimerTask)

	invoker.EXPECT().Destroy().Return()
	clusterInvoker.Destroy()
	require.False(t, clusterInvoker.enqueueRetry(retryTask))
}

// failed firstly, success later after one retry.
func TestFailbackRetryOneSuccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	invoker := mock.NewMockInvoker(ctrl)
	clusterInvoker := registerFailback(invoker).(*failbackClusterInvoker)

	invoker.EXPECT().GetURL().Return(failbackUrl).AnyTimes()
	invoker.EXPECT().IsAvailable().Return(true)

	// failed at first
	mockFailedResult := &result.RPCResult{Err: errors.New("error")}
	invoker.EXPECT().Invoke(gomock.Any(), gomock.Any()).Return(mockFailedResult)

	// success second
	var wg sync.WaitGroup
	wg.Add(1)
	now := time.Now()
	mockSuccResult := &result.RPCResult{Rest: clusterpkg.Rest{Tried: 0, Success: true}}
	invoker.EXPECT().IsAvailable().Return(true)
	invoker.EXPECT().Invoke(gomock.Any(), gomock.Any()).DoAndReturn(func(context.Context, base.Invocation) result.Result {
		delta := time.Since(now).Nanoseconds() / int64(time.Second)
		// with exponential backoff, first retry happens after ~1s instead of 5s
		assert.GreaterOrEqual(t, delta, int64(1))
		wg.Done()
		return mockSuccResult
	})

	result := clusterInvoker.Invoke(context.Background(), &invocation.RPCInvocation{})
	require.NoError(t, result.Error())
	assert.Nil(t, result.Result())
	assert.Empty(t, result.Attachments())

	// ensure the retry task has been executed
	assert.Equal(t, int64(1), clusterInvoker.taskList.Len())
	// wait until the retry task is executed, the taskList will be empty.
	wg.Wait()
	assert.Equal(t, int64(0), clusterInvoker.taskList.Len())

	invoker.EXPECT().Destroy().Return().AnyTimes()
	clusterInvoker.Destroy()

	assert.Equal(t, int64(0), clusterInvoker.taskList.Len())
}

// failed firstly, and failed again after each retry time.
func TestFailbackRetryFailed(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	invoker := mock.NewMockInvoker(ctrl)
	clusterInvoker := registerFailback(invoker).(*failbackClusterInvoker)

	invoker.EXPECT().GetURL().Return(failbackUrl).AnyTimes()
	invoker.EXPECT().IsAvailable().Return(true).AnyTimes()

	mockFailedResult := &result.RPCResult{Err: errors.New("error")}
	invoker.EXPECT().Invoke(gomock.Any(), gomock.Any()).Return(mockFailedResult)

	// Use atomic counter to safely track retries across goroutines.
	// With exponential backoff and randomization factor, timing is non-deterministic.
	var retryCount atomic.Int64
	targetRetries := int64(2)

	// add retry calls that eventually failed.
	invoker.EXPECT().Invoke(gomock.Any(), gomock.Any()).DoAndReturn(func(context.Context, base.Invocation) result.Result {
		retryCount.Add(1)
		return mockFailedResult
	}).MinTimes(int(targetRetries))

	// first call should failed.
	result := clusterInvoker.Invoke(context.Background(), &invocation.RPCInvocation{})
	require.NoError(t, result.Error())
	assert.Nil(t, result.Result())
	assert.Empty(t, result.Attachments())

	// Wait for at least targetRetries to complete, with bounded timeout to avoid hanging tests
	require.Eventually(t, func() bool {
		return retryCount.Load() >= targetRetries
	}, 10*time.Second, 100*time.Millisecond)

	// Wait for task to be re-queued after retries (with timeout)
	require.Eventually(t, func() bool {
		return clusterInvoker.taskList.Len() >= int64(1)
	}, 5*time.Second, 100*time.Millisecond)

	invoker.EXPECT().Destroy().Return()
	clusterInvoker.Destroy()
	// after destroy, the taskList will be empty
	assert.Equal(t, int64(0), clusterInvoker.taskList.Len())
}

// add 10 tasks but all failed firstly, and failed again with one retry.
func TestFailbackRetryFailed10Times(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	invoker := mock.NewMockInvoker(ctrl)
	clusterInvoker := registerFailback(invoker).(*failbackClusterInvoker)
	clusterInvoker.maxRetries = 10
	clusterInvoker.failbackTasks = 20

	invoker.EXPECT().IsAvailable().Return(true).AnyTimes()
	invoker.EXPECT().GetURL().Return(failbackUrl).AnyTimes()

	// 10 task should failed firstly.
	mockFailedResult := &result.RPCResult{Err: errors.New("error")}
	invoker.EXPECT().Invoke(gomock.Any(), gomock.Any()).Return(mockFailedResult).Times(10)

	// 10 task should retry and failed.
	// With exponential backoff (starting at ~1s), retries happen faster than the old fixed 5s interval.
	// Use atomic counter to safely track retries across goroutines.
	var retryCount atomic.Int64
	invoker.EXPECT().Invoke(gomock.Any(), gomock.Any()).DoAndReturn(func(context.Context, base.Invocation) result.Result {
		retryCount.Add(1)
		return mockFailedResult
	}).MinTimes(10)

	for range 10 {
		result := clusterInvoker.Invoke(context.Background(), &invocation.RPCInvocation{})
		require.NoError(t, result.Error())
		assert.Nil(t, result.Result())
		assert.Empty(t, result.Attachments())
	}

	// Wait for at least 10 retries to complete, with bounded timeout
	require.Eventually(t, func() bool {
		return retryCount.Load() >= 10
	}, 30*time.Second, 100*time.Millisecond)

	// Wait for tasks to be re-queued after retries
	require.Eventually(t, func() bool {
		return clusterInvoker.taskList.Len() >= int64(1)
	}, 5*time.Second, 100*time.Millisecond)

	invoker.EXPECT().Destroy().Return()
	clusterInvoker.Destroy()

	assert.Equal(t, int64(0), clusterInvoker.taskList.Len())
}

func TestFailbackOutOfLimit(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	invoker := mock.NewMockInvoker(ctrl)
	clusterInvoker := registerFailback(invoker).(*failbackClusterInvoker)
	clusterInvoker.failbackTasks = 1

	invoker.EXPECT().GetURL().Return(failbackUrl).AnyTimes()
	invoker.EXPECT().IsAvailable().Return(true).AnyTimes()

	mockFailedResult := &result.RPCResult{Err: errors.New("error")}
	invoker.EXPECT().Invoke(gomock.Any(), gomock.Any()).Return(mockFailedResult).Times(11)

	// reached limit
	result := clusterInvoker.Invoke(context.Background(), &invocation.RPCInvocation{})
	require.NoError(t, result.Error())
	assert.Nil(t, result.Result())
	assert.Empty(t, result.Attachments())

	// all will be out of limit
	for range 10 {
		result := clusterInvoker.Invoke(context.Background(), &invocation.RPCInvocation{})
		require.NoError(t, result.Error())
		assert.Nil(t, result.Result())
		assert.Empty(t, result.Attachments())

		assert.Equal(t, int64(1), clusterInvoker.taskList.Len())
	}

	invoker.EXPECT().Destroy().Return()
	clusterInvoker.Destroy()
}
