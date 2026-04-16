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

> Bidirectional service-to-edge gateway for long-lived connections

Frontier is an open-source gateway written in Go for **service <-> edge communication**. It lets backend services and edge nodes talk to each other over long-lived connections, with built-in **bidirectional RPC**, **messaging**, and **point-to-point streams**.

It is built for systems where both sides stay online and need to actively call, notify, or open streams to each other. Frontier is **not a reverse proxy** and **not just a message broker**. It is infrastructure for addressing and operating large fleets of connected edge nodes from backend services.

<p align="center">
  <img src="./docs/diagram/frontier.png" width="100%" alt="Frontier architecture overview">
</p>

## What Makes Frontier Different

- **Different from API gateways**: Frontier is designed for backend-to-edge communication, not just north-south HTTP traffic.
- **Different from MQ**: It gives you bidirectional RPC, messaging, and streams in one connectivity model.
- **Different from tunnels**: Services can address a specific online edge node instead of only exposing a port.
- **Made for real fleets**: Works for devices, agents, clients, and remote connectors that stay online for a long time.

## What You Can Build

<table>
  <tr>
    <td width="50%">
      <img src="./docs/diagram/rtmp.png" alt="Frontier stream relay example">
      <p><strong>Traffic relay and media streaming</strong><br>Open point-to-point streams for RTMP relay, file transfer, proxy traffic, or other custom protocols.</p>
    </td>
    <td width="50%">
      <img src="./docs/diagram/stream.png" alt="Frontier stream architecture">
      <p><strong>Remote agents and device fleets</strong><br>Keep edge nodes online, route service calls to a specific edge, and let the edge call backend services back.</p>
    </td>
  </tr>
</table>

## At A Glance

| You need to... | Frontier gives you... |
| --- | --- |
| Call a specific online device or agent from your backend | Service -> Edge RPC and messaging over long-lived connections |
| Let edge nodes initiate calls without opening inbound ports | Edge -> Service RPC on the same connection model |
| Move bytes, not just request/response payloads | Point-to-point streams between service and edge |
| Run one control plane for a large connected fleet | Presence, lifecycle hooks, control APIs, clustering |

## Table of Contents

- [Why Frontier](#why-frontier)
- [What Makes Frontier Different](#what-makes-frontier-different)
- [What You Can Build](#what-you-can-build)
- [At A Glance](#at-a-glance)
- [When to Use Frontier](#when-to-use-frontier)
- [Real-World Use Cases](#real-world-use-cases)
- [Comparison](#comparison)
- [Quick Start](#quick-start)
- [Docs](#docs)
- [Community](#community)
- [License](#license)

## Why Frontier

Most infrastructure is optimized for one of these models:

- **service -> service** via HTTP or gRPC
- **client -> service** via API gateways and reverse proxies
- **event fan-out** via message brokers

Frontier is optimized for a different model:

- **service <-> edge** over long-lived, stateful connections
- backend services calling a specific online edge node
- edge nodes calling backend services without exposing inbound ports
- opening direct streams between services and edge nodes when RPC is not enough

<p align="center">
  <img src="./docs/diagram/frontlas.png" width="88%" alt="Frontier clustering with Frontlas">
</p>

## When to Use Frontier

Use Frontier if you need:

- Backend services to call specific online devices, agents, clients, or connectors
- Edge nodes to call backend services over the same connectivity model
- Long-lived connections at large scale
- One data plane for RPC, messaging, and streams
- Cluster deployment and high availability for service-to-edge connectivity

Do not use Frontier if:

- You only need service-to-service RPC; gRPC is a simpler fit
- You only need HTTP ingress, routing, or proxying; use an API gateway or Envoy
- You only need pub/sub or event streaming; use NATS or Kafka
- You only need a generic tunnel; use frp or another tunneling tool

## Real-World Use Cases

- IoT and device fleets
- Remote agents and connectors
- IM and other real-time systems
- Game backends talking to online clients or edge nodes
- Zero-trust internal access based on connector-style agents
- File transfer, media relay, or traffic proxy over point-to-point streams

## Use Cases In One Screen

| Scenario | Why Frontier fits |
| --- | --- |
| Device control plane | Address a specific online edge node, push commands, receive state, and keep the link alive |
| Remote connector platform | Let connectors dial out, avoid inbound exposure, and keep service-side routing simple |
| Real-time apps | Maintain long-lived sessions and combine notifications, RPC, and streams in one path |
| Internal zero-trust access | Use agent-style edges as the last-mile bridge between backend systems and private resources |

## Comparison

| Capability | Frontier | gRPC | NATS | frp | Envoy |
| --- | --- | --- | --- | --- | --- |
| Built for service <-> edge communication | Yes | No | Partial | No | No |
| Backend can address a specific online edge node | Yes | No | Partial | Partial | No |
| Edge can call backend services | Yes | Partial | Yes | No | No |
| Point-to-point streams between service and edge | Yes | Partial | No | Tunnel only | No |
| Unified RPC + messaging + streams model | Yes | No | No | No | No |
| Long-lived connection fleet as a primary model | Yes | No | Partial | Partial | No |

`Partial` here means the capability can be assembled with extra patterns, but it is not the system's primary communication model.

## Quick Start

1. Start a single Frontier instance:

```bash
docker run -d --name frontier -p 30011:30011 -p 30012:30012 singchia/frontier:1.2.2
```

2. Build the examples:

```bash
make examples
```

3. Run the chatroom demo:

```bash
# Terminal 1
./bin/chatroom_service

# Terminal 2
./bin/chatroom_agent
```

The chatroom example shows the basic Frontier interaction model: long-lived connectivity, edge online/offline events, and service <-> edge messaging.

You can also run the RTMP example if you want to see Frontier's stream model used for traffic relay:

```bash
# Terminal 1
./bin/rtmp_service

# Terminal 2
./bin/rtmp_edge
```

Demo video:

https://github.com/singchia/frontier/assets/15531166/18b01d96-e30b-450f-9610-917d65259c30

## Docs

README is intentionally optimized for fast understanding and fast conversion. The implementation details, configuration surface, deployment playbooks, and cluster operations live in the docs.

Start here:

- [Usage Guide](./docs/USAGE.md)
- [Configuration Guide](./docs/CONFIGURATION.md)
- [Technical Docs Index](./docs/README.md)
- [Systemd Deployment](./dist/systemd/README.md)
- [Docker Compose Deployment](./dist/compose/README.md)
- [Helm and Operator Deployment](./dist/helm/README.md)
- [Roadmap](./ROADMAP.md)

If you are evaluating Frontier for production, the docs cover:

- architecture and communication model
- RPC, messaging, and point-to-point stream semantics
- deployment on Docker, Compose, Helm, and Operator
- cluster mode with Frontlas and Redis
- development workflow and contribution guidelines

## Community

<p align=center>
<img src="./docs/diagram/wechat.JPG" width="30%">
</p>

Join our WeChat group for discussions and support.

## License

 Released under the [Apache License 2.0](https://github.com/singchia/geminio/blob/main/LICENSE)

---
A Star ⭐️ would be greatly appreciated ♥️
