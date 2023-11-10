package client

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/global"
	"dubbo.apache.org/dubbo-go/v3/registry"
	"github.com/creasty/defaults"
	"strconv"
	"time"
)

type ClientOptions struct {
	Consumer   *global.ConsumerConfig
	Registries map[string]*global.RegistryConfig
}

func defaultClientOptions() *ClientOptions {
	return &ClientOptions{
		Consumer:   global.DefaultConsumerConfig(),
		Registries: make(map[string]*global.RegistryConfig),
	}
}

func (cliOpts *ClientOptions) init(opts ...ClientOption) error {
	for _, opt := range opts {
		opt(cliOpts)
	}
	if err := defaults.Set(cliOpts); err != nil {
		return err
	}
	return nil
}

type ClientOption func(*ClientOptions)

func WithClientCheck() ClientOption {
	return func(opts *ClientOptions) {
		opts.Consumer.Check = true
	}
}

func WithClientURL(url string) ClientOption {
	return func(opts *ClientOptions) {
		opts.Consumer.URL = url
	}
}

// todo(DMwangnima): change Filter Option like Cluster and LoadBalance
func WithClientFilter(filter string) ClientOption {
	return func(opts *ClientOptions) {
		opts.Consumer.Filter = filter
	}
}

// todo(DMwangnima): think about a more ideal configuration style
func WithClientRegistryIDs(registryIDs []string) ClientOption {
	return func(opts *ClientOptions) {
		if len(registryIDs) > 0 {
			opts.Consumer.RegistryIDs = registryIDs
		}
	}
}

func WithClientRegistry(opts ...registry.Option) ClientOption {
	regOpts := registry.NewOptions(opts...)

	return func(cliOpts *ClientOptions) {
		if cliOpts.Registries == nil {
			cliOpts.Registries = make(map[string]*global.RegistryConfig)
		}
		cliOpts.Registries[regOpts.ID] = regOpts.Registry
	}
}

//func WithClientShutdown(opts ...graceful_shutdown.Option) ClientOption {
//	sdOpts := graceful_shutdown.NewOptions(opts...)
//
//	return func(cliOpts *ClientOptions) {
//		cliOpts.Shutdown = sdOpts.Shutdown
//	}
//}

// ========== Cluster Strategy ==========

func WithClientClusterAvailable() ClientOption {
	return func(opts *ClientOptions) {
		opts.Consumer.Cluster = constant.ClusterKeyAvailable
	}
}

func WithClientClusterBroadcast() ClientOption {
	return func(opts *ClientOptions) {
		opts.Consumer.Cluster = constant.ClusterKeyBroadcast
	}
}

func WithClientClusterFailBack() ClientOption {
	return func(opts *ClientOptions) {
		opts.Consumer.Cluster = constant.ClusterKeyFailback
	}
}

func WithClientClusterFailFast() ClientOption {
	return func(opts *ClientOptions) {
		opts.Consumer.Cluster = constant.ClusterKeyFailfast
	}
}

func WithClientClusterFailOver() ClientOption {
	return func(opts *ClientOptions) {
		opts.Consumer.Cluster = constant.ClusterKeyFailover
	}
}

func WithClientClusterFailSafe() ClientOption {
	return func(opts *ClientOptions) {
		opts.Consumer.Cluster = constant.ClusterKeyFailsafe
	}
}

func WithClientClusterForking() ClientOption {
	return func(opts *ClientOptions) {
		opts.Consumer.Cluster = constant.ClusterKeyForking
	}
}

func WithClientClusterZoneAware() ClientOption {
	return func(opts *ClientOptions) {
		opts.Consumer.Cluster = constant.ClusterKeyZoneAware
	}
}

func WithClientClusterAdaptiveService() ClientOption {
	return func(opts *ClientOptions) {
		opts.Consumer.Cluster = constant.ClusterKeyAdaptiveService
	}
}

// ========== LoadBalance Strategy ==========

