<p align=center>
<img src="./docs/diagram/frontier-logo.png" width="30%" height="30%">
</p>

<div align="center">

[![Go](https://github.com/singchia/frontier/actions/workflows/go.yml/badge.svg)](https://github.com/singchia/frontier/actions/workflows/go.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/singchia/frontier)](https://goreportcard.com/report/github.com/singchia/frontier)
[![Go Reference](https://pkg.go.dev/badge/badge/github.com/singchia/frontier.svg)](https://pkg.go.dev/github.com/singchia/frontier/api/dataplane/v1/service)
[![License](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)

简体中文 | [English](./README_en.md)

</div>

Frontier是一个go开发的全双工开源长连接网关，旨在让微服务直达边缘节点或客户端，反之边缘节点或客户端也同样直达微服务。对于两者，提供了全双工的单双向RPC调用，消息发布和接收，以及点对点流的功能。Frontier符合云原生架构，可以使用Operator快速部署一个集群，具有高可用和弹性，轻松支撑百万边缘节点或客户端在线的需求。


## 特性

- **RPC**  微服务和边缘可以Call对方的函数（提前注册），并且在微服务侧支持负载均衡
- **消息**  微服务和边缘可以Publish对方的Topic，边缘可以Publish到外部MQ的Topic，微服务侧支持负载均衡
- **多路复用/流**  微服务可以直接在边缘节点打开一个流（连接），可以封装例如文件上传、代理等，天堑变通途
- **上线离线控制**  微服务可以注册边缘节点获取ID、上线离线回调，当这些事件发生，Frontier会调用这些函数
- **API简单**  在项目api目录下，分别对边缘和微服务提供了封装好的sdk，可以非常简单的基于这个sdk做开发
- **部署简单**  支持多种部署方式(docker docker-compose helm以及operator)来部署Frontier实例或集群
- **水平扩展**  提供了Frontiter和Frontlas集群，在单实例性能达到瓶颈下，可以水平扩展Frontier实例或集群
- **高可用**  支持集群部署，支持微服务和边缘节点永久重连sdk，在当前实例宕机情况时，切换新可用实例继续服务
- **支持控制面**  提供了gRPC和rest接口，允许运维人员对微服务和边缘节点查询或删除，删除即踢除目标下线

## 架构

### 组件Frontier

<img src="./docs/diagram/frontier.png" width="100%" height="100%">


- _Service End_：微服务侧的功能入口，默认连接
- _Edge End_：边缘节点或客户端侧的功能入口
- _Publish/Receive_：发布和接收消息
- _Call/Register_：调用和注册函数
- _OpenStream/AcceptStream_：打开和接收点到点流（连接）
-  _外部MQ_：Frontier支持将从边缘节点Publish的消息根据配置的Topic转发到外部MQ

Frontier需要微服务和边缘节点两方都主动连接到Frontier，Service和Edge的元信息（接收Topic，RPC，Service名等）可以在连接的时候携带过来。连接的默认端口是：

- ```:30011``` 提供给微服务连接，获取Service
- ```:30012``` 提供给边缘节点连接，获取Edge
- ```:30010``` 提供给运维人员或者程序使用的控制面


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

1. 所有的消息、RPC和Stream都是点到点的传递
	- 从微服务到边缘，一定要指定边缘节点ID
	- 从边缘到微服务，Frontier根据Topic和Method路由，最终哈希选择一个微服务或外部MQ，默认根据```edgeid```哈希，你也可以选择```random```或```srcip```
2. 消息需要接收方明确结束
	- 为了保障消息的传达语义，接收方一定需要msg.Done()或msg.Error(err)，保障传达一致性
3. Multiplexer打开的流在逻辑上是微服务与边缘节点的直接通信
	- 对方接收到流后，所有在这个流上功能都会直达对方，不会经过Frontierd的路由策略


## 使用

### 示例

目录[examples/chatroom](./examples/chatroom)下有简单的聊天室示例，仅100行代码实现一个的聊天室功能，可以通过

```
make examples
```

在bin目录下得到```chatroom_service```和```chatroom_egent```可执行程序，运行示例：

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
	svc.RegisterGetEdgeID(context.TODO(), getID)
	svc.RegisterEdgeOnline(context.TODO(), online)
	svc.RegisterEdgeOffline(context.TODO(), offline)
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
	msg := svc.NewMessage([]byte("test"))
	// 发布一条消息到ID为1001的边缘节点
	err := svc.Publish(context.TODO(), 1001, msg)
	// ...
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
		// 接收消息
		msg, err := svc.Receive(context.TODO())
		if err == io.EOF {
			// 收到EOF表示svc生命周期已终结，不可以再使用
			return
		}
		if err != nil {
			fmt.Println("receive err:", err)
			continue
		}
		// 处理完msg后，需要通知调用方已完成
		msg.Done()
	}
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
	svc, _ := service.NewService(dialer)
	req := svc.NewRequest([]byte("test"))
	// 调用ID为1001边缘节点的foo方法，前提是边缘节点需要预注册该方法
	rsp, err := svc.Call(context.TODO(), 1001, "foo", req)
	// ...
}
```

**微服务注册方法以供边缘节点调用**：

```golang
package main

import (
	"context"
	"net"
	"github.com/singchia/frontier/api/dataplane/v1/service"
	"github.com/singchia/geminio"
)

func main() {
	dialer := func() (net.Conn, error) {
		return net.Dial("tcp", "127.0.0.1:30011")
	}
	svc, _ := service.NewService(dialer)
	// 注册一个"echo"方法
	svc.Register(context.TODO(), "echo", echo)
	// ...
}

func echo(ctx context.Context, req geminio.Request, rsp geminio.Response) {
	value := req.Data()
	rsp.SetData(value)
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
	st, err := svc.OpenStream(context.TODO(), 1001)
}
```
基于这个新打开流，你可以用来传递文件、代理流量等。


**微服务接收流**

```golang
package main

import (
	"fmt"
	"io"
	"net"
	"github.com/singchia/frontier/api/dataplane/v1/service"
)

func main() {
	dialer := func() (net.Conn, error) {
		return net.Dial("tcp", "127.0.0.1:30011")
	}
	// 在获取svc时声明需要微服务名，在边缘打开流时需要指定该微服务名
	svc, _ := service.NewService(dialer, service.OptionServiceName("service-name"))
	for {
		st, err := svc.AcceptStream()
		if err == io.EOF {
			// 收到EOF表示svc生命周期已终结，不可以再使用
			return
		} else if err != nil {
			fmt.Println("accept stream err:", err)
			continue
		}
		// 使用stream，这个stream同时是个net.Conn，你可以Read/Write/Close，同时也可以RPC和消息
	}
}
```
基于这个新打开流，你可以用来传递文件、代理流量等。

**消息、RPC和流一起来吧！**

```golang
package main

import (
	"context"
	"fmt"
	"io"
	"net"
	"github.com/singchia/frontier/api/dataplane/v1/service"
	"github.com/singchia/geminio"
)

func main() {
	dialer := func() (net.Conn, error) {
		return net.Dial("tcp", "127.0.0.1:30011")
	}
	// 在获取svc时声明需要微服务名，在边缘打开流时需要指定该微服务名
	svc, _ := service.NewService(dialer, service.OptionServiceName("service-name"))

	// 接收流
	go func() {
		for {
			st, err := svc.AcceptStream()
			if err == io.EOF {
				// 收到EOF表示svc生命周期已终结，不可以再使用
				return
			} else if err != nil {
				fmt.Println("accept stream err:", err)
				continue
			}
			// 使用stream，这个stream同时是个net.Conn，你可以Read/Write/Close，同时也可以RPC和消息
		}
	}()

	// 注册一个"echo"方法
	svc.Register(context.TODO(), "echo", echo)

	// 接收消息
	for {
		msg, err := svc.Receive(context.TODO())
		if err == io.EOF {
			// 收到EOF表示svc生命周期已终结，不可以再使用
			return
		}
		if err != nil {
			fmt.Println("receive err:", err)
			continue
		}
		// 处理完msg后，需要通知调用方已完成
		msg.Done()
	}
}

func echo(ctx context.Context, req geminio.Request, rsp geminio.Response) {
	value := req.Data()
	rsp.SetData(value)
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
	eg, _ := edge.NewEdge(dialer)
	// 开始使用eg ...
}
```

**边缘节点发布消息到Topic**：

Service需要提前声明接收该Topic，或者在配置文件中配置外部MQ。

```golang
package main

import (
	"context"
	"net"
	"github.com/singchia/frontier/api/dataplane/v1/edge"
)

func main() {
	dialer := func() (net.Conn, error) {
		return net.Dial("tcp", "127.0.0.1:30012")
	}
	eg, _ := edge.NewEdge(dialer)
	// 开始使用eg
	msg := eg.NewMessage([]byte("test"))
	err := eg.Publish(context.TODO(), "foo", msg)
	// ...
}

```

**边缘节点接收消息**：

```golang
package main

import (
	"context"
	"fmt"
	"io"
	"net"
	"github.com/singchia/frontier/api/dataplane/v1/edge"
)

func main() {
	dialer := func() (net.Conn, error) {
		return net.Dial("tcp", "127.0.0.1:30012")
	}
	eg, _ := edge.NewEdge(dialer)
	for {
		// 接收消息
		msg, err := eg.Receive(context.TODO())
		if err == io.EOF {
			// 收到EOF表示eg生命周期已终结，不可以再使用
			return
		}
		if err != nil {
			fmt.Println("receive err:", err)
			continue
		}
		// 处理完msg后，需要通知调用方已完成
		msg.Done()
	}
	// ...
}

```

**边缘节点调用微服务RPC**：

```golang
package main

import (
	"context"
	"net"
	"github.com/singchia/frontier/api/dataplane/v1/edge"
)

func main() {
	dialer := func() (net.Conn, error) {
		return net.Dial("tcp", "127.0.0.1:30012")
	}
	eg, _ := edge.NewEdge(dialer)
	// 开始使用eg
	req := eg.NewRequest([]byte("test"))
	// 调用echo方法，Frontier会查找并转发请求到相应的微服务
	rsp, err := eg.Call(context.TODO(), "echo", req)
}

```

**边缘节点注册RPC**：

```golang
package main

import (
	"context"
	"net"
	"github.com/singchia/frontier/api/dataplane/v1/edge"
	"github.com/singchia/geminio"
)

func main() {
	dialer := func() (net.Conn, error) {
		return net.Dial("tcp", "127.0.0.1:30012")
	}
	eg, _ := edge.NewEdge(dialer)
	// 注册一个"echo"方法
	eg.Register(context.TODO(), "echo", echo)
	// ...
}

func echo(ctx context.Context, req geminio.Request, rsp geminio.Response) {
	value := req.Data()
	rsp.SetData(value)
}
```

**边缘节点打开微服务的点到点流**：

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
	eg, _ := edge.NewEdge(dialer)
	st, err := eg.OpenStream("service-name")
	// ...
}
```
基于这个新打开流，你可以用来传递文件、代理流量等。

**边缘节点接收流**：

```golang
package main

import (
	"net"
	"fmt"
	"io"
	"github.com/singchia/frontier/api/dataplane/v1/edge"
)

func main() {
	dialer := func() (net.Conn, error) {
		return net.Dial("tcp", "127.0.0.1:30012")
	}
	eg, _ := edge.NewEdge(dialer)
	for {
		stream, err := eg.AcceptStream()
		if err == io.EOF {
			// 收到EOF表示eg生命周期已终结，不可以再使用
			return
		} else if err != nil {
			fmt.Println("accept stream err:", err)
			continue
		}
		// 使用stream，这个stream同时是个net.Conn，你可以Read/Write/Close，同时也可以RPC和消息
	}
}
```

### 错误处理

<table><thead>
  <tr>
    <th>错误</th>
    <th>描述和处理</th>
  </tr></thead>
<tbody>
  <tr>
    <td>io.EOF</td>
    <td>收到EOF表示流或连接已关闭，需要退出Receive、AcceptStream等操作</td>
  </tr>
  <tr>
    <td>io.ErrShortBuffer</td>
    <td>发送端或者接收端的Buffer已满，可以设置OptionServiceBufferSize或OptionEdgeBufferSize来调整</td>
  </tr>
  <tr>
    <td>apis.ErrEdgeNotOnline</td>
    <td>表示该边缘节点不在线，需要检查边缘连接</td>
  </tr>
  <tr>
    <td>apis.ServiceNotOnline</td>
    <td>表示微服务不在线，需要检查微服务连接信息或连接</td>
  </tr>
  <tr>
    <td>apis.RPCNotOnline</td>
    <td>表示Call的RPC不在线</td>
  </tr>
  <tr>
    <td>apis.TopicOnline</td>
    <td>表示Publish的Topic不在线</td>
  </tr>
  <tr>
    <td>其他错误</td>
    <td>还存在Timeout、BufferFull等</td>
  </tr>
</tbody>
</table>

需要注意的是，如果关闭流，在流上正在阻塞的方法都会立即得到io.EOF，如果关闭入口（Service和Edge），则所有在此之上的流，阻塞的方法都会立即得到io.EOF。

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

例如你可以使用下面请求来踢除某个边缘节点下线：

```
curl -X DELETE http://127.0.0.1:30010/v1/edges/{edge_id} 
```
或查看某个微服务注册了哪些RPC：

```
curl -X GET http://127.0.0.1:30010/v1/services/rpcs?service_id={service_id}
```

**注意**：gRPC/Rest依赖dao backend，有两个选项```buntdb```和```sqlite```，都是使用的in-memory模式，为性能考虑，默认backend使用buntdb，并且列表接口返回字段count永远是-1，当你配置backend为sqlite3时，会认为你对在Frontier上连接的微服务和边缘节点有强烈的OLTP需求，例如在Frontier上封装web，此时count才会返回总数。


## Frontier配置

如果需要更近一步定制你的Frontier实例，可以在这一节了解各个配置是如何工作的。定制完你的配置，保存为```frontier.yaml```，挂载到容器```/usr/conf/frontier.yaml```位置生效。

### 最小化配置

简单起，你可以仅配置面向微服务和边缘节点的服务监听地址：

```yaml
# 微服务端配置
servicebound:
  # 监听网络
  listen:
    network: tcp
    # 监听地址
    addr: 0.0.0.0:30011
# 边缘节点端配置
edgebound:
  # 监听网络
  listen:
    network: tcp
    # 监听地址
    addr: 0.0.0.0:30012
  # 找不到注册的GetEdgeID时，是否允许Frontier分配edgeID
  edgeid_alloc_when_no_idservice_on: true
```

### TLS

对于用户来说，比较重要的TLS配置在微服务、边缘节点和控制面都是支持的，另支持mTLS，Frontier由此校验客户端携带的证书。

```yaml
servicebound:
  listen:
    addr: 0.0.0.0:30011
    network: tcp
    tls:
      # 是否开启TLS，默认不开启
      enable: false
      # 证书和私钥，允许配置多对证书，由客户端协商确定
      certs:
      - cert: servicebound.cert
        key: servicebound.key
      # 是否启用mtls，启动会校验客户端携带的证书是否由下面的CA签发
      mtls: false
      # CA证书，用于校验客户端证书
      ca_certs:
      - ca1.cert
edgebound:
  listen:
    addr: 0.0.0.0:30012
    network: tcp
    tls:
      # 是否开启TLS，默认不开启
      enable: false
      # 证书和私钥，允许配置多对证书，由客户端协商确定
      certs:
      - cert: edgebound.cert
        key: edgebound.key
      insecure_skip_verify: false
      # 是否启用mtls，启动会校验客户端携带的证书是否由下面的CA签发
      mtls: false
      # CA证书，用于校验客户端证书
      ca_certs:
      - ca1.cert
```

### 外部MQ

如果你需要配置外部MQ，Frontier也支持将相应的Topic转Publish到这些MQ。

**AMQP**

```yaml
mqm:
  amqp:
    # 是否允许
    enable: false
    # AMQP地址
    addrs: null
    # 生产者
    producer:
       # exchange名
      exchange: ""
      # 等于Frontier内Topic的概念，数组值
      routing_keys: null
```
对于AMQP来说，以上是最小配置，边缘节点Publish的消息Topic如果在routing_keys内，Frontier会Publish到exchange中，如果还有微服务或其他外部MQ也声明了该Topic，Frontier仍然会按照hashby来选择一个Publish。

**Kafka**

```yaml
mqm:
  kafka:
    # 是否允许
    enable: false
    # kafka地址
    addrs: null
    # 生产者
    producer:
       # 数组值
      topics: null
```
对于Kafka来说，以上是最小配置，边缘节点Publish的消息Topic如果在上面数组中，Frontier会Publish过来。如果还有微服务或其他外部MQ也声明了该Topic，Frontier仍然会按照hashby来选择一个Publish。

**NATS**

```yaml
mqm:
  nats:
    # 是否允许
    enable: false
    # NATS地址
    addrs: null
    producer:
      # 等于Frontier内Topic的概念，数组值
      subjects: null
    # 如果允许jetstream，会优先Publish到jetstream
    jetstream:
      enable: false
      # jetstream名
      name: ""
      producer:
        # 等于Frontier内Topic的概念，数组值
        subjects: null
```
NATS配置里，如果允许Jetstream，会优先使用Publish到Jetstream。如果还有微服务或其他外部MQ也声明了该Topic，Frontier仍然会按照hashby来选择一个Publish。

**NSQ**

```yaml
mqm:
  nsq:
    # 是否允许
    enable: false
    # NSQ地址
    addrs: null
    producer:
      # 数组值
      topics: null
```
NSQ的Topic里，如果还有微服务或其他外部MQ也声明了该Topic，Frontier仍然会按照hashby来选择一个Publish。

**Redis**

```yaml
mqm:
  redis:
    # 是否允许
    enable: false
    # Redis地址
    addrs: null
    # Redis DB
    db: 0
    # 密码
    password: ""
    producer:
      # 等于Frontier内Topic的概念，数组值
      channels: null
```
如果还有微服务或其他外部MQ也声明了该Topic，Frontier仍然会按照hashby来选择一个Publish。


### 其他配置

```yaml
daemon:
  # 是否开启PProf
  pprof:
    addr: 0.0.0.0:6060
    cpu_profile_rate: 0
    enable: true
  # 资源限制
  rlimit:
    enable: true
    nofile: 102400
  # 控制面开启
controlplane:
  enable: false
  listen:
    network: tcp
    addr: 0.0.0.0:30010
dao:
  # 支持buntdb和sqlite3，都使用的in-memory模式，保持无状态
  backend: buntdb
  # sqlite debug开启
  debug: false
exchange:
  # Frontier根据edgeid srcip或random的哈希策略转发边缘节点的消息、RPC和打开流到微服务，默认edgeid
  # 即相同的边缘节点总是会请求到相同的微服务。
  hashby: edgeid
```

更多详细配置见 [frontier_all.yaml](./etc/frontier_all.yaml)

## Frontier部署

在单Frontier实例下，可以根据环境选择以下方式部署你的Frontier实例。

### docker

```
docker run -d --name frontier -p 30011:30011 -p 30012:30012 singchia/frontier:1.1.0
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

### operator

见下面集群部署章节

## 集群

### Frontier + Frontlas架构

<img src="./docs/diagram/frontlas.png" width="100%" height="100%">

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


## k8s

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


## 许可证

Released under the [Apache License 2.0](https://github.com/singchia/geminio/blob/main/LICENSE)

---

已经看到这里，点个Star⭐️是莫大的认可♥️