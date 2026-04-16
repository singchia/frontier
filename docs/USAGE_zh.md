## 使用

理解 Frontier 最快的方式，不是把它当成一个通用网关，而是把它看成 **service <-> edge 的双向连接层**。

建议按下面顺序阅读：

1. 先建立心智模型
2. 先跑最贴近你场景的示例
3. 再复制需要的 Service 侧或 Edge 侧 SDK 用法

### 心智模型

- **Service 侧**连接 `:30011`
- **Edge 侧**连接 `:30012`
- **Service -> Edge** 通常需要指定具体 `edgeID`
- **Edge -> Service** 通常按 Method、Topic 或 ServiceName 路由
- **Stream** 在逻辑上就是 service 和 edge 之间的直连 `net.Conn`

如果只记住一句话，那就是：

> Frontier 适合“后端服务需要主动触达在线边缘节点，同时边缘节点也需要主动触达后端服务”的系统。

### 示例

先从最贴近你目标的示例开始。

#### 聊天室：消息和在线态

目录 [examples/chatroom](../examples/chatroom) 下有一个简单聊天室示例，大约 100 行代码，最适合用来理解：

- service <-> edge 的消息交互
- 边缘节点上线/离线通知
- 长连接下的基本交互模型

构建示例程序：

```
make examples
```

在 `bin` 目录下得到 `chatroom_service` 和 `chatroom_agent` 可执行程序，运行示例：

https://github.com/singchia/frontier/assets/15531166/18b01d96-e30b-450f-9610-917d65259c30

#### 直播代理：点对点流

目录 [examples/rtmp](../examples/rtmp) 下有一个简单直播代理示例，大约 80 行代码，最适合用来理解：

- service 到 edge 的点对点建流
- Frontier 不只是做 RPC，也可以做流承载
- RTMP 这类协议的中继或代理方式

运行后，使用 [OBS](https://obsproject.com/) 连接 `rtmp_edge` 即可做直播代理：

<img src="./diagram/rtmp.png" width="100%">

#### 应该先跑哪个示例？

- 如果你关心命令下发、通知、设备/Agent 消息交互，先看 **chatroom**
- 如果你关心文件传输、媒体中继或自定义协议代理，先看 **rtmp**
- 如果你已经明确要接 SDK，继续看下面的代码片段

### Service 侧常见模式

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

### Edge 侧常见模式

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

**GRPC**  详见[Protobuf定义](../api/controlplane/frontier/v1/controlplane.proto) 


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

**REST** Swagger详见[Swagger定义](../swagger/swagger.yaml)

例如你可以使用下面请求来踢除某个边缘节点下线：

```
curl -X DELETE http://127.0.0.1:30010/v1/edges/{edge_id} 
```
或查看某个微服务注册了哪些RPC：

```
curl -X GET http://127.0.0.1:30010/v1/services/rpcs?service_id={service_id}
```

**注意**：gRPC/Rest依赖dao backend，有两个选项```buntdb```和```sqlite```，都是使用的in-memory模式，为性能考虑，默认backend使用buntdb，并且列表接口返回字段count永远是-1，当你配置backend为sqlite3时，会认为你对在Frontier上连接的微服务和边缘节点有强烈的OLTP需求，例如在Frontier上封装web，此时count才会返回总数。
