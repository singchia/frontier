# WireGuard over frontier —— example 设计文档

- 分支：`feat/wireguard`
- 日期：2026-04-21
- 目录：`examples/wireguard/`

## 1. 目标与非目标

**目标**：在 `examples/wireguard/` 下新增一组示例，演示用 frontier 打通两台独立 host 上的 WireGuard peer，使它们即便双方都无法直接路由到对方（NAT / 跨网络）时，也能通过 frontier 作为中继完成 WG 握手与数据平面通信。

**非目标**（本 example 不做）：

- 不内嵌 WG 实现（不引 `wireguard-go`、不建 TUN 接口、不生成密钥）——WG 守护进程由使用者自备
- 不做多对 peer 的 mesh / 动态路由
- 不做鉴权、加密增强（WG 自带端到端机密性，frontier 只搬字节）
- 不做 metrics、健康检查、优雅 draining
- 不做性能压测（benchmark 后续工作）

## 2. 关键决策

| 决策 | 选择 | 理由 |
|---|---|---|
| WG 协议 | UDP-only（原生） | WG 规范只定义 UDP 传输 |
| 两端角色 | 两个 edge + 一个 router（service） | WG 语义上两端对等，避免强行给一方打 service 标签 |
| frontier 传输层 | UDP（`etc/frontier_udp.yaml`） | 用上本分支新加的 UDP listener；保留 `--frontier-network tcp` 作为排障开关 |
| 数据平面通道 | geminio stream（单长流） | Stream 模型匹配，`Publish` 的 per-message ack 不适合 VPN 数据面 |
| 报文边界 | 自加 2B big-endian length-prefix framing | geminio stream 是字节流；WG 报文无内置长度字段，必须外包 |
| 配对方式 | 首帧携带 pair-id 明文字符串 | 免改 edge SDK（今天的 `OpenStream` 不透传 Meta） |

## 3. 拓扑

```
       host-A                                              host-B
  ┌──────────────┐                                    ┌──────────────┐
  │ wg0          │                                    │ wg0          │
  │ Endpoint=    │                                    │ Endpoint=    │
  │ 127.0.0.1:   │                                    │ 127.0.0.1:   │
  │   51820      │                                    │   51820      │
  │   │ ▲        │                                    │   │ ▲        │
  │  UDP UDP     │                                    │  UDP UDP     │
  │   ▼ │        │                                    │   ▼ │        │
  │ ┌─────────┐  │                                    │ ┌─────────┐  │
  │ │ wg-edge │  │                                    │ │ wg-edge │  │
  │ │ --pair=x│  │                                    │ │ --pair=x│  │
  │ └────┬────┘  │                                    │ └────┬────┘  │
  └──────┼───────┘                                    └──────┼───────┘
         │ geminio stream (UDP transport)                    │
         │ first frame: pair-id "x"                          │
         ▼                                                   ▼
       ┌─────────────────────────────────────────────────────┐
       │                 frontier (UDP)                      │
       │   :30012 edgebound     :30011 servicebound          │
       └────────────────────┬────────────────────────────────┘
                            ▼
                    ┌──────────────────┐
                    │  wg-router       │
                    │  (service="wg")  │
                    │  pair-id → 配对   │
                    └──────────────────┘
```

三类进程：

- `frontier`：UDP 监听模式（复用 `etc/frontier_udp.yaml`，无改动）
- `wg-router`：以 service 角色连接 frontier，service name 默认 `wg`；接收两侧 edge 的 stream 首帧 pair-id，两两配对后双向转发 framed 字节
- `wg-edge`：两端各一份；本机 `ListenUDP` 给 WG 当 endpoint；启动即 `OpenStream("wg")`，先写 pair-id 首帧，然后 UDP ↔ stream 双向搬运

两个 edge 都是主动连接 frontier（edge SDK 一贯行为），都只 `OpenStream`，不 `AcceptStream`。`AcceptStream` 只出现在 router。

## 4. 代码结构

```
examples/wireguard/
├── Makefile                  # build 两个二进制 + udpping 工具
├── README.md                 # 跑法、wg0 配置示例、真实 WG 验证步骤
├── edge/
│   └── edge.go
├── router/
│   └── router.go
├── cmd/
│   └── udpping/
│       └── main.go           # 测试辅助：UDP echo + 主动探包
└── internal/
    └── frame/
        ├── frame.go          # 2B length-prefix 编解码
        └── frame_test.go
```

