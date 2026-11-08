# 动态 TLS 回调（实施决策）

复用 Geminio Service 回调，不新增 Cert HTTP 依赖。保留 file 默认模式，callback 模式仅用于 edgebound。

SDK 向每个连接独立注册，失败记录待重试；新增连接补注册。回调签名绑定随机请求 nonce、CSR、Frontier ID 和响应正文，Frontier 预置 Ed25519 提供者公钥；服务名仅用于路由。私钥在 Frontier 进程中生成，不传给提供者；重启重新生成 CSR。签发方必须另外核验 Frontier 的登记资格，CSR 不代表签发授权。

每 60 秒（±20%）刷新，失败 1 秒至 30 秒退避；配置含有效期限，断联不能无限延长信任。长期支持多个 CA，任一有效链均可通过，但这不替代 Edge 身份及业务准入回调。不在握手时发远程 RPC。禁用 TLS session tickets 防止恢复会话绕过更新后的信任配置。

首次无配置时监听端口可建立 TCP，但 TLS 握手拒绝；servicebound 不依赖动态证书。旧连接不因正常续期关闭。撤销旧连接仍由管理面 KickEdge 负责。callback 不可自动退回 file。

回滚：关闭 callback 配置，恢复现有文件配置并重启。不得回滚到过期或已撤销的信任材料。

## 配置示例

```yaml
edgebound:
  listen:
    network: tcp
    addr: ':30012'
    tls:
      enable: true
      mtls: true
      source: callback
      callback:
        service: cert
        provider_keys:
          - '<base64-encoded Ed25519 provider public key>'
        refresh_interval: 60s
        timeout: 5s
```

`source` 缺省或 `file` 保持现有行为。`callback` 与 `certs`、`ca_certs` 文件列表互斥，目前不支持 edgebound bypass。Servicebound 保留原有配置，不应依赖该回调，否则形成启动循环。Provider 公钥与 CA 是不同用途：前者验证配置来源，后者验证 TLS 对端。配置来源公钥可同时设置多个，通过部署渠道配置，不能接受未经认证的回调修改自身信任依据。

回调中返回 `TrustedCAs` 多个 PEM bundle，长期并存，Go x509 按任意一条可信链验证。它不限制为轮换临时使用。角色及节点身份仍由注册认证校验，不能把任意可信 CA 的任何证书视为同一个 Edge。

## Cert 侧 SDK

```go
svc, err := service.NewClusterService(frontlasAddress,
    service.OptionServiceName("cert"),
    service.OptionServiceRegistrationConfirmation(true),
)
// 检查 err；关闭时调用 svc.Close()。
err = service.RegisterGetTLSConfig(ctx, svc,
    func(ctx context.Context, req security.TLSConfigRequest) (security.SignedTLSConfig, error) {
        // 业务实现：验证登记资格、CSR 及所请求 Frontier 身份；不能仅凭 req.FrontierID 授权。
        // 查找/签发与 CSR 公钥匹配的服务端证书，读取完整信任集合。
        config, err := issuer.GetTLSConfig(ctx, req)
        if err != nil {
            return security.SignedTLSConfig{}, err
        }
        return security.SignTLSConfig(req, config, providerSigningKey)
    })
```

上述 `issuer` 是签发服务自己的实现，不是 Frontier 内置依赖。签发方应预登记 Frontier 公钥或提供独立部署身份验证；本补丁不实现 Cert 服务或自动授予签发资格。请求带 CSR 不等于已认证。

响应使用 `revision`、`certificate_chain`、`trusted_cas`、Unix 秒 `valid_until`；每次返回完整签名快照。签名绑定本次 nonce、CSR 和 Frontier ID，防止替换、跨实例使用和重放。无需把私钥放入请求/响应；短时间重复刷新应复用已签发证书，不要重复签发。同版本允许延长有效租约，但不允许变更证书或 CA 内容；内容变更必须升级版本。

有效租约最大 24 小时，实际也不超过服务端证书有效期。建议签发服务返回较短租约，按业务对离线可用性和撤销时延的取舍配置。刷新间隔默认 60 秒 ±20%，可设 1 秒至 5 分钟；失败退避 1 秒至 30 秒并加抖动。提供者注册或重连会触发刷新，配置接近失效时提前刷新。源认证仅保护返回的公开证书材料，不取代整个 Servicebound 网络的安全隔离。

## 注册可靠性和兼容性

集群 SDK 有界并发注册（最多 8 个任务），失败不停止其他 Frontier，按连接和方法记录待重试项。新连接发布与 Register 使用同一登记锁协调，避免建连快照和新注册并发时漏注册；不持锁做网络 IO。连接移除后清理待重试项。重复注册同一方法以最后提交的回调为准，正在执行的旧版本结束后补注册最新版本。

启用 `OptionServiceRegistrationConfirmation(true)` 后，注册还会调用本地 `frontier_registration_status_v1`，确认当前 Service 的仓库记录可见。仓库失败返回错误，集群 SDK 重试；仅返回 Geminio ACK 不代表仓库写入成功。此选项需要新版 Frontier，默认关闭以兼容旧服务端。旧 Geminio 报文和 Delegate 接口保持不变。单连接 `NewService` 的注册错误仍由调用者处理；自动失败补偿属于 `NewClusterService`。

客户端主动 Register 在并发替换时只保证接纳目标回调并尝试同步，后台继续收敛；不能把并发调用的一次 nil 返回解释为所有实例已永久收敛。没有在线 Frontier 时，本地保存回调，后续发现后补注册。

## 验证

根目录执行 `make test-dynamic-tls frontier frontlas`。覆盖多 CA 的真实 TLS 握手、移除 CA、无效签名/重放/回退版本、断联缓存、过期拒绝、刷新退出、SDK 部分失败和新实例补注册，以及真实 Geminio 回调和注册确认。

当前仍保留原有控制面职责：强制撤销已有连接使用 KickEdge，证书续期不主动中断旧连接；未新增 Cert 业务服务，也未替换现有部署证书。
