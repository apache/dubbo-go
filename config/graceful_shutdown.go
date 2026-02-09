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

package config

import (
	"os"
	"os/signal"
	"runtime/debug"
	"time"
)

import (
	gxset "github.com/dubbogo/gost/container/set"
	"github.com/dubbogo/gost/log/logger"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
)

/*
 * The key point is that find out the signals to handle.
 * The most important documentation is https://golang.org/pkg/os/signal/
 * From this documentation, we can know that:
 * 1. The signals SIGKILL and SIGSTOP may not be caught by signal package;
 * 2. SIGHUP, SIGINT, or SIGTERM signal causes the program to exit
 * 3. SIGQUIT, SIGILL, SIGTRAP, SIGABRT, SIGSTKFLT, SIGEMT, or SIGSYS signal causes the program to exit with a stack dump
 * 4. The invocation of Notify(signal...) will disable the default behavior of those signals.
 *
 * So the signals SIGKILL, SIGSTOP, SIGHUP, SIGINT, SIGTERM, SIGQUIT, SIGILL, SIGTRAP, SIGABRT, SIGSTKFLT, SIGEMT, SIGSYS
 * should be processed.
 * syscall.SIGEMT cannot be found in CI
 * It's seems that the Unix/Linux does not have the signal SIGSTKFLT. https://github.com/golang/go/issues/33381
 * So this signal will be ignored.
 * The signals are different on different platforms.
 * We define them by using 'package build' feature https://golang.org/pkg/go/build/
 */
const defaultShutDownTime = time.Second * 60

func gracefulShutdownInit() {
	//尝试从扩展点系统里拿到“优雅停机 consumer filter”和“优雅停机 provider filter”；拿不到就直接不启用优雅停机相关流程。
	gracefulShutdownConsumerFilter, exist := extension.GetFilter(constant.GracefulShutdownConsumerFilterKey)
	if !exist {
		return
	}
	gracefulShutdownProviderFilter, exist := extension.GetFilter(constant.GracefulShutdownProviderFilterKey)
	if !exist {
		return
	}
	// retrieve ShutdownConfig for gracefulShutdownFilter
	// 这份实现把“优雅停机”当成可插拔能力 ——
	// 如果你没有把相关 filter 包 import 进来（从而触发注册），
	// 那这里就拿不到 filter，于是优雅停机初始化直接跳过。
	if filter, ok := gracefulShutdownConsumerFilter.(Setter); ok && rootConfig.Shutdown != nil {
		filter.Set(constant.GracefulShutdownFilterShutdownConfig, GetShutDown())
	}

	//Setter 是一个接口 type Setter interface { Set(name string, config any) }
	//gracefulShutdownProviderFilter.(Setter) 不是所有 filter 都必须支持注入配置；只有实现了 Setter 的 filter 才能被动态塞配置进去。
	if filter, ok := gracefulShutdownProviderFilter.(Setter); ok && rootConfig.Shutdown != nil {
		filter.Set(constant.GracefulShutdownFilterShutdownConfig, GetShutDown())
	}
	//是否开启信号量监听
	if GetShutDown().GetInternalSignal() {
		//创建一个信号量的chan
		signals := make(chan os.Signal, 1)
		//将关机信号列表注册到该chan中
		signal.Notify(signals, ShutdownSignals...)
		//异步监听
		go func() {
			//阻塞等待第一个信号，收到后进入优雅停机流程。
			sig := <-signals
			logger.Infof("get signal %s, applicationConfig will shutdown.", sig)
			// gracefulShutdownOnce.Do(func() {
			// 再总下线时长之后, 执行系统退出函数
			// 这是一个兜底定时器：如果优雅停机流程卡住超过总超时，就直接 os.Exit(0) 强制退出。
			// 注意：这不是“等 BeforeShutdown 返回再退出”，而是一个并行计时炸弹。
			time.AfterFunc(totalTimeout(), func() {
				logger.Warn("Shutdown gracefully timeout, applicationConfig will shutdown immediately. ")
				os.Exit(0)
			})
			// 执行下线策略 真正的优雅停机主流程。
			BeforeShutdown()
			// those signals' original behavior is exit with dump ths stack, so we try to keep the behavior
			// 对某些“默认行为是带堆栈/转储退出”的信号，这里尝试保留行为：在退出前写 heap dump 到 stdout fd。
			for _, dumpSignal := range DumpHeapShutdownSignals {
				if sig == dumpSignal {
					debug.WriteHeapDump(os.Stdout.Fd())
				}
			}
			os.Exit(0)

		}()
	}
}