依赖：`github.com/singchia/frontier/api/dataplane/v1/{edge,service}`、标准库、`spf13/pflag`。不新增第三方依赖。

## 5. Wire format

一个 stream 内部按顺序写入：

```
┌──── 首帧 (pair-id，开流后唯一一次) ────┐
│  [2B length N]  [N bytes pair-id utf8]
└──────────────────────────────────────┘
┌──── 数据帧 (重复，双向对称) ────────────┐
│  [2B length M]  [M bytes WG packet]
│  ...
└──────────────────────────────────────┘
```

规则：

- Length 字段：`uint16` big-endian，长度不含自己
- 首帧：stream 开通后 edge 写的第一件事，router 必须先读出 pair-id 才入配对队列
- 数据帧：首帧之后，一帧对应一个 WG UDP datagram；bridge 阶段 router 透明转发，不解析内容
- Payload 范围：1 ≤ M ≤ 65535（length==0 视为协议错误并断流）；MVP 不额外加 sanity 上限（TODO 留口）
- pair-id：ASCII/UTF-8，长度 1–255；router 用 `--max-pair-id-len` 防恶意大首帧

`internal/frame` 对外 API：

```go
package frame

// WriteFrame 把 p 前加 2B length，一次写完。
// len(p) > 65535 返回 error。
func WriteFrame(w io.Writer, p []byte) error

// ReadFrame 读 2B length，再读 length 字节。
// length==0 返回 error。
func ReadFrame(r io.Reader) ([]byte, error)
```

## 6. 数据流

### 6.1 edge 内部（两 goroutine）

```
                   wg-edge 内部状态
            ┌───────────────────────────────┐
            │ lastSrc atomic.Pointer[UDPAddr]│
            └───────────────────────────────┘

   wg0 ──UDP──► ListenUDP ──goroutine 1──► WriteFrame ──► stream
   wg0 ◄─UDP── WriteToUDP ◄─goroutine 2 ◄── ReadFrame ◄─────┘
               (to lastSrc)
```

goroutine 1（UDP → stream）：

```
n, src, err := udpConn.ReadFromUDP(buf)
if err: return
lastSrc.Store(src)
frame.WriteFrame(stream, buf[:n])
```

goroutine 2（stream → UDP）：

```
pkt, err := frame.ReadFrame(stream)
if err: return
src := lastSrc.Load()
if src == nil: 丢包 (尚未知道回包地址)
udpConn.WriteToUDP(pkt, src)
```

启动顺序：

1. `ListenUDP` 本地端口
2. `edge.NewEdge(dialer, ...)`
3. `stream = OpenStream("wg")`
4. `WriteFrame(stream, []byte(pairID))` ← 首帧
5. 起两 goroutine，阻塞等任一 goroutine 退出

> OpenStream 必须在收到 WG 第一个包之前完成，否则首个握手 UDP 会被丢弃。

### 6.2 router 的配对状态机

```
AcceptStream → ReadFrame → pair-id "x"

lock:
  if pending["x"] 存在:
     other = pending["x"]
     delete(pending, "x")
     go bridge(stream, other)
  else:
     pending["x"] = stream
unlock
```

`bridge(a, b)`：两 goroutine 分别 `for { f, _ := ReadFrame(a); WriteFrame(b, f) }`，任一侧 err 则关双方。bridge 阶段不解析 frame 内容，原样转发。

### 6.3 配对异常

- **单侧 pending 超时**：`--pair-timeout`（默认 60s）到期，踢出 pending 并关 stream
- **同 pair-id 第三条 stream 到来**：拒绝（关新流），不替换已活跃的 pair
- **配对成功后断连**：bridge 关双方 → 两端 edge 感知 stream 关 → `edge.NewEdge` 自愈底层连接 + 我们自己的 reopen 循环重开 stream → 重入 pending → 下一轮配对

## 7. CLI 参数

**`wg-edge`**

```
--frontier-addr     1.2.3.4:30012   frontier edgebound
--frontier-network  udp             tcp | udp，默认 udp
--listen            127.0.0.1:51820 本机给 wg0 用的 UDP endpoint
--pair-id           hello           两端必须一致
--service-name      wg              OpenStream 目标，默认 "wg"
--name              host-a          日志展示的可读名（可选）
```

**`wg-router`**

