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
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/common/extension"
	"github.com/apache/dubbo-go/common/logger"
	"github.com/apache/dubbo-go/protocol"
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
	for {
		sig := <-signals

		gracefulShutdownOnce.Do(func() {
			logger.Infof("get signal %s, application is shutdown", sig.String())

			destroyAllRegistries()
			// waiting for a short time so that the clients have enough time to get the notification that server shutdowns
			// The value of configuration depends on how long the clients will get notification.
			waitAndAcceptNewRequests()

			// after this step, the new request will be rejected because the server is shutdown.
			destroyProviderProtocols()

			//


			logger.Infof("Execute the custom callbacks.")
			customCallbacks := extension.GetAllCustomShutdownCallbacks()
			for callback := customCallbacks.Front(); callback != nil; callback = callback.Next() {
				callback.Value.(func())()
			}
		})
	}
}

func destroyAllRegistries() {
	registryProtocol := extension.GetProtocol(constant.REGISTRY_KEY)
	registryProtocol.Destroy()
}

func destroyConsumerProtocols() {
	if consumerConfig == nil || consumerConfig.ProtocolConf == nil {
		return
	}
	destroyProtocols(consumerConfig.ProtocolConf)
}
func destroyProviderProtocols() {
	if providerConfig == nil || providerConfig.ProtocolConf == nil {
		return
	}
	destroyProtocols(providerConfig.ProtocolConf)
}

func destroyProtocols(protocolConf interface{}) {
	protocols := protocolConf.(map[interface{}]interface{})
	for name, _ := range protocols {
		protocol := extension.GetProtocol(name.(string))
		protocol.Destroy()
	}
}

func waitAndAcceptNewRequests() {

	if providerConfig == nil || providerConfig.ShutdownConfig == nil {
		return
	}
	shutdownConfig := providerConfig.ShutdownConfig

	timeout, err := strconv.ParseInt(shutdownConfig.AcceptNewRequestsTimeout, 0, 0)
	if err != nil {
		logger.Errorf("The timeout configuration of keeping accept new requests is invalid. Go next step!", err)
		return
	}

	// ignore this phase
	if timeout < 0 {
		return
	}

	var duration = time.Duration(timeout) * time.Millisecond

	time.Sleep(duration)
}

/**
 * this method will wait a short time until timeout or all requests have been processed.
 * this implementation use the active filter, so you need to add the filter into your application configuration
 * for example:
 * server.yml or client.yml
 *
 * filter: "active",
 *
 * see the ActiveFilter for more detail.
 * We use the bigger value between consumer's config and provider's config
 * if the application is both consumer and provider.
 * This method's behavior is a little bit complicated.
 */
func waitForProcessingRequest()  {
	var timeout int64 = 0
	if providerConfig != nil && providerConfig.ShutdownConfig != nil {
		timeout = waitingProcessedTimeout(providerConfig.ShutdownConfig)
	}

	if consumerConfig != nil && consumerConfig.ShutdownConfig != nil {
		consumerTimeout := waitingProcessedTimeout(consumerConfig.ShutdownConfig)
		if consumerTimeout > timeout {
			timeout = consumerTimeout
		}
	}
	if timeout <= 0{
		return
	}

	timeout = timeout * time.Millisecond.Nanoseconds()

	start := time.Now().UnixNano()

	for time.Now().UnixNano() - start < timeout && protocol.GetTotalActive() > 0  {
		// sleep 10 ms and then we check it again
		time.Sleep(10 * time.Millisecond)
	}
}

func waitingProcessedTimeout(shutdownConfig *ShutdownConfig) int64 {
	if len(shutdownConfig.WaitingProcessRequestsTimeout) <=0 {
		return 0
	}
	config, err := strconv.ParseInt(shutdownConfig.WaitingProcessRequestsTimeout, 0, 0)
	if err != nil {
		logger.Errorf("The configuration of shutdownConfig.WaitingProcessRequestsTimeout is invalid: %s",
			shutdownConfig.WaitingProcessRequestsTimeout)
		return 0
	}
	return config
}
