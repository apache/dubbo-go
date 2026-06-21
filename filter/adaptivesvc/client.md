# Client-Side Adaptive Service

This document covers the consumer-side `adaptiveService` cluster, P2C load
balancing, local metrics, and how provider `remaining` feedback is used.

## Related Files

Consumer-side cluster registration and creation:

- `cluster/cluster/adaptivesvc/cluster.go`
- `cluster/cluster/adaptivesvc/cluster_invoker.go`

Consumer-side load balancing and local metrics:

- `cluster/loadbalance/p2c/loadbalance.go`
- `cluster/loadbalance/p2c/loadbalance_test.go`
- `cluster/metrics/local_metrics.go`
- `cluster/metrics/utils.go`
- `cluster/metrics/metrics.go`

Configuration and cluster creation:

- `config/consumer_config.go`
- `config/reference_config.go`
- `client/options.go`
- `client/action.go`
- `registry/protocol/protocol.go`
- `registry/directory/directory.go`

Shared constants and error helpers:

- `common/constant/cluster.go`
- `common/constant/key.go`
- `cluster/utils/adaptivesvc.go`

## Configuration and Usage

### Configuration Files

Consumer-side config field:

```yaml
dubbo:
  consumer:
    adaptive-service: true
```

When `consumer.adaptive-service` is enabled, the reference configuration is
overridden to:

```text
cluster = adaptiveService
loadbalance = p2c
```

This is the simplest and safest consumer configuration path because it sets both
required values together.

### API Options

Consumer-side reference options:

```go
client.WithClusterAdaptiveService()
client.WithLoadBalanceP2C()
```

Consumer-side client-wide options:

```go
client.WithClientClusterAdaptiveService()
client.WithClientLoadBalanceP2C()
```

When using these explicit API options, make sure both the cluster and load
balancer are configured. The adaptive service cluster rejects load balancers other
than `p2c`.

### Required Imports

The adaptive service cluster and P2C load balancer are extension implementations
registered through package `init()` functions. Applications need to import the
all-in-one imports package or import the required implementations directly:

```go
import _ "dubbo.apache.org/dubbo-go/v3/imports"

// or import only what is needed
import _ "dubbo.apache.org/dubbo-go/v3/cluster/cluster/adaptivesvc"
import _ "dubbo.apache.org/dubbo-go/v3/cluster/loadbalance/p2c"
```

The provider side must also enable adaptive service. Otherwise the consumer can
still send the request, but the provider will not write
`adaptive-service.remaining` back into response attachments, so the consumer will
not get useful capacity metrics for later P2C decisions.

## Architecture

```text
consumer side
├── cluster registration and creation
│   ├── package init()
│   │   └── extension.SetCluster("adaptiveService", newAdaptiveServiceCluster)
│   ├── reference URL contains cluster=adaptiveService
│   │   └── registry/direct reference creation resolves the cluster key
│   └── adaptiveServiceCluster.Join(directory)
│       └── builds adaptiveServiceClusterInvoker
│
├── adaptiveServiceClusterInvoker
│   ├── Invoke()
│   │   ├── list provider invokers from Directory
│   │   ├── require loadbalance == p2c
│   │   ├── get p2c load balancer from extension registry
│   │   ├── select provider invoker with lb.Select()
│   │   ├── set request attachment
│   │   │   └── adaptive-service.enabled = "1"
│   │   ├── invoke selected provider
│   │   ├── read response attachment
│   │   │   └── adaptive-service.remaining
│   │   └── update LocalMetrics
│   │       └── provider URL + method + hill-climbing -> remaining
│   │
│   └── next Invoke()
│       └── p2c can use the previously stored remaining value
│
└── p2c load balancer
    ├── package init()
    │   └── extension.SetLoadbalance("p2c", newP2CLoadBalance)
    └── p2cLoadBalance.Select()
        ├── randomly pick two provider invokers
        ├── read LocalMetrics for both providers
        │   └── method + hill-climbing -> remaining
        └── select the provider with larger remaining capacity
```

## Adaptive Service Cluster

`cluster/cluster/adaptivesvc` registers the `adaptiveService` cluster, and
`cluster/loadbalance/p2c` registers the `p2c` load balancer. Consumer config
writes both keys into the reference URL; registry or direct-reference creation
then resolves the cluster key and calls `Join()` to build the final consumer
invoker.

### adaptiveServiceClusterInvoker

Initialization and invoker creation:

- `cluster/cluster/adaptivesvc/cluster.go` registers the cluster implementation
  under the cluster key `adaptiveService`.
- Consumer configuration writes `cluster=adaptiveService` and `loadbalance=p2c`
  into the reference URL.
- Registry reference creation reads the cluster key from the consumer service URL,
  resolves the cluster implementation through `extension.GetCluster()`, and calls
  `Join(registryDirectory)`.
- Direct-reference creation can build a static directory and call
  `Join(static.NewDirectory(invokers))`.

Cluster registration:

```go
extension.SetCluster(constant.ClusterKeyAdaptiveService, newAdaptiveServiceCluster)
```

Registry path:

```text
ReferenceConfig.Refer()
└── registryProtocol.Refer()
    ├── extension.GetDirectoryInstance(...)
    ├── directory.Subscribe(...)
    ├── extension.GetCluster("adaptiveService")
    └── adaptiveServiceCluster.Join(registryDirectory)
```

Direct URL path:

```text
ReferenceConfig.Refer()
└── adaptiveServiceCluster.Join(static.NewDirectory(invokers))
```

`Join()` returns an invoker:

