<p align=center>
<img src="./docs/diagram/frontier-logo.png" width="30%" height="30%">
</p>

Frontier是一个go开发的开源长连接网关，旨在让微服务直达边缘节点或客户端，反之边缘节点或客户端也同样直达微服务。对于两者，提供了单双向RPC调用，消息发布和接收，以及点对点通信的功能。Frontier符合云原生架构，可以使用Operator快速部署一个集群，具有高可用和弹性，轻松支撑百万边缘节点或客户端在线的需求。


## 特性

- **RPC**  微服务和边缘可以Call对方的函数（提前注册），并且在微服务侧支持负载均衡
- **消息**  微服务和边缘可以Publish对方的Topic，边缘可以Publish到外部MQ的Topic，微服务侧支持负载均衡
- **多路复用/流**  微服务可以直接在边缘节点打开一个流（连接），可以封装例如文件上传、代理等，天堑变通途
- **上线离线控制**  微服务可以注册边缘节点获取ID、上线离线回调，当这些事件发生，Frontier会调用这些方法
- **API简单**  在项目api目录下，分别对边缘和微服务提供了封装好的sdk，可以非常简单的基于这个sdk做开发
- **部署简单**  支持多种部署方式(docker docker-compose helm以及operator)来部署Frontier实例或集群
- **水平扩展**  提供了Frontiter和Frontlas集群，在单实例性能达到瓶颈下，可以水平扩展Frontier实例或集群
- **高可用**  支持集群部署，支持微服务和边缘节点永久重连sdk，在当前实例宕机情况时，切换新可用实例继续服务
- **支持控制面**  提供了gRPC和rest接口，允许运维人员对微服务和边缘节点查询或删除，删除即踢除目标下线

## 架构

### Frontier架构

<img src="./docs/diagram/frontier.png" width="100%" height="100%">


- _Service End_：微服务侧的功能入口，默认连接
- _Edge End_：边缘节点或客户端侧的功能入口
- _Publish/Receive_：发布和接收消息
- _Call/Register_：调用和注册函数
- _OpenStream/AcceptStream_：打开和接收点到点流（连接）

Frontier需要微服务和边缘节点两方都主动连接到Frontier，Service和Edge的元信息（接收Topic，RPC，Service名等）可以在连接的时候携带过来。连接的默认端口是：

- ```:30011``` 提供给微服务连接，获取Service
- ```:30012``` 提供给边缘节点连接，获取Edge
- ```:30010``` 提供给运维人员或者程序使用的控制面


### 功能

<table><thead>
  <tr>
    <th>功能</th>
    <th>发起方</th>
    <th>目标方</th>
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
    <td>必须Publish到Topic，由Frontier根据Topic选择具体Service或MQ</td>
  </tr>
  <tr>
    <td rowspan="2">RPCer</td>
    <td>Service</td>
    <td>Edge</td>
    <td>Call</td>
    <td>EdgeID+Method</td>
    <td>必须Call到具体的EdgeID，必须携带Method</td>
  </tr>
  <tr>
    <td>Edge</td>
    <td>Service</td>
    <td>Call</td>
    <td>Method</td>
    <td>必须Call到Method，由Frontier根据Method选择具体的Service</td>
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

## 使用

### 示例

目录[examples/chatroom](./examples/chatroom)下有简单的聊天室示例，仅100行代码实现一个的聊天室功能，可以通过

```
make examples
```

在bin目录下得到```chatroom_service```和```chatroom_client```可执行程序，运行示例：

https://github.com/singchia/frontier/assets/15531166/18b01d96-e30b-450f-9610-917d65259c30

在这个示例你可以看到上线离线通知，消息Publish等功能。


### 微服务如何使用

**微服务侧获取Service**：

```golang
package main

import (
	"net"
	"github.com/singchia/frontier/api/dataplane/v1/service"
)

func main() {
	dialer := func() (net.Conn, error) {
		return net.Dial("tcp", "127.0.0.1:30011")
	}
	svc, err := service.NewService(dialer)
	// 开始使用service
}
```

**微服务接收获取ID、上线/离线通知**：

```golang
package main

import (
	"context"
	"net"
	"github.com/singchia/frontier/api/dataplane/v1/service"
)

func main() {
	dialer := func() (net.Conn, error) {
		return net.Dial("tcp", "127.0.0.1:30011")
	}
	svc, _ := service.NewService(dialer)
	srv.RegisterGetEdgeID(context.TODO(), getID)
	srv.RegisterEdgeOnline(context.TODO(), online)
	srv.RegisterEdgeOffline(context.TODO(), offline)
}

// service可以根据meta分配id给edge
func getID(meta []byte) (uint64, error) {
	return 0, nil
}

// edge上线
func online(edgeID uint64, meta []byte, addr net.Addr) error {
	return nil
}

// edge离线
func offline(edgeID uint64, meta []byte, addr net.Addr) error {
	return nil
}
```

