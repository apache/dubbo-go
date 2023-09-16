package graceful_shutdown

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/config"
	"dubbo.apache.org/dubbo-go/v3/global"
	"github.com/dubbogo/gost/log/logger"
	"os"
	"os/signal"
	"runtime/debug"
	"sync"
	"time"
)

const (
	defaultTimeout                     = 60 * time.Second
	defaultStepTimeout                 = 3 * time.Second
	defaultConsumerUpdateWaitTime      = 3 * time.Second
	defaultOfflineRequestWindowTimeout = 3 * time.Second
)

var (
	initOnce       sync.Once
	compatShutdown *config.ShutdownConfig

	proMu     sync.Mutex
	protocols map[string]struct{}
)

func Init(cfg *global.ShutdownConfig) {
	compatShutdown = compatShutdownConfig(cfg)
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
}

// RegisterProtocol registers protocol which would be destroyed before shutdown
func RegisterProtocol(name string) {
	proMu.Lock()
	protocols[name] = struct{}{}
	proMu.Unlock()
}

func totalTimeout() time.Duration {
	result, err := time.ParseDuration(compatShutdown.Timeout)
	if err != nil {
		logger.Errorf("The Timeout configuration is invalid: %s, and we will use the default value: %s, err: %v",
			compatShutdown.Timeout, defaultTimeout.String(), err)
		result = defaultTimeout
	}
	if result <= defaultTimeout {
		result = defaultTimeout
	}

	return result
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

	waitTime, err := time.ParseDuration(compatShutdown.ConsumerUpdateWaitTime)
	if err != nil {
		logger.Errorf("The ConsumerUpdateTimeout configuration is invalid: %s, and we will use the default value: %s, err: %v",
			compatShutdown.ConsumerActiveCount.Load(), defaultConsumerUpdateWaitTime.String(), err)
		waitTime = defaultConsumerUpdateWaitTime
	}
	time.Sleep(waitTime)

	timeout, err := time.ParseDuration(compatShutdown.StepTimeout)
	if err != nil {
		logger.Errorf("The StepTimeout configuration is invalid: %s, and we will use the default value: %s, err: %v",
			compatShutdown.StepTimeout, defaultStepTimeout.String(), err)
		timeout = defaultStepTimeout
	}

	// ignore this step
	if timeout < 0 {
		return
	}
	waitingProviderProcessedTimeout(timeout)
}

func waitingProviderProcessedTimeout(timeout time.Duration) {
	deadline := time.Now().Add(timeout)

	offlineRequestWindowTimeout, err := time.ParseDuration(compatShutdown.OfflineRequestWindowTimeout)
	if err != nil {
		logger.Errorf("The OfflineRequestWindowTimeout configuration is invalid: %s, and we will use the default value: %s, err: %v",
			compatShutdown.OfflineRequestWindowTimeout, defaultOfflineRequestWindowTimeout.String(), err)
		offlineRequestWindowTimeout = defaultOfflineRequestWindowTimeout
	}

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
	timeout, err := time.ParseDuration(compatShutdown.StepTimeout)
	if err != nil {
		logger.Errorf("The StepTimeout configuration is invalid: %s, and we will use the default value: %s, err: %v",
			compatShutdown.StepTimeout, defaultStepTimeout.String(), err)
		timeout = defaultStepTimeout
	}
	if timeout <= 0 {
		return
	}
	deadline := time.Now().Add(timeout)

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
