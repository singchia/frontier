# RFC-001: 云原生部署优化专项

> 完整规范见 `~/.claude/skills/gospec/spec/01-requirement/technical-rfc.md`。
>
> 本 RFC 聚焦 Frontier / Frontlas 在 Kubernetes 云原生场景下的部署形态、CRD/Operator 健壮性、可观测性与交付链路。属纯技术重构，用户侧使用方式不变。

## 元信息

- 状态：草稿
- 作者：singchia
- 日期：2026-05-01
- 关联 ADR：待 M2 启动时拆分（CRD v1alpha2 引入策略、TLS 证书生命周期、可观测性指标命名）
- 关联 issue：—

## 背景

Frontier 现已交付 Helm chart、Dockerfile 和基于 kubebuilder 的 Operator（FrontierCluster CRD），但在云原生生产场景下存在以下痛点：

### 1. 数据正确性 bug（P0）

代码审计发现 4 处真 bug：

- `pkg/operator/internal/controller/frontiercluster_deployment.go:114` — TLS Cert/Key volume 错挂成 CA Secret，证书目录里塞的是 CA。
- `pkg/operator/internal/controller/frontiercluster_tls.go:99` — `getEBCAFromSecret` 把 Secret 读取错误吞成空字符串 + nil err，CA 不存在的报错变成"成功但 CA 为空"。
- `pkg/operator/api/v1alpha1/frontiercluster_fields.go:117,124` — Operator 自管 Secret 名拼接 `fc.Name + "edgebound-..."`，缺连字符。
- `pkg/operator/api/v1alpha1/frontiercluster_fields.go:76-78` — Frontlas 的 `fpport`（frontier-plane 端口）硬编码 40012，CRD 里改不动。

### 2. 生产可用底座缺失

- Frontier deployment **没 liveness、没 readiness、没 preStop、没 terminationGracePeriodSeconds**。长连接型网关在滚动更新时秒断 edge。
- CRD 不暴露 `Resources / Tolerations / TopologySpread / ImagePullPolicy / ImagePullSecrets / ServiceAccountName / Annotations / PriorityClassName`，生产用户无法做基本的资源治理与调度配置。
- `Redis.Password` 是 spec 里的明文字段，违反 gospec 安全红线"密钥禁止进代码仓库"，且 `kubectl describe pod` 会打出环境变量明文。
- 容器以 root 运行，违反 gospec 安全红线。
- `ImagePullPolicy: PullAlways` 硬编码，启动慢且强依赖 registry。
- PodAntiAffinity 用了 `RequiredDuringSchedulingIgnoredDuringExecution` 硬反亲和，replicas 超 node 数即不可调度。

### 3. 可观测性不达标

gospec 红线："所有对外服务必须暴露 `/healthz`、`/readyz`、`/metrics`"。当前：

- Frontier 完全没有 health 端点；Frontlas 只有 readiness（`/cluster/v1/health`）。
- 配置结构体里有 `metrics` 字段（`pkg/frontier/config/config.go:247`）但**未实现** Prometheus exporter。
- MQ 投递、Frontlas RPC 延迟、cmux 路由分布、edge 在线数等关键运行指标全部缺失。
- 日志未结构化（用 klog），未贯通 trace_id。

### 4. 交付链路不规范

gospec 红线："所有构建 / 产物 / 部署目标必须由根 Makefile 作为唯一入口"。当前：

- Helm chart 只有 frontier 模板，**没有 frontlas 模板**，生产用户得自己拼。
- 镜像无签名、CI 无 govulncheck、无依赖漏洞扫描。
- Operator 不发 Kubernetes Event；Status 用 Phase（Running/Failed/Pending）而非现代的 `[]metav1.Condition`。

### 为什么现在做

1. ROADMAP.md 已把 "Helm / Operator / Atlas cluster" 全部勾选为完成，但实际不完整，文档与代码状态不一致，会误导生产用户。
2. 1.2.4 已发布、当前主分支处于功能稳定期，正适合做横切的非功能性优化，不与新功能 PR 抢 review 资源。
3. M1 的 4 个 bug 影响 mTLS 用户上线，**已经是阻塞问题**。