```
--frontier-addr     1.2.3.4:30011   frontier servicebound
--frontier-network  udp
--service-name      wg
--pair-timeout      60s             pending 单侧等待上限
--max-pair-id-len   256             首帧 length 的 sanity 上限
```

**`udpping`**（测试辅助）

```
--mode              send | echo     主动发 / 被动 echo
--listen            127.0.0.1:7000  本机 UDP
--target            127.0.0.1:51820 对端（send 模式下）
--interval          1s              send 间隔
```

## 8. 生命周期与错误处理

**edge**：

- 使用 `edge.NewEdge`（带底层重连）
- 自己维护 stream 重开循环，包含 pair-id 首帧写入；失败指数退避 1s → 30s
- UDP listener 一直存活，不因 stream 断开关闭；重连期间到达的 WG 包丢弃（WG 自带重传）
- `lastSrc` 跨 stream 重连保留

**router**：

- 使用带重连的 `service.NewService`；信任 `AcceptStream` 在底层重连后自动恢复
- pending map 用 mutex 保护；pair-timeout 到期清理，进程退出时关闭所有 pending + active stream
- bridge goroutine 不 recover panic——让错误冒出来好定位

**信号**：两个进程都 handle SIGINT/SIGTERM，关闭活动 stream、关闭 SDK 连接、直接 Exit；不做 draining。

## 9. 不做的事（YAGNI 列表）

- metrics / prometheus
- 健康检查 HTTP 端点
- 多 pair-id 并发限流
- pair-id 鉴权（HMAC / PSK）
- 主动 keepalive（WG 自有）
- frame payload 大小 sanity 上限（uint16 自限已够）
- benchmark / 压测
- CI 自动跑真实 WG 验证（需要 root + wg tools）

## 10. 测试计划

**层 1 — `internal/frame` 单元测试**

- `TestRoundTrip`：随机多帧往返
- `TestMaxPayload`：65535 字节
- `TestEmptyPayload`：length==0 视为错误
- `TestOversizedPayload`：>65535 时 WriteFrame 返回 error 且不写出
- `TestShortRead`：不完整帧返回 `io.ErrUnexpectedEOF`
- `go test ./examples/wireguard/internal/frame -race`

**层 2 — UDP echo 端到端**

本机起 frontier + wg-router + 两 wg-edge + 两 udpping（一端 send 一端 echo）。验证双向 datagram 往返成功。

```
udpping(send) → wg-edge-A → frontier → wg-router → frontier → wg-edge-B → udpping(echo)
                                                                            │
udpping(send) ← wg-edge-A ← frontier ← wg-router ← frontier ← wg-edge-B ←───┘
```

**层 3 — 真实 WG 人工验证**

README 给出 Linux 两机（或两 netns）配置步骤：生成 keypair、`wg0.conf`、`wg-quick up`、`ping 10.0.0.2` 走 WG。一次人工跑通即可 sign off，不入 CI。

## 11. 验收标准

1. `go build ./examples/wireguard/...` 通过
2. `go test ./examples/wireguard/internal/frame -race` 全绿
3. 按 README 一键起所有进程，udpping 双向 echo 成功
4. README 真实 WG 步骤人工走通

## 12. 风险与未决事项

| 项 | 说明 | 处理 |
|---|---|---|
| geminio-over-pion-UDP 的 MTU 行为 | 若 geminio 单次写 > UDP MTU 会被截断 | 测试中监控；如发现则 fallback `--frontier-network tcp` 或给 frontier 加 packet-mode 通道（超出 example 范围） |
| pair-id 明文 | 无鉴权，任何知道 pair-id 的 edge 可接入 | README disclaimer；后续可加 HMAC over pair-id |
| 单 pair 的单侧断连期间丢包 | WG 包被丢，依赖 WG 重传 | 可接受；WG 天然抗丢包 |
| edge 宿主机 WG 源端口漂移 | `lastSrc` 可能指向过期地址 | WG 默认固定 ListenPort，实务不会；若发生，下一次 WG 出包自动纠正 |
| B 侧从未出过包的死锁 | 若 B 的 WG 不主动发，B 的 `lastSrc` 永远 nil，A 的包被丢 | WG 两端都配 PersistentKeepalive 即可；README 标注 |

## 13. 复用分支

直接在 `feat/wireguard` 上加目录，不新开分支。该分支现有的一个 commit（UDP listener 支持）与本 example 正交，不冲突。
