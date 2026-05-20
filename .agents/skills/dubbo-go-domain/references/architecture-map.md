# dubbo-go Architecture Map

## Subsystem Ownership

| Area | Primary paths | Owns | Related skill |
| --- | --- | --- | --- |
| Configuration | `loader.go`, `config/`, `global/`, `config_center/`, `common/config/` | YAML or option loading, `RootConfig`, `InstanceOptions`, config center, hot reload | `dubbo-go-config` |
| Runtime API | `dubbo.go`, `client/`, `server/`, `proxy/`, `graceful_shutdown/` | Public client/server construction, service registration, reference creation, export/refer lifecycle | `dubbo-go-runtime` |
| Protocols | `protocol/`, `remoting/`, `protocol/protocolwrapper/`, `common/url.go` | `Protocol`, `Invoker`, `Exporter`, codecs, transport clients and servers, URL semantics | `dubbo-go-protocol` |
| Registry discovery | `registry/`, `registry/servicediscovery/`, `registry/directory/`, `remoting/{nacos,zookeeper,etcdv3,polaris}/` | Registry factories, register/unregister, subscribe/unsubscribe, service instance notifications | `dubbo-go-registry` |
| Metadata | `metadata/`, `metadata/info/`, `metadata/report/`, `metadata/mapping/` | MetadataInfo, metadata service export, metadata reports, service name mapping, revisions | `dubbo-go-metadata` |
| Cluster and routing | `cluster/cluster/`, `cluster/directory/`, `cluster/router/`, `cluster/loadbalance/` | Fault tolerance, directory listing, route chain, load balancing, retries | `dubbo-go-cluster` |
| Filters and observability | `filter/`, `metrics/`, `otel/`, `logger/`, `protocol/protocolwrapper/` | Filter chain, metrics, tracing, logging, TPS and auth interception | `dubbo-go-observability-filters` |
| Tools and codegen | `tools/`, `Makefile` | CLI, schema generation, protoc plugins, import formatting, RPC contract scanner | `dubbo-go-tools` |

## Common End-to-End Flows

Provider startup:
1. Config is loaded into `InstanceOptions` or `RootConfig`.
2. `Instance.NewServer` or `server.NewServer` builds server options.
3. `ServiceOptions.Export` registers service metadata in `common.ServiceMap`, builds an export URL, wraps the protocol with filters, exports the service, and registers provider URLs or service instances.
4. Metadata is collected and may be published through a metadata report.

Consumer startup:
1. Config is loaded into client and reference options.
2. `ReferenceOptions.Refer` builds an interface-level URL, loads direct or registry URLs, creates registry or protocol invokers, and builds the service proxy.
3. Registry notifications update directory invokers.
4. Cluster, router chain, load balancer, filters, and protocol invoker handle each invocation.

Protocol invocation:
1. Runtime code builds a `common.URL` with transport params and non-transport attributes.
2. `extension.GetProtocol` resolves the registered protocol.
3. `protocolwrapper.FILTER` wraps provider or consumer invokers with configured filters.
4. The concrete protocol handles codec, remoting, request mapping, response mapping, and destroy semantics.

Registry and metadata:
1. Provider export adds service URLs to metadata and registers URLs or application-level service instances.
2. Consumers subscribe through registry directories or service discovery registries.
3. Service name mapping and metadata reports connect interface names, application names, and metadata revisions.

## Boundary Rules

- Public API behavior belongs to runtime even when the root cause is config.
- Transport behavior belongs to protocol even when filters observe it.
- Service instance registration belongs to registry; metadata revision calculation and report storage belong to metadata.
- Routing decisions belong to cluster; rule source or dynamic config loading belongs to config or registry depending on the producer.
