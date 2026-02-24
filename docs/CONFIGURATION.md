## Configuration

If you need to further customize your Frontier instance, you can learn how various configurations work in this section. Customize your configuration, save it as ```frontier.yaml```, and mount it to the container at ```/usr/conf/frontier.yaml``` to take effect.

### Minimal Configuration

To get started, you can simply configure the service listening addresses for microservices and edge nodes:

```yaml
# Microservice configuration
servicebound:
  # Listening network
  listen:
    network: tcp
    # Listening address
    addr: 0.0.0.0:30011
# Edge node configuration
edgebound:
  # Listening network
  listen:
    network: tcp
    # Listening address
    addr: 0.0.0.0:30012
  # Whether to allow Frontier to allocate edgeID if no ID service is registered
  edgeid_alloc_when_no_idservice_on: true
```

### TLS

TLS configuration is supported for microservices, edge nodes, and control planes. mTLS is also supported, where Frontier verifies the client certificate.

```yaml
servicebound:
  listen:
    addr: 0.0.0.0:30011
    network: tcp
    tls:
      # Whether to enable TLS, default is disabled
      enable: false
      # Certificates and private keys, multiple pairs of certificates are allowed for client negotiation
      certs:
      - cert: servicebound.cert
        key: servicebound.key
      # Whether to enable mTLS, client certificates will be verified by the following CA
      mtls: false
      # CA certificates for verifying client certificates
      ca_certs:
      - ca1.cert
edgebound:
  listen:
    addr: 0.0.0.0:30012
    network: tcp
    tls:
      # Whether to enable TLS, default is disabled
      enable: false
      # Certificates and private keys, multiple pairs of certificates are allowed for client negotiation
      certs:
      - cert: edgebound.cert
        key: edgebound.key
      insecure_skip_verify: false
      # Whether to enable mTLS, client certificates will be verified by the following CA
      mtls: false
      # CA certificates for verifying client certificates
      ca_certs:
      - ca1.cert
```

### External MQ

If you need to configure an external MQ, Frontier supports publishing the corresponding topic to these MQs.

**AMQP**

```yaml
mqm:
  amqp:
    # Whether to allow
    enable: false
    # AMQP addresses
    addrs: null
    # Producer
    producer:
       # Exchange name
      exchange: ""
      # Equivalent to Frontier's internal topic concept, array values
      routing_keys: null
```

For AMQP, the above is the minimal configuration. If the topic of the message published by the edge node is in `routing_keys`, Frontier will publish to the `exchange.` If there are also microservices or other external MQs that declare the topic, Frontier will still choose one to publish based on hashby.

**Kafka**

```yaml
mqm:
  kafka:
    # Whether to allow
    enable: false
    # Kafka addresses
    addrs: null
    # Producer
       # Array values
      topics: null
```

For Kafka, the above is the minimal configuration. If the topic of the message published by the edge node is in the above array, Frontier will publish it. If there are also microservices or other external MQs that declare the topic, Frontier will still choose one to publish based on hashby.

**NATS**

```yaml
mqm:
  nats:
    # Whether to allow
    enable: false
    # NATS addresses
    addrs: null
    producer:
      # Equivalent to Frontier's internal topic concept, array values
      subjects: null
    # If Jetstream is allowed, it will be prioritized for publishing
    jetstream:
      enable: false
      # Jetstream name
      name: ""
      producer:
        # Equivalent to Frontier's internal topic concept, array values
        subjects: null
```

In NATS configuration, if Jetstream is allowed, it will be prioritized for publishing. If there are also microservices or other external MQs that declare the topic, Frontier will still choose one to publish based on hashby.

**NSQ**

```yaml
mqm:
  nsq:
    # Whether to allow
    enable: false
    # NSQ addresses
    addrs: null
    producer:
      # Array values
      topics: null
```
In NSQ's topics, if there are also microservices or other external MQs that declare the topic, Frontier will still choose one to publish based on hashby.

**Redis**

```yaml
mqm:
  redis:
    # Whether to allow
    enable: false
    # Redis addresses
    addrs: null
    # Redis DB
    db: 0
    # Password
    password: ""
    producer:
      # Equivalent to Frontier's internal topic concept, array values
      channels: null
```

If there are also microservices or other external MQs that declare the topic, Frontier will still choose one to publish based on hashby.

**Other Configurations**

```yaml
daemon:
  # Whether to enable PProf
  pprof:
    addr: 0.0.0.0:6060
    cpu_profile_rate: 0
    enable: true
  # Resource limits
  rlimit:
    enable: true
    nofile: 102400
  # Control plane enable
controlplane:
  enable: false
  listen:
    network: tcp
    addr: 0.0.0.0:30010
dao:
  # Supports buntdb and sqlite3, both use in-memory mode to remain stateless
  backend: buntdb
  # SQLite debug enable
  debug: false
exchange:
  # Frontier forwards edge node messages, RPCs, and open streams to microservices based on hash strategy: edgeid, srcip, or random, default is edgeid.
  # That is, the same edge node will always request the same microservice.
  hashby: edgeid
```

For more detailed configurations, see [frontier_all.yaml](../etc/frontier_all.yaml).