**微服务发布消息到边缘节点**：

前提需要该Edge在线，否则会找不到Edge

```golang
package main

import (
	"context"
	"net"
	"github.com/singchia/frontier/api/dataplane/v1/service"
)

func main() {
	dialer := func() (net.Conn, error) {
		return net.Dial("tcp", "127.0.0.1:30011")
	}
	svc, _ := service.NewService(dialer)
	msg := srv.NewMessage([]byte("test"))
	// 发布一条消息到ID为1001的边缘节点
	err = srv.Publish(context.TODO(), 1001, msg)
}
```

**微服务调用边缘节点的RPC**：

```golang
package main

import (
	"context"
	"net"
	"github.com/singchia/frontier/api/dataplane/v1/service"
)

func main() {
	dialer := func() (net.Conn, error) {
		return net.Dial("tcp", "127.0.0.1:30011")
	}
	req := srv.NewRequest([]byte("test"))
	// 调用ID为1001边缘节点的foo方法，前提是边缘节点需要预注册该方法
	rsp, err := srv.Call(context.TODO(), edgeID, "foo", req)
}
```

**微服务打开边缘节点的点到点流**：

```golang
package main

import (
	"context"
	"net"
	"github.com/singchia/frontier/api/dataplane/v1/service"
)

func main() {
	dialer := func() (net.Conn, error) {
		return net.Dial("tcp", "127.0.0.1:30011")
	}
	svc, _ := service.NewService(dialer)
	// 打开ID为1001边缘节点的新流（同时st也是一个net.Conn），前提是edge需要AcceptStream接收该流
	st, err := srv.OpenStream(context.TODO(), 1001)
}
```
基于这个新打开流，你可以用来传递文件、代理流量等。

**微服务注册方法以供边缘节点调用**：

```golang
package main

import (
	"context"
	"net"
	"github.com/singchia/frontier/api/dataplane/v1/service"
)

func main() {
	dialer := func() (net.Conn, error) {
		return net.Dial("tcp", "127.0.0.1:30011")
	}
	svc, _ := service.NewService(dialer)
	// 注册一个"echo"方法
	srv.Register(context.TODO(), "echo", echo)
}

func echo(ctx context.Context, req geminio.Request, rsp geminio.Response) {
	value := req.Data()
	rsp.SetData(value)
}
```

**微服务声明接收Topic**：

```golang
package main

import (
	"context"
	"fmt"
	"io"
	"net"
	"github.com/singchia/frontier/api/dataplane/v1/service"
)

func main() {
	dialer := func() (net.Conn, error) {
		return net.Dial("tcp", "127.0.0.1:30011")
	}
	// 在获取svc时声明需要接收的topic
	svc, _ := service.NewService(dialer, service.OptionServiceReceiveTopics([]string{"foo"}))
	for {
		msg, err := srv.Receive(context.TODO())
		if err == io.EOF {
			return
		}
		if err != nil {
			fmt.Println("\n> receive err:", err)
			continue
		}
		msg.Done()
		fmt.Printf("> receive msg, edgeID: %d streamID: %d data: %s\n", msg.ClientID(), msg.StreamID(), string(value))
	}
}
```

### 边缘节点/客户端如何使用

**边缘节点侧获取Edge**：

```golang
package main

import (
	"net"
	"github.com/singchia/frontier/api/dataplane/v1/edge"
)

func main() {
	dialer := func() (net.Conn, error) {
		return net.Dial("tcp", "127.0.0.1:30012")
	}
	eg, err := edge.NewEdge(dialer)
	// 开始使用eg
}
```

**边缘节点发布消息到Topic**：

Service需要提前声明接收该Topic，或者在配置文件中配置外部MQ。

```golang
package main

import (
	"net"
	"github.com/singchia/frontier/api/dataplane/v1/edge"
)

func main() {
	dialer := func() (net.Conn, error) {
		return net.Dial("tcp", "127.0.0.1:30012")
	}
	eg, err := edge.NewEdge(dialer)
	// 开始使用eg
	msg := cli.NewMessage([]byte("test"))
	err = cli.Publish(context.TODO(), "foo", msg)
}
```

**边缘节点调用微服务RPC**：

```golang
package main

import (
	"net"
	"github.com/singchia/frontier/api/dataplane/v1/edge"
)

func main() {
	dialer := func() (net.Conn, error) {
		return net.Dial("tcp", "127.0.0.1:30012")
	}
	eg, err := edge.NewEdge(dialer)
	// 开始使用eg
	msg := cli.NewMessage([]byte("test"))
	// 调用echo方法，Frontier会查找并转发请求到相应的微服务
	rsp, err := cli.Call(context.TODO(), "echo", req)
}
```