// BeforeShutdown provides processing flow before shutdown
// 优雅停机主流程（四步）
func BeforeShutdown() {
	// 1. 销毁所有注册中心（摘流量） 目标: 外部不再把新请求路由到这个实例。
	destroyAllRegistries()
	// waiting for a short time so that the clients have enough time to get the notification that server shutdowns
	// The value of configuration depends on how long the clients will get notification.

	// 2. 等待所有消费者调用完成（等流量自然迁移完） 这个过程是阻塞的，会等待所有消费者调用完成。
	// 目标:留一个“客户端感知下线”的缓冲时间 + 等待 provider 处理完
	waitAndAcceptNewRequests()

	// reject sending/receiving the new request, but keeping waiting for accepting requests
	// 3.开始拒绝新请求 + 等待 consumer 在途请求结束
	waitForSendingAndReceivingRequests()

	// destroy all protocols
	// 4. 销毁所有协议（关闭所有网络连接） 目标: 所有网络连接都被关闭。
	destroyProtocols()

	// 5. 执行自定义回调 目标: 自定义逻辑（如通知监控系统、清理资源等）。
	// 框架允许其他模块注册“进程退出前回调”（清理资源、flush、上报、关闭自建 goroutine 等）。
	logger.Info("Graceful shutdown --- Execute the custom callbacks.")
	customCallbacks := extension.GetAllCustomShutdownCallbacks()
	for callback := customCallbacks.Front(); callback != nil; callback = callback.Next() {
		callback.Value.(func())()
	}
}

func destroyAllRegistries() {
	logger.Info("Graceful shutdown --- Destroy all registriesConfig. ")
	registryProtocol := extension.GetProtocol(constant.RegistryProtocol)
	registryProtocol.Destroy()
}

// destroyProtocols destroys protocols.
// First we destroy provider's protocols, and then we destroy the consumer protocols.
func destroyProtocols() {
	logger.Info("Graceful shutdown --- Destroy protocols. ")

	if rootConfig.Protocols == nil {
		return
	}

	consumerProtocols := getConsumerProtocols()

	destroyProviderProtocols(consumerProtocols)
	destroyConsumerProtocols(consumerProtocols)
}

// destroyProviderProtocols destroys the provider's protocol.
// if the protocol is consumer's protocol too, we will keep it
// provider 协议销毁：跳过“同时被 consumer 使用的协议”
func destroyProviderProtocols(consumerProtocols *gxset.HashSet) {
	logger.Info("Graceful shutdown --- First destroy provider's protocols. ")
	for _, protocol := range rootConfig.Protocols {
		// the protocol is the consumer's protocol too, we can not destroy it.
		// 如果某个协议也在 consumerProtocols 里，说明 consumer 还要用它（或者共用资源），先不 destroy。
		if consumerProtocols.Contains(protocol.Name) {
			continue
		}
		extension.GetProtocol(protocol.Name).Destroy()
	}
}

func destroyConsumerProtocols(consumerProtocols *gxset.HashSet) {
	logger.Info("Graceful shutdown --- Second Destroy consumer's protocols. ")
	for name := range consumerProtocols.Items {
		extension.GetProtocol(name.(string)).Destroy()
	}
}

