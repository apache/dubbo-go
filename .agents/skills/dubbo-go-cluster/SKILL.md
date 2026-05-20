---
name: dubbo-go-cluster
description: >-
  Implements and reviews dubbo-go cluster, routing, directory, and load-balancing changes. Use when the user asks about
  failover, failfast, failsafe, failback, forking, broadcast, available, zoneaware, adaptive service, Directory,
  RouterChain, condition, tag, script, affinity or Polaris routers, load balance algorithms, retries, provider
  selection, or packages cluster/cluster/, cluster/directory/, cluster/router/, and cluster/loadbalance/. Do not use
  for registry backend mechanics unless the directory snapshot contract is affected.
---

# dubbo-go Cluster

## Purpose

Use this skill to change or explain how provider invokers are listed, routed, load-balanced, retried, and invoked by cluster fault-tolerance strategies.

## When to use

Use for cluster invokers, directory snapshots, router chain behavior, dynamic routing rules, tag or condition matching, load balancing, retries, and adaptive service behavior.

Do not use for registry backend code unless registry notifications affect directory snapshots.

## Inputs

Required:
- Cluster strategy, router, load balancer, directory, or invocation selection behavior.
- Intended routing/selection behavior or observed mismatch.

Optional:
- Consumer URL, provider URLs, invocation attachments, method name, route rule, or registry event sequence.

If missing, start at `cluster/router/chain/chain.go` for routing or `cluster/cluster/` for fault tolerance.

## Workflow

1. Classify the behavior as directory update, router filtering, load balance selection, or cluster fault tolerance.
2. Read `cluster/directory/` for provider invoker snapshots and registry notification handling.
3. Read `cluster/router/chain/chain.go` and the relevant router package for filtering behavior.
4. Read `cluster/loadbalance/` for provider selection.
5. Read the relevant `cluster/cluster/*/cluster_invoker.go` for retry and error behavior.
6. Check extension registration under `common/extension/cluster.go`, `loadbalance.go`, and `router_factory.go`.
7. Read `references/routing-cluster-flow.md` before changing selection semantics.

## Output format

Return directory source, router chain result, load balance or cluster strategy, changed files, and validation performed.

## Validation

- Confirm router priority and stable ordering still produce deterministic results.
- Confirm directory snapshots are copied safely and not mutated by callers.
- Confirm retry count and business-error handling match the strategy.
- Run focused tests such as `go test ./cluster/...`; use `make test` for broad invocation behavior.

## Edge cases

- RouterChain first narrows invokers by service key and falls back to all invokers if no service-key match exists.
- Static router config is injected after builtin routers are created.
- Failover retries non-business errors but returns business errors directly.
- Method-level URL params can override interface-level routing, retries, and load balance settings.

## References

- Read `references/routing-cluster-flow.md` for directory, router, load balance, and cluster invocation details.
