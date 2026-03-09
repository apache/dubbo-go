# 昨天提交审查计划（简明版）

## 先说结论

昨天这批 `graceful_shutdown` 相关改动里，我现在确认的情况是：

- `gracefulShutdownCallbacks` 那个 map 的未加锁问题，你刚刚已经修了，这个点不再算待修问题。
- `customShutdownCallbacks` 现在还是直接返回 live list，但按当前仓库用法看，基本都是启动期注册；我暂时不把它当成这轮必须修的问题。
- 现在真正还要盯的，主要还有 3 个点：
  1. `consumer_filter` 在短路返回时，可能把 `ConsumerActiveCount` 减成负数
  2. invoker 被标记成不可用后，TTL 到期不会恢复
  3. `triple` 的 graceful shutdown 回调改动，需要再做一次回归验证，确保没有类型断言问题

## 这次不处理什么

- 不重构整个 graceful shutdown 流程
- 不改 `protocol_filter_wrapper` 的整体语义
- 不把范围扩大到 auth、token、输入校验这些无关模块
- 不把 `customShutdownCallbacks` 当成这轮已确认 bug 强行处理

## 要做的 3 件事

### 1. 验证 triple 的 graceful shutdown 回调修复

要做什么：
- 保留你现在在 `protocol/triple/triple.go` 里的修复
- 补一个很小的回归测试，确认 graceful shutdown callback 被调用时不会 panic

完成标准：
- `go test ./protocol/grpc ./protocol/triple` 通过

建议提交：
- `fix(triple): stabilize graceful shutdown callback registration`

---

### 2. 修 consumer 计数错误

要做什么：
- 修 `filter/graceful_shutdown/consumer_filter.go`
- 保证只有真正加过 `ConsumerActiveCount` 的请求，最后才会减回去
- 避免 closing 场景下“没加过却减了”

为什么要修：
- 现在 filter 被短路时，`OnResponse` 还是会走
- 这会让 `ConsumerActiveCount` 出现错误，极端情况下可能变负数

完成标准：
- 补一个失败优先测试，证明短路路径不会把计数减成负数
- `go test ./filter/graceful_shutdown` 通过

建议提交：
- `fix(graceful-shutdown): balance consumer active counts`

---

### 3. 修 invoker 不恢复的问题

要做什么：
- 修 `filter/graceful_shutdown/consumer_filter.go`
- invoker 因 closing 被临时标记为不可用后，等 TTL 到期要恢复 `SetAvailable(true)`
- 同时保留你现在对 closing 错误的收窄判断，不要把 `EOF`、`connection reset by peer` 这类普通网络错误误判成 closing

为什么要修：
- 现在的逻辑会把 invoker 拉黑式地排除出去
- 但 TTL 到了以后没有恢复，可能导致这个节点一直选不中

完成标准：
- 补测试证明 TTL 到期后 invoker 会恢复 available
- 补测试证明 `EOF` 和 `connection reset by peer` 不会触发隔离
- `go test ./filter/graceful_shutdown ./protocol/base` 通过

建议提交：
- `fix(graceful-shutdown): recover invoker availability after quarantine`

## 验证方式

至少跑这些：

```bash
go test ./protocol/grpc ./protocol/triple
go test ./filter/graceful_shutdown
go test ./filter/graceful_shutdown ./protocol/base
```

如果当前环境支持 race：

```bash
go test -race ./filter/graceful_shutdown
```

如果不支持 race（比如 `CGO_ENABLED=0`），那就以补的回归测试为准，不强行要求跑 `-race`。

## 推荐执行顺序

1. 先做 triple 回调验证
2. 再修 consumer 计数
3. 最后修 TTL 恢复和错误分类

## 最终目标

做到这几点就算完成：

- 已修好的 callback map 不被误改回去
- `ConsumerActiveCount` 不再记错
- invoker 不会被“临时隔离后永久失联”
- `triple` graceful shutdown 回调路径有测试兜底
