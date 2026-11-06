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
