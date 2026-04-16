## Usage

Frontier is easiest to understand if you think about it as **service <-> edge connectivity**, not as a generic gateway.

Use this guide in the following order:

1. Understand the mental model
2. Run the example closest to your use case
3. Copy the SDK pattern you need on the service side or edge side

### Mental Model

- **Service side** connects to `:30011`
- **Edge side** connects to `:30012`
- **Service -> Edge** operations usually target a specific `edgeID`
- **Edge -> Service** operations route by declared method, topic, or service name
- **Streams** behave like direct `net.Conn` links between service and edge

If you only remember one thing, remember this:

> Frontier is for systems where backend services need to actively reach online edge nodes, and edge nodes also need to actively reach backend services.

### Examples

Start with the example that matches the job you want Frontier to do.

#### Chatroom: messaging and presence

In [examples/chatroom](../examples/chatroom), there is a simple chatroom example implemented in about 100 lines of code. It is the fastest way to understand:

- service <-> edge messaging
- edge online/offline notifications
- the basic long-lived connection model

Build the example binaries:

```
make examples
```

Run the demo:

https://github.com/singchia/frontier/assets/15531166/18b01d96-e30b-450f-9610-917d65259c30

#### RTMP: point-to-point streams

In [examples/rtmp](../examples/rtmp), there is a simple live streaming example implemented in about 80 lines of code. It is the fastest way to understand:

- service -> edge stream opening
- using Frontier as a stream transport rather than only RPC
- traffic relay for protocols such as RTMP

After running, use [OBS](https://obsproject.com/) to connect to `rtmp_edge` for live streaming proxy:

<img src="./diagram/rtmp.png" width="100%">

#### Which example should you start with?

- If you care about commands, notifications, or device/agent messaging, start with **chatroom**
- If you care about file transfer, media relay, or custom protocol tunneling, start with **rtmp**
- If you want production integration patterns, continue with the SDK snippets below

### Service-Side SDK Patterns

**Getting Service on the Microservice Side**:

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
	// Start using the service
}
```

**Receiving ID, Online/Offline Notifications on Microservice Side**:

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

// The service can assign IDs to edges based on metadata
func getID(meta []byte) (uint64, error) {
	return 0, nil
}

// Edge goes online
func online(edgeID uint64, meta []byte, addr net.Addr) error {
	return nil
}

// Edge goes offline
func offline(edgeID uint64, meta []byte, addr net.Addr) error {
	return nil
}
```

**Microservice Publishing Messages to Edge Nodes**:

The edge must be online beforehand, otherwise the edge cannot be found.
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
	// Publish a message to the edge node with ID 1001
	err := svc.Publish(context.TODO(), 1001, msg)
	// ...
}
```

**Microservice Declaring Topic to Receive**:

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
	// Declare the topic to receive when getting the service
	svc, _ := service.NewService(dialer, service.OptionServiceReceiveTopics([]string{"foo"}))
	for {
		// Receive messages
		msg, err := svc.Receive(context.TODO())
		if err == io.EOF {
			// Receiving EOF indicates the lifecycle of the service has ended and it can no longer be used
			return
		}
		if err != nil {
			fmt.Println("receive err:", err)
			continue
		}
		// After processing the message, notify the caller it is done
		msg.Done()
	}
}
```

**Microservice Calling Edge Node RPC**:

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
	// Call the "foo" method on the edge node with ID 1001. The edge node must have pre-registered this method.
	rsp, err := svc.Call(context.TODO(), 1001, "foo", req)
	// ...
}
```

**Microservice Registering Methods for Edge Nodes to Call**:

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
	// Register an "echo" method
	svc.Register(context.TODO(), "echo", echo)
	// ...
}

func echo(ctx context.Context, req geminio.Request, rsp geminio.Response) {
	value := req.Data()
	rsp.SetData(value)
}
```

**Microservice Opening Point-to-Point Stream on Edge Node**:

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
	// Open a new stream to the edge node with ID 1001 (st is also a net.Conn). The edge must accept the stream with AcceptStream.
	st, err := svc.OpenStream(context.TODO(), 1001)
}
```
Based on this newly opened stream, you can transfer files, proxy traffic, etc.

**Microservice Receives Stream**:

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
	// Declare the service name when getting the service, required when the edge opens a stream to specify the service name.
	svc, _ := service.NewService(dialer, service.OptionServiceName("service-name"))
	for {
		st, err := svc.AcceptStream()
		if err == io.EOF {
			// Receiving EOF indicates the lifecycle of the service has ended and it can no longer be used
			return
		} else if err != nil {
			fmt.Println("accept stream err:", err)
			continue
		}
		// Use the stream. This stream is also a net.Conn. You can Read/Write/Close, and also use RPC and messaging.
	}
}
```
Based on this newly opened stream, you can transfer files, proxy traffic, etc.

