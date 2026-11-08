# 动态 TLS 联通验证（2026-11-08）

环境：本机进程内真实 Frontier edgebound / servicebound / exchange，真实 TCP、TLS 和 Geminio；测试签发回调模拟 Cert 业务。发现测试使用真实 gRPC server 提供可变的 Frontlas 拓扑响应，未使用生产 Frontlas 集群。没有替换现有部署。

命令：`make test-registration-id-precision test-dynamic-tls-lifecycle`，所有测试启用 race detector。生命周期与扩容发现各重复三次；每轮生命周期在恢复后执行 8 并发 × 8 个短连接及首个 echo RPC。刷新间隔缩短为 1 秒，信任配置租约 6 秒，以验证过期行为；不是生产默认值。

## 已验证

- 没有初始证书时拒绝 Edge TLS 接入。
- 两个独立 CA 签发的 Edge 均可接入并完成 RPC，不受信任的 CA 被拒绝。
- 回调把服务端证书从 serial 1 更新到 2；新连接实际观测到 serial 2。
- 移除 CA-A 后，CA-A 新连接拒绝，CA-B 新连接正常；CA-A 已有连接继续完成 RPC。
- Cert Service 关闭后，在租约内使用缓存继续接入；租约过期拒绝新连接，已有连接仍工作。
- Cert Service 重连并注册回调后，TLS 接入恢复。
- SDK 先在空拓扑中注册回调，再新增 Frontier；每轮约 10 秒自动发现、建连、补注册并成功调用回调。

## 发现并修复：64 位 ID 索引精度

最初关闭 Cert Service 后，业务 Service 的 `get_edge_id` 注册记录消失，新 Edge 报 `service not online`，但已连接 Edge 的 echo 仍成功。不是 TLS 缓存失效。

BuntDB 的 `IndexJSON` 对数值使用 float64 比较。Geminio 默认生成的 ID 大于 2^53，相邻 ID 可能被索引当成相等。按 Service ID 删除 RPC / Topic 时，会误删其他 Service 的记录；Edge RPC 同样受影响。

新增 `TestLargeIDsKeepIndependentRegistrations` 在旧代码下失败，改为精确 uint64 索引后通过。覆盖 Service RPC、Service Topic、Edge RPC，以及 2^53 和 MaxUint64 边界。仅更换索引比较函数，不改 ID 编码或线协议。

## 仍未通过：短连接首个 RPC 时序

修复后连续三轮：两轮完整通过；一轮有一个 `no such rpc: echo`。192 次短连接有 1 次首个 RPC 失败，整个测试命令因此返回非零，不能报告全绿。

初步定位：Geminio 应用层可以在 Frontier 的 `handleConn` 返回并安装 `Hijack` 转发器前处理请求。客户端建连成功并不保证 Frontier 的应用转发器已就绪。需要进一步通过初始化顺序或明确的连接就绪机制修复；测试没有用 sleep 或自动重试掩盖失败。

证据日志：本机 `/private/tmp/frontier-tls-lifecycle-fixed.log`。复现入口及断言已保留在 `pkg/frontier/exchange/tls_lifecycle_test.go`。该现象尚未完成无 TLS 对照验证，因此不把初步定位视为最终根因结论。
