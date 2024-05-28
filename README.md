<p align=center>
<img src="./docs/diagram/frontier-logo.png" width="30%" height="30%">
</p>

Frontier是一个go开发的开源长连接网关，能让微服务直接连通边缘节点或客户端，反之边缘节点或客户端也能直接连通到微服务。对于微服务或者边缘节点，提供了单双向RPC调用，消息发布和接收，以及直接点对点拨通通信的特性。Frontier采用云原生架构，可以使用Operator快速部署一个集群，轻松实现百万连接。

## 特性

- **RPC**（远程过程调用）；微服务可以直接Call到边缘节点的注册函数，同样的边缘节点也可以Call到微服务的注册函数，并且在微服务侧支持负载均衡。
- **消息**（发布和接收）；微服务可以直接Publish到边缘节点的topic，同样的边缘节点也可以Publish到微服务的topic，或外部消息队列的topic，微服务侧的topic，支持负载均衡。
- **多路复用**（点到点拨通）；微服务可以直接在边缘节点上打开一个新流（连接），你可以在这个新连接上封装例如流读写、拷贝文件、代理等，天堑变通途。
- **上线离线控制**（边缘节点）；微服务可以注册边缘节点获取ID、上线和下线回调，当边缘节点请求ID、上线或者离线时，Frontier会调用这个回调。
- **API简单**（SDK提供）；在项目的api目录下，分别对边缘和微服务提供了封装好的sdk，你可以非常简单和轻量的基于这个sdk做开发。
- **部署简单**；支持多种部署方式，如docker、docker-compose、k8s-helm以及operator来部署和管理你的Frontier实例。
- **水平扩展**；提供了Frontiter和Frontlas集群，在单实例性能达到瓶颈下，可以水平扩展Frontier实例。
- **高可用**；Frontlas具有集群视角，你可以使用微服务和边缘节点永久重连的sdk，在当前Frontier宕机情况下，新选择一个可用Frontier实例继续服务。

## 架构

### Frontier架构

<img src="./docs/diagram/frontier.png" width="100%" height="100%">


- Service End：微服务侧的功能入口
- Edge End：边缘节点或客户端侧的功能入口
- Publish/Receive：发布和接收消息
	- Topic：发布和接收消息的主题
- Call/Register：调用和注册函数
	- Method：调用和注册的函数名
- OpenStream/AcceptStream：打开和接收点到点连接

## 使用

### Service

**微服务侧获取Service**：

```golang
package main

import "github.com/singchia/frontier/api/dataplane/v1/service"

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

import "github.com/singchia/frontier/api/dataplane/v1/service"

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

func online(edgeID uint64, meta []byte, addr net.Addr) error {
	return nil
}

func offline(edgeID uint64, meta []byte, addr net.Addr) error {
	return nil
}
```

**Service发布消息到Edge**：

```golang
package main

import "github.com/singchia/frontier/api/dataplane/v1/service"

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

import "github.com/singchia/frontier/api/dataplane/v1/service"

func main() {
	dialer := func() (net.Conn, error) {
		return net.Dial("tcp", "127.0.0.1:30011")
	}
	svc, _ := service.NewService(dialer)
	req := srv.NewRequest([]byte("test"))
	// 调用ID为1001边缘节点的foo方法，前提是边缘节点需要预注册该方法
	rsp, err := srv.Call(context.TODO(), edgeID, "foo", req)
}
```

**Service打开Edge的点到点流**：

```golang
package main

import "github.com/singchia/frontier/api/dataplane/v1/service"

func main() {
	dialer := func() (net.Conn, error) {
		return net.Dial("tcp", "127.0.0.1:30011")
	}
	svc, _ := service.NewService(dialer)
	// 打开ID为1001边缘节点的新流（同时st也是一个net.Conn），前提是edge需要AcceptStream接收该流
	st, err := srv.OpenStream(context.TODO(), 1001)
}
```

**Service注册方法以供Edge调用**：

```golang
package main

import "github.com/singchia/frontier/api/dataplane/v1/service"

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

### Edge

**边缘节点侧获取Edge**：

```golang
package main

import "github.com/singchia/frontier/api/dataplane/v1/edge"

func main() {
	dialer := func() (net.Conn, error) {
		return net.Dial("tcp", "127.0.0.1:30012")
	}
	eg, err := edge.NewEdge(dialer)
	// 开始使用eg
}
```

## 部署

### 默认端口

- 30011：提供给微服务连接，获取Service的端口
- 30012：提供给边缘节点连接，获取Edge的端口
- 30010：提供给运维人员或者程序使用的控制面端口

### docker

```

```

### docker-compose

```

```

### helm

```

```

## 集群

### Frontier + Frontlas架构

<img src="./docs/diagram/frontlas.png" width="100%" height="100%">


## 参与开发

如果你发现任何Bug，请提出Issue，项目Maintainers会及时响应相关问题。
 
 如果你希望能够提交Feature，更快速解决项目问题，满足以下简单条件下欢迎提交PR：
 
 * 代码风格保持一致
 * 每次提交一个Feature
 * 提交的代码都携带单元测试

## 许可证

Released under the [Apache License 2.0](https://github.com/singchia/geminio/blob/main/LICENSE)