## 方案

分四个里程碑，每个里程碑独立 PR、独立可回滚。

### M1 — 数据正确性兜底（1-2 天）

纯 bug 修复，不改 API、不改 CRD schema：

| 位置 | 修复 |
|---|---|
| `frontiercluster_deployment.go:114` | Cert/Key volume 改用 `EBTLSOperatorCertKeyNamespacedName()` |
| `frontiercluster_tls.go:99` | `return "", err` 替代 `return "", nil` |
| `frontiercluster_fields.go:117,124` | 拼接修正为 `fc.Name + "-edgebound-..."`；保留旧名查找做软迁移 |
| `frontiercluster_fields.go:64-95` + `types.go:73-76` | `ControlPlane.FrontierPlanePort` 字段补到 CRD spec，与 ControlPlane.Port 解耦 |

### M2 — 生产可用底座（CRD v1alpha2，1-2 周）

引入 `v1alpha2`，老的 `v1alpha1` 保留 served=true 直到 M4 完成后下线。CRD 暴露：

```go
type PodOverrides struct {
    Resources           *corev1.ResourceRequirements
    Tolerations         []corev1.Toleration
    TopologySpread      []corev1.TopologySpreadConstraint
    NodeSelector        map[string]string
    Affinity            *corev1.Affinity   // 替代单独的 NodeAffinity
    PriorityClassName   string
    ServiceAccountName  string
    ImagePullSecrets    []corev1.LocalObjectReference
    ImagePullPolicy     corev1.PullPolicy  // 默认 IfNotPresent
    Annotations         map[string]string
    Labels              map[string]string
    SecurityContext     *corev1.PodSecurityContext  // 强制 nonRoot
}

type Redis struct {
    // ... 旧字段保留
    PasswordSecret *corev1.SecretKeySelector  // 新增；与 Password 互斥，优先用此
}
```

代码侧：

- Frontier 增加 `/healthz`、`/readyz` HTTP endpoint（监听独立调试端口或 cmux 复用 30010）。
- Frontier signal 处理：SIGTERM → 从 Frontlas 注销自己 → close edge listener → 等存量连接自然结束（`min(graceWindow, terminationGracePeriod-5s)`）→ exit。
- Operator 给 frontier deployment 加 `livenessProbe`（TCP 30010 或 gRPC health）+ `readinessProbe`（注册到 Frontlas 成功后 ready）+ `preStop`（sleep 让 endpoint 摘除）+ `terminationGracePeriodSeconds: 60` 默认值，可被 PodOverrides 覆盖。
- 容器 `securityContext.runAsNonRoot: true`，Dockerfile 改 distroless/nonroot UID。
- Default ImagePullPolicy 改 IfNotPresent。
- PodAntiAffinity 默认改 `PreferredDuringSchedulingIgnoredDuringExecution`，可被 Affinity 覆盖。

### M3 — 可观测性达标（1-2 周）

按 gospec `10-observability/` 规范：

- 实现 `pkg/frontier/config/config.go` 里 metrics 字段对应的 Prometheus exporter，导出指标：
  - `frontier_edge_connections_total{state="online|disconnected"}`
  - `frontier_servicebound_rpc_duration_seconds`（histogram）
  - `frontier_frontlas_rpc_duration_seconds`
  - `frontier_mq_publish_total{backend, topic, result}`
  - `frontier_cmux_route_total{route="grpc|tcp"}`
  - 严守 gospec 红线：高基数字段（edge_id、user_id、url）禁止做 label。
- 替换 klog → `log/slog`，输出 JSON，注入 `trace_id`/`edge_id`/`pod_name`。
- `/healthz`（liveness：进程存活）、`/readyz`（readiness：Frontlas 已注册 + 关键 listener up）、`/metrics` 三个端点统一暴露。
- Helm 加 `ServiceMonitor` 模板，开关在 values.yaml。
- 默认 PrometheusRule：edge 连接异常下跌、Frontlas RPC P99 延迟、MQ 投递失败率。每条告警带 Runbook 链接（M3 末把 Runbook 写到 `docs/runbooks/`）。

### M4 — 交付与发布规范（1 周）

