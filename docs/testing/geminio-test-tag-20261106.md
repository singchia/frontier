# Geminio 测试 tag 集成

基线为远端 main `dda09a10387750fba85aab0853439081715ffcc6`（包含 Edge 旧会话重连修复）。依赖固定为 `github.com/singchia/geminio v1.3.0-test.20261106.1`，通过 Go module 下载并校验；不是未提交的临时 Geminio 源码。

最新 main 仅升级依赖时，跨 Frontier reset 回归失败：接收端得到 EOF，而不是 ErrStreamAborted。将此前的三处本地接入修改迁入：

- Service SDK 使用 OpenStreamContext，并贯穿集群查找/首次拨号取消。
- 缓存映射存在而 Frontier 连接缺失时重新查询地址。
- raw relay 异常复制时 AbortStream 配对两端，正常 EOF 仍优雅关闭。

验证：
- Linux arm64 Frontier 和 Frontlas 通过根 Makefile 编译。
- SDK/exchange race 回归通过。
- 慢流隔离和跨 Frontier reset 连续三轮 race 通过（106.665 秒）。
- 最新 Edge 重连相关测试连续五轮 race 通过。
- Frontlas config/repo race 通过；其余 Frontlas 包暂无测试。
- SDK/exchange/Frontlas 静态检查通过。

已知基线问题：完整 edgebound 单测的 TestEdgeManagerStream 传入 nil exchange，newEdgeManager 中 exchange.AddEdgebound(em) 在创建 Geminio 连接之前即 panic。本分支没有修改此项，也不将完整包测试标记通过。

本分支不包含 Frontlas 断线后的注册状态对账修复，也不包含 Geminio 跨连接旧请求 ClientID 修复。

## 隔离数据面实测

使用本分支 Frontier/Frontlas，以及 Trustio `195d09c`，在独立 Docker Compose 项目中启动 Redis、双 Frontier、双 Entry、双 Edge 和两个应用。通过根 Makefile 构建；Geminio replace 指向从 Go module 下载的上述 tag 目录。没有修改现有部署。

- Linux 容器真实 TUN → gVisor → MASQUE → Entry → Frontier → Edge → 应用：双入口 × 双边缘均通过 TCP echo/half-close、UDP 0/64/1200 字节和 ICMP Echo；未授权 ICMP 目标被拒绝。这里不代表 macOS 桌面 TUN 测试。
- 原生 macOS MASQUE 客户端经 Docker 发布端口进行 120 秒负载：同一 QUIC 连接上 4 条长流、120 次负载流取消、86 次新探测流全部通过。此项未经过 TUN，也不是独占环境性能基准。
- 将 Frontlas 停机 35 秒（元数据 TTL 30 秒）再启动，等待 12 秒后：Redis 的两个 Frontier alive key 均存在，而两个 Edge metadata key 均不存在。随后既有拓扑上的新客户端 TCP/UDP 探测仍成功。因此只能确认已有路径可用，不能认定发现注册恢复正常；缓存路径可能掩盖缺失注册。本分支尚未修复重连后的元数据对账。

测试输出保存于本目录的 `geminio-test-tag-20261106/`。

后续 Edge 心跳补建修复和复测结果见 [Edge 心跳恢复注册](edge-heartbeat-recovery.md)。上面的故障记录为修复前基线。