func WithClientLoadBalanceConsistentHashing() ClientOption {
	return func(opts *ClientOptions) {
		opts.Consumer.Loadbalance = constant.LoadBalanceKeyConsistentHashing
	}
}

func WithClientLoadBalanceLeastActive() ClientOption {
	return func(opts *ClientOptions) {
		opts.Consumer.Loadbalance = constant.LoadBalanceKeyLeastActive
	}
}

func WithClientLoadBalanceRandom() ClientOption {
	return func(opts *ClientOptions) {
		opts.Consumer.Loadbalance = constant.LoadBalanceKeyRandom
	}
}

func WithClientLoadBalanceRoundRobin() ClientOption {
	return func(opts *ClientOptions) {
		opts.Consumer.Loadbalance = constant.LoadBalanceKeyRoundRobin
	}
}

func WithClientLoadBalanceP2C() ClientOption {
	return func(opts *ClientOptions) {
		opts.Consumer.Loadbalance = constant.LoadBalanceKeyP2C
	}
}

func WithClientLoadBalanceXDSRingHash() ClientOption {
	return func(opts *ClientOptions) {
		opts.Consumer.Loadbalance = constant.LoadBalanceKeyLeastActive
	}
}

func WithClientRetries(retries int) ClientOption {
	return func(opts *ClientOptions) {
		opts.Consumer.Retries = strconv.Itoa(retries)
	}
}

func WithClientGroup(group string) ClientOption {
	return func(opts *ClientOptions) {
		opts.Consumer.Group = group
	}
}

func WithClientVersion(version string) ClientOption {
	return func(opts *ClientOptions) {
		opts.Consumer.Version = version
	}
}

func WithClientJSON() ClientOption {
	return func(opts *ClientOptions) {
		opts.Consumer.Serialization = constant.JSONSerialization
	}
}

func WithClientProvidedBy(providedBy string) ClientOption {
	return func(opts *ClientOptions) {
		opts.Consumer.ProvidedBy = providedBy
	}
}

// todo(DMwangnima): implement this functionality
//func WithAsync() ClientOption {
//	return func(opts *ClientOptions) {
//		opts.Consumer.Async = true
//	}
//}

func WithClientParams(params map[string]string) ClientOption {
	return func(opts *ClientOptions) {
		opts.Consumer.Params = params
	}
}

// todo(DMwangnima): implement this functionality
//func WithClientGeneric(generic bool) ClientOption {
//	return func(opts *ClientOptions) {
//		if generic {
//			opts.Consumer.Generic = "true"
//		} else {
//			opts.Consumer.Generic = "false"
//		}
//	}
//}

func WithClientSticky(sticky bool) ClientOption {
	return func(opts *ClientOptions) {
		opts.Consumer.Sticky = sticky
	}
}

// ========== Protocol to consume ==========

func WithClientProtocolDubbo() ClientOption {
	return func(opts *ClientOptions) {
		opts.Consumer.Protocol = constant.Dubbo
	}
}

func WithClientProtocolTriple() ClientOption {
	return func(opts *ClientOptions) {
		opts.Consumer.Protocol = "tri"
	}
}

func WithClientProtocolJsonRPC() ClientOption {
	return func(opts *ClientOptions) {
		opts.Consumer.Protocol = "jsonrpc"
	}
}

func WithClientProtocol(protocol string) ClientOption {
	return func(opts *ClientOptions) {
		opts.Consumer.Protocol = protocol
	}
}

func WithClientRequestTimeout(timeout time.Duration) ClientOption {
	return func(opts *ClientOptions) {
		opts.Consumer.RequestTimeout = timeout.String()
	}
}

func WithClientForce(force bool) ClientOption {
	return func(opts *ClientOptions) {
		opts.Consumer.ForceTag = force
	}
}

func WithClientMeshProviderPort(port int) ClientOption {
	return func(opts *ClientOptions) {
		opts.Consumer.MeshProviderPort = port
	}
}
