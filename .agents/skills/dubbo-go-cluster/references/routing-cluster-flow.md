# Routing and Cluster Flow

## Invocation Path

1. A proxy or protocol caller invokes a cluster invoker.
2. The cluster invoker asks its directory for invokers matching the invocation.
3. Directory output is routed through `RouterChain`.
4. The cluster strategy chooses or retries invokers using a load balancer.
5. The selected protocol invoker performs the remote call.

## Directory

- Static directory holds fixed invokers.
- Registry directory reacts to registry notifications and rebuilds invoker snapshots.
- Directory output should be treated as a snapshot for one invocation path.

## Router Chain

- `RouterChain.Route` copies invokers, filters by service key when possible, then applies routers in priority order.
- Router factories are registered through `extension.SetRouterFactory`.
- Static router configs are injected through routers implementing `StaticConfigSetter`.
- Dynamic routers may update internal state on `Notify`.

## Load Balance

- Load balance implementations live under `cluster/loadbalance/`.
- Selection receives candidate invokers and the invocation.
- Method-level URL params can affect algorithm choice.

## Cluster Strategies

- `failover`: retry non-business errors and reselect providers.
- `failfast`: fail immediately.
- `failsafe`: swallow failures according to strategy.
- `failback`: retry in background.
- `forking`, `broadcast`, `available`, `zoneaware`, and `adaptivesvc` have specialized selection behavior.

## Contract Checks

- Keep router output deterministic for the same snapshot and invocation.
- Avoid mutating shared invoker slices from callers.
- Preserve distinction between business errors and transport or provider errors.