func waitAndAcceptNewRequests() {
	logger.Info("Graceful shutdown --- Keep waiting and accept new requests for a short time. ")
	if rootConfig.Shutdown == nil {
		return
	}

	time.Sleep(rootConfig.Shutdown.GetConsumerUpdateWaitTime())

	timeout := rootConfig.Shutdown.GetStepTimeout()
	// ignore this step
	if timeout < 0 {
		return
	}
	waitingProviderProcessedTimeout(rootConfig.Shutdown)
}

// 等待 provider 侧请求“自然收敛”。
func waitingProviderProcessedTimeout(shutdownConfig *ShutdownConfig) {
	timeout := shutdownConfig.GetStepTimeout()
	if timeout <= 0 {
		return
	}
	deadline := time.Now().Add(timeout)

	offlineRequestWindowTimeout := shutdownConfig.GetOfflineRequestWindowTimeout()
	// 阻塞等待 provider 侧请求“自然收敛”。
	//		ProviderActiveCount > 0 仍有正在处理的 provider 调用
	//		当前时间 < ProviderLastReceivedRequestTime + offlineRequestWindowTimeout
	//		虽然活跃数可能为 0，但距离“最后一次收到请求”还没过窗口期。
	// 			窗口期意义: 防止出现“刚刚还有流量抖动/刚接到请求”的情况，给一个短窗确保流量真的停了。
	for time.Now().Before(deadline) &&
		(shutdownConfig.ProviderActiveCount.Load() > 0 || time.Now().Before(shutdownConfig.ProviderLastReceivedRequestTime.Load().Add(offlineRequestWindowTimeout))) {
		// sleep 10 ms and then we check it again
		time.Sleep(10 * time.Millisecond)
		logger.Infof("waiting for provider active invocation count = %d, provider last received request time: %v",
			shutdownConfig.ProviderActiveCount.Load(), shutdownConfig.ProviderLastReceivedRequestTime.Load())
	}
}

// for provider. It will wait for processing receiving requests
func waitForSendingAndReceivingRequests() {
	logger.Info("Graceful shutdown --- Keep waiting until sending/accepting requests finish or timeout. ")
	if rootConfig == nil || rootConfig.Shutdown == nil {
		// ignore this step
		return
	}
	//这就是“进入拒绝新请求阶段”的全局开关。
	rootConfig.Shutdown.RejectRequest.Store(true)
	waitingConsumerProcessedTimeout(rootConfig.Shutdown)
}

// 然后最多等待 stepTimeout，让 consumer 的在途调用收敛到 0。
func waitingConsumerProcessedTimeout(shutdownConfig *ShutdownConfig) {
	timeout := shutdownConfig.GetStepTimeout()
	if timeout <= 0 {
		return
	}
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) && shutdownConfig.ConsumerActiveCount.Load() > 0 {
		// sleep 10 ms and then we check it again
		time.Sleep(10 * time.Millisecond)
		logger.Infof("waiting for consumer active invocation count = %d", shutdownConfig.ConsumerActiveCount.Load())
	}
}

// 总超时计算
func totalTimeout() time.Duration {
	timeout := defaultShutDownTime // 60s
	//它只在 “GetTimeout() > 60s” 时覆盖；如果你配置 10s，这里仍然按 60s 兜底。
	if rootConfig.Shutdown != nil && rootConfig.Shutdown.GetTimeout() > timeout {
		timeout = rootConfig.Shutdown.GetTimeout()
	}

	return timeout
}

// we can not get the protocols from consumerConfig because some protocol don't have configuration, like jsonrpc.
// 获取所有协议
func getConsumerProtocols() *gxset.HashSet {
	result := gxset.NewSet()
	if rootConfig.Consumer == nil || rootConfig.Consumer.References == nil {
		return result
	}

	for _, reference := range rootConfig.Consumer.References {
		result.Add(reference.Protocol)
	}
	return result
}
