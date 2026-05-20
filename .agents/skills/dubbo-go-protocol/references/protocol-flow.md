# Protocol Flow

## Core Interfaces

- `protocol/base/base_protocol.go`: `Protocol` with `Export`, `Refer`, and `Destroy`.
- `protocol/base/base_invoker.go`: `Invoker`, availability, destroyed state, safe URL access.
- `protocol/base/base_exporter.go`: `Exporter` and unexport behavior.
- `protocol/base/invocation.go` and `protocol/result/`: invocation data and result contract.

## Extension Registration

- Concrete protocols register through `common/extension/protocol.go`.
- The filter wrapper registers the synthetic `filter` protocol.
- Runtime export usually calls `extension.GetProtocol(protocolwrapper.FILTER)` so service and reference filters wrap the concrete protocol.

## Export Flow

1. Runtime creates an invoker with a service URL.
2. `protocolwrapper.FILTER.Export` resolves the concrete protocol from `invoker.GetURL().Protocol`.
3. It wraps the invoker with provider filters from `service.filter`.
4. The concrete protocol exports the invoker and stores an exporter key.
5. Server/remoting code receives requests and invokes the exported invoker.

## Refer Flow

1. Runtime or registry directory creates a URL for a remote provider.
2. `protocolwrapper.FILTER.Refer` resolves the concrete protocol from `url.Protocol`.
3. The concrete protocol creates a remote invoker.
4. The wrapper applies consumer filters from `reference.filter`.
5. Cluster and directory code call `Invoke` on the resulting invoker.

## URL Rules

- Use params for values that belong in Dubbo URLs or are needed by remote-compatible components.
- Use attributes for in-process objects such as service implementations, TLS configs, application config, Triple config, shutdown config, and generated service info.
- Treat `common.URL` as immutable after construction except for narrowly scoped builder/setup paths.

## Destroy Rules

- `BaseProtocol.Destroy` destroys invokers and unexports exporters.
- Concrete protocols should close remoting clients, servers, and exporter maps without double-unregistering unless implementing registry-specific cleanup.
- Avoid holding locks while calling external `Destroy` or `UnExport` code.