按 gospec `08-delivery/` + `12-operations/` 规范：

- **Makefile 唯一入口**：`make build`、`make image`、`make helm-package`、`make e2e`、`make release`，CI 一律 `make <target>`，README 同步。
- Helm chart 补 `templates/frontlas/{deployment,service,configmap,servicemonitor}.yaml` + 可选 redis 子 chart 依赖（`bitnami/redis`）。
- CI 加 `govulncheck`、`trivy image`、`cosign sign`（Keyless OIDC）。
- Operator Status 从 `Phase` 改为 `[]metav1.Condition`（Available / Progressing / Degraded），保留 Phase 字段做软迁移直到 v1alpha1 下线。
- Operator 增加 `record.EventRecorder`，重要状态变更（TLS Secret 缺失、Frontlas not ready、副本不齐）发 K8s Event。
- 写 `docs/runbooks/frontier-rollback.md` + `docs/runbooks/redis-loss-recovery.md`。

## 备选方案

| 方案 | 优点 | 缺点 | 是否采用 |
|---|---|---|---|
| **A. 分四个里程碑、各自 PR、CRD 走 v1alpha2** | 风险隔离、可分阶段回滚、bug 修复可立即放出 | 总跨度长，需要 v1alpha1/v1alpha2 共存期 | ✅ |
| B. 一个大 PR 全量重构 | 一次到位 | 评审困难、回滚成本高、阻塞用户上线 | ❌ |
| C. 仅修 M1 bug，其它写 ROADMAP 留坑 | 改动最小 | gospec 红线长期违反、生产用户继续踩坑 | ❌ |
| D. 直接 v1，不走 v1alpha2 | 一步到位 | 字段还在演进、过早承诺兼容会形成包袱 | ❌ |
| E. v1alpha1 加 deprecated 字段就地扩展，不发新版本 | 不引入版本管理负担 | 字段语义混乱、后续退不出去 | ❌ |

## 影响范围

- **代码范围**：
  - `pkg/operator/api/v1alpha1/`（M1 修复 + M2 起共存）
  - `pkg/operator/api/v1alpha2/`（M2 新增）
  - `pkg/operator/internal/controller/`（M1 + M2 + M4）
  - `pkg/frontier/`（M2 优雅停机、M3 metrics/log）
  - `pkg/frontlas/`（M3 metrics/log 对齐）
  - `dist/helm/`（M3 ServiceMonitor、M4 frontlas 模板）
  - `images/Dockerfile.*`（M2 distroless / nonroot）
  - `Makefile`（M4 重构入口）
  - `.github/workflows/`（M4 govulncheck / trivy / cosign）

- **运行时影响**：
  - M1 修复后 mTLS 用户证书目录内容会变（之前是错的），需要在 CHANGELOG 提示用户重启 frontier pod。
  - M2 新增 preStop + terminationGracePeriod 会让滚动更新慢约 60 秒，但避免长连接秒断。
  - M2 切非 root 用户，使用 hostPath 或要求 root 的部署需要在 values.yaml 主动覆盖 SecurityContext。
  - M3 暴露 `/metrics` 默认开启会增加少量 CPU/内存（可关）。

- **迁移成本**：
  - 已部署 v1alpha1 用户：M2 发布后无强制迁移，v1alpha1 与 v1alpha2 共存至少一个 minor 版本。
  - Helm 用户：M4 后建议切换到 chart 内置的 frontlas 模板。

- **回滚预案**：
  - M1：单个 PR revert 即可，无 schema 变更。
  - M2：Operator 镜像降级；CRD 不需要回滚，v1alpha2 字段全部 optional 且向后兼容。
  - M3：metrics/healthz endpoint 通过 config 关闭即可不影响业务。
  - M4：Status conditions 与 Phase 同时写，回滚 controller 镜像即恢复 Phase-only 行为。

## 风险

