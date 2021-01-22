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
)

import (
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/common/extension"
	"github.com/apache/dubbo-go/common/logger"
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

// nolint
func GracefulShutdownInit() {

	signals := make(chan os.Signal, 1)

	signal.Notify(signals, ShutdownSignals...)

	go func() {
		select {
		case sig := <-signals:
			logger.Infof("get signal %s, application will shutdown.", sig)
			// gracefulShutdownOnce.Do(func() {
			time.AfterFunc(totalTimeout(), func() {
				logger.Warn("Shutdown gracefully timeout, application will shutdown immediately. ")
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

// BeforeShutdown provides processing flow before shutdown
func BeforeShutdown() {

	destroyAllRegistries()
	// waiting for a short time so that the clients have enough time to get the notification that server shutdowns
	// The value of configuration depends on how long the clients will get notification.
	waitAndAcceptNewRequests()

	// reject the new request, but keeping waiting for accepting requests
	waitForReceivingRequests()

	// we fetch the protocols from Consumer.References. Consumer.ProtocolConfig doesn't contains all protocol, like jsonrpc
	consumerProtocols := getConsumerProtocols()

	// If this application is not the provider, it will do nothing
	destroyProviderProtocols(consumerProtocols)

	// reject sending the new request, and waiting for response of sending requests
	waitForSendingRequests()

	// If this application is not the consumer, it will do nothing
	destroyConsumerProtocols(consumerProtocols)

	logger.Info("Graceful shutdown --- Execute the custom callbacks.")
	customCallbacks := extension.GetAllCustomShutdownCallbacks()
	for callback := customCallbacks.Front(); callback != nil; callback = callback.Next() {
		callback.Value.(func())()
	}
}

func destroyAllRegistries() {
	logger.Info("Graceful shutdown --- Destroy all registries. ")
	registryProtocol := extension.GetProtocol(constant.REGISTRY_KEY)
	registryProtocol.Destroy()
}

func destroyConsumerProtocols(consumerProtocols *gxset.HashSet) {
	logger.Info("Graceful shutdown --- Destroy consumer's protocols. ")
	for name := range consumerProtocols.Items {
		extension.GetProtocol(name.(string)).Destroy()
	}
}

// destroyProviderProtocols destroys the provider's protocol.
// if the protocol is consumer's protocol too, we will keep it
func destroyProviderProtocols(consumerProtocols *gxset.HashSet) {

	logger.Info("Graceful shutdown --- Destroy provider's protocols. ")

	if providerConfig == nil || providerConfig.Protocols == nil {
		return
	}

	for _, protocol := range providerConfig.Protocols {

		// the protocol is the consumer's protocol too, we can not destroy it.
		if consumerProtocols.Contains(protocol.Name) {
			continue
		}
		extension.GetProtocol(protocol.Name).Destroy()
	}
}

func waitAndAcceptNewRequests() {

	logger.Info("Graceful shutdown --- Keep waiting and accept new requests for a short time. ")
	if providerConfig == nil || providerConfig.ShutdownConfig == nil {
		return
	}

	timeout := providerConfig.ShutdownConfig.GetStepTimeout()

	// ignore this step
	if timeout < 0 {
		return
	}
	time.Sleep(timeout)
}

// for provider. It will wait for processing receiving requests
func waitForReceivingRequests() {
	logger.Info("Graceful shutdown --- Keep waiting until accepting requests finish or timeout. ")
	if providerConfig == nil || providerConfig.ShutdownConfig == nil {
		// ignore this step
		return
	}
	waitingProcessedTimeout(providerConfig.ShutdownConfig)
}

// for consumer. It will wait for the response of sending requests
func waitForSendingRequests() {
	logger.Info("Graceful shutdown --- Keep waiting until sending requests getting response or timeout ")
	if consumerConfig == nil || consumerConfig.ShutdownConfig == nil {
		// ignore this step
		return
	}
	waitingProcessedTimeout(consumerConfig.ShutdownConfig)
}

func waitingProcessedTimeout(shutdownConfig *ShutdownConfig) {
	timeout := shutdownConfig.GetStepTimeout()
	if timeout <= 0 {
		return
	}
	start := time.Now()

	for time.Now().After(start.Add(timeout)) && !shutdownConfig.RequestsFinished {
		// sleep 10 ms and then we check it again
		time.Sleep(10 * time.Millisecond)
	}
}

func totalTimeout() time.Duration {
	var providerShutdown = defaultShutDownTime
	if providerConfig != nil && providerConfig.ShutdownConfig != nil {
		providerShutdown = providerConfig.ShutdownConfig.GetTimeout()
	}

	var consumerShutdown time.Duration
	if consumerConfig != nil && consumerConfig.ShutdownConfig != nil {
		consumerShutdown = consumerConfig.ShutdownConfig.GetTimeout()
	}

	var timeout = providerShutdown
	if consumerShutdown > providerShutdown {
		timeout = consumerShutdown
	}
	return timeout
}

// we can not get the protocols from consumerConfig because some protocol don't have configuration, like jsonrpc.
func getConsumerProtocols() *gxset.HashSet {
	result := gxset.NewSet()
	if consumerConfig == nil || consumerConfig.References == nil {
		return result
	}

	for _, reference := range consumerConfig.References {
		result.Add(reference.Protocol)
	}
	return result
}
