# Service 心跳恢复与独立租约

## 行为

ServiceHeartbeat 现在携带 Service、Addr，Frontlas 用原子 Lua 操作补建当前 Frontier 的映射，重复心跳不增加 service_count。ServiceOffline 也携带 Addr，避免旧连接的下线消息删除同 Frontier 的新连接。

同一 service_id 可同时映射多个 Frontier；每个 Frontier 的有效期独立计算。在线实例不能替其他实例续期。查询会清除过期映射，返回所有仍在线的实例；单实例 GetService API 按 FrontierID 排序选择第一个。原先部分查询对 Hash 执行 GET/MGET、扫描了 Frontier 前缀，现已统一为租约感知的 Hash 查询；批量查询通过 Pipeline 合并 Redis 往返。

## Redis 数据

- `frontlas:services:<service_id>`：Hash，FrontierID → JSON 元数据。
- `frontlas:service-leases:<service_id>`：Sorted Set，FrontierID → 截止毫秒。
- `frontlas:alive:services:<service_id>`：兼容现有聚合在线标记，仅表示至少一个实例在线。

期限采用 Redis TIME，避免不同 Frontlas 的主机时钟偏差。单实例期限取心跳有效期（当前 30 秒）与正值 ServiceMeta 的较小值。Hash/ZSet/聚合标记的整 key TTL 取最后一个有效实例的截止时间；查询、心跳、上下线都会原子剔除已过期实例。因此 Redis 中旧 Hash 字段可能保留到下一次操作，但查询不会把它作为在线实例返回，无需后台清扫进程。所有实例失联后整组 key 自动到期。

Online 只为首次加入的映射加计数，Offline 只对实际删除的映射减计数，心跳不加计数。过期/恢复后的计数仍由既有 FrontierStats 周期校正，需要启用 Frontier 的 metrics 上报。

## 兼容及边界

新增协议字段可选；旧 Frontier 的心跳只能续期已存在的 JSON 注册，缺失映射时明确失败。新 Frontier 的完整心跳可以把旧非 JSON Hash 值转换为 JSON，并创建独立租约。无独立租约的旧字段不作为在线记录返回；因此升级时短暂等待一次完整心跳或重新注册。

部署应先统一升级 Frontlas，再升级 Frontier；不要长期混跑新旧 Frontlas，旧服务端不会维护独立租约。回滚应成对回滚 Frontier/Frontlas 并重新注册 Service，以恢复旧格式与整 Hash TTL 语义。旧 ExpireService 接口已弃用且返回错误，因为仅 service_id 无法安全续期单个实例；使用 HeartbeatService。

未改变 Redis Cluster 跨 slot 限制。本次仍使用 FrontierID + Addr 区分已有连接，不是持久化会话世代；所有元数据丢失后的旧消息、地址复用，以及旧格式 Offline 没有地址的场景，不能提供严格 fencing 保证。没有修改 Frontier 自身的注册租约或 Geminio 旧请求跨连接问题。

## 验证

- `make test-service-heartbeat`：Frontlas 全包及 Informer 连续三轮 race 回归、静态检查。
- `make test-service-heartbeat-integration FRONTLAS_TEST_REDIS_ADDR=...`：独立真实 Redis 三轮测试，不清库，只清理测试创建的键。
- 回归覆盖 A 活跃/B 过期、最后实例失联、并发补建、计数幂等、旧心跳/Offline、新旧协议、分页/批量/单实例查询与 Redis 故障。

真实 Redis 7.4 隔离集群：两个 Entry、两个 Frontier、两个 Edge。基线有 4 个 Service 映射；仅停止 Frontlas 35 秒后映射数变为 0。启动 Frontlas 后每 5 秒轮询，第 5 次观测到全部 4 个映射恢复（约 20 秒），期间没有重启 Entry/Frontier。随后双入口 × 双边缘的 Linux 容器真实 TUN TCP half-close、UDP 0/64/1200 字节、ICMP Echo 全部通过，未授权目标仍被拒绝。此处不代表 macOS 桌面 TUN 测试。

原始输出见 `service-heartbeat-recovery/`。
