<p align=center>
<img src="./docs/diagram/frontier-logo.png" width="30%">
</p>

<div align="center">

[![Go](https://github.com/singchia/frontier/actions/workflows/go.yml/badge.svg)](https://github.com/singchia/frontier/actions/workflows/go.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/singchia/frontier)](https://goreportcard.com/report/github.com/singchia/frontier)
[![Go Reference](https://pkg.go.dev/badge/badge/github.com/singchia/frontier.svg)](https://pkg.go.dev/github.com/singchia/frontier/api/dataplane/v1/service)
[![License](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)

[English](./README.md) | 简体中文 | [日本語](./README_ja.md) | [한국어](./README_ko.md) | [Español](./README_es.md) | [Français](./README_fr.md) | [Deutsch](./README_de.md)

</div>

> 面向长连接场景的 service-to-edge 双向通信网关

Frontier 是一个使用 Go 编写的开源网关，专门用于 **service <-> edge** 通信。它让后端服务和边缘节点可以在长连接上直接双向交互，内置 **双向 RPC**、**消息收发** 和 **点对点流**。

它适用于两端都需要长期在线，并且需要主动互相调用、通知或建流的系统。Frontier **不是反向代理**，也 **不只是消息队列**。它更像是一层基础设施，让后端服务能够寻址并管理大规模在线边缘节点。

<p align="center">
  <img src="./docs/diagram/frontier.png" width="100%" alt="Frontier 架构总览">
</p>

## Frontier 的独特之处

- **不是 API Gateway 那一类产品**：Frontier 解决的是 backend-to-edge 通信，不只是南北向 HTTP 流量。
- **不是单纯 MQ**：它把双向 RPC、消息和流统一到同一套连接模型里。
- **不是普通隧道**：服务端可以直接寻址某个在线边缘节点，而不只是暴露一个端口。
- **天然面向在线节点集群**：适合设备、Agent、客户端、Connector 这种需要长期在线的场景。

## 你可以拿它做什么

<table>
  <tr>
    <td width="50%">
      <img src="./docs/diagram/rtmp.png" alt="Frontier 流量中继示例">
      <p><strong>流量中继和媒体传输</strong><br>通过点对点流承载 RTMP、中继代理、文件传输，或者任意自定义协议。</p>
    </td>
    <td width="50%">
      <img src="./docs/diagram/stream.png" alt="Frontier 流模型">
      <p><strong>远程 Agent 与设备集群</strong><br>让边缘节点保持在线，服务端可定向调用某个 edge，edge 也能反向调用后端服务。</p>
    </td>
  </tr>
</table>

## 一眼看懂

| 你的需求 | Frontier 提供的能力 |
| --- | --- |
| 后端要调用某个具体在线设备或 Agent | 基于长连接的 Service -> Edge RPC 与消息能力 |
| 边缘节点不能暴露入站端口，但要反向调用服务 | 同一套连接模型下的 Edge -> Service 调用 |
| 不只是 request/response，而是要传连续字节流 | Service 与 Edge 之间的点对点流 |
| 需要统一管理大规模在线节点 | 在线态、生命周期回调、控制面 API 与集群能力 |

## 目录

- [为什么是 Frontier](#为什么是-frontier)
- [Frontier 的独特之处](#frontier-的独特之处)
- [你可以拿它做什么](#你可以拿它做什么)
- [一眼看懂](#一眼看懂)
- [什么时候适合用 Frontier](#什么时候适合用-frontier)
- [真实场景](#真实场景)
- [对比](#对比)
- [快速开始](#快速开始)
- [文档](#文档)
- [社区](#社区)
- [许可证](#许可证)

## 为什么是 Frontier

大多数基础设施更偏向下面几种通信模型：

- **service -> service**，例如 HTTP 或 gRPC
- **client -> service**，例如 API Gateway 或反向代理
- **事件广播/分发**，例如消息队列

Frontier 面向的是另一类问题：

- **service <-> edge** 之间的长连接、双向、状态化通信
- 后端服务需要主动调用某个在线的边缘节点
- 边缘节点需要在不暴露入站端口的情况下主动调用后端服务
- RPC 不够时，还需要在服务和边缘之间打开直连流

<p align="center">
  <img src="./docs/diagram/frontlas.png" width="88%" alt="Frontier 集群与 Frontlas">
</p>

## 什么时候适合用 Frontier

如果你需要下面这些能力，Frontier 是合适的：

- 后端服务主动调用在线的设备、Agent、客户端或 Connector
- 边缘节点通过同一套连接模型主动调用后端服务
- 大规模长连接在线
- 用同一套数据面处理 RPC、消息和流
- 面向 service-to-edge 连接的集群部署和高可用

如果你只是下面这些需求，就不一定要用 Frontier：

- 只是做 service-to-service RPC，那么 gRPC 更简单
- 只是做 HTTP 入口、路由或代理，那么用 API Gateway 或 Envoy
- 只是做 pub/sub 或事件流，那么用 NATS 或 Kafka
- 只是做通用隧道，那么用 frp 或其他隧道工具

## 真实场景

- IoT 设备和终端集群
- 远程 Agent 和 Connector
- IM 和其他实时系统
- 游戏后端与在线客户端或边缘节点通信
- 基于 Connector 模式的零信任内网接入
- 通过点对点流做文件传输、媒体中继或流量代理

## 场景速览

| 场景 | 为什么适合 Frontier |
| --- | --- |
| 设备控制面 | 能直接寻址在线 edge，推指令、收状态，并维持长连接在线 |
| 远程 Connector 平台 | Connector 主动外连，不暴露入站端口，服务侧路由也更简单 |
| 实时业务系统 | 通知、RPC 和流都走同一条链路，适合长期在线连接 |
| 零信任内网接入 | 用 Agent 风格的 edge 做后端系统到私有资源的最后一跳桥接 |

## 对比

| 能力 | Frontier | gRPC | NATS | frp | Envoy |
| --- | --- | --- | --- | --- | --- |
| 以 service <-> edge 通信为核心模型 | 是 | 否 | 部分 | 否 | 否 |
| 后端可直接寻址某个在线边缘节点 | 是 | 否 | 部分 | 部分 | 否 |
| 边缘节点可主动调用后端服务 | 是 | 部分 | 是 | 否 | 否 |
| 支持 service 和 edge 之间的点对点流 | 是 | 部分 | 否 | 仅隧道 | 否 |
| 统一的 RPC + 消息 + 流模型 | 是 | 否 | 否 | 否 | 否 |
| 以大规模长连接在线为主要设计目标 | 是 | 否 | 部分 | 部分 | 否 |

这里的“部分”表示能力可以通过额外模式拼出来，但不是该系统的主通信模型。

## 快速开始

1. 启动单实例 Frontier：

```bash
docker run -d --name frontier -p 30011:30011 -p 30012:30012 singchia/frontier:1.2.2
```

2. 构建示例程序：

```bash
make examples
```

3. 运行 chatroom 示例：

```bash
# 终端 1
./bin/chatroom_service

# 终端 2
./bin/chatroom_agent
```

chatroom 示例展示的是 Frontier 最基础的交互模型：长连接在线、边缘上下线事件，以及 service <-> edge 的消息交互。

如果你想看点对点流的能力，也可以运行 RTMP 示例：

```bash
# 终端 1
./bin/rtmp_service

# 终端 2
./bin/rtmp_edge
```

演示视频：

https://github.com/singchia/frontier/assets/15531166/18b01d96-e30b-450f-9610-917d65259c30

## 文档

README 现在刻意只承担“快速看懂、快速转化”的职责。实现细节、配置面、部署方案和集群运维都放在文档里。

建议从这里开始：

- [使用指南](./docs/USAGE_zh.md)
- [配置指南](./docs/CONFIGURATION_zh.md)
- [技术文档索引](./docs/README.md)
- [Systemd 部署](./dist/systemd/README.md)
- [Docker Compose 部署](./dist/compose/README.md)
- [Helm 与 Operator 部署](./dist/helm/README.md)
- [路线图](./ROADMAP.md)

如果你在评估 Frontier 是否适合生产环境，文档里已经覆盖：

- 架构与通信模型
- RPC、消息和点对点流的语义
- Docker、Compose、Helm、Operator 的部署方式
- 基于 Frontlas 和 Redis 的集群模式
- 开发流程与贡献规范

## 社区

<p align=center>
<img src="./docs/diagram/wechat.JPG" width="30%">
</p>

欢迎加入微信交流群讨论和反馈。

## 许可证

基于 [Apache License 2.0](https://github.com/singchia/geminio/blob/main/LICENSE) 发布。

---
如果这个项目对你有帮助，欢迎点一个 Star ⭐
