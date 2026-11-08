# Edge 迁移后的流路由恢复

关联：https://github.com/singchia/frontier/issues/115

Service SDK 的 `OpenStream(ctx, edgeID)` 现在会在交出流之前确认 Frontier 已连接到目标 Edge。仅当 Frontier 明确返回本机没有该 Edge 时，集群 SDK 按 Edge 失效旧缓存，向 Frontlas 重查归属，并最多向新的 Frontier 重试一次。归属未变则返回 `service.ErrEdgeNotOnline`；调用方可用 `errors.Is` 判断。

此过程不会关闭共享 Frontier 连接，也不会重放业务数据。目标应用的错误、下一跳建流失败、普通 EOF、超时和取消都不能作为迁移证据；下一跳建流失败返回 `service.ErrEdgeStreamOpen`。已建立流不进行迁移或重放。

## 协议与兼容

- 继续使用现有 Geminio stream metadata 和 RPC 包，没有新增 Geminio 包类型或更改依赖。
- 新 SDK 设置 stream metadata 为 `frontier.stream-setup.v1`，然后在该流上调用保留 RPC `__frontier_stream_setup_v1`。
- 新 Frontier 仅为该 metadata 启用就绪确认。确认结果使用固定版本化响应值，区分 ready、edge-not-online、edge-open-failed；不向调用方泄露底层错误内容。
- 原始字节流没有前缀，应用不需要修改。旧 SDK 不带标记，继续走原有转发路径。
- 新 SDK 连接旧 Frontier 时，就绪 RPC 会转发到 Edge；对于 Geminio 的精确 `no such rpc: __frontier_stream_setup_v1` 响应回退到旧行为，不对 EOF/超时/其他错误进行猜测式回退。该 RPC 名称保留给 SDK，业务不应注册或拦截它。旧 Frontier 的历史无类型关闭行为仍不能获得自动迁移恢复，因此需要升级 Frontier 和 SDK 才能完整启用修复。
- 每次新建流增加一次就绪 RPC；连接内复用的业务请求不增加额外往返。尚未量化新建流吞吐影响，不能声称零性能开销。

## 缓存一致性

失效操作比较旧归属后再删除，避免迟到的 A 失败删除已经更新为 B 的缓存。BiMap 更新归属时同时清理旧反向索引，按 value 删除与读取 key 集合时遍历 map 的 key，而非 `struct{}` value。这也修复了旧的 Frontier 整体清理无法删除 Edge 映射的问题。

## 验证与上线

回归使用真实 TCP/Geminio 连接和 Frontier exchange：同一 SDK 在 A→B 迁移后恢复，A 上既有其他流继续可用；覆盖四种新旧 SDK/服务器组合、离线/归属未变、下一跳失败、context 取消、并发恢复及缓存清理。兼容测试的旧路径由跳过新 metadata 分支模拟，并非对所有历史 release 的穷举。

滚动更新可先部署 Frontier 再升级 SDK；回滚 SDK 或 Frontier 后保留原有转发能力，但不再保证迁移自动恢复。本变更未解决高并发压测中的其他连接拒绝或 Frontlas 的事件时序问题。

本次 `make verify-pr` 已通过（构建、流阻塞/重置、心跳与多 Edge 回归）。验证入口：`make test-edge-route-recovery`（包含 `-race -count=5` 与 `go vet`），并已接入 `make verify-pr`。另外尝试完整运行 servicebound/edgebound 包时，旧 `TestServiceManager` / `TestEdgeManagerStream` 会因 fixture 传入 nil exchange 而 panic；在未修改的 1a591c4 基线同样复现，不将其报告为本次回归通过。

追加多 Frontier 回归：同一 Service 同时保持 A/B/C 三条共享连接，三个固定 Edge 的既有流持续回显，第四个 Edge 循环迁移 18 次，并额外打开 54 条固定 Edge 的新流。每轮校验三条共享连接的对象未被替换，归属查询恰好为 4 次冷缓存查询加 18 次迁移重查。`-race -count=20` 全部通过，共 360 次迁移、1,080 条固定 Edge 新流。这里 Frontlas 查询由线程安全的测试替身提供，数据面使用真实 TCP/Geminio；不代表 Docker 部署级或百万 Edge 容量测试。
