<p align=center>
<img src="./docs/diagram/frontier-logo.png" width="30%">
</p>

<div align="center">

[![Go](https://github.com/singchia/frontier/actions/workflows/go.yml/badge.svg)](https://github.com/singchia/frontier/actions/workflows/go.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/singchia/frontier)](https://goreportcard.com/report/github.com/singchia/frontier)
[![Go Reference](https://pkg.go.dev/badge/badge/github.com/singchia/frontier.svg)](https://pkg.go.dev/github.com/singchia/frontier/api/dataplane/v1/service)
[![License](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)

[English](./README.md) | [简体中文](./README_zh.md) | [日本語](./README_ja.md) | 한국어 | [Español](./README_es.md) | [Français](./README_fr.md) | [Deutsch](./README_de.md)

</div>


Frontier는 Go로 작성된 **전이중(full-duplex)** 오픈소스 장기 연결 게이트웨이입니다. 마이크로서비스가 엣지 노드나 클라이언트에 직접 도달할 수 있도록 하고, 그 반대 방향도 지원합니다. 전이중 **양방향 RPC**, **메시징**, **점대점 스트림**을 제공합니다. Frontier는 **클라우드 네이티브** 아키텍처 원칙을 따르며, Operator를 통한 빠른 클러스터 배포를 지원하고, **고가용성**과 수백만 온라인 엣지 노드/클라이언트에 대한 **탄력적 확장**을 위해 설계되었습니다.

## 목차

- [기능](#기능)
- [빠른 시작](#빠른-시작)
- [아키텍처](#아키텍처)
- [사용법](#사용법)
- [구성](#구성)
- [배포](#배포)
- [클러스터](#클러스터)
- [Kubernetes](#kubernetes)
- [개발](#개발)
- [테스트](#테스트)
- [커뮤니티](#커뮤니티)
- [라이선스](#라이선스)

## 빠른 시작

1. 단일 Frontier 인스턴스 실행:

```bash
docker run -d --name frontier -p 30011:30011 -p 30012:30012 singchia/frontier:1.1.0
```

2. 예제 빌드 및 실행:

```bash
make examples
```

채팅방 예제 실행:

```bash
# 터미널 1
./bin/chatroom_service

# 터미널 2
./bin/chatroom_agent
```

데모 영상:

https://github.com/singchia/frontier/assets/15531166/18b01d96-e30b-450f-9610-917d65259c30

## 기능

- **양방향 RPC**: 서비스와 엣지가 로드 밸런싱과 함께 서로 호출할 수 있습니다.
- **메시징**: 서비스, 엣지, 외부 MQ 간의 토픽 기반 게시/수신.
- **점대점 스트림**: 프록시, 파일 전송, 사용자 정의 트래픽을 위한 직접 스트림 개설.
- **클라우드 네이티브 배포**: Docker, Compose, Helm 또는 Operator를 통한 실행.
- **고가용성 및 확장**: 재연결, 클러스터링, Frontlas를 통한 수평 확장 지원.
- **인증 및 프레즌스**: 엣지 인증과 온라인/오프라인 알림.
- **컨트롤 플레인 API**: 온라인 노드 조회 및 관리를 위한 gRPC 및 REST API.


## 아키텍처

**Frontier 컴포넌트**

<img src="./docs/diagram/frontier.png" width="100%">

- _Service End_: 마이크로서비스 기능의 진입점, 기본적으로 연결됩니다.
- _Edge End_: 엣지 노드 또는 클라이언트 기능의 진입점.
- _Publish/Receive_: 메시지 게시 및 수신.
- _Call/Register_: 함수 호출 및 등록.
- _OpenStream/AcceptStream_: 점대점 스트림(연결)의 개설 및 수락.
- _External MQ_: Frontier는 구성에 따라 엣지 노드에서 게시된 메시지를 외부 MQ 토픽으로 전달할 수 있습니다.


Frontier는 마이크로서비스와 엣지 노드 모두 능동적으로 Frontier에 연결해야 합니다. 연결 시 Service와 Edge의 메타데이터(수신 토픽, RPC, 서비스 이름 등)를 실어 보낼 수 있습니다. 기본 연결 포트는 다음과 같습니다:

- :30011: 마이크로서비스가 Service를 얻기 위해 연결.
- :30012: 엣지 노드가 Edge를 얻기 위해 연결.
- :30010: 운영자 또는 프로그램이 컨트롤 플레인을 사용하기 위해 연결.


### 기능 상세

<table><thead>
  <tr>
    <th>기능</th>
    <th>개시자</th>
    <th>수신자</th>
    <th>메서드</th>
    <th>라우팅 방식</th>
    <th>설명</th>
  </tr></thead>
<tbody>
  <tr>
    <td rowspan="2">Messager</td>
    <td>Service</td>
    <td>Edge</td>
    <td>Publish</td>
    <td>EdgeID+Topic</td>
    <td>특정 EdgeID로 게시해야 하며, 기본 토픽은 비어 있습니다. 엣지는 Receive를 호출해 메시지를 받고, 처리 후 메시지 일관성을 보장하기 위해 msg.Done() 또는 msg.Error(err)를 반드시 호출해야 합니다.</td>
  </tr>
  <tr>
    <td>Edge</td>
    <td>Service 또는 External MQ</td>
    <td>Publish</td>
    <td>Topic</td>
    <td>토픽으로 게시해야 하며, Frontier가 토픽을 기반으로 특정 Service 또는 MQ를 선택합니다.</td>
  </tr>
  <tr>
    <td rowspan="2">RPCer</td>
    <td>Service</td>
    <td>Edge</td>
    <td>Call</td>
    <td>EdgeID+Method</td>
    <td>특정 EdgeID를 메서드 이름과 함께 호출해야 합니다.</td>
  </tr>
  <tr>
    <td>Edge</td>
    <td>Service</td>
    <td>Call</td>
    <td>Method</td>
    <td>메서드를 호출해야 하며, Frontier가 메서드 이름을 기반으로 특정 Service를 선택합니다.</td>
  </tr>
  <tr>
    <td rowspan="2">Multiplexer</td>
    <td>Service</td>
    <td>Edge</td>
    <td>OpenStream</td>
    <td>EdgeID</td>
    <td>특정 EdgeID로 스트림을 개설해야 합니다.</td>
  </tr>
  <tr>
    <td>Edge</td>
    <td>Service</td>
    <td>OpenStream</td>
    <td>ServiceName</td>
    <td>ServiceName으로 스트림을 개설해야 하며, Service 초기화 시 service.OptionServiceName으로 지정합니다.</td>
  </tr>
</tbody></table>

**주요 설계 원칙**:

1. 모든 메시지, RPC, 스트림은 점대점 전송입니다.
	- 마이크로서비스에서 엣지로 보낼 때는 엣지 노드 ID를 지정해야 합니다.
	- 엣지에서 마이크로서비스로는 Frontier가 Topic과 Method를 기반으로 라우팅하며, 최종적으로 해싱을 통해 마이크로서비스나 외부 MQ를 선택합니다. 기본값은 edgeid 기반 해싱이지만 random 또는 srcip를 선택할 수 있습니다.
2. 메시지는 수신자의 명시적 확인이 필요합니다.
	- 메시지 전달 시맨틱을 보장하기 위해 수신자는 msg.Done() 또는 msg.Error(err)를 호출해 전달 일관성을 보장해야 합니다.
3. Multiplexer가 개설한 스트림은 논리적으로 마이크로서비스와 엣지 노드 간의 직접 통신을 나타냅니다.
	- 상대방이 스트림을 받으면, 이 스트림의 모든 기능은 Frontier의 라우팅 정책을 우회하여 상대방에게 직접 도달합니다.

## 사용법

상세 사용 가이드: [docs/USAGE.md](./docs/USAGE.md)

## 구성

상세 구성 가이드: [docs/CONFIGURATION.md](./docs/CONFIGURATION.md)

## 배포

단일 Frontier 인스턴스의 경우, 환경에 따라 다음 방법 중 하나로 Frontier 인스턴스를 배포할 수 있습니다.

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

Kubernetes 환경에서는 Helm으로 빠르게 인스턴스를 배포할 수 있습니다.

```bash
git clone https://github.com/singchia/frontier.git
cd dist/helm
helm install frontier ./ -f values.yaml
```

마이크로서비스는 ```service/frontier-servicebound-svc:30011```에 연결해야 하고, 엣지 노드는 `:30012`가 위치한 NodePort에 연결할 수 있습니다.

### Systemd

전용 Systemd 문서 참고:

[dist/systemd/README.md](./dist/systemd/README.md)

### Operator

아래 클러스터 배포 섹션을 참조하세요.

## 클러스터

### Frontier + Frontlas 아키텍처

<img src="./docs/diagram/frontlas.png" width="100%">

추가 Frontlas 컴포넌트는 클러스터 구성에 사용됩니다. Frontlas도 스테이트리스 컴포넌트이며 메모리에 기타 정보를 저장하지 않기 때문에 Redis에 대한 추가 의존성이 필요합니다. Frontlas에 Redis 연결 정보를 제공해야 하며, `redis`, `sentinel`, `redis-cluster`를 지원합니다.

- _Frontier_: 마이크로서비스와 엣지 데이터 플레인 간 통신 컴포넌트.
- _Frontlas_: Frontier Atlas의 약자로, 마이크로서비스와 엣지의 메타데이터 및 활성 정보를 Redis에 기록하는 클러스터 관리 컴포넌트.

Frontier는 Frontlas에 능동적으로 연결하여 자신, 마이크로서비스, 엣지의 활성 상태를 보고해야 합니다. Frontlas의 기본 포트:

- `:40011` 마이크로서비스 연결용, 단일 Frontier 인스턴스의 30011 포트를 대체합니다.
- `:40012` Frontier가 상태 보고를 위해 연결.

필요한 만큼 Frontier 인스턴스를 배포할 수 있으며, Frontlas의 경우 상태를 저장하지 않고 일관성 문제가 없기 때문에 두 인스턴스를 별도로 배포하면 HA(고가용성)를 보장할 수 있습니다.

### 구성

**Frontier**의 `frontier.yaml`에 다음 구성을 추가해야 합니다:

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
  # Frontier 클러스터 내에서 고유한 ID
  frontier_id: frontier01
```

Frontier는 Frontlas에 연결하여 자신, 마이크로서비스, 엣지의 활성 상태를 보고해야 합니다.

**Frontlas**의 `frontlas.yaml` 최소 구성:

```yaml
control_plane:
  listen:
    # 마이크로서비스는 이 주소에 연결해 클러스터 내 엣지를 발견합니다
    network: tcp
    addr: 0.0.0.0:40011
frontier_plane:
  # Frontier가 이 주소에 연결합니다
  listen:
    network: tcp
    addr: 0.0.0.0:40012
  expiration:
    # Redis 내 마이크로서비스 메타데이터 만료 시간
    service_meta: 30
    # Redis 내 엣지 메타데이터 만료 시간
    edge_meta: 30
redis:
  # standalone, sentinel, cluster 연결 지원
  mode: standalone
  standalone:
    network: tcp
    addr: redis:6379
    db: 0
```

### 사용법

Frontlas가 사용 가능한 Frontier를 발견하는 데 사용되므로, 마이크로서비스는 다음과 같이 조정해야 합니다:

**마이크로서비스가 Service 획득**

```golang
package main

import (
  "net"
  "github.com/singchia/frontier/api/dataplane/v1/service"
)

func main() {
  // NewClusterService로 Service 획득
  svc, err := service.NewClusterService("127.0.0.1:40011")
  // service 사용 시작, 나머지는 그대로
}
```

**엣지 노드의 연결 주소 획득**

엣지 노드는 여전히 Frontier에 연결하지만, Frontlas에서 사용 가능한 Frontier 주소를 얻을 수 있습니다. Frontlas는 Frontier 인스턴스를 나열하는 인터페이스를 제공합니다:

```bash
curl -X GET http://127.0.0.1:40011/cluster/v1/frontiers
```

이 인터페이스를 래핑해 엣지 노드에 로드 밸런싱 또는 고가용성을 제공하거나, mTLS를 추가해 엣지 노드에 직접 제공할 수 있습니다(권장하지 않음).

컨트롤 플레인 gRPC는 [Protobuf 정의](./api/controlplane/frontlas/v1/cluster.proto) 참조.

Frontlas 컨트롤 플레인은 Frontier와 다르게 클러스터 지향 컨트롤 플레인이며, 현재는 클러스터에 대한 읽기 인터페이스만 제공합니다.

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

**CRD 및 Operator 설치**

다음 단계에 따라 .kubeconfig 환경에 Operator를 설치 및 배포합니다:

```bash
git clone https://github.com/singchia/frontier.git
cd dist/crd
kubectl apply -f install.yaml
```

CRD 확인:

```bash
kubectl get crd frontierclusters.frontier.singchia.io
```

Operator 확인:

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
    # 단일 인스턴스 Frontier
    replicas: 2
    # 마이크로서비스 측 포트
    servicebound:
      port: 30011
    # 엣지 노드 측 포트
    edgebound:
      port: 30012
  frontlas:
    # 단일 인스턴스 Frontlas
    replicas: 1
    # 컨트롤 플레인 포트
    controlplane:
      port: 40011
    redis:
      # 의존하는 Redis 구성
      addrs:
        - rfs-redisfailover:26379
      password: your-password
      masterName: mymaster
      redisType: sentinel
```

`frontiercluster.yaml`로 저장하고,

```
kubectl apply -f frontiercluster.yaml
```

1분 이내에 2 인스턴스 Frontier + 1 인스턴스 Frontlas 클러스터가 준비됩니다.

다음 명령으로 리소스 배포 상태를 확인:

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

마이크로서비스는 `service/frontiercluster-frontlas-svc:40011`에 연결하고, 엣지 노드는 `:30012`가 위치한 NodePort에 연결할 수 있습니다.

## 개발

### 로드맵

[ROADMAP](./ROADMAP.md) 참조.

### 기여

버그를 발견하면 issue를 열어주세요. 프로젝트 메인테이너가 신속히 응답합니다.

기능을 제출하거나 프로젝트 이슈를 더 빠르게 해결하고 싶다면, 다음의 간단한 조건에 따라 PR을 환영합니다:

- 코드 스타일 일관성 유지
- 각 제출은 하나의 기능 포함
- 제출한 코드에 단위 테스트 포함

## 테스트

### Stream 기능

<img src="./docs/diagram/stream.png" width="100%">

## 커뮤니티

<p align=center>
<img src="./docs/diagram/wechat.JPG" width="30%">
</p>

토론과 지원을 위해 WeChat 그룹에 참여하세요.

## 라이선스

 [Apache License 2.0](https://github.com/singchia/geminio/blob/main/LICENSE)에 따라 배포됩니다.

---
Star ⭐️ 부탁드립니다 ♥️
