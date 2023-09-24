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

package graceful_shutdown

import (
	"os"
	"os/signal"
	"runtime/debug"
	"sync"
	"time"
)

import (
	"github.com/dubbogo/gost/log/logger"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/config"
)

const (
	// todo(DMwangnima): these descriptions and defaults could be wrapped by functions of Options
	defaultTimeout                     = 60 * time.Second
	defaultStepTimeout                 = 3 * time.Second
	defaultConsumerUpdateWaitTime      = 3 * time.Second
	defaultOfflineRequestWindowTimeout = 3 * time.Second

	timeoutDesc                     = "Timeout"
	stepTimeoutDesc                 = "StepTimeout"
	consumerUpdateWaitTimeDesc      = "ConsumerUpdateWaitTime"
	offlineRequestWindowTimeoutDesc = "OfflineRequestWindowTimeout"
)

var (
	initOnce       sync.Once
	compatShutdown *config.ShutdownConfig

	proMu     sync.Mutex
	protocols map[string]struct{}
)

func Init(opts ...Option) {
	initOnce.Do(func() {
		protocols = make(map[string]struct{})
		newOpts := DefaultOptions()
		for _, opt := range opts {
			opt(newOpts)
		}
		compatShutdown = compatShutdownConfig(newOpts.Shutdown)
		// retrieve ShutdownConfig for gracefulShutdownFilter
		cGracefulShutdownFilter, existcGracefulShutdownFilter := extension.GetFilter(constant.GracefulShutdownConsumerFilterKey)
		if !existcGracefulShutdownFilter {
			return
		}
		sGracefulShutdownFilter, existsGracefulShutdownFilter := extension.GetFilter(constant.GracefulShutdownProviderFilterKey)
		if !existsGracefulShutdownFilter {
			return
		}
		if filter, ok := cGracefulShutdownFilter.(config.Setter); ok {
			filter.Set(constant.GracefulShutdownFilterShutdownConfig, compatShutdown)
		}

		if filter, ok := sGracefulShutdownFilter.(config.Setter); ok {
			filter.Set(constant.GracefulShutdownFilterShutdownConfig, compatShutdown)
		}

		if compatShutdown.InternalSignal != nil && *compatShutdown.InternalSignal {
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
					beforeShutdown()
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
	})
}

// RegisterProtocol registers protocol which would be destroyed before shutdown.
// Please make sure that Init function has been invoked before, otherwise this
// function would not make any sense.
func RegisterProtocol(name string) {
	proMu.Lock()
	protocols[name] = struct{}{}
	proMu.Unlock()
}

func totalTimeout() time.Duration {
	timeout := parseDuration(compatShutdown.Timeout, timeoutDesc, defaultTimeout)
	if timeout < defaultTimeout {
		timeout = defaultTimeout
	}

	return timeout
}

func beforeShutdown() {
	destroyRegistries()
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

// destroyRegistries destroys RegistryProtocol directly.
func destroyRegistries() {
	logger.Info("Graceful shutdown --- Destroy all registriesConfig. ")
	registryProtocol := extension.GetProtocol(constant.RegistryProtocol)
	registryProtocol.Destroy()
}

func waitAndAcceptNewRequests() {
	logger.Info("Graceful shutdown --- Keep waiting and accept new requests for a short time. ")

	updateWaitTime := parseDuration(compatShutdown.ConsumerUpdateWaitTime, consumerUpdateWaitTimeDesc, defaultConsumerUpdateWaitTime)
	time.Sleep(updateWaitTime)

	stepTimeout := parseDuration(compatShutdown.StepTimeout, stepTimeoutDesc, defaultStepTimeout)

	// ignore this step
	if stepTimeout < 0 {
		return
	}
	waitingProviderProcessedTimeout(stepTimeout)
}

func waitingProviderProcessedTimeout(timeout time.Duration) {
	deadline := time.Now().Add(timeout)

	offlineRequestWindowTimeout := parseDuration(compatShutdown.OfflineRequestWindowTimeout, offlineRequestWindowTimeoutDesc, defaultOfflineRequestWindowTimeout)

	for time.Now().Before(deadline) &&
		(compatShutdown.ProviderActiveCount.Load() > 0 || time.Now().Before(compatShutdown.ProviderLastReceivedRequestTime.Load().Add(offlineRequestWindowTimeout))) {
		// sleep 10 ms and then we check it again
		time.Sleep(10 * time.Millisecond)
		logger.Infof("waiting for provider active invocation count = %d, provider last received request time: %v",
			compatShutdown.ProviderActiveCount.Load(), compatShutdown.ProviderLastReceivedRequestTime.Load())
	}
}

// for provider. It will wait for processing receiving requests
func waitForSendingAndReceivingRequests() {
	logger.Info("Graceful shutdown --- Keep waiting until sending/accepting requests finish or timeout. ")
	compatShutdown.RejectRequest.Store(true)
	waitingConsumerProcessedTimeout()
}

func waitingConsumerProcessedTimeout() {
	stepTimeout := parseDuration(compatShutdown.StepTimeout, stepTimeoutDesc, defaultStepTimeout)

	if stepTimeout <= 0 {
		return
	}
	deadline := time.Now().Add(stepTimeout)

	for time.Now().Before(deadline) && compatShutdown.ConsumerActiveCount.Load() > 0 {
		// sleep 10 ms and then we check it again
		time.Sleep(10 * time.Millisecond)
		logger.Infof("waiting for consumer active invocation count = %d", compatShutdown.ConsumerActiveCount.Load())
	}
}

// destroyProtocols destroys protocols that have been registered.
func destroyProtocols() {
	logger.Info("Graceful shutdown --- Destroy protocols. ")

	proMu.Lock()
	// extension.GetProtocol might panic
	defer proMu.Unlock()
	for name := range protocols {
		extension.GetProtocol(name).Destroy()
	}
}
