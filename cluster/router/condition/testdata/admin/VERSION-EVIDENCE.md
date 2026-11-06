# Condition and tag router version evidence

Recorded 2026-11-06 on macOS arm64 (Go 1.26.1). Base:
`415f5d17ed0ec451ba8686cd30dfdc508734bba6` (`develop`) plus this commit's
test/fixture/doc changes only. No product source or `go.mod` / `go.sum` edits.

SourceOnly is inspection. RuntimePass / RuntimeFail below for historical tags
are **local router tests**, not config-center traffic. B3 is only
`configVersion < 3.1`. B4 is provider/consumer application lookup.
#3251 implemented B4; #3772 added compatibility coverage; #3773 fixed B3 on
develop. GOST / Nacos SDK versions are those pinned by each tag's unmodified
go.mod.

## Formal tags versus develop

| SDK commit (formal tag) | GOST / Nacos v2 | B3 SourceOnly | B3 local | B4 SourceOnly | B4 local |
|---|---|---|---|---|---|
| [012a247](https://github.com/apache/dubbo-go/blob/012a247993ba63a9e9033761c85ac9ccc950c594/cluster/router/condition/dynamic_router.go#L247) (v3.3.0) | v1.14.0 / v2.2.2 | swapped ~L254 | RuntimeFail | [lookup](https://github.com/apache/dubbo-go/blob/012a247993ba63a9e9033761c85ac9ccc950c594/cluster/router/tag/router.go#L56) | RuntimeFail (uncached) |
| [489ee6a](https://github.com/apache/dubbo-go/blob/489ee6aed49b9a08cd0fb63232a4eaec5fce6e1a/cluster/router/condition/dynamic_router.go#L247) (v3.3.1) | v1.14.3 / v2.2.5 | swapped ~L254 | RuntimeFail | [lookup](https://github.com/apache/dubbo-go/blob/489ee6aed49b9a08cd0fb63232a4eaec5fce6e1a/cluster/router/tag/router.go#L56) | RuntimeFail (uncached) |
| [cdda54b](https://github.com/apache/dubbo-go/blob/cdda54be1f353e372a80940729cdb380702c325e/cluster/router/condition/dynamic_router.go#L289) (v3.3.2) | v1.14.3 / v2.2.5 | swapped ~L296 | RuntimeFail | [lookup](https://github.com/apache/dubbo-go/blob/cdda54be1f353e372a80940729cdb380702c325e/cluster/router/tag/router.go#L77) | RuntimePass (cached + uncached) |
| [415f5d17](https://github.com/apache/dubbo-go/blob/415f5d17ed0ec451ba8686cd30dfdc508734bba6/cluster/router/condition/dynamic_router.go#L341) (develop base) | v1.15.0 / v2.3.5 | corrected ~L348–349 | RuntimePass | [lookup](https://github.com/apache/dubbo-go/blob/415f5d17ed0ec451ba8686cd30dfdc508734bba6/cluster/router/tag/router.go#L77) | RuntimePass (cached + uncached) |

On each historical tag, asymmetric v3.0 pairs (`enabled=true,force=false` and
`enabled=false,force=true`) failed correct provider selection; symmetric pairs
passed. The structured v3.1 control passed on all three. Omitting `enabled`
panicked on each tag (separate from the swapped-flag bug). v3.3.0 / v3.3.1 B4
failed at the first enabled assertion, so later lifecycle steps did not run.

**Upgrade note:** v3.3.2 covers B4 only. B3 still needs a later formal release
that includes the develop fix. This task does not decide backport scope.

## Fixture provenance

| Fixture(s) | Admin revision / writer | Resource; transport |
|---|---|---|
| condition `service-*.yaml` | `71c70cf9…`, #1595 `meshresource.MarshalRuleSpec` | `org.apache.dubbo.condition.CompatService:1.0.0:compat.condition-router`; Nacos data ID = resource, group `dubbo`; ZK `/dubbo/config/<resource>` |
| condition `application-*.yaml` | same + `CreateConditionRule` / `UpdateInstanceTrafficStatus` | `condition-compat-provider.condition-router`; same group/path |
| condition `consumer-application-initial.yaml` | same | `condition-compat-consumer.condition-router` |
| condition `legacy-enabled-omitted.yaml` | `b9ee1b77…`, old `MarshalRuleSpec` | same service resource path |
| tag `enabled` / `updated` / `force` / `legacy-disabled-omitted` | `b9ee1b77…`, `RuleGovernor` Create/Update; captured from ZK | `tag-compat-provider.tag-router`; ZK `/dubbo/config/…`, Nacos group `dubbo` |
| tag `fixed-disabled` | `71c70cf9…`, same governor / #1595 writer; captured from ZK | same |
| inline v3.0 / v3.1 rules in `admin_compatibility_test.go` | **parser fixtures** (hand-written; no Admin writer SHA) | none |

Committed fixture payloads are unchanged apart from license comments. RPC tests
retarget application names and provider addresses per run; those derived bytes
are test inputs, not Admin captures. See the two fixture READMEs.

Local B3 coverage: matching and missing host for all four enabled/force pairs
plus omitted fields. Local B4 coverage: reuse `#3772` lifecycle (different apps,
enabled/disabled, update, delete, force/fallback, cached/uncached). Config-center
RPC harness uses native watchers and selected Triple providers (no external
`Process`).

## Config-center matrix (develop base)

Nacos 2.5.2 and ZooKeeper 3.9.4 via `testdata/admin/compose.yaml`. Configured
rows use Nacos group `dubbo` or ZK root `/dubbo` + namespace `config`.

| Case | Observed location | Result |
|---|---|---|
| condition Nacos (configured) | unique `*.condition-router` data IDs under group `dubbo` | RuntimePass (create/update/delete/recreate + selected RPC; app disable/restore; forced empty; consumer create-after-subscribe) |
| condition ZK (configured) | `/dubbo/config/` + unique keys | RuntimePass (same checks) |
| tag Nacos (configured) | unique `*.tag-router` under group `dubbo` | RuntimePass (identities, cache paths, update/disable/force/fallback, delete/recreate, create-after-subscribe) |
| tag ZK (configured) | `/dubbo/config/<unique>.tag-router` | RuntimePass (same) |
| tag ZK **defaults** | Admin writes `/dubbo/config/<key>`; SDK queries `/dubbo/config/dubbo/<key>` | RuntimeFail before selected RPC ([Admin #1625](https://github.com/apache/dubbo-admin/issues/1625), [#1629](https://github.com/apache/dubbo-admin/issues/1629), [#1523](https://github.com/apache/dubbo-admin/issues/1523)) |
| condition ZK defaults | same path mismatch | NOT_RUN (SourceOnly) |
| historical tags × real config-center | n/a | NOT_RUN |
| temporary modfile | n/a | NOT_RUN |

Configured Pass does **not** mean default interoperability is fixed.

## Commands and exit codes (rerun 2026-11-06)

Ports `18848` / `21811` were free; compose `up -d --wait` exited 0.
`export PATH=$PATH:/opt/homebrew/bin`.

| Command | Exit | Result |
|---|---:|---|
| `go test ./cluster/router/condition/... ./cluster/router/tag/... -count=1` | 0 | PASS |
| `go test -race ./cluster/router/condition ./cluster/router/tag -count=1` | 0 | PASS |
| `DUBBO_GO_CONDITION_NACOS=127.0.0.1:18848 DUBBO_GO_CONDITION_ZK=127.0.0.1:21811 go test ./cluster/router/condition -run '^TestAdminConditionConfigCenterRPC$' -v -count=1` | 0 | PASS |
| `DUBBO_TAG_NACOS=127.0.0.1:18848 DUBBO_TAG_ZOOKEEPER=127.0.0.1:21811 go test -tags adminintegration ./cluster/router/tag -run '^TestAdminTagConfigCenterRPC$' -v -count=1` | 0 | PASS |
| `DUBBO_TAG_ZOOKEEPER=127.0.0.1:21811 go test -tags adminintegration ./cluster/router/tag -run '^TestAdminTagZooKeeperDefaultRPC$' -v -count=1` | 1 | product FAIL (default path) |
| `make fmt` | 0 | PASS |
| `make test` | 0 | PASS |

PowerShell (after compose is up):

```powershell
$env:DUBBO_GO_CONDITION_NACOS='127.0.0.1:18848'; $env:DUBBO_GO_CONDITION_ZK='127.0.0.1:21811'
go test ./cluster/router/condition -run '^TestAdminConditionConfigCenterRPC$' -v -count=1
$env:DUBBO_TAG_NACOS='127.0.0.1:18848'; $env:DUBBO_TAG_ZOOKEEPER='127.0.0.1:21811'
go test -tags adminintegration ./cluster/router/tag -run '^TestAdminTagConfigCenterRPC$' -v -count=1
```

## Replaying historical local tests

Isolated worktrees; overlay **only** test files. Adapt import /
`NewApplicationRouter` / cache APIs on older tags; do not change product source.
Fixture YAML for tag tests is read from this checkout via
`DUBBO_3793_TAG_FIXTURES`. Observed exits after adaptation (v3.3.0, v3.3.1,
v3.3.2): condition explicit/control `1,1,1`; omitted-enabled panic `1,1,1`;
tag lifecycle `1,1,0`. Unadapted overlays on 3.3.0/3.3.1 were BLOCKED at build.

```sh
repo_root=$(pwd)
for release_tag in v3.3.0 v3.3.1 v3.3.2; do
  git worktree add --detach "/tmp/dubbo3793-$release_tag" "$release_tag"
  cp cluster/router/condition/admin_compatibility_test.go \
    "/tmp/dubbo3793-$release_tag/cluster/router/condition/admin_compatibility_test.go"
  cp cluster/router/tag/admin_compatibility_test.go \
    "/tmp/dubbo3793-$release_tag/cluster/router/tag/admin_compatibility_test.go"
done
# See prior commit evidence / PR discussion for the small Python API adaptation
# (protocol import, NewApplicationRouter(), cache path, fixture env).
for release_tag in v3.3.0 v3.3.1 v3.3.2; do
  (
    cd "/tmp/dubbo3793-$release_tag" || exit 1
    go test ./cluster/router/condition \
      -run '^TestConditionVersionFlagRouting/(enabled_force|enabled_fallback|disabled_force|disabled_fallback)$|^TestConditionVersion31Control$' \
      -count=1 -v; printf 'condition exit=%s\n' "$?"
    go test ./cluster/router/condition -run '^TestConditionVersionFlagRouting/omitted_enabled$' \
      -count=1 -v; printf 'omitted exit=%s\n' "$?"
    DUBBO_3793_TAG_FIXTURES="$repo_root/cluster/router/tag/testdata/admin" \
      go test ./cluster/router/tag -run '^TestAdminTagRuleLifecycle$' -count=1 -v
    printf 'tag exit=%s\n' "$?"
  )
done
```

Inline v3.0 parser fixture shape (flags and host substituted in the test):

```yaml
configVersion: v3.0
scope: service
key: org.apache.dubbo.condition.CompatService
runtime: true
enabled: true
force: false
conditions:
  - '=>host=127.0.0.2'
```

The v3.1 control fixture is inline in `TestConditionVersion31Control`.
