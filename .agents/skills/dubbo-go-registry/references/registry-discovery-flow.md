# Registry and Discovery Flow

## Interface Contracts

- `registry.Registry` owns provider `Register`/`UnRegister` and consumer `Subscribe`/`UnSubscribe`.
- `registry.NotifyListener` receives `Notify` and `NotifyAll`.
- `registry.ServiceDiscovery` owns application-level service instance registration and watching.
- `registry.ServiceDiscoveryRegistry` combines normal registry behavior with service instance registration.

## Extension Points

- Registry factories register with `extension.SetRegistry`.
- Service discovery implementations register with `extension.SetServiceDiscovery`.
- Service name mapping and instance customizers are separate extension points used by application-level discovery.

## Provider Registration

1. Runtime builds service export URLs and loads registry URLs.
2. Interface-level registries store provider URLs directly.
3. Application-level registry uses metadata to create `ServiceInstance` values.
4. Remote metadata may be published before service instances are registered.

## Consumer Subscription

1. Runtime builds an interface-level consumer URL.
2. Registry directory subscribes and initially loads instances when needed.
3. Registry implementation emits complete or incremental service events.
4. Directory updates invoker snapshots.
5. Router chain and cluster selection consume the new snapshot on invocation.

## Application-Level Discovery

- `registry/servicediscovery/service_discovery_registry.go` maps interface names to application names and listens by application service names.
- Service instances include metadata storage type and revision properties.
- Metadata reports may be required to resolve provider service URLs.

## Notification Rules

- Treat `NotifyAll` as a full replacement list.
- Treat single `Notify` events as incremental unless the concrete registry documents otherwise.
- Keep listener callbacks non-blocking where possible because subscription paths can be shared by many references.
