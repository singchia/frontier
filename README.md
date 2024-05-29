<p align=center>
<img src="./docs/diagram/frontier-logo.png" width="30%" height="30%">
</p>

Frontier是一个go开发的开源长连接网关，能让微服务直接连通边缘节点或客户端，反之边缘节点或客户端也能直接连通到微服务。对于微服务或者边缘节点，提供了单双向RPC调用，消息发布和接收，以及直接点对点拨通通信的特性。Frontier采用云原生架构，可以使用Operator快速部署一个集群，轻松实现百万连接。


## 特性

- **RPC**  微服务和边缘可以Call对方的函数（提前注册），并且在微服务侧支持负载均衡
- **消息**  微服务和边缘可以Publish对方的Topic，边缘可以Publish到外部MQ的Topic，微服务侧支持负载均衡
- **多路复用**  微服务可以直接在边缘节点打开一个新流（连接），可以封装例如文件上传、代理等，天堑变通途
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

Frontier需要微服务和边缘节点两方都主动连接到Frontier，这种设计的优势在不需要Frontier主动连接任何一个地址，Service和Edge的元信息可以在连接的时候携带过来。连接的默认端口是：

- ```30011```：提供给微服务连接，获取Service
- ```30012```：提供给边缘节点连接，获取Edge
- ```30010```：提供给运维人员或者程序使用的控制面

详情见部署章节


## 使用

### 示例

> examples/chatroom

该目录下有简单的聊天室示例，仅100行代码来实现一个聊天室功能，可以通过

```
make examples
```

在bin目录下得到```chatroom_service```和```chatroom_client```可执行程序，运行示例：

https://github.com/singchia/frontier/assets/15531166/18b01d96-e30b-450f-9610-917d65259c30

可以看到上线离线通知，消息Publish等功能

### Service

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

**Service接收获取ID、上线/离线通知**：

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

**Service发布消息到Edge**：

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

**Service调用Edge的RPC**：

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

**Service打开Edge的点到点流**：

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

**Service注册方法以供Edge调用**：

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

**Service声明接收Topic**：

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

### Edge

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

Frontier控制面提供gRPC和rest接口，运维人员可以使用这些api来确定本实例的连接情况，定义在 

> api/controlplane/frontier/v1/controlplane.proto

```protobuf
service ControlPlane {
    // 列举所有边缘节点
    rpc ListEdges(ListEdgesRequest) returns (ListEdgesResponse);
    // 获取边缘节点详情
    rpc GetEdge(GetEdgeRequest) returns (Edge);
    // 踢除某个边缘节点下线
    rpc KickEdge(KickEdgeRequest) returns (KickEdgeResponse);
    // 列举边缘节点注册的RPC
    rpc ListEdgeRPCs(ListEdgeRPCsRequest) returns (ListEdgeRPCsResponse);

    // 列举所有微服务
    rpc ListServices(ListServicesRequest) returns (ListServicesResponse);
    // 获取微服务详情
    rpc GetService(GetServiceRequest) returns (Service);
    // 提出某个微服务下线
    rpc KickService(KickServiceRequest) returns (KickServiceResponse);
    // 列举微服务注册的RPC
    rpc ListServiceRPCs(ListServiceRPCsRequest) returns (ListServiceRPCsResponse);
    // 列举微服务接收的Topic
    rpc ListServiceTopics(ListServiceTopicsRequest) returns (ListServiceTopicsResponse);
}
```

Swagger文档请见[swagger](./docs/swagger/swagger.yaml)

**注意**：

当你配置dao backend使用sqlite3时，count才会返回总数，默认使用buntdb，为性能考虑，这个值返回-1


## Frontier配置

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

然后你可以使用上面所说的iclm示例来测试功能性

### docker-compose

```
git clone https://github.com/singchia/frontier.git
cd dist/compose
docker-compose up -d frontier
```

### helm

```
git clone https://github.com/singchia/frontier.git
cd dist/helm
helm install frontier ./ -f values.yaml
```


## 集群

### Frontier + Frontlas架构

<img src="./docs/diagram/frontlas.png" width="100%" height="100%">

新增Frontlas组件用于构建集群，Frontlas同样也是无状态组件，并不在内存里留存其他信息，因此需要额外依赖Redis，你需要

- _Frontier_：微服务和边缘数据面通信组件
- _Frontlas_：集群管理组件，将微服务和边缘的元信息、活跃信息记录在Redis里

### 高可用

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