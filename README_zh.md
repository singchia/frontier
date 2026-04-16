<p align=center>
<img src="./docs/diagram/frontier-logo.png" width="30%">
</p>

<div align="center">

[![Go](https://github.com/singchia/frontier/actions/workflows/go.yml/badge.svg)](https://github.com/singchia/frontier/actions/workflows/go.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/singchia/frontier)](https://goreportcard.com/report/github.com/singchia/frontier)
[![Go Reference](https://pkg.go.dev/badge/badge/github.com/singchia/frontier.svg)](https://pkg.go.dev/github.com/singchia/frontier/api/dataplane/v1/service)
[![License](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)

[English](./README.md) | 简体中文

</div>

# Frontier

> 面向长连接场景的 service-to-edge 双向通信网关

Frontier 是一个使用 Go 编写的开源网关，专门用于 **service <-> edge** 通信。它让后端服务和边缘节点可以在长连接上直接双向交互，内置 **双向 RPC**、**消息收发** 和 **点对点流**。

它适用于两端都需要长期在线，并且需要主动互相调用、通知或建流的系统。Frontier **不是反向代理**，也 **不只是消息队列**。它更像是一层基础设施，让后端服务能够寻址并管理大规模在线边缘节点。

## 目录

- [为什么是 Frontier](#为什么是-frontier)
- [什么时候适合用 Frontier](#什么时候适合用-frontier)
- [真实场景](#真实场景)
- [对比](#对比)
- [快速开始](#快速开始)
- [特性](#特性)
- [架构](#架构)
- [使用](#使用)
- [配置](#配置)
- [部署](#部署)
- [集群](#集群)
- [Kubernetes](#kubernetes)
- [开发](#开发)
- [测试](#测试)
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

演示视频: 

https://github.com/singchia/frontier/assets/15531166/18b01d96-e30b-450f-9610-917d65259c30

## 特性

- **双向 RPC**：服务可以调用边缘，边缘也可以调用服务
- **消息**：服务和边缘之间可以收发消息，边缘发布的 Topic 也可以转发到外部 MQ
- **点对点流**：可直接打开流做代理、文件传输、媒体中继或自定义流量承载
- **在线态与生命周期**：支持边缘 ID 分配以及上下线回调
- **控制面 API**：通过 gRPC 和 REST 查询、管理在线节点
- **集群与高可用**：结合 Frontlas 做扩展、重连和高可用部署
- **云原生部署**：支持 Docker、Compose、Helm 和 Operator

## 架构

Frontier 位于后端服务和边缘节点之间。两端都通过出站长连接接入 Frontier，然后由 Frontier 提供统一的 RPC、消息和流模型。

<img src="./docs/diagram/frontier.png" width="100%">


- _Service End_：后端服务入口
- _Edge End_：边缘节点或客户端入口
- _Publish/Receive_：消息发布与接收
- _Call/Register_：RPC 调用与方法注册
- _OpenStream/AcceptStream_：点对点流建立
- _外部MQ_：可选地将边缘发布的消息转发到外部 MQ

默认端口如下：

- `:30011` 提供给后端服务连接，获取 Service
- `:30012` 提供给边缘节点连接，获取 Edge
- `:30010` 提供给运维人员或程序使用的控制面


### 功能

<table><thead>
  <tr>
    <th>功能</th>
    <th>发起方</th>
    <th>接收方</th>
    <th>方法</th>
    <th>路由方式</th>
    <th>描述</th>
  </tr></thead>
<tbody>
  <tr>
    <td rowspan="2">Messager</td>
    <td>Service</td>
    <td>Edge</td>
    <td>Publish</td>
    <td>EdgeID+Topic</td>
    <td>必须Publish到具体的EdgeID，默认Topic为空，Edge调用Receive接收，接收处理完成后必须调用msg.Done()或msg.Error(err)保障消息一致性</td>
  </tr>
  <tr>
    <td>Edge</td>
    <td>Service或外部MQ</td>
    <td>Publish</td>
    <td>Topic</td>
    <td>必须Publish到Topic，由Frontier根据Topic选择某个Service或MQ</td>
  </tr>
  <tr>
    <td rowspan="2">RPCer</td>
    <td>Service</td>
    <td>Edge</td>
    <td>Call</td>
    <td>EdgeID+Method</td>
    <td>必须Call到具体的EdgeID，需要携带Method</td>
  </tr>
  <tr>
    <td>Edge</td>
    <td>Service</td>
    <td>Call</td>
    <td>Method</td>
    <td>必须Call到Method，由Frontier根据Method选择某个的Service</td>
  </tr>
  <tr>
    <td rowspan="2">Multiplexer</td>
    <td>Service</td>
    <td>Edge</td>
    <td>OpenStream</td>
    <td>EdgeID</td>
    <td>必须OpenStream到具体的EdgeID</td>
  </tr>
  <tr>
    <td>Edge</td>
    <td>Service</td>
    <td>OpenStream</td>
    <td>ServiceName</td>
    <td>必须OpenStream到ServiceName，该ServiceName由Service初始化时携带的service.OptionServiceName指定</td>
  </tr>
</tbody></table>

主要遵守以下设计原则：

1. 所有消息、RPC 和 Stream 都是点对点传递
   - 从服务到边缘，必须指定边缘节点 ID
   - 从边缘到服务，Frontier 会按 Topic 或 Method 路由，再通过哈希选择某个服务或外部 MQ；默认哈希键为 `edgeid`，也支持 `random` 和 `srcip`
2. 消息需要由接收方显式结束
   - 接收方需要调用 `msg.Done()` 或 `msg.Error(err)`，以保证消息传达语义
3. Multiplexer 打开的流在逻辑上是服务和边缘之间的直连通信
   - 一旦对端接受该流，这条流上的数据将绕过 Frontier 的上层路由策略，直接到达对方


## 使用

详细使用文档: [docs/USAGE_zh.md](./docs/USAGE_zh.md)

## 配置

详细配置文档: [docs/CONFIGURATION_zh.md](./docs/CONFIGURATION_zh.md)

## 部署

在单Frontier实例下，可以根据环境选择以下方式部署你的Frontier实例。

### docker

```
docker run -d --name frontier -p 30011:30011 -p 30012:30012 singchia/frontier:1.2.2
```


### docker-compose

```
git clone https://github.com/singchia/frontier.git
cd dist/compose
docker-compose up -d frontier
```

### helm

如果你是在k8s环境下，可以使用helm快速部署一个实例

```
git clone https://github.com/singchia/frontier.git
cd dist/helm
helm install frontier ./ -f values.yaml
```

你的微服务应该连接`service/frontier-servicebound-svc:30011`，你的边缘节点可以连接`:30012`所在的NodePort。

### Systemd

使用独立的 Systemd 文档：

[dist/systemd/README_cn.md](./dist/systemd/README_cn.md)

### operator

见下面集群部署章节

## 集群

### Frontier + Frontlas架构

<img src="./docs/diagram/frontlas.png" width="100%">

新增Frontlas组件用于构建集群，Frontlas同样也是无状态组件，并不在内存里留存其他信息，因此需要额外依赖Redis，你需要提供一个Redis连接信息给到Frontlas，支持 ```redis``` ```sentinel```和```redis-cluster```。

- _Frontier_：微服务和边缘数据面通信组件
- _Frontlas_：命名取自Frontier Atlas，集群管理组件，将微服务和边缘的元信息、活跃信息记录在Redis里

Frontier需要主动连接Frontlas以上报自己、微服务和边缘的活跃和状态，默认Frontlas的端口是：

- ```:40011``` 提供给微服务连接，代替微服务在单Frontier实例下连接的30011端口
- ```:40012``` 提供给Frontier连接，上报状态

你可以根据需要部署任意多个Frontier实例，而对于Frontlas，分开部署两个即可保障HA（高可用），因为不存储状态没有一致性问题。

### 配置

**Frontier**的frontier.yaml需要添加如下配置：

```yaml
frontlas:
  enable: true
  dial:
    network: tcp
    addr:
      - 127.0.0.1:40012
    tls:
  metrics:
    enable: false
    interval: 0
daemon:
  # Frontier集群内的唯一ID
  frontier_id: frontier01
```
Frontier需要连接Frontlas，用来上报自己、微服务和边缘的活跃和状态。


**Frontlas**的frontlas.yaml最小化配置：

```yaml
control_plane:
  listen:
    # 微服务改连接这个地址，用来发现集群的边缘节点所在的Frontier
    network: tcp
    addr: 0.0.0.0:40011
frontier_plane:
  # Frontier连接这个地址
  listen:
    network: tcp
    addr: 0.0.0.0:40012
  expiration:
    # 微服务在redis内元信息的过期时间
    service_meta: 30
    # 边缘节点在redis内元信息的过期时间
    edge_meta: 30
redis:
  # 支持连接standalone、sentinel和cluster
  mode: standalone
  standalone:
    network: tcp
    addr: redis:6379
    db: 0
```

更多详细配置见 [frontlas_all.yaml](./etc/frontlas_all.yaml)

### 使用

由于使用Frontlas来发现可用的Frontier，因此微服务需要做出调整如下：

**微服务获取Service**

```golang
package main

import (
	"net"
	"github.com/singchia/frontier/api/dataplane/v1/service"
)

func main() {
	// 改使用NewClusterService来获取Service
	svc, err := service.NewClusterService("127.0.0.1:40011")
	// 开始使用service，其他一切保持不变
}
```

**边缘节点获取连接地址**

对于边缘节点来说，依然连接Frontier，不过可以从Frontlas来获取可用的Frontier地址，Frontlas提供了列举Frontier实例接口：

```
curl -X http://127.0.0.1:40011/cluster/v1/frontiers
```
你可以在这个接口上封装一下，提供给边缘节点做负载均衡或者高可用，或加上mTLS直接提供给边缘节点（不建议）。

**控制面GRPC** 详见[Protobuf定义](./api/controlplane/frontlas/v1/cluster.proto) 

Frontlas控制面与Frontier不同，是面向集群的控制面，目前只提供了读取集群的接口

```protobuf
service ClusterService {
    rpc GetFrontierByEdge(GetFrontierByEdgeIDRequest) returns (GetFrontierByEdgeIDResponse);
    rpc ListFrontiers(ListFrontiersRequest) returns (ListFrontiersResponse);

    rpc ListEdges(ListEdgesRequest) returns (ListEdgesResponse);
    rpc GetEdgeByID(GetEdgeByIDRequest) returns (GetEdgeByIDResponse);
    rpc GetEdgesCount(GetEdgesCountRequest) returns (GetEdgesCountResponse);

    rpc ListServices(ListServicesRequest) returns (ListServicesResponse) ;
    rpc GetServiceByID(GetServiceByIDRequest) returns (GetServiceByIDResponse) ;
    rpc GetServicesCount(GetServicesCountRequest) returns (GetServicesCountResponse) ;
}
```


## Kubernetes

### Operator

**安装CRD和Operator**

按照以下步骤安装和部署Operator到你的.kubeconfig环境中：

```
git clone https://github.com/singchia/frontier.git
cd dist/crd
kubectl apply -f install.yaml
```

查看CRD：

```
kubectl get crd frontierclusters.frontier.singchia.io
```

查看Operator：

```
kubectl get all -n frontier-system
```

**FrontierCluster集群**

```yaml
apiVersion: frontier.singchia.io/v1alpha1
kind: FrontierCluster
metadata:
  labels:
    app.kubernetes.io/name: frontiercluster
    app.kubernetes.io/managed-by: kustomize
  name: frontiercluster
spec:
  frontier:
    # 单实例Frontier
    replicas: 2
    # 微服务侧端口
    servicebound:
      port: 30011
    # 边缘节点侧端口
    edgebound:
      port: 30012
  frontlas:
    # 单实例Frontlas
    replicas: 1
    # 控制面端口
    controlplane:
      port: 40011
    redis:
      # 依赖的Redis配置
      addrs:
        - rfs-redisfailover:26379
      password: your-password
      masterName: mymaster
      redisType: sentinel
```

保存为`frontiercluster.yaml`，执行

```
kubectl apply -f frontiercluster.yaml
```

1分钟，你即可拥有一个2实例Frontier+1实例Frontlas的集群。

通过一下来检查资源部署情况 

> kubectl get all -l app=frontiercluster-frontier  
> kubectl get all -l app=frontiercluster-frontlas


```
NAME                                           READY   STATUS    RESTARTS   AGE
pod/frontiercluster-frontier-57d565c89-dn6n8   1/1     Running   0          7m22s
pod/frontiercluster-frontier-57d565c89-nmwmt   1/1     Running   0          7m22s
NAME                                       TYPE        CLUSTER-IP      EXTERNAL-IP   PORT(S)          AGE
service/frontiercluster-edgebound-svc      NodePort    10.233.23.174   <none>        30012:30012/TCP  8m7s
service/frontiercluster-servicebound-svc   ClusterIP   10.233.29.156   <none>        30011/TCP        8m7s
NAME                                       READY   UP-TO-DATE   AVAILABLE   AGE
deployment.apps/frontiercluster-frontier   2/2     2            2           7m22s
NAME                                                 DESIRED   CURRENT   READY   AGE
replicaset.apps/frontiercluster-frontier-57d565c89   2         2         2       7m22s
```

```
NAME                                            READY   STATUS    RESTARTS   AGE
pod/frontiercluster-frontlas-85c4fb6d9b-5clkh   1/1     Running   0          8m11s
NAME                                   TYPE        CLUSTER-IP    EXTERNAL-IP   PORT(S)               AGE
service/frontiercluster-frontlas-svc   ClusterIP   10.233.0.23   <none>        40011/TCP,40012/TCP   8m11s
NAME                                       READY   UP-TO-DATE   AVAILABLE   AGE
deployment.apps/frontiercluster-frontlas   1/1     1            1           8m11s
NAME                                                  DESIRED   CURRENT   READY   AGE
replicaset.apps/frontiercluster-frontlas-85c4fb6d9b   1         1         1       8m11s
```

你的微服务应该连接`service/frontiercluster-frontlas-svc:40011`，你的边缘节点可以连接`:30012`所在的NodePort。

## 开发

### 路线图
 
 详见 [ROADMAP](./ROADMAP.md)

### 贡献

如果你发现任何Bug，请提出Issue，项目Maintainers会及时响应相关问题。
 
 如果你希望能够提交Feature，更快速解决项目问题，满足以下简单条件下欢迎提交PR：
 
 * 代码风格保持一致
 * 每次提交一个Feature
 * 提交的代码都携带单元测试


## 测试

### 流功能测试

<img src="./docs/diagram/stream.png" width="100%">


## 社区

<p align=center>
<img src="./docs/diagram/wechat.JPG" width="30%">
</p>

添加以加入微信群组

## 许可证

Released under the [Apache License 2.0](https://github.com/singchia/geminio/blob/main/LICENSE)

---

已经看到这里，点个Star⭐️吧♥️
