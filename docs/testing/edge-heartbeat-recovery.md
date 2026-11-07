# Edge 心跳恢复注册

## 修复范围

Frontlas 的 EdgeHeartbeat 原先只执行 EXPIRE；元数据或 alive key 过期后无法补建，Redis 返回 false 又被当作成功。现在 Frontier 在心跳里附带已有的连接地址，Frontlas 原子校验归属、补建缺失元数据并恢复 alive key。

- 现有 Edge 记录的 FrontierID 或连接地址与新心跳不符时返回错误，不覆盖、不续期。
- 记录不存在时用心跳的完整信息恢复，不需要 Edge 断线重连。
- 不修改 edge_count，避免重复心跳放大计数；由既有 FrontierStats 周期同步数量。
- Edge 元数据 TTL 改用 EdgeMeta（此前误用 ServiceMeta）。
- 新增 JSON 字段 addr 为可选字段，旧 Frontlas 忽略该字段。旧 Frontier 对已有同 Frontier 记录仍可续期；记录缺失时明确报错，不能补建。完整恢复能力需要两端升级。

本记录对应 Edge 修复；ServiceHeartbeat 的后续修复见 [Service 心跳恢复与独立租约](service-heartbeat-recovery.md)。没有修改 Frontier 的 30 秒注册租约机制，也没有修改 Geminio 的跨连接旧请求问题。

## 并发边界

归属比较与写入在一个 Redis Lua 脚本中完成，保证心跳不能覆盖已存在的新归属。这里的 FrontierID + Addr 不是持久化会话世代：所有元数据丢失时，无法判断一个延迟到达的旧心跳是否已经失效；相同地址复用也无法区分。若要求跨元数据丢失的严格旧会话隔离，需要额外设计会话世代与 fencing，不能把本次修复当作这一保证。

沿用现有 Redis key 格式和多 key 原子操作部署前提；没有新增 Redis Cluster 跨 slot 支持。

## 验证

`make test-edge-heartbeat` 连续三轮 race 测试覆盖：

- 元数据过期后 20 个并发心跳补建，计数不膨胀。
- 只有 alive key 缺失时恢复。
- 旧格式心跳兼容续期、缺失记录明确失败。
- 跨 Frontier 迁移和同 Frontier 新地址建立后，旧心跳不能覆盖或续期。
- Redis 错误传播；EdgeMeta 与 ServiceMeta 不同时使用正确 TTL。

真实 Redis 7.4 + 双 Frontier/Entry/Edge 隔离集群实测通过：

1. 基线 TCP/UDP 探测通过。
2. 单独停止 Frontlas 35 秒，两个 Edge 元数据 key 均过期（EXISTS = 0）。
3. 启动 Frontlas，不重启 Frontier/Edge；按 5 秒间隔轮询，第 5 次观测到两个元数据和两个 alive key 全部恢复（约 20 秒）。
4. 重启两个 Entry 清空缓存，再跑 Linux 容器真实 TUN：双入口 × 双边缘 TCP half-close、UDP 0/64/1200 字节、ICMP Echo 全部通过；未授权目标仍被拒绝。

这不是 macOS 桌面 TUN 测试。原始结果见 `edge-heartbeat-recovery/`。

准备阶段重建测试镜像时，曾遇到旧 Frontier 注册租约尚未过期而启动被拒绝；待租约过期后重启测试 Frontier/Edge 建立基线。上述故障注入阶段只停止 Frontlas，未重启 Frontier/Edge；此修复没有解决租约等待问题。

## 回滚

回滚 Frontier/Frontlas 到本分支前一个版本即可，未修改持久数据结构；回滚后恢复原来只续期、不能补建的行为。测试 Geminio tag 保持不变。
