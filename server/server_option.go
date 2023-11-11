package server

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"strconv"
	"time"
)

// ---------- For user ----------

// ========== LoadBalance Strategy ==========

func WithServerLoadBalanceConsistentHashing() ServerOption {
	return func(opts *ServerOptions) {
		opts.Provider.Loadbalance = constant.LoadBalanceKeyConsistentHashing
	}
}

func WithServerLoadBalanceLeastActive() ServerOption {
	return func(opts *ServerOptions) {
		opts.Provider.Loadbalance = constant.LoadBalanceKeyLeastActive
	}
}

func WithServerLoadBalanceRandom() ServerOption {
	return func(opts *ServerOptions) {
		opts.Provider.Loadbalance = constant.LoadBalanceKeyRandom
	}
}

func WithServerLoadBalanceRoundRobin() ServerOption {
	return func(opts *ServerOptions) {
		opts.Provider.Loadbalance = constant.LoadBalanceKeyRoundRobin
	}
}

func WithServerLoadBalanceP2C() ServerOption {
	return func(opts *ServerOptions) {
		opts.Provider.Loadbalance = constant.LoadBalanceKeyP2C
	}
}

func WithServerLoadBalanceXDSRingHash() ServerOption {
	return func(opts *ServerOptions) {
		opts.Provider.Loadbalance = constant.LoadBalanceKeyLeastActive
	}
}

// warmUp is in seconds
func WithServerWarmUp(warmUp time.Duration) ServerOption {
	return func(opts *ServerOptions) {
		warmUpSec := int(warmUp / time.Second)
		opts.Provider.Warmup = strconv.Itoa(warmUpSec)
	}
}

// ========== Cluster Strategy ==========

func WithServerClusterAvailable() ServerOption {
	return func(opts *ServerOptions) {
		opts.Provider.Cluster = constant.ClusterKeyAvailable
	}
}

func WithServerClusterBroadcast() ServerOption {
	return func(opts *ServerOptions) {
		opts.Provider.Cluster = constant.ClusterKeyBroadcast
	}
}

func WithServerClusterFailBack() ServerOption {
	return func(opts *ServerOptions) {
		opts.Provider.Cluster = constant.ClusterKeyFailback
	}
}

func WithServerClusterFailFast() ServerOption {
	return func(opts *ServerOptions) {
		opts.Provider.Cluster = constant.ClusterKeyFailfast
	}
}

func WithServerClusterFailOver() ServerOption {
	return func(opts *ServerOptions) {
		opts.Provider.Cluster = constant.ClusterKeyFailover
	}
}

func WithServerClusterFailSafe() ServerOption {
	return func(opts *ServerOptions) {
		opts.Provider.Cluster = constant.ClusterKeyFailsafe
	}
}

func WithServerClusterForking() ServerOption {
	return func(opts *ServerOptions) {
		opts.Provider.Cluster = constant.ClusterKeyForking
	}
}

func WithServerClusterZoneAware() ServerOption {
	return func(opts *ServerOptions) {
		opts.Provider.Cluster = constant.ClusterKeyZoneAware
	}
}

func WithServerClusterAdaptiveService() ServerOption {
	return func(opts *ServerOptions) {
		opts.Provider.Cluster = constant.ClusterKeyAdaptiveService
	}
}

func WithServerGroup(group string) ServerOption {
	return func(opts *ServerOptions) {
		opts.Provider.Group = group
	}
}

func WithServerVersion(version string) ServerOption {
	return func(opts *ServerOptions) {
		opts.Provider.Version = version
	}
}

func WithServerJSON() ServerOption {
	return func(opts *ServerOptions) {
		opts.Provider.Serialization = constant.JSONSerialization
	}
}

// WithToken should be used with WithFilter("token")
func WithServerToken(token string) ServerOption {
	return func(opts *ServerOptions) {
		opts.Provider.Token = token
	}
}

func WithServerNotRegister() ServerOption {
	return func(opts *ServerOptions) {
		opts.Provider.NotRegister = true
	}
}
