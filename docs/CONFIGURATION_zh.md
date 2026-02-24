## 配置

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

更多详细配置见 [frontier_all.yaml](../etc/frontier_all.yaml)