### 控制面

Frontier控制面提供gRPC和Rest接口，运维人员可以使用这些api来确定本实例的连接情况，gRPC和Rest都由默认端口```:30010```提供服务。

**GRPC**  详见[Protobuf定义](./api/controlplane/frontier/v1/controlplane.proto) 


```protobuf
service ControlPlane {
    rpc ListEdges(ListEdgesRequest) returns (ListEdgesResponse);
    rpc GetEdge(GetEdgeRequest) returns (Edge);
    rpc KickEdge(KickEdgeRequest) returns (KickEdgeResponse);
    rpc ListEdgeRPCs(ListEdgeRPCsRequest) returns (ListEdgeRPCsResponse);
    rpc ListServices(ListServicesRequest) returns (ListServicesResponse);
    rpc GetService(GetServiceRequest) returns (Service);
    rpc KickService(KickServiceRequest) returns (KickServiceResponse);
    rpc ListServiceRPCs(ListServiceRPCsRequest) returns (ListServiceRPCsResponse);
    rpc ListServiceTopics(ListServiceTopicsRequest) returns (ListServiceTopicsResponse);
}
```

**REST** Swagger详见[Swagger定义](./docs/swagger/swagger.yaml)

例如你可以使用下面请求来踢出某个边缘节点下线：

```
curl -X DELETE http://127.0.0.1:30010/v1/edges/{edge_id} 
```
或查看某个微服务注册了哪些RPC：

```
curl -X GET http://127.0.0.1:30010/v1/services/rpcs?service_id={service_id}
```

注意：gRPC/Rest依赖dao backend，有两个选项```buntdb```和```sqlite```，都是使用的in-memory模式，为性能考虑，默认backend使用buntdb，并且列表接口返回字段count永远是-1，当你配置backend为sqlite3时，会认为你对在Frontier上连接的微服务和边缘节点有强烈的OLTP需求，例如在Frontier上封装web，此时count才会返回总数。


## Frontier配置

如果需要更近一步定制你的Frontier实例，可以在这一节了解各个配置是如何工作的。

### TLS

```
tls:
  enable: false
  mtls: false
  ca_certs:
  - ca1.cert
  - ca2.cert
  certs:
  - cert: edgebound.cert
    key: edgebound.key
  insecure_skip_verify: false
```

## 部署

在单Frontier实例下，可以根据环境选择以下方式部署你的Frontier实例。

### docker

```
docker run -d --name frontier -p 30011:30011 -p 30012:30012 singchia/frontier:1.0.0
```


### docker-compose

```
git clone https://github.com/singchia/frontier.git
cd dist/compose
docker-compose up -d frontier
```

### helm

如果你是在k8s环境下，可以使用helm快速部署一个实例，默认

```
git clone https://github.com/singchia/frontier.git
cd dist/helm
helm install frontier ./ -f values.yaml
```


## 集群

### Frontier + Frontlas架构

<img src="./docs/diagram/frontlas.png" width="100%" height="100%">

新增Frontlas组件用于构建集群，Frontlas同样也是无状态组件，并不在内存里留存其他信息，因此需要额外依赖Redis，你需要提供一个Redis连接信息给到Frontlas，支持 ```redis``` ```sentinel```和```redis-cluster```。

- _Frontier_：微服务和边缘数据面通信组件
- _Frontlas_：集群管理组件，将微服务和边缘的元信息、活跃信息记录在Redis里

Frontier需要主动连接Frontlas以上报自己、微服务和边缘的活跃和状态，默认Frontlas的端口是：

- ```:40011``` 提供给微服务连接，代替微服务在单Frontier实例下连接的30011端口
- ```:40012``` 提供给Frontier连接，上报状态

### 使用

### 分布式

当部署多个Frontier实例时，跨实例的微服务和边缘节点寻址一定需要分布式存储，如果没有Frontlas，这部分的存储工作

### 高可用

### 水平扩展

## k8s

### Operator

## 开发

### 路线图
 
 详见 [ROADMAP](./ROADMAP.md)

### Bug和Feature

如果你发现任何Bug，请提出Issue，项目Maintainers会及时响应相关问题。
 
 如果你希望能够提交Feature，更快速解决项目问题，满足以下简单条件下欢迎提交PR：
 
 * 代码风格保持一致
 * 每次提交一个Feature
 * 提交的代码都携带单元测试

## 测试

### 流
<img src="./docs/diagram/stream.png" width="100%">


## 许可证

Released under the [Apache License 2.0](https://github.com/singchia/geminio/blob/main/LICENSE)