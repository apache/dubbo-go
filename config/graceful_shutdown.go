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
	// retrieve ShutdownConfig for gracefulShutdownFilter
	cGracefulShutdownFilter, existcGracefulShutdownFilter := extension.GetFilter(constant.GracefulShutdownConsumerFilterKey)
	if !existcGracefulShutdownFilter {
		return
	}
	sGracefulShutdownFilter, existsGracefulShutdownFilter := extension.GetFilter(constant.GracefulShutdownProviderFilterKey)
	if !existsGracefulShutdownFilter {
		return
	}
	if filter, ok := cGracefulShutdownFilter.(Setter); ok && rootConfig.Shutdown != nil {
		filter.Set(constant.GracefulShutdownFilterShutdownConfig, GetShutDown())
	}

	if filter, ok := sGracefulShutdownFilter.(Setter); ok && rootConfig.Shutdown != nil {
		filter.Set(constant.GracefulShutdownFilterShutdownConfig, GetShutDown())
	}

	if GetShutDown().InternalSignal {
		signals := make(chan os.Signal, 1)
		signal.Notify(signals, ShutdownSignals...)

		go func() {
			select {
			case sig := <-signals:
				logger.Infof("get signal %s, applicationConfig will shutdown.", sig)
				// gracefulShutdownOnce.Do(func() {
				time.AfterFunc(totalTimeout(), func() {
					logger.Warn("Shutdown gracefully timeout, applicationConfig will shutdown immediately. ")
					os.Exit(0)
				})
				BeforeShutdown()
				// those signals' original behavior is exit with dump ths stack, so we try to keep the behavior
				for _, dumpSignal := range DumpHeapShutdownSignals {
					if sig == dumpSignal {
						debug.WriteHeapDump(os.Stdout.Fd())
					}
				}
				os.Exit(0)
			}
		}()
	}
}

// BeforeShutdown provides processing flow before shutdown
func BeforeShutdown() {
	destroyAllRegistries()
	// waiting for a short time so that the clients have enough time to get the notification that server shutdowns
	// The value of configuration depends on how long the clients will get notification.
	waitAndAcceptNewRequests()

	// reject sending/receiving the new request, but keeping waiting for accepting requests
	waitForSendingAndReceivingRequests()

	// destroy all protocols
	destroyProtocols()

	logger.Info("Graceful shutdown --- Execute the custom callbacks.")
	customCallbacks := extension.GetAllCustomShutdownCallbacks()
	for callback := customCallbacks.Front(); callback != nil; callback = callback.Next() {
		callback.Value.(func())()
	}
}

func destroyAllRegistries() {
	logger.Info("Graceful shutdown --- Destroy all registriesConfig. ")
	registryProtocol := extension.GetProtocol(constant.RegistryKey)
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
func destroyProviderProtocols(consumerProtocols *gxset.HashSet) {
	logger.Info("Graceful shutdown --- First destroy provider's protocols. ")
	for _, protocol := range rootConfig.Protocols {
		// the protocol is the consumer's protocol too, we can not destroy it.
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

func waitingProviderProcessedTimeout(shutdownConfig *ShutdownConfig) {
	timeout := shutdownConfig.GetStepTimeout()
	if timeout <= 0 {
		return
	}
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) && shutdownConfig.ProviderActiveCount.Load() > 0 {
		// sleep 10 ms and then we check it again
		time.Sleep(10 * time.Millisecond)
		logger.Infof("waiting for provider active invocation count = %d", shutdownConfig.ProviderActiveCount.Load())
	}
}

//for provider. It will wait for processing receiving requests
func waitForSendingAndReceivingRequests() {
	logger.Info("Graceful shutdown --- Keep waiting until sending/accepting requests finish or timeout. ")
	if rootConfig == nil || rootConfig.Shutdown == nil {
		// ignore this step
		return
	}
	rootConfig.Shutdown.RejectRequest.Store(true)
	waitingConsumerProcessedTimeout(rootConfig.Shutdown)
}

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

func totalTimeout() time.Duration {
	timeout := defaultShutDownTime
	if rootConfig.Shutdown != nil && rootConfig.Shutdown.GetTimeout() > timeout {
		timeout = rootConfig.Shutdown.GetTimeout()
	}

	return timeout
}

// we can not get the protocols from consumerConfig because some protocol don't have configuration, like jsonrpc.
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