| 风险 | 影响 | 概率 | 缓解措施 |
|---|---|---|---|
| M1 修复 TLS volume 后老用户重启即生效，mTLS 证书目录变化可能导致连接中断 | 中 | 中 | CHANGELOG 显式说明；建议用户低峰期滚动重启 |
| v1alpha2 conversion webhook 实施复杂 | 中 | 中 | M2 不引入 webhook，依赖字段全部 optional 走默认值；conversion 推迟到 v1beta1 |
| 优雅停机 + 从 Frontlas 注销若 Frontlas 不可达可能阻塞 SIGTERM | 中 | 低 | 注销加 5s 超时，超时后强制关闭 listener |
| Prometheus 指标基数过高拖垮 prom server | 高 | 低 | gospec 红线已禁止高基数 label；M3 加单元测试断言 label set |
| Distroless 镜像缺 shell 排查困难 | 低 | 中 | 提供 `:debug` tag 方案 + 文档化 ephemeral container 排查 |
| Makefile 重构破坏现有 CI | 中 | 低 | M4 先双轨运行（旧 target 加 deprecation 提示）一个版本周期再下线 |

## 排期

- 预计开始：2026-05-01
- 预计完成：2026-06-19（7 周）

里程碑节点：

| 里程碑 | 起止 | 产出 |
|---|---|---|
| M1 | 2026-05-01 → 2026-05-04 | 1 PR，4 个 bug 修复 + 单元测试 |
| M2 | 2026-05-05 → 2026-05-22 | 1 PR，CRD v1alpha2 + 优雅停机 + 探针 + 非 root |
| M3 | 2026-05-25 → 2026-06-12 | 1 PR，metrics + slog + ServiceMonitor + Runbook 初稿 |
| M4 | 2026-06-15 → 2026-06-19 | 1 PR，Makefile + frontlas chart + 镜像签名 + Conditions |

## 验收标准

### M1
- [ ] 4 个 bug 全部有对应单元测试
- [ ] `make test` 全绿
- [ ] mTLS e2e 场景验证证书目录内容正确

### M2
- [ ] `kubectl apply` v1alpha2 sample 可成功 reconcile
- [ ] frontier pod 滚动更新期间 edge 长连接不被秒断（手工验证）
- [ ] 容器内 `id` 命令显示非 root
- [ ] Redis 密码可走 SecretKeySelector，`kubectl describe pod` 不打印明文

### M3
- [ ] `curl :30010/metrics` 返回 Prometheus 格式
- [ ] `curl :30010/healthz` 进程存活时返回 200
- [ ] `curl :30010/readyz` 在未注册 Frontlas 时返回 503
- [ ] 所有 metrics label 经单元测试断言无高基数
- [ ] 日志为 JSON，含 `trace_id`、`pod_name`、`level`

### M4
- [ ] `make build` / `make image` / `make helm-package` / `make e2e` 全部可用
- [ ] CI 包含 `govulncheck`、`trivy`、`cosign sign`
- [ ] Helm chart `helm install frontier dist/helm` 可一键拉起 frontier + frontlas + redis
- [ ] `kubectl describe fc` 显示 Conditions
- [ ] `kubectl get events` 能看到 Operator 发的事件

## 实施任务

颗粒度按 1-3 天拆，每个里程碑落到具体 PR：

### M1（任务 #3）
1. 修 deployment.go:114 TLS volume 错挂 + 单元测试
2. 修 tls.go:99 吞错误 + 单元测试
3. 修 fields.go:117/124 命名 + 兼容旧名查找
4. CRD types 加 `Frontlas.ControlPlane.FrontierPlanePort`，fields.go 用之

### M2（任务 #4）
拆为：v1alpha2 类型定义 / Reconciler 切版本路由 / Frontier 优雅停机 / 容器非 root / Redis SecretRef / preStop+probes

### M3（任务 #5）
拆为：metrics 包搭建 / frontier 接入指标 / frontlas 接入指标 / log 切 slog / health endpoints / ServiceMonitor + 默认 PrometheusRule + Runbook

### M4（任务 #6）
拆为：Makefile 改造 / Helm frontlas 模板 / CI govulncheck+trivy / cosign 签名 / Status Conditions / EventRecorder

## 变更记录

| 日期 | 变更人 | 变更内容 | 原因 |
|---|---|---|---|
| 2026-05-01 | singchia | 初稿 | 立项 |
