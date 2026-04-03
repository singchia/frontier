# Frontier 测试计划

**文档版本:** 1.1
**创建日期:** 2026-04-01
**测试执行:** Claude Code

---

## 目录

1. [项目概述与测试范围](#一项目概述与测试范围)
2. [测试分类与编号规则](#二测试分类与编号规则)
3. [单元测试](#三单元测试)
4. [基准测试](#四基准测试)
5. [端到端测试](#五端到端测试)
6. [安全测试](#六安全测试)
7. [测试覆盖矩阵](#七测试覆盖矩阵)
8. [执行命令速查](#八执行命令速查)

---

## 一、项目概述与测试范围

### 架构简述

Frontier 是一个面向边缘节点的反向代理与消息总线，核心数据流如下：

```
Edge (边缘节点)
    │  TCP/TLS
    ▼
┌─────────────────────────────────┐
│           Frontier              │
│  ┌───────────┐  ┌────────────┐  │
│  │ Edgebound │  │Servicebound│  │
│  └─────┬─────┘  └─────┬──────┘  │
│        └──────┬────────┘        │
│          ┌────▼────┐            │
│          │Exchange │            │
│          └─────────┘            │
└─────────────────────────────────┘
    │  TCP/TLS
    ▼
Service (业务服务)
```

**核心能力（测试重点）：**
- **Edgebound**：接受 Edge 接入，管理 Edge 连接生命周期
- **Servicebound**：接受 Service 接入，管理 Service 注册与路由
- **Exchange**：Edge ↔ Service 之间的 RPC 转发、消息转发、Stream 透传
- **Repo（DAO）**：内存数据库（buntdb / sqlite）存储 Edge/Service 元数据

### 测试范围

| 包含 | 不包含 |
|------|--------|
| frontier 核心数据面（edgebound / servicebound / exchange） | frontlas（集群控制面） |
| Repo DAO（membuntdb / memsqlite） | Kubernetes Operator |
| 配置加载（config） | MQ 外部依赖集成（Kafka/NATS/NSQ 等） |
| 基准测试（bench / batch） | 控制面 REST/gRPC API |

---

## 二、测试分类与编号规则

### 2.1 测试类别

| 类别编码 | 类别名称 | 测试工具 | 目录 |
|---------|---------|---------|------|
| UNIT | 单元测试 | `go test` | `pkg/frontier/...` |
| BENCH | 基准测试 | `go test -bench` / 独立二进制 | `test/bench/`, `test/batch/` |
| E2E | 端到端测试 | `go test` + 本地 frontier 实例 | `test/e2e/`（待创建）|
| SEC | 安全测试 | `go test -race` / `go test -fuzz` | `test/security/`（待创建）|

### 2.2 编号规则

格式：`[类别]-[模块]-[序号]`

| 缩写 | 模块 |
|------|------|
| EDGE | Edgebound |
| SVC  | Servicebound |
| EXCH | Exchange |
| REPO | Repo/DAO |
| CONF | Config |
| CONN | 连接管理 |
| RPC  | RPC 转发 |
| MSG  | 消息转发 |
| STRM | Stream 透传 |

---

## 三、单元测试

### 3.1 已有测试（`pkg/`）

| 编号 | 测试名称 | 文件 | 验证点 |
|------|---------|------|--------|
| UNIT-CONF-001 | TestGenDefaultConfig | `pkg/frontier/config/config_test.go` | 默认配置序列化到 YAML |
| UNIT-CONF-002 | TestGenAllConfig | `pkg/frontier/config/config_test.go` | 完整配置序列化到 YAML |
| UNIT-EDGE-001 | TestEdgeManager | `pkg/frontier/edgebound/edge_manager_test.go` | Edge 接入→在线→断开完整流程 |
| UNIT-EDGE-002 | TestEdgeManagerStream | `pkg/frontier/edgebound/edge_dataplane_test.go` | Edge 批量创建 Stream（1000条） |
| UNIT-SVC-001  | TestServiceManager | `pkg/frontier/servicebound/service_manager_test.go` | Service 接入→在线→断开完整流程 |
| UNIT-REPO-001 | TestListEdges（buntdb） | `pkg/frontier/repo/dao/membuntdb/dao_edge_test.go` | Edge 列表按地址前缀/时间范围查询 |
| UNIT-REPO-002 | TestListEdgeRPCs（buntdb） | `pkg/frontier/repo/dao/membuntdb/dao_edge_test.go` | EdgeRPC 多条件查询 |
| UNIT-REPO-003 | TestListServices（buntdb） | `pkg/frontier/repo/dao/membuntdb/dao_service_test.go` | Service 列表查询及分页 |
| UNIT-REPO-004 | TestDeleteService（buntdb） | `pkg/frontier/repo/dao/membuntdb/dao_service_test.go` | Service 删除后数量校验 |
| UNIT-REPO-005 | TestListServiceRPCs（buntdb） | `pkg/frontier/repo/dao/membuntdb/dao_service_test.go` | ServiceRPC 按 ID/时间查询 |
| UNIT-REPO-006 | TestListServiceTopics（buntdb） | `pkg/frontier/repo/dao/membuntdb/dao_service_test.go` | ServiceTopic 多条件查询 |
| UNIT-REPO-007 | TestCreateEdge（sqlite） | `pkg/frontier/repo/dao/memsqlite/dao_edge_test.go` | Edge 写入 sqlite |
| UNIT-REPO-008 | TestCountEdges（sqlite） | `pkg/frontier/repo/dao/memsqlite/dao_edge_test.go` | 批量写入后计数校验（10000条）|
| UNIT-REPO-009 | BenchmarkCreateEdge（sqlite） | `pkg/frontier/repo/dao/memsqlite/dao_edge_test.go` | Edge 写入并发性能基线 |
| UNIT-REPO-010 | BenchmarkGetEdge（sqlite） | `pkg/frontier/repo/dao/memsqlite/dao_edge_test.go` | Edge 读取并发性能基线 |
| UNIT-REPO-011 | BenchmarkListEdges（sqlite） | `pkg/frontier/repo/dao/memsqlite/dao_edge_test.go` | 10万条数据分页查询性能 |
| UNIT-REPO-012 | TestListServices（sqlite） | `pkg/frontier/repo/dao/memsqlite/dao_service_test.go` | Service 与 RPC/Topic 联合查询 |

### 3.2 待补充测试

| 编号 | 建议测试名称 | 目标文件 | 验证点 |
|------|------------|---------|--------|
| UNIT-EXCH-001 | TestExchangeForwardRPCToService | `pkg/frontier/exchange/` | RPC 从 Edge 转发到 Service 全流程 |
| UNIT-EXCH-002 | TestExchangeForwardMessageToService | `pkg/frontier/exchange/` | 消息从 Edge 转发到 Service 全流程 |
| UNIT-EXCH-003 | TestExchangeForwardRPCToEdge | `pkg/frontier/exchange/` | RPC 从 Service 转发到指定 Edge |
| UNIT-EXCH-004 | TestExchangeForwardMessageToEdge | `pkg/frontier/exchange/` | 消息从 Service 投递到指定 Edge |
| UNIT-EXCH-005 | TestExchangeStreamToService | `pkg/frontier/exchange/` | Stream 从 Edge 透传到 Service |
| UNIT-EXCH-006 | TestExchangeStreamToEdge | `pkg/frontier/exchange/` | Stream 从 Service 透传到 Edge |
| UNIT-EXCH-007 | TestExchangeEdgeOnlineOffline | `pkg/frontier/exchange/` | Edge 上下线事件通知 Service |
| UNIT-EDGE-003 | TestEdgeManagerMultiple | `pkg/frontier/edgebound/` | 多 Edge 并发接入，ID 分配唯一性 |
| UNIT-SVC-002  | TestServiceManagerRouting | `pkg/frontier/servicebound/` | 按 RPC/Topic/Name 查找 Service |

---

## 四、基准测试

基准测试为**独立二进制**，需先启动一个本地 frontier 实例，再运行对应客户端程序。

### 4.1 已有基准测试

| 编号 | 测试方法 | 文件 | 场景 | 
|------|---------|------|------|
| BENCH-CALL-001 | `BenchmarkEdgeCallService` | `test/bench/benchmark_test.go` | Edge 端通过 Frontier 调用 Service 的 RPC 的并发吞吐 (QPS) |
| BENCH-PUB-001  | `BenchmarkEdgePublishMessage` | `test/bench/benchmark_test.go` | Edge 端通过 Frontier 发布消息的并发吞吐 (QPS) |
| BENCH-OPEN-001 | `BenchmarkEdgeOpenStream` | `test/bench/benchmark_test.go` | Edge 端通过 Frontier 打开并关闭 Stream 的并发吞吐 (QPS) |
| BENCH-EDGE-001 | `edges` (独立二进制) | `test/batch/edges/edges.go` | 大规模边缘节点长连接模拟 |

### 4.2 待补充基准测试

| 编号 | 建议测试名称 | 目录 | 场景 |
|------|------------|------|------|
| BENCH-CONN-001 | `BenchmarkConnect` | `test/bench/benchmark_test.go` | 测量每秒可接入 Edge 连接数（TPS） |
| BENCH-STRM-001 | `BenchmarkStreamTransfer` | `test/bench/benchmark_test.go` | Stream 双向数据传输带��测试 |

### 4.3 基准测试执行方式

```bash
# 运行所有性能基准测试并打印内存分配情况
go test -bench=. -benchmem -v ./test/bench/...

# 运行特定模块基准测试并设定测试时间（如 10s）
go test -bench=BenchmarkEdgeCallService -benchtime=10s ./test/bench/...

# 大规模连接模拟（独立二进制）
cd test/batch/edges && make
./edges --address 127.0.0.1:30011 --count 10000 --nseconds 30
```

---

## 五、端到端测试

E2E 测试在进程内启动 frontier（不依赖外部进程），验证 Edge → Frontier → Service 的完整链路。

### 5.1 测试目录结构（待创建）

```
test/e2e/
├── main_test.go        # TestMain：启动/停止嵌入式 frontier
├── helper.go           # 公共 dialer、frontier 启动工具函数
├── conn_test.go        # 连接生命周期测试
├── rpc_test.go         # RPC 转发测试
├── message_test.go     # 消息转发测试
└── stream_test.go      # Stream 透传测试
```

### 5.2 E2E 测试用例

#### 连接管理（CONN）

| 编号 | 测试名称 | 验证点 |
|------|---------|--------|
| E2E-CONN-001 | TestEdgeConnect | Edge 成功接入 frontier，edgeID 非零 |
| E2E-CONN-002 | TestEdgeConnectAndClose | Edge 正常关闭，frontier 侧资源清理完毕 |
| E2E-CONN-003 | TestEdgeConnectWithMeta | Edge 携带 meta 接入，Service 侧通过 `EdgeOnline` 回调获得正确 meta |
| E2E-CONN-004 | TestMultiEdgeConnect | 100 个 Edge 并发接入，全部成功且 edgeID 唯一 |
| E2E-CONN-005 | TestServiceConnect | Service 成功接入并注册 RPC/Topic |
| E2E-CONN-006 | TestServiceConnectAndClose | Service 下线后，Edge 侧 RPC 调用返回 `ErrServiceNotOnline` |

#### RPC 转发（RPC）

| 编号 | 测试名称 | 验证点 |
|------|---------|--------|
| E2E-RPC-001 | TestEdgeCallService | Edge 通过 frontier 调用 Service 注册的 RPC，返回正确响应 |
| E2E-RPC-002 | TestServiceCallEdge | Service 通过 frontier 调用 Edge 注册的 RPC，指定 edgeID |
| E2E-RPC-003 | TestRPCEdgeIDCarry | Service 调用 Edge RPC 时，frontier 正确在 Custom 字段附加 edgeID |
| E2E-RPC-004 | TestRPCTargetEdgeOffline | 目标 Edge 已下线时，Service 调用返回 `ErrEdgeNotOnline` |
| E2E-RPC-005 | TestRPCTargetRPCNotFound | Service 调用不存在的 RPC 方法时，Edge 返回错误 |
| E2E-RPC-006 | TestRPCConcurrent | 10 个 Edge 同时调用 Service RPC，无错误，响应数据一致 |

#### 消息转发（MSG）

| 编号 | 测试名称 | 验证点 |
|------|---------|--------|
| E2E-MSG-001 | TestEdgePublishToService | Edge Publish 消息，Service 通过已注册 Topic 正确 Receive |
| E2E-MSG-002 | TestServicePublishToEdge | Service Publish 消息到指定 edgeID，Edge 正确 Receive |
| E2E-MSG-003 | TestMessageTopicRoute | 多个 Service 注册不同 Topic，Edge 消息按 Topic 路由到正确 Service |
| E2E-MSG-004 | TestMessageTopicNotFound | Edge 发布不存在 Topic 的消息，返回 `ErrTopicNotOnline` |
| E2E-MSG-005 | TestMessageConcurrent | 10 个 Edge 并发 Publish，消息不丢失，数量一致 |

#### Stream 透传（STRM）

| 编号 | 测试名称 | 验证点 |
|------|---------|--------|
| E2E-STRM-001 | TestEdgeOpenStreamToService | Edge OpenStream 到指定 Service，Service AcceptStream 收到 |
| E2E-STRM-002 | TestServiceOpenStreamToEdge | Service OpenStream 到指定 edgeID，Edge AcceptStream 收到 |
| E2E-STRM-003 | TestStreamRawDataForward | Stream 内 Raw IO 双向传输，数据内容完整一致 |
| E2E-STRM-004 | TestStreamMessageForward | Stream 内 Message 双向转发，数据内容完整一致 |
| E2E-STRM-005 | TestStreamRPCForward | Stream 内 RPC 双向调用，返回值正确 |
| E2E-STRM-006 | TestStreamClose | Stream 一端关闭，另一端收到 EOF，资源正确清理 |
| E2E-STRM-007 | TestStreamTargetEdgeOffline | 目标 Edge 不在线时，Service OpenStream 返回错误 |

#### 资源管理（RES）

| 编号 | 测试名称 | 验证点 |
|------|---------|--------|
| E2E-RES-001 | TestResourceCleanupOnEdgeClose | Edge 关闭后，Repo 中 Edge 及其 RPC 记录被删除 |
| E2E-RES-002 | TestResourceCleanupOnServiceClose | Service 关闭后，Repo 中 Service 及其 RPC/Topic 记录被删除 |
| E2E-RES-003 | TestGoroutineNoLeak | 100 次 Edge 接入/断开循环后，goroutine 数量回落到基线 |

### 5.3 E2E 执行方式

```bash
# 运行所有 E2E 测试
go test -v -timeout 5m ./test/e2e/

# 带竞态检测
go test -race -v -timeout 5m ./test/e2e/

# 运行单个用例
go test -v -run TestEdgeCallService ./test/e2e/
```

---

## 六、安全测试

### 6.1 测试目录结构（待创建）

```
test/security/
├── main_test.go
├── input_test.go       # 输入合法性验证
├── boundary_test.go    # 边界值测试
├── race_test.go        # 并发竞态测试
└── fuzz_test.go        # 模糊测试
```

### 6.2 安全测试用例

#### 输入合法性（INPUT）

| 编号 | 测试名称 | 验证点 |
|------|---------|--------|
| SEC-INPUT-001 | TestLargePayloadRPC | RPC 请求携带 64MB payload，frontier 不崩溃，返回正常错误或正常响应 |
| SEC-INPUT-002 | TestEmptyPayloadRPC | RPC/消息 payload 为空（nil / 0字节），frontier 正常处理 |
| SEC-INPUT-003 | TestSpecialCharactersMeta | Edge meta 包含特殊字符（换行、空字节、Unicode），frontier 正常接受 |
| SEC-INPUT-004 | TestNilMessageData | Edge 发送 nil data 的消息，frontier 不 panic |

#### 边界值（BOUND）

| 编号 | 测试名称 | 验证点 |
|------|---------|--------|
| SEC-BOUND-001 | TestMaxEdgeConnections | 超大量 Edge 并发接入（如 65535），系统不崩溃，超出限制时返回可预期错误 |
| SEC-BOUND-002 | TestMaxStreamsPerEdge | 单个 Edge 打开 10000 个 Stream，frontier 不崩溃，资源可释放 |
| SEC-BOUND-003 | TestEdgeIDOverflow | edgeID 为 0 / MaxUint64 等边界值，frontier 正确拒绝或处理 |

#### 并发竞态（RACE）

| 编号 | 测试名称 | 执行方式 | 验证点 |
|------|---------|---------|--------|
| SEC-RACE-001 | TestRaceEdgeConnectClose | `-race` | 并发 Connect 和 Close，无 data race |
| SEC-RACE-002 | TestRaceMultipleEdgeClose | `-race` | 同一 Edge 被多个 goroutine 并发 Close，无 panic / data race |
| SEC-RACE-003 | TestRaceServiceRegisterUnregister | `-race` | Service 并发注册/注销 RPC，无 data race |
| SEC-RACE-004 | TestRaceForwardAndClose | `-race` | Edge 正在转发 RPC 时同时 Close，无 panic |

#### 模糊测试（FUZZ，Go 1.18+）

| 编号 | 测试名称 | 验证点 |
|------|---------|--------|
| SEC-FUZZ-001 | FuzzEdgeMeta | 随机 meta 字节序列作为 Edge 接入 meta，frontier 不 panic |
| SEC-FUZZ-002 | FuzzRPCPayload | 随机 payload 通过 RPC 调用，frontier 不 panic |
| SEC-FUZZ-003 | FuzzMessagePayload | 随机 payload 通过 Publish 发送，frontier 不 panic |

### 6.3 安全测试执行方式

```bash
# 带竞态检测运行安全测试
go test -race -v ./test/security/

# 运行 fuzzing（至少跑 60 秒）
go test -fuzz=FuzzEdgeMeta    -fuzztime=60s ./test/security/
go test -fuzz=FuzzRPCPayload  -fuzztime=60s ./test/security/
go test -fuzz=FuzzMessagePayload -fuzztime=60s ./test/security/
```

---

## 七、测试覆盖矩阵

| 功能模块 | 单元测试 | 基准测试 | E2E测试 | 安全测试 |
|---------|:-------:|:-------:|:------:|:-------:|
| Edgebound（连接接入）| ✅ 已有 | ✅ 已有 | 🔲 待建 | 🔲 待建 |
| Servicebound（连接接入）| ✅ 已有 | ✅ 已有 | 🔲 待建 | 🔲 待建 |
| Exchange RPC 转发 | 🔲 待建 | ✅ 已有 | 🔲 待建 | 🔲 待建 |
| Exchange 消息转发 | 🔲 待建 | ✅ 已有 | 🔲 待建 | 🔲 待建 |
| Exchange Stream 透传 | 🔲 待建 | ✅ 已有 | 🔲 待建 | 🔲 待建 |
| Exchange 上下线通知 | 🔲 待建 | — | 🔲 待建 | — |
| Repo / DAO（buntdb）| ✅ 已有 | — | — | — |
| Repo / DAO（sqlite）| ✅ 已有 | ✅ 已有 | — | — |
| Config 加载 | ✅ 已有 | — | — | — |
| 竞态安全 | — | — | — | 🔲 待建 |
| 边界/模糊 | — | — | — | 🔲 待建 |

---

## 八、执行命令速查

```bash
# ── 单元测试 ──────────────────────────────────────────────
# 运行所有单元测试
go test ./pkg/frontier/...

# 带竞态检测
go test -race ./pkg/frontier/...

# 带覆盖率
go test -coverprofile=coverage.out ./pkg/frontier/...
go tool cover -html=coverage.out -o coverage.html

# ── 基准测试（go test bench 方式）───────────────
go test -bench=. -benchmem -v ./test/bench/...

# 大规模连接模拟
cd test/batch/edges && make && ./edges --count 10000 --nseconds 30

# ── E2E 测试（待创建）────────────────────────────────────
go test -v -timeout 5m ./test/e2e/
go test -race -v -timeout 5m ./test/e2e/

# ── 安全测试（待创建）────────────────────────────────────
go test -race -v ./test/security/
go test -fuzz=FuzzEdgeMeta -fuzztime=60s ./test/security/
```

---

## 附录

### 文档更新记录

| 日期 | 版本 | 修改内容 |
|-----|------|---------|
| 2026-04-01 | 1.0 | 初始版本 |
| 2026-04-01 | 1.1 | 去除 frontlas / Operator，聚焦 frontier 数据面；细化 E2E 和安全测试用例 |
