# Client and Server Lifecycle

## Public Entry Points

- `dubbo.NewInstance` initializes `InstanceOptions`.
- `Instance.NewClient` and `Instance.NewServer` clone config into runtime options.
- `dubbo.Load` starts configured consumer and provider services.
- `client.NewClient` and `server.NewServer` are lower-level constructors.

## Consumer Reference Flow

1. `ReferenceOptions.refer` resolves interface, methods, and optional generated `ClientInfo`.
2. It applies adaptive service overrides and mesh URL rewriting when enabled.
3. It builds an interface-level `common.URL` with reference params and runtime attributes.
4. It calls `processURL` to choose direct URL mode or registry URL loading.
5. It calls `buildInvoker` to create protocol or cluster invokers.
6. It creates a proxy through the configured proxy factory and implements the user service when needed.
7. It registers the reference protocol for graceful shutdown.

## Provider Service Flow

1. `ServiceOptions.Export` validates TPS and related extension names.
2. It loads registry URLs unless `NotRegister` is set.
3. It registers service methods in `common.ServiceMap`.
4. It builds one export URL per protocol config.
5. It adds in-process attributes such as RPC service, TLS config, Triple config, application, shutdown, provider, and registry config.
6. It exports through `extension.GetProtocol(protocolwrapper.FILTER)` so filters wrap the concrete protocol.
7. It records metadata and registers provider URLs or service instances.

## Proxy Flow

- `proxy.Proxy` is used for generated or reflective client callbacks.
- Proxy factories are registered under `common/extension/proxy_factory.go`.
- `proxy/proxy_factory/default.go` handles reflective calls; pass-through proxy handles direct invocation.

## Shutdown Flow

- Runtime registers protocols with `graceful_shutdown`.
- Protocol `Destroy` should close invokers and unexport providers.
- Registry-only unregister behavior should use `base.RegistryUnregisterer` when supported.

## Contract Checks

- Keep params and attributes consistent between consumer and provider URL construction.
- Do not assume one protocol per service.
- Keep direct URL and registry URL behavior equivalent after URL normalization.
