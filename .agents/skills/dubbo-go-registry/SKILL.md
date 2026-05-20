---
name: dubbo-go-registry
description: >-
  Implements and reviews dubbo-go registry and service discovery changes. Use when the user asks about registry
  factories, Register, UnRegister, Subscribe, UnSubscribe, Notify, NotifyAll, LoadSubscribeInstances, service discovery,
  registry directory, Nacos, Zookeeper, Etcd, Polaris, interface-level or application-level registration, service
  instance listeners, or packages registry/, registry/servicediscovery/, registry/directory/, and remoting/* registry
  integrations. Do not use for metadata revision or report storage unless registry behavior depends on it.
---

# dubbo-go Registry

## Purpose

Use this skill to trace service registration and subscription from runtime URLs through registry factories, concrete registry implementations, service discovery, directory updates, and listener notifications.

## When to use

Use for provider registration, consumer subscription, registry extension registration, application-level service discovery, registry directory behavior, and registry provider integrations.

Do not use for metadata revision calculation, metadata report storage, or cluster routing unless registry notification data is the contract.

## Inputs

Required:
- Registry protocol, registry package path, URL, listener, or event behavior.
- Intended registration/subscription behavior or observed discovery bug.

Optional:
- Provider URL, consumer URL, service instance metadata, registry backend, or notification sequence.

If missing, start at `registry/registry.go` and follow the concrete protocol through `common/extension/registry.go`.

## Workflow

1. Read `registry/registry.go` and `registry/service_discovery.go` for the interface contract.
2. Check extension registration in `common/extension/registry.go` or `service_discovery.go`.
3. Trace URL loading through runtime or `internal.LoadRegistries` before entering concrete registry code.
4. For interface-level registries, inspect the concrete implementation under `registry/{nacos,zookeeper,etcdv3,polaris}` and matching `remoting/`.
5. For application-level discovery, inspect `registry/servicediscovery/service_discovery_registry.go`.
6. For consumer updates, follow listener output into `registry/directory/` and cluster routing.
7. Read `references/registry-discovery-flow.md` before changing notification semantics.

## Output format

Return registry protocol, registration or subscription flow, listener and directory impact, changed files, and validation commands or gaps.

## Validation

- Confirm `Register` and `UnRegister` use full URL matching semantics.
- Confirm `Subscribe`, `UnSubscribe`, and `LoadSubscribeInstances` preserve callback behavior.
- Confirm `NotifyAll` represents a complete list and `Notify` can represent incremental events.
- Run targeted tests such as `go test ./registry/...`; include matching `remoting/...` tests for backend-specific changes.

## Edge cases

- Application-level service discovery combines `ServiceDiscovery`, service name mapping, and metadata.
- `shouldRegister` and `shouldSubscribe` gates can skip work for some URL categories.
- Registry notification order affects directory snapshots and router updates.
- Some registry packages rely on blank imports to register extensions.

## References

- Read `references/registry-discovery-flow.md` for registry, service discovery, listener, and directory details.
