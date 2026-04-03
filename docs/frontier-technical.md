# Frontier 技术原理

## 目录

1. [客户端认证](#客户端认证)
2. [客户端上下线](#客户端上下线)
3. [Exchange 原理](#exchange-原理)

---

## 客户端认证

### 概述

Frontier 支持两种类型的客户端连接：
- **Service（服务端）**：后端服务，负责处理业务逻辑
- **Edge（边缘端）**：客户端，通常是终端设备或用户应用

### Service 客户端认证

Service 客户端在连接时通过 `Meta` 结构体传递认证信息：

```go
type Meta struct {
    Service string   `json:"service"`  // 服务名称
    Topics  []string `json:"topics"`   // 订阅的主题列表
}
```

**连接流程：**

1. Service 客户端建立 TCP 连接
2. 通过 Geminio 协议握手，在 `Meta` 中携带服务名称和订阅的 Topics
3. Frontier 解析 `Meta` 并分配 `ServiceID`
4. 注册服务到内存缓存和数据库
5. 注册服务声明的 Topics 和 RPCs

**关键代码位置：**
- `pkg/frontier/servicebound/service_manager.go:handleConn()` - 处理连接
- `pkg/frontier/servicebound/service_manager.go:online()` - 上线处理
- `pkg/frontier/servicebound/service_onoff.go:GetClientID()` - ID 分配

### Edge 客户端认证

Edge 客户端连接时通过 `Meta` 传递元数据（通常是用户标识等信息）：

**连接流程：**

1. Edge 客户端建立 TCP 连接
2. 通过 Geminio 协议握手，携带 `Meta` 信息
3. Frontier 通过 Exchange 向 Service 请求分配 `EdgeID`
   - 如果 Service 不在线，根据配置决定是否自动分配 ID
4. 注册 Edge 到内存缓存和数据库

**EdgeID 分配机制：**

```go
func (em *edgeManager) GetClientID(_ uint64, meta []byte) (uint64, error) {
    // 优先从 Exchange 获取 EdgeID（通过 Service）
    if em.exchange != nil {
        edgeID, err := em.exchange.GetEdgeID(meta)
        if err == nil {
            return edgeID, nil
        }
    }
    
    // 如果 Service 不在线，根据配置决定是否自动分配
    if em.conf.Edgebound.EdgeIDAllocWhenNoIDServiceOn {
        return em.idFactory.GetID(), nil
    }
    
    return 0, err
}
```

**关键代码位置：**
- `pkg/frontier/edgebound/edge_manager.go:handleConn()` - 处理连接
- `pkg/frontier/edgebound/edge_manager.go:online()` - 上线处理
- `pkg/frontier/edgebound/edge_onoff.go:GetClientID()` - ID 分配

### 认证特点

1. **基于 Geminio 协议**：使用 Geminio 作为底层通信协议
2. **Meta 信息传递**：通过连接时的 Meta 字段传递认证和配置信息
3. **ID 分配策略**：
   - Service：支持指定 ID 或自动分配
   - Edge：优先从 Service 获取，支持降级自动分配
4. **连接复用**：同一 ServiceID/EdgeID 的旧连接会被新连接踢下线

---

## 客户端上下线

### Service 上线流程

**时序图：**

```
Service Client          Servicebound Manager
     |                          |
     |---- TCP Connect -------->|
     |                          |
     |---- Geminio Handshake -->|
     |                          |
     |<--- Parse Meta ----------|
     |                          |
     |---- Allocate ServiceID ->|
     |                          |
     |---- Register Service -->|
     |                          |
     |---- Register Topics ----->|
     |                          |
     |---- Register RPCs ------>|
     |                          |
     |---- Add to MQM --------->|
     |                          |
     |---- ConnOnline Event --->|
     |                          |
     |---- Forward Setup ------->|
```

**详细步骤：**

1. **连接建立** (`handleConn`)
   - 接受 TCP 连接
   - 创建 Geminio End
   - 解析 Meta 信息

2. **注册 Topics** (`remoteReceiveClaim`)
   - 将服务声明的 Topics 注册到数据库
   - 创建 `ServiceTopic` 记录

3. **添加到 MQM** (Message Queue Manager)
   - 将 End 添加到消息队列管理器
   - 用于后续消息路由

4. **上线处理** (`online`)
   - 检查是否存在旧连接，如果存在则关闭
   - 添加到内存缓存 `services[serviceID] = end`
   - 创建数据库记录 `Service`
   - 触发 `ConnOnline` 事件

5. **设置转发** (`forward`)
   - 调用 Exchange 设置 Service -> Edge 的转发

**关键代码：**

```go
func (sm *serviceManager) handleConn(conn net.Conn) error {
    // 创建 Geminio End
    end, err := server.NewEndWithConn(conn, opt)
    
    // 解析 Meta
    meta := &apis.Meta{}
    json.Unmarshal(end.Meta(), meta)
    
    // 注册 Topics
    sm.remoteReceiveClaim(end.ClientID(), meta.Topics)
    
    // 添加到 MQM
    sm.mqm.AddMQByEnd(meta.Topics, end)
    
    // 上线处理
    sm.online(end, meta)
    
    // 设置转发
    sm.forward(meta, end)
}
```

### Service 下线流程

**触发时机：**
- 客户端主动关闭连接
- 网络断开
- 连接超时

**处理步骤：**

1. **ConnOffline 事件**
   - Geminio 协议层检测到连接断开
   - 触发 `ConnOffline` 回调

2. **清理缓存** (`offline`)
   - 从内存缓存 `services` 中删除
   - 验证地址匹配，避免误删

3. **清理数据库**
   - 删除 `Service` 记录
   - 删除 `ServiceRPCs` 记录
   - 删除 `ServiceTopics` 记录

4. **清理 MQM**
   - 从消息队列管理器中移除

5. **通知其他组件**
   - 触发 `ServiceOffline` 事件
   - 通知 Informer

**关键代码：**

```go
func (sm *serviceManager) ConnOffline(d delegate.ConnDescriber) error {
    serviceID := d.ClientID()
    addr := d.RemoteAddr()
    
    // 清理缓存和数据库
    err := sm.offline(serviceID, addr)
    
    // 通知其他组件
    if sm.informer != nil {
        sm.informer.ServiceOffline(serviceID, meta, addr)
    }
    
    return nil
}
```

### Edge 上线流程

**时序图：**

```
Edge Client          Edgebound Manager          Exchange          Service
     |                      |                      |                 |
     |---- TCP Connect ---->|                      |                 |
     |                      |                      |                 |
     |---- Geminio -------->|                      |                 |
     |   Handshake          |                      |                 |
     |                      |                      |                 |
     |<--- Get EdgeID ------|                      |                 |
     |                      |---- GetEdgeID ------->|                 |
     |                      |                      |---- RPC ------->|
     |                      |                      |<--- EdgeID -----|
     |                      |<--- EdgeID ----------|                 |
     |<--- EdgeID ----------|                      |                 |
     |                      |                      |                 |
     |                      |---- Register Edge --->|                 |
     |                      |---- ConnOnline ------>|                 |
     |                      |---- Forward Setup --->|                 |
```

**详细步骤：**

1. **连接建立** (`handleConn`)
   - 接受 TCP 连接
   - 创建 Geminio End

2. **EdgeID 分配** (`GetClientID`)
   - 优先通过 Exchange 向 Service 请求 EdgeID
   - 如果 Service 不在线，根据配置决定是否自动分配

3. **上线处理** (`online`)
   - 检查是否存在旧连接，如果存在则关闭
   - 添加到内存缓存 `edges[edgeID] = end`
   - 创建数据库记录 `Edge`
   - 触发 `ConnOnline` 事件
   - 通知 Exchange Edge 上线

4. **设置转发** (`forward`)
   - 调用 Exchange 设置 Edge -> Service 的转发

**关键代码：**

```go
func (em *edgeManager) handleConn(conn net.Conn) error {
    // 创建 Geminio End
    end, err := server.NewEndWithConn(conn, opt)
    
    // 上线处理（内部会分配 EdgeID）
    em.online(end)
    
    // 设置转发
    em.forward(end)
}
```

### Edge 下线流程

**处理步骤：**

1. **ConnOffline 事件**
   - Geminio 协议层检测到连接断开

2. **清理缓存** (`offline`)
   - 从内存缓存 `edges` 中删除
   - 验证地址匹配

3. **清理数据库**
   - 删除 `Edge` 记录
   - 删除 `EdgeRPCs` 记录

4. **通知其他组件**
   - 触发 `EdgeOffline` 事件
   - 通知 Exchange Edge 下线
   - 通知 Informer

**关键代码：**

```go
func (em *edgeManager) ConnOffline(d delegate.ConnDescriber) error {
    edgeID := d.ClientID()
    meta := d.Meta()
    addr := d.RemoteAddr()
    
    // 清理缓存和数据库
    err := em.offline(edgeID, meta, addr)
    
    return nil
}
```

### 上下线特性

1. **并发安全**
   - 使用 `sync.RWMutex` 保护内存缓存
   - 使用 `synchub.SyncHub` 同步旧连接清理

2. **幂等性**
   - 通过地址匹配避免误删
   - 支持同一 ID 的重复连接（会踢掉旧连接）

3. **数据一致性**
   - 内存缓存和数据库同步更新
   - 使用事务保证数据一致性（TODO）

4. **事件通知**
   - 通过 Informer 通知其他组件
   - 支持自定义事件处理

---

## Exchange 原理

### 概述

Exchange 是 Frontier 的核心组件，负责在 Service 和 Edge 之间转发消息、RPC 调用和 Stream。它实现了 Service 和 Edge 的解耦，使得两者可以独立扩展。

### 架构设计

```
                    Exchange
                       |
        +--------------+--------------+
        |                             |
   Edgebound                    Servicebound
        |                             |
    Edge Clients                Service Clients
```

**核心组件：**

- `Edgebound`: 管理所有 Edge 连接
- `Servicebound`: 管理所有 Service 连接
- `MQM`: 消息队列管理器，负责消息路由
- `Exchange`: 转发引擎

### Service -> Edge 转发

#### 消息转发 (Message Forwarding)

**流程：**

1. Service 发送消息，在 `Custom` 字段末尾携带目标 `EdgeID`（8字节）
2. Exchange 截取 `EdgeID` 并查找对应的 Edge
3. 如果 Edge 在线，将消息转发给 Edge
4. 如果 Edge 不在线，返回错误

**时序图：**

```
Service          Exchange         Edgebound        Edge
  |                 |                 |              |
  |--Publish------->|                 |              |
  |                 |                 |              |
  |                 |--GetEdgeByID--->|              |
  |                 |                 |              |
  |                 |<--Edge End------|              |
  |                 |                 |              |
  |                 |-----------------|--Publish---->|
  |                 |                 |              |
  |                 |<----------------|--Done--------|
  |                 |                 |              |
  |<--Done----------|                 |              |
  |                 |                 |              |
  
  说明: Publish Message 携带 Custom+EdgeID
        Exchange 提取 EdgeID 后查找 Edge
        如果 Edge 在线，转发消息并返回 Done
        如果 Edge 不在线，返回 Error(ErrEdgeNotOnline)
  
  或者（Edge不在线）:
  
Service          Exchange         Edgebound        Edge
  |                 |                 |              |
  |--Publish------->|                 |              |
  |                 |                 |              |
  |                 |--GetEdgeByID--->|              |
  |                 |                 |              |
  |                 |<--nil-----------|              |
  |                 |                 |              |
  |<--Error----------|                 |              |
```

**关键代码：**

```go
func (ex *exchange) forwardMessageToEdge(end geminio.End) {
    serviceID := end.ClientID()
    go func() {
        for {
            msg, err := end.Receive(context.TODO())
            if err != nil {
                return
            }
            
            // 从 Custom 末尾提取 EdgeID
            custom := msg.Custom()
            edgeID := binary.BigEndian.Uint64(custom[len(custom)-8:])
            msg.SetCustom(custom[:len(custom)-8])
            
            // 查找 Edge
            edge := ex.Edgebound.GetEdgeByID(edgeID)
            if edge == nil {
                msg.Error(apis.ErrEdgeNotOnline)
                return
            }
            
            // 转发消息
            mopt := options.NewMessage()
            mopt.SetCustom(msg.Custom())
            mopt.SetTopic(msg.Topic())
            newmsg := edge.NewMessage(msg.Data(), mopt)
            edge.Publish(context.TODO(), newmsg, popt)
            msg.Done()
        }
    }()
}
```

#### RPC 转发 (RPC Forwarding)

**流程：**

1. Service 发起 RPC 调用，在 `Custom` 字段末尾携带目标 `EdgeID`
2. Exchange 拦截 RPC（通过 Hijack）
3. 提取 `EdgeID` 并查找 Edge
4. 转发 RPC 调用到 Edge
5. 将响应返回给 Service，并在 `Custom` 中携带 `EdgeID`

**时序图：**

```
Service          Exchange         Edgebound        Edge
  |                 |                 |              |
  |--Call RPC------>|                 |              |
  |                 |                 |              |
  |                 |--GetEdgeByID--->|              |
  |                 |                 |              |
  |                 |<--Edge End------|              |
  |                 |                 |              |
  |                 |-----------------|--Call RPC-->|
  |                 |                 |              |
  |                 |<----------------|--Response----|
  |                 |                 |              |
  |<--Response-------|                 |              |
  |                 |                 |              |
  
  说明: Call RPC 携带 Custom+EdgeID
        Exchange Hijack 拦截并提取 EdgeID
        如果 Edge 在线，转发 RPC 并返回 Response(Data+Custom+EdgeID)
        如果 Edge 不在线，返回 Error(ErrEdgeNotOnline)
  
  或者（Edge不在线）:
  
Service          Exchange         Edgebound        Edge
  |                 |                 |              |
  |--Call RPC------>|                 |              |
  |                 |                 |              |
  |                 |--GetEdgeByID--->|              |
  |                 |                 |              |
  |                 |<--nil-----------|              |
  |                 |                 |              |
  |<--Error----------|                 |              |
```

**关键代码：**

```go
func (ex *exchange) forwardRPCToEdge(end geminio.End) {
    end.Hijack(func(ctx context.Context, method string, r1 geminio.Request, r2 geminio.Response) {
        serviceID := end.ClientID()
        
        // 提取 EdgeID
        custom := r1.Custom()
        edgeID := binary.BigEndian.Uint64(custom[len(custom)-8:])
        r1.SetCustom(custom[:len(custom)-8])
        
        // 查找 Edge
        edge := ex.Edgebound.GetEdgeByID(edgeID)
        if edge == nil {
            r2.SetError(apis.ErrEdgeNotOnline)
            return
        }
        
        // 转发 RPC
        r3 := edge.NewRequest(r1.Data(), ropt)
        r4, err := edge.Call(ctx, method, r3, copt)
        
        // 返回响应，携带 EdgeID
        tail := make([]byte, 8)
        binary.BigEndian.PutUint64(tail, edgeID)
        r2.SetCustom(append(r4.Custom(), tail...))
        r2.SetData(r4.Data())
    })
}
```

### Edge -> Service 转发

#### 消息转发 (Message Forwarding)

**流程：**

1. Edge 发送消息到指定 Topic
2. Exchange 接收消息
3. 通过 MQM 将消息投递到订阅该 Topic 的 Service
4. MQM 负责消息路由和负载均衡

**时序图：**

```
Edge         Exchange         MQM         Servicebound      Service
  |              |              |              |              |
  |--Publish---->|              |              |              |
  |              |              |              |              |
  |              |--Produce---->|              |              |
  |              |              |              |              |
  |              |              |--GetServices>|              |
  |              |              |              |              |
  |              |              |<--ServiceList|              |
  |              |              |              |              |
  |              |              |--Deliver---->|              |
  |              |              |              |              |
  |              |              |<--Done-------|              |
  |              |              |              |              |
  |              |<--Success----|              |              |
  |              |              |              |              |
  |<--Done-------|              |              |              |
  
  说明: Edge 发送消息到指定 Topic
        Exchange 通过 MQM 投递消息
        MQM 查找订阅 Topic 的 Services 并负载均衡投递
```

**关键代码：**

```go
func (ex *exchange) forwardMessageToService(end geminio.End) {
    edgeID := end.ClientID()
    go func() {
        for {
            msg, err := end.Receive(context.TODO())
            if err != nil {
                return
            }
            
            topic := msg.Topic()
            
            // 通过 MQM 投递消息
            err = ex.MQM.Produce(topic, msg.Data(),
                apis.WithOrigin(msg),
                apis.WithEdgeID(edgeID),
                apis.WithAddr(end.RemoteAddr()))
            
            if err != nil {
                msg.Error(err)
                continue
            }
            msg.Done()
        }
    }()
}
```

#### RPC 转发 (RPC Forwarding)

**流程：**

1. Edge 发起 RPC 调用，指定 RPC 方法名
2. Exchange 拦截 RPC
3. 根据 RPC 方法名查找提供该方法的 Service（可能有多个）
4. 使用哈希算法选择目标 Service（负载均衡）
5. 转发 RPC 调用，在 `Custom` 中携带 `EdgeID`
6. 将响应返回给 Edge

**时序图：**

```
Edge         Exchange         Servicebound      Service
  |              |                  |              |
  |--Call RPC-->|                  |              |
  |              |                  |              |
  |              |--GetServicesByRPC|              |
  |              |                  |              |
  |              |<--ServiceList----|              |
  |              |                  |              |
  |              |------------------|--Call RPC-->|
  |              |                  |              |
  |              |<-----------------|--Response---|
  |              |                  |              |
  |<--Response----|                  |              |
  |              |                  |              |
  
  说明: Edge 发起 RPC 调用指定 method
        Exchange Hijack 拦截并查找提供该 RPC 的 Services
        使用哈希算法 Hash(edgeID, addr) 选择 Service
        在 Custom 中追加 EdgeID 后转发 RPC
        返回 Response(Data+Custom)
```

**负载均衡策略：**

```go
// 使用哈希算法选择 Service
index := misc.Hash(ex.conf.Exchange.HashBy, len(svcs), edgeID, addr)
svc := svcs[index]
```

**关键代码：**

```go
func (ex *exchange) forwardRPCToService(end geminio.End) {
    edgeID := end.ClientID()
    addr := end.RemoteAddr()
    
    end.Hijack(func(ctx context.Context, method string, r1 geminio.Request, r2 geminio.Response) {
        // 查找提供该 RPC 的 Services
        svcs, err := ex.Servicebound.GetServicesByRPC(method)
        if err != nil {
            r2.SetError(err)
            return
        }
        
        // 负载均衡选择 Service
        index := misc.Hash(ex.conf.Exchange.HashBy, len(svcs), edgeID, addr)
        svc := svcs[index]
        
        // 在 Custom 中携带 EdgeID
        tail := make([]byte, 8)
        binary.BigEndian.PutUint64(tail, edgeID)
        custom := append(r1.Custom(), tail...)
        
        // 转发 RPC
        r3 := svc.NewRequest(r1.Data(), ropt)
        r4, err := svc.Call(ctx, method, r3, copt)
        
        r2.SetData(r4.Data())
        r2.SetCustom(r4.Custom())
    })
}
```

### Stream 转发

Stream 用于建立 Service 和 Edge 之间的双向数据流。

#### Service -> Edge Stream

**流程：**

1. Service 创建 Stream，在 `Peer` 字段中指定目标 `EdgeID`
2. Exchange 解析 `EdgeID`
3. 查找对应的 Edge
4. 在 Edge 端创建对应的 Stream
5. 双向转发 Stream 数据（Raw、Message、RPC）

**时序图：**

```
Service          Exchange         Edgebound        Edge
  |                 |                 |              |
  |--OpenStream---->|                 |              |
  |                 |                 |              |
  |                 |--GetEdgeByID--->|              |
  |                 |                 |              |
  |                 |<--Edge End------|              |
  |                 |                 |              |
  |                 |-----------------|--OpenStream->|
  |                 |                 |              |
  |                 |<----------------|--EdgeStream--|
  |                 |                 |              |
  |<--Connected-----|                 |              |
  |                 |                 |              |
  |<==============Bidirectional Data==========>|
  |                 |                 |              |
  
  说明: Service 创建 Stream，Peer=EdgeID
        Exchange 解析 EdgeID 并查找 Edge
        如果 Edge 在线，建立双向 Stream 并转发数据
        如果 Edge 不在线，关闭 Stream
  
  或者（Edge不在线）:
  
Service          Exchange         Edgebound        Edge
  |                 |                 |              |
  |--OpenStream---->|                 |              |
  |                 |                 |              |
  |                 |--GetEdgeByID--->|              |
  |                 |                 |              |
  |                 |<--nil-----------|              |
  |                 |                 |              |
  |<--Close Stream--|                 |              |
```

**关键代码：**

```go
func (ex *exchange) StreamToEdge(serviceStream geminio.Stream) {
    // 从 Peer 中解析 EdgeID
    peer := serviceStream.Peer()
    edgeID, err := strconv.ParseUint(peer, 10, 64)
    
    // 查找 Edge
    edge := ex.Edgebound.GetEdgeByID(edgeID)
    if edge == nil {
        serviceStream.Close()
        return
    }
    
    // 创建 Edge Stream
    edgeStream, err := edge.OpenStream()
    
    // 双向转发
    ex.streamForward(serviceStream, edgeStream)
}
```

#### Edge -> Service Stream

**流程：**

1. Edge 创建 Stream，在 `Peer` 字段中指定目标 Service 名称
2. Exchange 根据 Service 名称查找 Service
3. 在 Service 端创建对应的 Stream
4. 双向转发 Stream 数据

**时序图：**

```
Edge         Exchange         Servicebound      Service
  |              |                  |              |
  |--OpenStream->|                  |              |
  |              |                  |              |
  |              |--GetServiceByName|              |
  |              |                  |              |
  |              |<--Service End----|              |
  |              |                  |              |
  |              |------------------|--OpenStream->|
  |              |                  |              |
  |              |<-----------------|--ServiceStream|
  |              |                  |              |
  |<--Connected---|                  |              |
  |              |                  |              |
  |<============Bidirectional Data==========>|
  |              |                  |              |
  
  说明: Edge 创建 Stream，Peer=ServiceName
        Exchange 解析 Service 名称并查找 Service
        如果 Service 在线，建立双向 Stream 并转发数据
        如果 Service 不在线，关闭 Stream
  
  或者（Service不在线）:
  
Edge         Exchange         Servicebound      Service
  |              |                  |              |
  |--OpenStream->|                  |              |
  |              |                  |              |
  |              |--GetServiceByName|              |
  |              |                  |              |
  |              |<--Error-----------|              |
  |              |                  |              |
  |<--Close Stream|                  |              |
```

**关键代码：**

```go
func (ex *exchange) StreamToService(edgeStream geminio.Stream) {
    // 从 Peer 中解析 Service 名称
    peer := edgeStream.Peer()
    svc, err := ex.Servicebound.GetServiceByName(peer)
    
    // 创建 Service Stream
    serviceStream, err := svc.OpenStream()
    
    // 双向转发
    ex.streamForward(edgeStream, serviceStream)
}
```

#### Stream 数据转发

Stream 支持三种数据类型的转发：

1. **Raw 数据**：原始字节流双向转发
2. **Message**：消息双向转发
3. **RPC**：RPC 调用双向转发

**时序图：**

```
Stream A        Exchange        Stream B
    |               |               |
    |--Raw Data---->|               |
    |               |               |
    |               |--Raw Data---->|
    |               |               |
    |<--Raw Data----|               |
    |               |               |
    |               |<--Raw Data----|
    |               |               |
    |--Message----->|               |
    |               |               |
    |               |--Publish Msg->|
    |               |               |
    |<--Message-----|               |
    |               |               |
    |               |<--Message------|
    |               |               |
    |--RPC Request->|               |
    |               |               |
    |               |--Call RPC----->|
    |               |               |
    |               |<--RPC Response-|
    |               |               |
    |<--RPC Response|               |
  
  说明: Stream 支持三种数据类型的双向转发
        - Raw 数据: 原始字节流双向转发
        - Message: 消息双向转发
        - RPC: RPC 调用双向转发
```

**关键代码：**

```go
func (ex *exchange) streamForward(left, right geminio.Stream) {
    // Raw 数据转发
    ex.streamForwardRaw(left, right)
    // Message 转发
    ex.streamForwardMessage(left, right)
    // RPC 转发
    ex.streamForwardRPC(left, right)
}
```

### Exchange 特性

1. **透明转发**
   - Service 和 Edge 无需知道对方的具体位置
   - 通过 ID 或名称进行路由

2. **负载均衡**
   - RPC 调用支持多 Service 负载均衡
   - 使用哈希算法保证相同 Edge 的请求路由到同一 Service

3. **错误处理**
   - 目标不在线时返回明确错误
   - 支持超时控制（默认 30 秒）

4. **Custom 字段传递**
   - 通过 Custom 字段传递路由信息（EdgeID）
   - 保持原始 Custom 数据，仅在末尾追加路由信息

5. **异步处理**
   - 消息转发使用 goroutine 异步处理
   - RPC 转发同步等待响应

### 数据流向总结

```
Service -> Edge:
  - Message: Service 指定 EdgeID -> Exchange 转发 -> Edge
  - RPC: Service 指定 EdgeID -> Exchange 转发 -> Edge -> 响应返回
  - Stream: Service 指定 EdgeID -> Exchange 建立双向流

Edge -> Service:
  - Message: Edge 指定 Topic -> Exchange -> MQM -> 订阅的 Service
  - RPC: Edge 指定方法名 -> Exchange 负载均衡 -> Service -> 响应返回
  - Stream: Edge 指定 Service 名称 -> Exchange 建立双向流
```

---

---

## Frontier + Frontlas 集群模式

### 概述

Frontier 集群模式通过引入 Frontlas（Frontier Atlas）组件实现多 Frontier 实例的协调管理。Frontlas 是一个无状态的集群管理组件，使用 Redis 存储 Frontier、Service 和 Edge 的元数据信息。

**架构图：**

```
                    Frontlas (集群管理)
                         |
            +------------+------------+
            |                         |
        Redis (元数据存储)        gRPC/REST API
            |                         |
    +-------+-------+         +-------+-------+
    |               |         |               |
Frontier-1    Frontier-2    Service-1    Service-2
    |               |         |               |
Edge-1         Edge-2         |               |
                              ...             ...
```

**核心组件：**

- **Frontier**: 无状态的数据平面组件，可以水平扩展
- **Frontlas**: 无状态的集群管理组件，使用 Redis 存储元数据
- **Redis**: 存储 Frontier、Service、Edge 的元数据和存活信息

### 多 Frontier 下的连接管理

#### Service 连接管理

在集群模式下，Service 通过 `clusterServiceEnd` 管理多个 Frontier 连接。

**连接池管理：**

```go
type clusterServiceEnd struct {
    cc clusterv1.ClusterServiceClient  // gRPC 客户端，连接 Frontlas
    
    edgefrontiers *mapmap.BiMap        // EdgeID <-> FrontierID 双向映射
    frontiers     sync.Map              // FrontierID -> frontierNend 连接池
}
```

**关键机制：**

1. **定期更新 Frontier 列表**
   - 每 10 秒通过 gRPC 调用 `ListFrontiers` 获取最新的 Frontier 列表
   - 对比当前连接池，识别新增和删除的 Frontier

2. **动态连接管理**
   - **新增 Frontier**: 自动创建连接并加入连接池
   - **删除 Frontier**: 关闭连接并从连接池移除
   - **连接漂移**: 当 Frontier 地址变化时，关闭旧连接，创建新连接

**时序图：**

```
Service         Frontlas         Frontier-1      Frontier-2
  |                |                 |              |
  |--ListFrontiers>|                 |              |
  |                |                 |              |
  |<--FrontierList-|                 |              |
  |                |                 |              |
  |--Compare Pool->|                 |              |
  |                |                 |              |
  |--New Frontier->|                 |              |
  |                |                 |              |
  |----------------|-----------------|--Connect---->|
  |                |                 |              |
  |<----------------|-----------------|--Connected---|
  |                |                 |              |
  |--Old Frontier->|                 |              |
  |                |                 |              |
  |----------------|-----------------|--Close------>|
  |                |                 |              |
```

**流程：**

1. Service 启动时连接 Frontlas（gRPC）
2. 调用 `ListFrontiers` 获取所有 Frontier 列表
3. 为每个 Frontier 创建 `serviceEnd` 连接
4. 定期（10秒）更新 Frontier 列表
5. 对比差异：
   - **新增**: 创建新连接
   - **删除**: 关闭旧连接，清理 EdgeID 映射
   - **变更**: 关闭旧连接，创建新连接

**关键代码：**

```go
func (end *clusterServiceEnd) update() error {
    // 获取最新 Frontier 列表
    rsp, err := end.cc.ListFrontiers(context.TODO(), &clusterv1.ListFrontiersRequest{})
    
    // 对比当前连接池
    keeps := []string{}
    removes := []*frontierNend{}
    news := []*clusterv1.Frontier{}
    
    // 识别需要删除的 Frontier
    end.frontiers.Range(func(key, value interface{}) bool {
        // 如果不在新列表中，标记为删除
        if !foundInNewList {
            removes = append(removes, frontierNend)
        }
        return true
    })
    
    // 识别新增的 Frontier
    for _, frontier := range rsp.Frontiers {
        if !foundInKeeps {
            news = append(news, frontier)
        }
    }
    
    // 异步处理连接变更
    go func() {
        // 关闭旧连接
        for _, remove := range removes {
            remove.end.Close()
            end.edgefrontiers.DelValue(remove.frontier.FrontierId)
        }
        // 创建新连接
        for _, new := range news {
            serviceEnd, err := end.newServiceEnd(new.AdvertisedSbAddr)
            end.frontiers.Swap(new.FrontierId, &frontierNend{
                frontier: new,
                end:      serviceEnd,
            })
        }
    }()
}
```

#### Edge 路由查找

当 Service 需要与特定 Edge 通信时，需要查找该 Edge 所在的 Frontier。

**查找机制：**

1. **缓存查找**: 首先从 `edgefrontiers` 双向映射中查找 EdgeID 对应的 FrontierID
2. **Frontlas 查询**: 如果缓存未命中，调用 `GetFrontierByEdge` 查询 Frontlas
3. **连接获取**: 从连接池中获取对应的 Frontier 连接
4. **连接创建**: 如果连接池中没有，动态创建连接

**时序图：**

```
Service         Frontlas         Redis           Frontier
  |                |                |              |
  |--Publish(edgeID)>|                |              |
  |                |                |              |
  |--Check Cache-->|                |              |
  |                |                |              |
  |<--Cache Miss----|                |              |
  |                |                |              |
  |--GetFrontierByEdge>|                |              |
  |                |                |              |
  |                |--GetEdge------->|              |
  |                |                |              |
  |                |<--Edge Info----|              |
  |                |                |              |
  |                |--GetFrontier-->|              |
  |                |                |              |
  |                |<--Frontier Info|              |
  |                |                |              |
  |<--Frontier Info-|                |              |
  |                |                |              |
  |--Get Connection>|                |              |
  |                |                |              |
  |----------------|----------------|--Publish---->|
  |                |                |              |
```

**关键代码：**

```go
func (end *clusterServiceEnd) lookup(edgeID uint64) (string, *serviceEnd, error) {
    // 1. 从缓存查找
    frontierID, ok := end.edgefrontiers.GetValue(edgeID)
    if !ok {
        // 2. 从 Frontlas 查询
        rsp, err := end.cc.GetFrontierByEdge(context.TODO(), &clusterv1.GetFrontierByEdgeIDRequest{
            EdgeId: edgeID,
        })
        frontierID = rsp.Fontier.FrontierId
        // 3. 更新缓存
        end.edgefrontiers.Set(edgeID, frontierID)
    }
    
    // 4. 从连接池获取连接
    fe, ok := end.frontiers.Load(frontierID)
    if !ok {
        // 5. 动态创建连接
        serviceEnd, err := end.newServiceEnd(frontier.AdvertisedSbAddr)
        end.frontiers.Swap(frontierID, &frontierNend{
            frontier: frontier,
            end:      serviceEnd,
        })
    }
    
    return frontierID, serviceEnd, nil
}
```

#### 连接漂移处理

**场景：**

1. **Frontier 实例重启**: IP 地址可能变化（K8s Pod）
2. **Frontier 实例迁移**: 节点故障导致 Pod 迁移
3. **Frontier 配置变更**: 端口或地址配置变化

**处理机制：**

1. **定期更新检测**: 每 10 秒更新 Frontier 列表，检测地址变化
2. **连接对比**: 通过 `frontierEqual` 比较 FrontierID 和地址
3. **优雅切换**: 
   - 先创建新连接
   - 再关闭旧连接
   - 使用 `Swap` 保证原子性

**关键代码：**

```go
func frontierEqual(a, b *clusterv1.Frontier) bool {
    return a.AdvertisedSbAddr == b.AdvertisedSbAddr &&
           a.FrontierId == b.FrontierId
}

// 在 update() 中
prev, ok := end.frontiers.Swap(new.FrontierId, &frontierNend{
    frontier: new,
    end:      serviceEnd,
})
if ok {
    // 关闭旧连接
    prev.(*frontierNend).end.Close()
}
```

#### Edge 连接管理

在集群模式下，Edge 直接连接到 Frontier 实例。Edge 可以连接到任意 Frontier，当连接失败时可以重试或切换到其他 Frontier。

**连接流程：**

1. **Edge 初始连接**
   - Edge 通过 Dialer 连接到指定的 Frontier 地址
   - Frontier 接受连接并分配 EdgeID
   - Edge 上线处理

2. **Edge 上线通知**
   - Frontier 在本地注册 Edge
   - Frontier 通过 Exchange 通知 Service（如果 Service 在线）
   - Frontier 向 Frontlas 报告 Edge 上线

3. **心跳续期**
   - Edge 每 30 秒向 Frontier 发送心跳
   - Frontier 转发心跳到 Frontlas
   - Frontlas 续期 Edge 的存活标记

4. **连接失败处理**
   - Edge 连接失败时，根据配置决定是否重试
   - 使用 `NewRetryEdge` 时，会自动重连
   - 可以配置连接到不同的 Frontier 地址

**时序图：**

```
Edge            Frontier-1        Exchange         Frontlas         Redis
  |                 |                 |                |              |
  |--Connect------->|                 |                |              |
  |                 |                 |                |              |
  |                 |--Allocate EdgeID>|                |              |
  |                 |                 |                |              |
  |                 |<--EdgeID--------|                |              |
  |                 |                 |                |              |
  |<--Connected-----|                 |                |              |
  |                 |                 |                |              |
  |                 |--EdgeOnline---->|                |              |
  |                 |                 |                |              |
  |                 |                 |--EdgeOnline---->|              |
  |                 |                 |                |              |
  |                 |                 |                |--SetEdgeAndAlive>|
  |                 |                 |                |              |
  |                 |                 |                |<--Success-----|
  |                 |                 |                |              |
  |                 |                 |<--Success------|              |
  |                 |                 |                |              |
  |                 |<--Success-------|                |              |
  |                 |                 |                |              |
  |--Heartbeat------|                 |                |              |
  |                 |                 |                |              |
  |                 |--EdgeHeartbeat->|                |              |
  |                 |                 |                |              |
  |                 |                 |--EdgeHeartbeat>|              |
  |                 |                 |                |              |
  |                 |                 |                |--ExpireEdge-->|
  |                 |                 |                |              |
  |                 |                 |                |<--Success-----|
  |                 |                 |                |              |
  |                 |                 |<--Success------|              |
  |                 |                 |                |              |
  |                 |<--Success-------|                |              |
  |                 |                 |                |              |
```

**Edge 重连场景：**

当 Edge 连接失败或 Frontier 故障时，Edge 可以重连到其他 Frontier。

**时序图：**

```
Edge            Frontier-1        Frontier-2        Frontlas         Redis
  |                 |                 |                |              |
  |--Connect------->|                 |                |              |
  |                 |                 |                |              |
  |                 |<--Connection Failed|                |              |
  |                 |                 |                |              |
  |--Retry Connect->|                 |                |              |
  |                 |                 |                |              |
  |                 |<--Connection Failed|                |              |
  |                 |                 |                |              |
  |--Connect--------|---------------->|                |              |
  |                 |                 |                |              |
  |                 |                 |--Allocate EdgeID>|              |
  |                 |                 |                |              |
  |                 |                 |<--EdgeID--------|              |
  |                 |                 |                |              |
  |<--Connected-----|                 |                |              |
  |                 |                 |                |              |
  |                 |                 |--EdgeOnline---->|              |
  |                 |                 |                |              |
  |                 |                 |                |--SetEdgeAndAlive>|
  |                 |                 |                |              |
  |                 |                 |                |<--Success-----|
  |                 |                 |                |              |
  |                 |                 |<--Success------|              |
  |                 |                 |                |              |
  |                 |                 |--Delete Old Edge>|              |
  |                 |                 |                |              |
  |                 |                 |                |--DeleteEdge-->|
  |                 |                 |                |              |
  |                 |                 |                |<--Success-----|
  |                 |                 |                |              |
  |                 |                 |<--Success------|              |
```

**Edge 下线流程：**

**时序图：**

```
Edge            Frontier         Exchange         Frontlas         Redis
  |                 |                 |                |              |
  |--Disconnect---->|                 |                |              |
  |                 |                 |                |              |
  |                 |--EdgeOffline-->|                |              |
  |                 |                 |                |              |
  |                 |                 |--EdgeOffline-->|              |
  |                 |                 |                |              |
  |                 |                 |                |--DeleteEdge->|
  |                 |                 |                |              |
  |                 |                 |                |<--Success-----|
  |                 |                 |                |              |
  |                 |                 |<--Success------|              |
  |                 |                 |                |              |
  |                 |<--Success-------|                |              |
  |                 |                 |                |              |
  |                 |--Clean Local---->|                |              |
  |                 |                 |                |              |
```

**关键机制：**

1. **EdgeID 分配**
   - Edge 连接时，Frontier 通过 Exchange 向 Service 请求 EdgeID
   - 如果 Service 不在线，根据配置决定是否自动分配 EdgeID
   - EdgeID 在集群中全局唯一

2. **连接选择**
   - Edge 可以连接到任意 Frontier 实例
   - 通常通过负载均衡器或 DNS 选择 Frontier
   - 支持配置多个 Frontier 地址进行重试

3. **状态同步**
   - Edge 上线/下线时，Frontier 同步状态到 Frontlas
   - Frontlas 更新 Redis 中的 Edge 元数据
   - Service 通过 Frontlas 查询 Edge 所在的 Frontier

4. **心跳机制**
   - Edge 每 30 秒发送心跳到 Frontier
   - Frontier 转发心跳到 Frontlas
   - Frontlas 续期 Redis 中的存活标记（TTL 30秒）

**关键代码：**

```go
// Edge 上线处理
func (em *edgeManager) online(end geminio.End) error {
    // 1. 检查是否存在旧连接
    old, ok := em.edges[end.ClientID()]
    if ok {
        oldend.Close()  // 关闭旧连接
    }
    
    // 2. 添加到本地缓存
    em.edges[end.ClientID()] = end
    
    // 3. 创建数据库记录
    edge := &model.Edge{
        EdgeID: end.ClientID(),
        Meta:   string(end.Meta()),
        Addr:   end.RemoteAddr().String(),
    }
    em.repo.CreateEdge(edge)
    
    // 4. 通知 Exchange
    if em.exchange != nil {
        em.exchange.EdgeOnline(end.ClientID(), end.Meta(), end.RemoteAddr())
    }
}

// Edge 上线通知 Frontlas
func (fm *FrontierManager) EdgeOnline(ctx context.Context, req geminio.Request, rsp geminio.Response) {
    edgeOnline := &gapis.EdgeOnline{}
    json.Unmarshal(req.Data(), edgeOnline)
    
    // 更新 Redis
    fm.repo.SetEdgeAndAlive(edgeOnline.EdgeID, &repo.Edge{
        FrontierID: edgeOnline.FrontierID,
        Addr:       edgeOnline.Addr,
    }, edgeHeartbeatInterval)
}
```

### 水平扩展原理

#### Frontier 无状态设计

Frontier 实例是无状态的，所有状态信息存储在：
- **内存**: 当前连接的 Service 和 Edge 信息（重启后丢失）
- **Redis**: 通过 Frontlas 持久化的元数据

**无状态特性：**

1. **无本地存储**: Frontier 不存储任何持久化数据
2. **无会话绑定**: Service 和 Edge 可以连接到任意 Frontier 实例
3. **动态路由**: 通过 Frontlas 查询 Edge 所在的 Frontier

#### 水平扩展流程

**扩展步骤：**

1. **添加 Frontier 实例**
   - 新 Frontier 启动并连接 Frontlas
   - 在 Redis 中注册 Frontier 信息（FrontierID、地址等）
   - 设置存活标记（TTL 30秒）

2. **Service 发现新 Frontier**
   - Service 定期调用 `ListFrontiers` 获取最新列表
   - 检测到新 Frontier，自动创建连接
   - 新连接加入连接池

3. **Edge 连接分配**
   - 新 Edge 可以连接到任意 Frontier 实例
   - 通过负载均衡或随机选择
   - Edge 信息记录到 Redis（关联 FrontierID）

4. **流量自动分配**
   - 新 Edge 的请求自动路由到对应的 Frontier
   - Service 通过 `lookup` 查找 Edge 所在的 Frontier
   - 实现负载均衡

**时序图：**

```
NewFrontier     Frontlas         Redis           Service
  |                |                |              |
  |--Connect------>|                |              |
  |  (geminio)     |                |              |
  |                |                |              |
  |--ConnOnline--->|                |              |
  |  (FrontierID,  |                |              |
  |   Addr)        |                |              |
  |                |                |              |
  |                |--SetFrontierAndAlive>|              |
  |                |  (Hash + TTL)  |              |
  |                |                |              |
  |                |<--Success------|              |
  |                |                |              |
  |<--Registered---|                |              |
  |                |                |              |
  |                |<--ListFrontiers|              |
  |                |  (gRPC)        |              |
  |                |                |              |
  |                |--GetAllFrontiers>|              |
  |                |                |              |
  |                |<--FrontierList-|              |
  |                |                |              |
  |                |--FrontierList->|              |
  |                |                |              |
  |                |                |--Compare Pool>|
  |                |                |              |
  |                |                |--New Frontier>|
  |                |                |              |
  |                |                |--Connect----->|
  |                |                |  (geminio)    |
  |                |                |              |
  |                |                |<--Connected---|
```

#### 负载均衡策略

**Edge 连接分配：**

- **随机分配**: Edge 随机选择 Frontier 连接
- **负载均衡**: 根据 Frontier 的 Edge 数量分配（需要额外实现）

**Service 请求路由：**

- **精确路由**: 通过 EdgeID 查找对应的 Frontier
- **缓存优化**: EdgeID -> FrontierID 映射缓存，减少 Frontlas 查询

#### 高可用性

**Frontier 故障处理：**

1. **心跳检测**: Frontier 每 30 秒向 Frontlas 发送心跳
2. **TTL 过期**: Redis 中的存活标记过期后，Frontier 被视为离线
3. **自动清理**: Frontlas 清理过期的 Frontier 信息
4. **连接重建**: Service 检测到 Frontier 离线，关闭连接并清理缓存

**Frontlas 高可用：**

- **无状态设计**: Frontlas 实例无状态，可以部署多个
- **Redis 高可用**: 使用 Redis Sentinel 或 Cluster 模式
- **负载均衡**: 多个 Frontlas 实例通过负载均衡器提供服务

### Redis 数据模型

Frontlas 使用 Redis 存储 Frontier、Service 和 Edge 的元数据和存活信息。所有数据通过 TTL（Time To Live）机制管理生命周期。

#### Frontier 数据模型

**存储结构：**

1. **元数据（Hash）**
   - Key: `frontlas:frontiers:{frontierID}`
   - Type: Hash
   - Fields:
     - `advertised_sb_addr`: Servicebound 地址
     - `advertised_eb_addr`: Edgebound 地址
     - `edge_count`: Edge 数量
     - `service_count`: Service 数量
   - TTL: 由配置的 `service_meta` 决定（默认 30 秒）

2. **存活标记（String）**
   - Key: `frontlas:alive:frontiers:{frontierID}`
   - Type: String
   - Value: `1`
   - TTL: 30 秒（通过心跳续期）

**示例：**

```
# Frontier 元数据
frontlas:frontiers:frontier01
  advertised_sb_addr: "192.168.1.10:30011"
  advertised_eb_addr: "192.168.1.10:30012"
  edge_count: "100"
  service_count: "5"

# Frontier 存活标记
frontlas:alive:frontiers:frontier01 = "1" (TTL: 30s)
```

**操作：**

- **创建**: `SetFrontierAndAlive()` - 使用 Lua 脚本原子性创建元数据和存活标记
- **更新**: `ExpireFrontier()` - 更新存活标记的 TTL
- **删除**: `DeleteFrontier()` - 删除存活标记，保留元数据（edge_count 设为 0）

#### Service 数据模型

**存储结构：**

1. **元数据（String/JSON）**
   - Key: `frontlas:services:{serviceID}`
   - Type: String
   - Value: JSON 格式
     ```json
     {
       "service": "user-service",
       "frontier_id": "frontier01",
       "addr": "192.168.1.20:54321",
       "update_time": 1234567890
     }
     ```
   - TTL: 由配置的 `service_meta` 决定（默认 30 秒）

2. **存活标记（String）**
   - Key: `frontlas:alive:services:{serviceID}`
   - Type: String
   - Value: `1`
   - TTL: 30 秒（通过心跳续期）

**示例：**

```
# Service 元数据
frontlas:services:12345 = '{"service":"user-service","frontier_id":"frontier01","addr":"192.168.1.20:54321","update_time":1234567890}' (TTL: 30s)

# Service 存活标记
frontlas:alive:services:12345 = "1" (TTL: 30s)
```

**操作：**

- **创建**: `SetServiceAndAlive()` - 使用 Lua 脚本原子性创建元数据和存活标记，并更新 Frontier 的 service_count
- **更新**: `ExpireService()` - 更新元数据和存活标记的 TTL
- **删除**: `DeleteService()` - 使用 Lua 脚本删除存活标记，更新 Frontier 的 service_count

#### Edge 数据模型

**存储结构：**

1. **元数据（String/JSON）**
   - Key: `frontlas:edges:{edgeID}`
   - Type: String
   - Value: JSON 格式
     ```json
     {
       "frontier_id": "frontier01",
       "addr": "192.168.1.30:54322",
       "update_time": 1234567890
     }
     ```
   - TTL: 由配置的 `edge_meta` 决定（默认 30 秒）

2. **存活标记（String）**
   - Key: `frontlas:alive:edges:{edgeID}`
   - Type: String
   - Value: `1`
   - TTL: 30 秒（通过心跳续期）

**示例：**

```
# Edge 元数据
frontlas:edges:67890 = '{"frontier_id":"frontier01","addr":"192.168.1.30:54322","update_time":1234567890}' (TTL: 30s)

# Edge 存活标记
frontlas:alive:edges:67890 = "1" (TTL: 30s)
```

**操作：**

- **创建**: `SetEdgeAndAlive()` - 使用 Pipeline 原子性创建元数据和存活标记，并更新 Frontier 的 edge_count
- **更新**: `ExpireEdge()` - 更新元数据和存活标记的 TTL
- **删除**: `DeleteEdge()` - 使用 Lua 脚本删除存活标记，更新 Frontier 的 edge_count

#### 数据关系

**Frontier <-> Service 关系：**

- Service 元数据中包含 `frontier_id` 字段
- Frontier 的 Hash 中包含 `service_count` 字段
- 通过 `frontier_id` 可以查找 Service 所在的 Frontier

**Frontier <-> Edge 关系：**

- Edge 元数据中包含 `frontier_id` 字段
- Frontier 的 Hash 中包含 `edge_count` 字段
- 通过 `frontier_id` 可以查找 Edge 所在的 Frontier

**查询路径：**

```
EdgeID -> frontlas:edges:{edgeID} -> frontier_id -> frontlas:frontiers:{frontierID} -> advertised_sb_addr
```

#### TTL 和心跳机制

**TTL 策略：**

1. **元数据 TTL**: 配置项控制（默认 30 秒），用于清理长期不活跃的数据
2. **存活标记 TTL**: 固定 30 秒，通过心跳续期

**心跳机制：**

1. **Frontier 心跳**: 每 30 秒向 Frontlas 发送心跳，续期 `frontlas:alive:frontiers:{frontierID}`
2. **Service 心跳**: 每 30 秒向 Frontier 发送心跳，Frontier 转发给 Frontlas，续期 `frontlas:alive:services:{serviceID}`
3. **Edge 心跳**: 每 30 秒向 Frontier 发送心跳，Frontier 转发给 Frontlas，续期 `frontlas:alive:edges:{edgeID}`

**过期处理：**

- 当存活标记过期时，对应的资源被视为离线
- Frontlas 会清理过期的存活标记
- Service 通过定期查询检测到 Frontier 离线，自动清理连接

#### 数据一致性

**原子性操作：**

- 使用 Redis Lua 脚本保证创建/删除操作的原子性
- 使用 Pipeline 批量操作保证一致性

**关键 Lua 脚本：**

1. **frontier_create.lua**: 创建 Frontier 元数据和存活标记
2. **service_create.lua**: 创建 Service 元数据、存活标记，并更新 Frontier 计数
3. **service_delete.lua**: 删除 Service 存活标记，并更新 Frontier 计数
4. **edge_delete.lua**: 删除 Edge 存活标记，并更新 Frontier 计数

### CRD 部分原理

#### CRD 定义

Frontier 使用 Kubernetes Operator 模式，通过 CRD（Custom Resource Definition）定义集群配置。

**CRD 结构：**

```yaml
apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: frontierclusters.frontier.singchia.io
spec:
  group: frontier.singchia.io
  names:
    kind: FrontierCluster
    plural: frontierclusters
  scope: Namespaced
  versions:
    - name: v1alpha1
```

**CRD Spec 定义：**

```go
type FrontierClusterSpec struct {
    Frontier Frontier `json:"frontier"`
    Frontlas Frontlas `json:"frontlas"`
}

type Frontier struct {
    Replicas     int                 `json:"replicas,omitempty"`
    Servicebound Servicebound        `json:"servicebound"`
    Edgebound    Edgebound           `json:"edgebound"`
    Image        string              `json:"image,omitempty"`
    NodeAffinity corev1.NodeAffinity `json:"nodeAffinity,omitempty"`
}

type Frontlas struct {
    Replicas     int                 `json:"replicas,omitempty"`
    ControlPlane ControlPlane        `json:"controlplane,omitempty"`
    Redis        Redis               `json:"redis"`
    Image        string              `json:"image,omitempty"`
}
```

#### Controller 工作原理

**Reconcile 循环：**

Controller 通过 Reconcile 函数实现声明式配置管理。

**工作流程：**

1. **监听 CRD 变更**: Controller 监听 `FrontierCluster` 资源的创建、更新、删除
2. **Reconcile 触发**: 当 CRD 变更时，触发 Reconcile 函数
3. **状态对比**: 对比期望状态（Spec）和实际状态（Status）
4. **资源创建/更新**: 创建或更新 Deployment、Service 等资源
5. **状态更新**: 更新 CRD 的 Status 字段

**时序图：**

```
用户            K8s API        Controller      Deployment    Service
  |                |                |              |            |
  |--Apply CRD---->|                |              |            |
  |                |                |              |            |
  |                |--Event-------->|              |            |
  |                |                |              |            |
  |                |                |--Reconcile-->|            |
  |                |                |              |            |
  |                |                |--Ensure Service>|            |
  |                |                |              |            |
  |                |                |--Create/Update>|            |
  |                |                |              |            |
  |                |<--Create Service|              |            |
  |                |                |              |            |
  |                |                |--Ensure Deployment>|            |
  |                |                |              |            |
  |                |                |--Create/Update>|            |
  |                |                |              |            |
  |                |<--Create Deployment|              |            |
  |                |                |              |            |
  |                |                |--Check Ready->|            |
  |                |                |              |            |
  |                |                |<--Ready--------|            |
  |                |                |              |            |
  |                |                |--Update Status>|            |
  |                |                |              |            |
  |                |<--Status Update|              |            |
```

**关键代码：**

```go
func (r *FrontierClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
    // 1. 获取 CRD
    frontiercluster := frontierv1alpha1.FrontierCluster{}
    r.Get(ctx, req.NamespacedName, &frontiercluster)
    
    // 2. 确保 Service 存在
    r.ensureService(ctx, frontiercluster)
    
    // 3. 确保 TLS Secret 存在
    r.ensureTLS(ctx, frontiercluster)
    
    // 4. 确保 Deployment 存在
    ready, err := r.ensureDeployment(ctx, frontiercluster)
    
    // 5. 更新状态
    status.Update(ctx, r.client.Status(), &frontiercluster, statusOptions().
        withMessage(Info, "Good to go!").
        withRunningPhase())
}
```

#### 资源管理

**Deployment 创建：**

Controller 根据 CRD Spec 创建 Frontier 和 Frontlas 的 Deployment。

**关键配置：**

1. **环境变量注入**:
   - Frontier 地址和端口
   - Frontlas 地址
   - Redis 配置

2. **Pod 反亲和性**:
   - 确保 Frontier Pod 分布在不同节点
   - 提高可用性

3. **资源限制**:
   - 可配置 CPU 和内存限制

**关键代码：**

```go
func (r *FrontierClusterReconciler) ensureFrontierDeployment(ctx context.Context, fc v1alpha1.FrontierCluster) error {
    // 构建容器配置
    container := container.Builder().
        SetName("frontier").
        SetImage(image).
        SetEnvs([]corev1.EnvVar{
            {Name: FrontierServiceboundPortEnv, Value: strconv.Itoa(sbport)},
            {Name: FrontierEdgeboundPortEnv, Value: strconv.Itoa(ebport)},
            {Name: FrontlasAddrEnv, Value: frontlasAddr},
        })
    
    // 构建 Pod 模板
    podTemplateSpec := podtemplatespec.Builder().
        SetPodAntiAffinity(&corev1.PodAntiAffinity{
            RequiredDuringSchedulingIgnoredDuringExecution: []corev1.PodAffinityTerm{
                {TopologyKey: "kubernetes.io/hostname"},
            },
        })
    
    // 创建 Deployment
    deploy := deployment.Builder().
        SetReplicas(fc.FrontierReplicas()).
        SetPodTemplateSpec(podTemplateSpec).
        Build()
    
    deployment.CreateOrUpdate(ctx, r.client, deploy)
}
```

#### 自动扩缩容

**水平扩缩容：**

1. **修改 Replicas**: 用户修改 CRD 中的 `replicas` 字段
2. **Controller 检测**: Controller 检测到 Spec 变更
3. **更新 Deployment**: 更新 Deployment 的 Replicas
4. **K8s 调度**: Kubernetes 自动创建或删除 Pod
5. **状态同步**: Controller 更新 CRD Status

**垂直扩缩容：**

- 通过修改 Deployment 的资源限制实现
- 需要重启 Pod，影响较大

#### CRD Status 管理

**Status 字段：**

```go
type FrontierClusterStatus struct {
    Phase   Phase  `json:"phase"`      // Running, Failed, Pending
    Message string `json:"message"`    // 状态描述
}
```

**状态转换：**

- **Pending**: Deployment 未就绪
- **Running**: 所有 Deployment 就绪
- **Failed**: 创建或更新资源失败

**关键代码：**

```go
func (r *FrontierClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
    // 检查 Deployment 就绪状态
    frontierIsReady := deployment.IsReady(currentFrontierDeployment, fc.FrontierReplicas())
    frontlasIsReady := deployment.IsReady(currentFrontlasDeployment, fc.FrontlasReplicas())
    
    if !frontierIsReady || !frontlasIsReady {
        // 更新为 Pending 状态
        status.Update(ctx, r.client.Status(), &frontiercluster, statusOptions().
            withMessage(Info, "Deployment is not yet ready").
            withPendingPhase(10))
        return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
    }
    
    // 更新为 Running 状态
    status.Update(ctx, r.client.Status(), &frontiercluster, statusOptions().
        withMessage(Info, "Good to go!").
        withRunningPhase())
}
```

### 集群模式特性总结

1. **无状态设计**
   - Frontier 和 Frontlas 都是无状态的
   - 状态信息存储在 Redis 中
   - 支持水平扩展

2. **自动发现和路由**
   - Service 自动发现所有 Frontier 实例
   - 通过 Frontlas 查询 Edge 所在的 Frontier
   - 支持连接漂移和自动重连

3. **高可用性**
   - 多 Frontier 实例提供冗余
   - Frontlas 支持多实例部署
   - Redis 支持高可用模式

4. **声明式管理**
   - 通过 CRD 声明集群配置
   - Controller 自动管理资源
   - 支持自动扩缩容

---

## 总结

Frontier 通过以下机制实现了高效的客户端管理和消息转发：

1. **灵活的认证机制**：支持 Meta 信息传递和灵活的 ID 分配策略
2. **可靠的上下线管理**：保证数据一致性和并发安全
3. **高效的 Exchange 转发**：实现 Service 和 Edge 的解耦，支持消息、RPC 和 Stream 的透明转发
4. **集群模式支持**：通过 Frontlas 实现多 Frontier 实例的协调管理，支持水平扩展和高可用

这些设计使得 Frontier 能够支持大规模、高并发的边缘计算场景。
