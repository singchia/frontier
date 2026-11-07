# 测试服务器重新部署

部署来源：`test/geminio-flow-reset` 的 `32b36b7`，Geminio `v1.3.0-test.20261106.1`，通过根 Makefile 编译 Linux AMD64 Frontier/Frontlas。

运行镜像 `trustio-server:frontlas-32b36b7` 基于服务器现有运行镜像，仅替换 Frontier/Frontlas 二进制。更新 Frontlas 和两个 Frontier，保留其他服务镜像、凭据、证书和数据。服务器留存原 Compose 与旧镜像，并提供回滚 Makefile 目标。

二进制 SHA-256：

- Frontier：`728874dc578612a28c8abd79c23ce369c440a7c9a84de3ff9b1a0c8ac16aa035`
- Frontlas：`b3759f5e3d5131732d683eba1f7bd61820307bb175a1220e76a00a69ac492ba0`

部署后验收遇到并处理两个问题：

1. 管理网关已启用 HTTPS，但 Resource 下发的管理 origin 仍为 HTTP。保留原主机/端口，修正 scheme 并重启 Resource；没有手工修改测试 Edge 身份文件。
2. 旧 Entry 进程在 Frontier 重建后没有恢复 `get_edge_id` RPC 注册，新 Edge 被拒绝为 `service not online`。重启两个 Entry 后恢复。此次仅处理部署衔接，没有声称修复 Entry 自动恢复 RPC 注册的代码缺陷。

验收使用原生下载安装的临时 Edge 和真实 TCP/UDP echo 服务，通过服务器上的 MASQUE 探针验证两个 Entry。不是 macOS 桌面 TUN、公网端到端或性能基准测试。测试使用全新实体 ID，结束后恢复原空应用/节点目录并停止临时进程。

结果：两个入口 TCP/UDP 与未授权端口拒绝均通过。Frontlas 停机 35 秒后，Service 映射由 4 变为 0，测试 Edge 的元数据/alive 两个 key 由 2 变为 0。启动后按 2 秒间隔轮询，第 8 次观测到全部 Service 映射恢复（约 14 秒），第 11 次 Edge 两个 key 恢复（约 20 秒）。此阶段没有重启 Entry、Frontier 或 Edge。恢复后的双入口探测全部通过。

结束时 14 个常驻容器运行，所有配置了健康检查的服务 healthy。远端运行二进制校验值与本地构建一致。原始输出见 [验收日志](remote-redeployment-20261106.txt)。
