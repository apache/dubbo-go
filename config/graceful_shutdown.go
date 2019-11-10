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
	"sync"
	"syscall"
	"time"

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
 * It's seems that the Unix/Linux does not have the signal SIGSTKFLT. https://github.com/golang/go/issues/33381
 * So this signal will be ignored.
 *
 */
var gracefulShutdownOnce = sync.Once{}

func GracefulShutdownInit() {

	signals := make(chan os.Signal, 1)

	signal.Notify(signals, os.Interrupt, os.Kill, syscall.SIGKILL, syscall.SIGSTOP,
		syscall.SIGHUP, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGILL, syscall.SIGTRAP,
		syscall.SIGABRT, syscall.SIGEMT, syscall.SIGSYS,
	)

	go func() {
		select {
		case sig := <-signals:
			logger.Infof("get signal %s, application will shutdown.", sig.String())
			// gracefulShutdownOnce.Do(func() {
			BeforeShutdown()

			switch sig {
			// those signals' original behavior is exit with dump ths stack, so we try to keep the behavior
			case syscall.SIGQUIT, syscall.SIGILL, syscall.SIGTRAP,
				syscall.SIGABRT, syscall.SIGEMT, syscall.SIGSYS:
				debug.WriteHeapDump(os.Stdout.Fd())
			default:
				time.AfterFunc(totalTimeout(), func() {
					logger.Warn("Shutdown gracefully timeout, application will shutdown immediately. ")
					os.Exit(0)
				})
			}
			os.Exit(0)
		}
	}()
}

func totalTimeout() time.Duration {
	var providerShutdown time.Duration = 0
	if providerConfig != nil && providerConfig.ShutdownConfig != nil {
		providerShutdown = providerConfig.ShutdownConfig.GetTimeout()
	}

	var consumerShutdown time.Duration = 0
	if consumerConfig != nil && consumerConfig.ShutdownConfig != nil {
		consumerShutdown = consumerConfig.ShutdownConfig.GetTimeout()
	}

	var timeout = providerShutdown
	if consumerShutdown > providerShutdown {
		timeout = consumerShutdown
	}
	return timeout
}

func BeforeShutdown() {

	destroyAllRegistries()
	// waiting for a short time so that the clients have enough time to get the notification that server shutdowns
	// The value of configuration depends on how long the clients will get notification.
	waitAndAcceptNewRequests()

	// reject the new request, but keeping waiting for accepting requests
	waitForReceivingRequests()

	// If this application is not the provider, it will do nothing
	destroyProviderProtocols()

	// waiting for accepted requests to be processed.

	// after this step, the response from other providers will be rejected.
	// If this application is not the consumer, it will do nothing
	destroyConsumerProtocols()

	logger.Infof("Execute the custom callbacks.")
	customCallbacks := extension.GetAllCustomShutdownCallbacks()
	for callback := customCallbacks.Front(); callback != nil; callback = callback.Next() {
		callback.Value.(func())()
	}
}

func destroyAllRegistries() {
	logger.Infof("Graceful shutdown --- Destroy all registries. ")
	registryProtocol := extension.GetProtocol(constant.REGISTRY_KEY)
	registryProtocol.Destroy()
}

func destroyConsumerProtocols() {
	logger.Info("Graceful shutdown --- Destroy consumer's protocols. ")
	if consumerConfig == nil || consumerConfig.ProtocolConf == nil {
		return
	}
	destroyProtocols(consumerConfig.ProtocolConf)
}

/**
 * destroy the provider's protocol.
 * if the protocol is consumer's protocol too, we will keep it.
 */
func destroyProviderProtocols() {

	logger.Info("Graceful shutdown --- Destroy provider's protocols. ")

	if providerConfig == nil || providerConfig.ProtocolConf == nil {
		return
	}
	destroyProtocols(providerConfig.ProtocolConf)
}

func destroyProtocols(protocolConf interface{}) {
	protocols := protocolConf.(map[interface{}]interface{})
	for name, _ := range protocols {
		extension.GetProtocol(name.(string)).Destroy()
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
func waitForSendingRequests()  {
	logger.Info("Graceful shutdown --- Keep waiting until sending requests getting response or timeout ")
	if consumerConfig == nil || consumerConfig.ShutdownConfig == nil {
		// ignore this step
		return
	}
}

func waitingProcessedTimeout(shutdownConfig *ShutdownConfig) {
	timeout := shutdownConfig.GetStepTimeout()
	if timeout <= 0 {
		return
	}
	start := time.Now().UnixNano()
	for time.Now().UnixNano()-start < timeout.Nanoseconds() && !shutdownConfig.RequestsFinished {
		// sleep 10 ms and then we check it again
		time.Sleep(10 * time.Millisecond)
	}
}