**Messages, RPC, and Streams Together!**:

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
	// Declare the service name when getting the service, required when the edge opens a stream to specify the service name.
	svc, _ := service.NewService(dialer, service.OptionServiceName("service-name"))

	// Receive streams
	go func() {
		for {
			st, err := svc.AcceptStream()
			if err == io.EOF {
				// Receiving EOF indicates the lifecycle of the service has ended and it can no longer be used
				return
			} else if err != nil {
				fmt.Println("accept stream err:", err)
				continue
			}
			// Use the stream. This stream is also a net.Conn. You can Read/Write/Close, and also use RPC and messaging.
		}
	}()

	// Register an "echo" method
	svc.Register(context.TODO(), "echo", echo)

	// Receive messages
	for {
		msg, err := svc.Receive(context.TODO())
		if err == io.EOF {
			// Receiving EOF indicates the lifecycle of the service has ended and it can no longer be used
			return
		}
		if err != nil {
			fmt.Println("receive err:", err)
			continue
		}
		// After processing the message, notify the caller it is done
		msg.Done()
	}
}

func echo(ctx context.Context, req geminio.Request, rsp geminio.Response) {
	value := req.Data()
	rsp.SetData(value)
}
```

### Edge-Side SDK Patterns

**Getting Edge on the Edge Node Side**:

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
	// Start using eg ...
}
```

**Edge Node Publishes Message to Topic**:

The service needs to declare receiving the topic in advance, or configure an external MQ in the configuration file.

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
	// Start using eg
	msg := eg.NewMessage([]byte("test"))
	err := eg.Publish(context.TODO(), "foo", msg)
	// ...
}
```

**Edge Node Receives Messages**:

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
		// Receive messages
		msg, err := eg.Receive(context.TODO())
		if err == io.EOF {
			// Receiving EOF indicates the lifecycle of eg has ended and it can no longer be used
			return
		}
		if err != nil {
			fmt.Println("receive err:", err)
			continue
		}
		// After processing the message, notify the caller it is done
		msg.Done()
	}
	// ...
}
```

**Edge Node Calls RPC on Microservice**:

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
	// Start using eg
	req := eg.NewRequest([]byte("test"))
	// Call the "echo" method. Frontier will look up and forward the request to the corresponding microservice.
	rsp, err := eg.Call(context.TODO(), "echo", req)
}
```

**Edge Node Registers RPC**:

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
	// Register an "echo" method
	eg.Register(context.TODO(), "echo", echo)
	// ...
}

func echo(ctx context.Context, req geminio.Request, rsp geminio.Response) {
	value := req.Data()
	rsp.SetData(value)
}
```

**Edge Node Opens Point-to-Point Stream to Microservice**:

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

Based on this newly opened stream, you can transfer files, proxy traffic, etc.

**Edge Node Receives Stream**:

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
			// Receiving EOF indicates the lifecycle of eg has ended and it can no longer be used
			return
		} else if err != nil {
			fmt.Println("accept stream err:", err)
			continue
		}
		// Use the stream. This stream is also a net.Conn. You can Read/Write/Close, and also use RPC and messaging.
	}
}
```

### Error Handling

<table><thead>
  <tr>
    <th>Error</th>
    <th>Description and Handling</th>
  </tr></thead>
<tbody>
  <tr>
    <td>io.EOF</td>
    <td>Receiving EOF indicates that the stream or connection has been closed, and you need to exit operations such as Receive and AcceptStream.</td>
  </tr>
  <tr>
    <td>io.ErrShortBuffer</td>
    <td>The buffer on the sender or receiver is full. You can adjust the buffer size by setting OptionServiceBufferSize or OptionEdgeBufferSize.</td>
  </tr>
  <tr>
    <td>apis.ErrEdgeNotOnline</td>
    <td>This indicates that the edge node is not online, and you need to check the edge connection.</td>
  </tr>
  <tr>
    <td>apis.ServiceNotOnline</td>
    <td>This indicates that the microservice is not online, and you need to check the microservice connection information or connection.</td>
  </tr>
  <tr>
    <td>apis.RPCNotOnline</td>
    <td>This indicates that the RPC called is not online.</td>
  </tr>
  <tr>
    <td>apis.TopicNotOnline</td>
    <td>This indicates that the topic to be published is not online.</td>
  </tr>
  <tr>
    <td>Other Errors</td>
    <td>There are also errors like Timeout, BufferFull, etc.</td>
  </tr>
</tbody>
</table>
It should be noted that if the stream is closed, any blocking methods on the stream will immediately receive io.EOF. If the entry point (Service and Edge) is closed, all streams on it will immediately receive io.EOF for blocking methods.

### Control Plane

The Frontier control plane provides gRPC and REST interfaces. Operators can use these APIs to determine the connection status of the current instance. Both gRPC and REST are served on the default port :`30010`.

**GRPC**  See[Protobuf Definition](../api/controlplane/frontier/v1/controlplane.proto) 

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

REST Swagger definition can be found at [Swagger Definition](../swagger/swagger.yaml)

For example, you can use the following request to kick an edge node offline:

```
curl -X DELETE http://127.0.0.1:30010/v1/edges/{edge_id} 
```

Or check which RPCs a microservice has registered:


```
curl -X GET http://127.0.0.1:30010/v1/services/rpcs?service_id={service_id}
```

Note: gRPC/REST depends on the DAO backend, with two options: ```buntdb``` and ```sqlite3```. Both use in-memory mode. For performance considerations, the default backend uses buntdb, and the count field in the list interface always returns -1. When you configure the backend to ```sqlite3```, it means you have a strong OLTP requirement for connected microservices and edge nodes on Frontier, such as encapsulating the web on Frontier. In this case, the count will return the total number.