```go
func (c *adaptiveServiceCluster) Join(directory directory.Directory) base.Invoker {
	return clusterpkg.BuildInterceptorChain(newAdaptiveServiceClusterInvoker(directory))
}
```

After this step, the consumer proxy holds the returned
`adaptiveServiceClusterInvoker`. `Join()` is called during reference
initialization, not for every RPC request.

Runtime use:

- `adaptiveServiceClusterInvoker.Invoke()` receives the per-request
  `invocation`. The invocation carries method name, arguments, attachments, and
  local attributes through the consumer cluster, load balancer, filters, and
  protocol invoker.
- The invoker lists provider invokers from its `Directory`.
- It requires the load balancer key to be `p2c`.
- It resolves the P2C load balancer and delegates provider selection to
  `lb.Select(invokers, invocation)`.
- It sets `adaptive-service.enabled = "1"` in request attachments before invoking
  the selected provider.
- It reads `adaptive-service.remaining` from response attachments and updates
  local method metrics.

```text
adaptiveServiceClusterInvoker.Invoke(ctx, invocation)
├── list available provider invokers
│   └── ivk.Directory.List(invocation)
├── validate invokers
│   └── ivk.CheckInvokers(invokers, invocation)
├── get load balance key from provider URL
│   └── loadbalance defaults to p2c
├── reject unsupported load balancers
│   └── adaptive service currently supports only p2c
├── load P2C load balancer
│   └── extension.GetLoadbalance("p2c")
├── select one provider invoker
│   └── lb.Select(invokers, invocation)
├── mark the request as adaptive-service enabled
│   └── invocation.SetAttachment("adaptive-service.enabled", "1")
├── invoke the selected provider
│   └── invoker.Invoke(ctx, invocation)
├── skip metric updates if adaptive service itself failed
├── read response attachment
│   └── adaptive-service.remaining
├── parse remaining as an integer
└── update consumer local method metrics
    └── metrics.LocalMetrics.SetMethodMetrics(...)
```

The metric update stores the provider's remaining capacity under this tuple:

```text
provider URL + method name + hill-climbing -> remaining
```

The next P2C selection can then use this local metric to prefer providers with
more remaining capacity.

### p2c load balancer

Initialization:

- `cluster/loadbalance/p2c/loadbalance.go` registers the load balancer under the
  loadbalance key `p2c`.
- The adaptive service cluster requires this load balancer. If the reference URL
  contains another loadbalance key, `adaptiveServiceClusterInvoker.Invoke()`
  returns an error.

Load balancer registration:

```go
extension.SetLoadbalance(constant.LoadBalanceKeyP2C, ...)
```

Runtime use:

- `adaptiveServiceClusterInvoker` loads P2C with `extension.GetLoadbalance(lbKey)`.
- `adaptive-service.remaining` is returned by the provider in response
  attachments. The adaptive cluster parses it, stores it in `LocalMetrics`, and
  P2C uses that stored value on later requests.
- P2C randomly samples two provider invokers instead of scanning all providers.
- It reads local method metrics for both sampled providers using
  `invocation.ActualMethodName()` and the metric key `hill-climbing`.
- If one sampled provider has no local metrics, P2C selects that provider
  directly so it can receive traffic and report its first capacity sample.
- If both providers have metrics, P2C compares their remaining capacity and
  selects the provider with the larger value.

```text
p2cLoadBalance.Select(invokers, invocation)
├── randomly pick two provider invokers
├── read LocalMetrics for provider i
│   └── provider URL + ActualMethodName() + hill-climbing
├── read LocalMetrics for provider j
│   └── provider URL + ActualMethodName() + hill-climbing
├── select a provider directly if its metrics are missing
└── otherwise select the provider with larger remaining capacity
```

The consumer stores provider `remaining` with this shape:

```go
metrics.LocalMetrics.SetMethodMetrics(
	invoker.GetURL(),
	invocation.MethodName(),
	metrics.HillClimbing,
	uint64(remaining),
)
```

The local metrics key is conceptually:

```text
provider instance key + provider invoker key + method name + hill-climbing
```

`hill-climbing` here means the metric value comes from the provider-side
HillClimbing limiter. The consumer does not run the Hill Climbing algorithm.

The feedback loop is one request behind:

```text
request N returns provider remaining
request N+1 and later can use that remaining during P2C selection
```

## Attachments

Adaptive service uses the following attachment keys:

| Key | Direction | Meaning |
| --- | --- | --- |
| `adaptive-service.enabled` | consumer -> provider | Marks this request as participating in adaptive service. The value is `1`. |
| `adaptive-service.remaining` | provider -> consumer | Provider method limiter's remaining capacity after the request is processed. |
| `adaptive-service.inflight` | provider -> consumer | Provider method limiter's current in-flight request count. |

The provider filter also uses this local invocation attribute:

| Key | Scope | Meaning |
| --- | --- | --- |
| `adaptive-service.updater` | provider process only | Stores the limiter updater returned by `Acquire()` so `OnResponse()` can call `DoUpdate()`. |

Only `adaptive-service.remaining` is currently consumed by the consumer-side
adaptive service cluster. `adaptive-service.inflight` is returned by the provider
but is not used by P2C today.

## End-to-End Client Flow

```text
consumer adaptive cluster
├── selects provider with p2c
├── sends adaptive-service.enabled = 1
└── invokes provider

provider adaptive filter
├── sees adaptive-service.enabled = 1
├── acquires method-level limiter
├── invokes business logic
├── updates limiter on response
└── returns adaptive-service.remaining and adaptive-service.inflight

consumer adaptive cluster
├── reads adaptive-service.remaining
├── stores it in LocalMetrics
└── lets later P2C calls use it for provider selection
```
