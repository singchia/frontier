<p align=center>
<img src="./docs/diagram/frontier-logo.png" width="30%">
</p>

<div align="center">

[![Go](https://github.com/singchia/frontier/actions/workflows/go.yml/badge.svg)](https://github.com/singchia/frontier/actions/workflows/go.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/singchia/frontier)](https://goreportcard.com/report/github.com/singchia/frontier)
[![Go Reference](https://pkg.go.dev/badge/badge/github.com/singchia/frontier.svg)](https://pkg.go.dev/github.com/singchia/frontier/api/dataplane/v1/service)
[![License](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)

English | [简体中文](./README_zh.md)

</div>


# Frontier

Frontier is a **full-duplex**, open-source long-connection gateway written in Go. It enables microservices to directly reach edge nodes or clients, and vice versa. It provides full-duplex **bidirectional RPC**, **messaging**, and **point-to-point streams**. Frontier follows **cloud-native** architecture principles, supports fast cluster deployment via Operator, and is built for **high availability** and **elastic scaling** to millions of online edge nodes or clients.

## Table of Contents

- [Features](#features)
- [Quick Start](#quick-start)
- [Architecture](#architecture)
- [Usage](#usage)
- [Configuration](#configuration)
- [Deployment](#deployment)
- [Cluster](#cluster)
- [Kubernetes](#kubernetes)
- [Development](#development)
- [Testing](#testing)
- [Community](#community)
- [License](#license)

## Quick Start

1. Run a single Frontier instance:

```bash
docker run -d --name frontier -p 30011:30011 -p 30012:30012 singchia/frontier:1.1.0
```

2. Build and run examples:

```bash
make examples
```

Run the chatroom example:

```bash
# Terminal 1
./bin/chatroom_service

# Terminal 2
./bin/chatroom_agent
```

Demo video:

https://github.com/singchia/frontier/assets/15531166/18b01d96-e30b-450f-9610-917d65259c30

## Features

- **Bidirectional RPC**: Services and edges can call each other with load balancing.
- **Messaging**: Topic-based publish/receive between services, edges, and external MQ.
- **Point-to-Point Streams**: Open direct streams for proxying, file transfer, and custom traffic.
- **Cloud-Native Deployment**: Run via Docker, Compose, Helm, or Operator.
- **High Availability and Scaling**: Support for reconnect, clustering, and horizontal scale with Frontlas.
- **Auth and Presence**: Edge auth and online/offline notifications.
- **Control Plane APIs**: gRPC and REST APIs for querying and managing online nodes.


## Architecture

**Frontier Component**

<img src="./docs/diagram/frontier.png" width="100%">

- _Service End_: The entry point for microservice functions, connecting by default.
- _Edge End_: The entry point for edge node or client functions.
- _Publish/Receive_: Publishing and receiving messages.
- _Call/Register_: Calling and registering functions.
- _OpenStream/AcceptStream_: Opening and accepting point-to-point streams (connections).
- _External MQ_: Frontier supports forwarding messages published from edge nodes to external MQ topics based on configuration.


Frontier requires both microservices and edge nodes to actively connect to Frontier. The metadata of Service and Edge (receiving topics, RPC, service names, etc.) can be carried during the connection. The default connection ports are:

- :30011: For microservices to connect and obtain Service.
- :30012: For edge nodes to connect and obtain Edge.
- :30010: For operations personnel or programs to use the control plane.


### Functionality

<table><thead>
  <tr>
    <th>Function</th>
    <th>Initiator</th>
    <th>Receiver</th>
    <th>Method</th>
    <th>Routing Method</th>
    <th>Description</th>
  </tr></thead>
<tbody>
  <tr>
    <td rowspan="2">Messager</td>
    <td>Service</td>
    <td>Edge</td>
    <td>Publish</td>
    <td>EdgeID+Topic</td>
    <td>Must publish to a specific EdgeID, the default topic is empty. The edge calls Receive to receive the message, and after processing, must call msg.Done() or msg.Error(err) to ensure message consistency.</td>
  </tr>
  <tr>
    <td>Edge</td>
    <td>Service or External MQ</td>
    <td>Publish</td>
    <td>Topic</td>
    <td>Must publish to a topic, and Frontier selects a specific Service or MQ based on the topic.</td>
  </tr>
  <tr>
    <td rowspan="2">RPCer</td>
    <td>Service</td>
    <td>Edge</td>
    <td>Call</td>
    <td>EdgeID+Method</td>
    <td>Must call a specific EdgeID, carrying the method name.</td>
  </tr>
  <tr>
    <td>Edge</td>
    <td>Service</td>
    <td>Call</td>
    <td>Method</td>
    <td>Must call a method, and Frontier selects a specific Service based on the method name.</td>
  </tr>
  <tr>
    <td rowspan="2">Multiplexer</td>
    <td>Service</td>
    <td>Edge</td>
    <td>OpenStream</td>
    <td>EdgeID</td>
    <td>Must open a stream to a specific EdgeID.</td>
  </tr>
  <tr>
    <td>Edge</td>
    <td>Service</td>
    <td>OpenStream</td>
    <td>ServiceName</td>
    <td>Must open a stream to a ServiceName, specified by service.OptionServiceName during Service initialization.</td>
  </tr>
</tbody></table>

**Key design principles include**:

1. All messages, RPCs, and Streams are point-to-point transmissions.
	- From microservices to edges, the edge node ID must be specified.
	- From edges to microservices, Frontier routes based on Topic and Method, and finally selects a microservice or external MQ through hashing. The default is hashing based on edgeid, but you can choose random or srcip.
2. Messages require explicit acknowledgment by the receiver.
	- To ensure message delivery semantics, the receiver must call msg.Done() or msg.Error(err) to ensure delivery consistency.
3. Streams opened by the Multiplexer logically represent direct communication between microservices and edge nodes.
	- Once the other side receives the stream, all functionalities on this stream will directly reach the other side, bypassing Frontier's routing policies.

## Usage

Detailed usage guide: [docs/USAGE.md](./docs/USAGE.md)

## Configuration

Detailed configuration guide: [docs/CONFIGURATION.md](./docs/CONFIGURATION.md)

## Deployment

In a single Frontier instance, you can choose the following methods to deploy your Frontier instance based on your environment.

### Docker

```bash
docker run -d --name frontier -p 30011:30011 -p 30012:30012 singchia/frontier:1.1.0
```

### Docker-Compose

```bash
git clone https://github.com/singchia/frontier.git
cd dist/compose
docker-compose up -d frontier
```

### Helm

If you are in a Kubernetes environment, you can use Helm to quickly deploy an instance.

```bash
git clone https://github.com/singchia/frontier.git
cd dist/helm
helm install frontier ./ -f values.yaml
```

Your microservice should connect to ```service/frontier-servicebound-svc:30011```, and your edge node can connect to the NodePort where `:30012` is located.

### Systemd

Use the dedicated Systemd docs:

- English: [dist/systemd/README_en.md](./dist/systemd/README_en.md)

### Operator

See the cluster deployment section below.

## Cluster

### Frontier + Frontlas Architecture

<img src="./docs/diagram/frontlas.png" width="100%">

The additional Frontlas component is used to build the cluster. Frontlas is also a stateless component and does not store other information in memory, so it requires additional dependency on Redis. You need to provide a Redis connection information to Frontlas, supporting `redis`, `sentinel`, and `redis-cluster`.

- _Frontier_: Communication component between microservices and edge data planes.
- _Frontlas_: Named Frontier Atlas, a cluster management component that records metadata and active information of microservices and edges in Redis.

Frontier needs to proactively connect to Frontlas to report its own, microservice, and edge active and status. The default ports for Frontlas are:

- `:40011` for microservices connection, replacing the 30011 port in a single Frontier instance.
- `:40012` for Frontier connection to report status.

You can deploy any number of Frontier instances as needed, and for Frontlas, deploying two instances separately can ensure HA (High Availability) since it does not store state and has no consistency issues.

### Configuration

**Frontier**'s `frontier.yaml` needs to add the following configuration:

```yaml
frontlas:
  enable: true
  dial:
    network: tcp
    addr:
      - 127.0.0.1:40012
  metrics:
    enable: false
    interval: 0
daemon:
  # Unique ID within the Frontier cluster
  frontier_id: frontier01
```

Frontier needs to connect to Frontlas to report its own, microservice, and edge active and status.

**Frontlas**'s `frontlas.yaml` minimal configuration:

```yaml
control_plane:
  listen:
    # Microservices connect to this address to discover edges in the cluster
    network: tcp
    addr: 0.0.0.0:40011
frontier_plane:
  # Frontier connects to this address
  listen:
    network: tcp
    addr: 0.0.0.0:40012
  expiration:
    # Expiration time for microservice metadata in Redis
    service_meta: 30
    # Expiration time for edge metadata in Redis
    edge_meta: 30
redis:
  # Support for standalone, sentinel, and cluster connections
  mode: standalone
  standalone:
    network: tcp
    addr: redis:6379
    db: 0
```

### Usage

Since Frontlas is used to discover available Frontiers, microservices need to adjust as follows:

**Microservice Getting Service**

```golang
package main

import (
  "net"
  "github.com/singchia/frontier/api/dataplane/v1/service"
)

func main() {
  // Use NewClusterService to get Service
  svc, err := service.NewClusterService("127.0.0.1:40011")
  // Start using service, everything else remains unchanged
}
```

**Edge Node Getting Connection Address**

For edge nodes, they still connect to Frontier but can get available Frontier addresses from Frontlas. Frontlas provides an interface to list Frontier instances:

```bash
curl -X GET http://127.0.0.1:40011/cluster/v1/frontiers
```

You can wrap this interface to provide load balancing or high availability for edge nodes, or add mTLS to directly provide to edge nodes (not recommended).

Control Plane gRPC See [Protobuf Definition](./api/controlplane/frontlas/v1/cluster.proto).

The Frontlas control plane differs from Frontier as it is a cluster-oriented control plane, currently providing only read interfaces for the cluster.

```protobuf
service ClusterService {
    rpc GetFrontierByEdge(GetFrontierByEdgeIDRequest) returns (GetFrontierByEdgeIDResponse);
    rpc ListFrontiers(ListFrontiersRequest) returns (ListFrontiersResponse);

    rpc ListEdges(ListEdgesRequest) returns (ListEdgesResponse);
    rpc GetEdgeByID(GetEdgeByIDRequest) returns (GetEdgeByIDResponse);
    rpc GetEdgesCount(GetEdgesCountRequest) returns (GetEdgesCountResponse);

    rpc ListServices(ListServicesRequest) returns (ListServicesResponse);
    rpc GetServiceByID(GetServiceByIDRequest) returns (GetServiceByIDResponse);
    rpc GetServicesCount(GetServicesCountRequest) returns (GetServicesCountResponse);
}
```

## Kubernetes

### Operator

**Install CRD and Operator**

Follow these steps to install and deploy the Operator to your .kubeconfig environment:

```bash
git clone https://github.com/singchia/frontier.git
cd dist/crd
kubectl apply -f install.yaml
```

Check CRD:

```bash
kubectl get crd frontierclusters.frontier.singchia.io
```

Check Operator:

```bash
kubectl get all -n frontier-system
```

**FrontierCluster**

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
    # Single instance Frontier
    replicas: 2
    # Microservice side port
    servicebound:
      port: 30011
    # Edge node side port
    edgebound:
      port: 30012
  frontlas:
    # Single instance Frontlas
    replicas: 1
    # Control plane port
    controlplane:
      port: 40011
    redis:
      # Dependent Redis configuration
      addrs:
        - rfs-redisfailover:26379
      password: your-password
      masterName: mymaster
      redisType: sentinel
```

Save as`frontiercluster.yaml`，and

```
kubectl apply -f frontiercluster.yaml
```

In 1 minute, you will have a 2-instance Frontier + 1-instance Frontlas cluster.

Check resource deployment status with:

```bash
kubectl get all -l app=frontiercluster-frontier  
kubectl get all -l app=frontiercluster-frontlas
```

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

Your microservice should connect to `service/frontiercluster-frontlas-svc:40011`, and your edge node can connect to the NodePort where `:30012` is located.

## Development

### Roadmap

See [ROADMAP](./ROADMAP.md)

### Contributions

If you find any bugs, please open an issue, and project maintainers will respond promptly.

If you wish to submit features or more quickly address project issues, you are welcome to submit PRs under these simple conditions:

- Code style remains consistent
- Each submission includes one feature
- Submitted code includes unit tests

## Testing

### Stream Function

<img src="./docs/diagram/stream.png" width="100%">

## Community

<p align=center>
<img src="./docs/diagram/wechat.JPG" width="30%">
</p>

Join our WeChat group for discussions and support.

## License

 Released under the [Apache License 2.0](https://github.com/singchia/geminio/blob/main/LICENSE)

---
A Star ⭐️ would be greatly appreciated ♥️
