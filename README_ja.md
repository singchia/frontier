<p align=center>
<img src="./docs/diagram/frontier-logo.png" width="30%">
</p>

<div align="center">

[![Go](https://github.com/singchia/frontier/actions/workflows/go.yml/badge.svg)](https://github.com/singchia/frontier/actions/workflows/go.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/singchia/frontier)](https://goreportcard.com/report/github.com/singchia/frontier)
[![Go Reference](https://pkg.go.dev/badge/badge/github.com/singchia/frontier.svg)](https://pkg.go.dev/github.com/singchia/frontier/api/dataplane/v1/service)
[![License](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)

[English](./README.md) | [简体中文](./README_zh.md) | 日本語 | [한국어](./README_ko.md) | [Español](./README_es.md) | [Français](./README_fr.md) | [Deutsch](./README_de.md)

</div>


Frontier は Go で書かれた**全二重**のオープンソース長時間接続ゲートウェイです。マイクロサービスからエッジノードやクライアントへの直接通信、およびその逆方向の通信を可能にします。全二重の**双方向 RPC**、**メッセージング**、**ポイントツーポイントストリーム**を提供します。Frontier は**クラウドネイティブ**アーキテクチャの原則に従い、Operator による高速なクラスタデプロイをサポートし、**高可用性**と数百万のオンラインエッジノード／クライアントへの**弾性スケーリング**を実現します。

## 目次

- [特徴](#特徴)
- [クイックスタート](#クイックスタート)
- [アーキテクチャ](#アーキテクチャ)
- [使い方](#使い方)
- [設定](#設定)
- [デプロイ](#デプロイ)
- [クラスタ](#クラスタ)
- [Kubernetes](#kubernetes)
- [開発](#開発)
- [テスト](#テスト)
- [コミュニティ](#コミュニティ)
- [ライセンス](#ライセンス)

## クイックスタート

1. 単一の Frontier インスタンスを起動:

```bash
docker run -d --name frontier -p 30011:30011 -p 30012:30012 singchia/frontier:1.1.0
```

2. サンプルをビルドして実行:

```bash
make examples
```

チャットルームのサンプルを実行:

```bash
# ターミナル 1
./bin/chatroom_service

# ターミナル 2
./bin/chatroom_agent
```

デモ動画:

https://github.com/singchia/frontier/assets/15531166/18b01d96-e30b-450f-9610-917d65259c30

## 特徴

- **双方向 RPC**: サービスとエッジが互いに呼び出し可能、ロードバランシング対応。
- **メッセージング**: サービス、エッジ、外部 MQ 間でのトピックベースのパブリッシュ／受信。
- **ポイントツーポイントストリーム**: プロキシ、ファイル転送、カスタムトラフィックのための直接ストリームを開始。
- **クラウドネイティブデプロイ**: Docker、Compose、Helm、Operator での実行が可能。
- **高可用性とスケーリング**: 再接続、クラスタリング、Frontlas による水平スケーリングをサポート。
- **認証とプレゼンス**: エッジ認証およびオンライン／オフライン通知。
- **コントロールプレーン API**: オンラインノードの問い合わせと管理のための gRPC および REST API。


## アーキテクチャ

**Frontier コンポーネント**

<img src="./docs/diagram/frontier.png" width="100%">

- _Service End_: マイクロサービス機能のエントリポイント、デフォルトで接続します。
- _Edge End_: エッジノードまたはクライアント機能のエントリポイント。
- _Publish/Receive_: メッセージのパブリッシュと受信。
- _Call/Register_: 関数の呼び出しと登録。
- _OpenStream/AcceptStream_: ポイントツーポイントストリーム（接続）の開始と受付。
- _External MQ_: Frontier は設定に基づき、エッジノードから発行されたメッセージを外部 MQ トピックへ転送できます。


Frontier はマイクロサービスとエッジノードの双方が能動的に Frontier へ接続する必要があります。接続時に Service と Edge のメタデータ（受信トピック、RPC、サービス名など）を運ぶことができます。デフォルトの接続ポートは以下の通りです:

- :30011: マイクロサービスが Service を取得するために接続するポート。
- :30012: エッジノードが Edge を取得するために接続するポート。
- :30010: 運用担当者またはプログラムがコントロールプレーンを利用するポート。


### 機能

<table><thead>
  <tr>
    <th>機能</th>
    <th>発信側</th>
    <th>受信側</th>
    <th>メソッド</th>
    <th>ルーティング方式</th>
    <th>説明</th>
  </tr></thead>
<tbody>
  <tr>
    <td rowspan="2">Messager</td>
    <td>Service</td>
    <td>Edge</td>
    <td>Publish</td>
    <td>EdgeID+Topic</td>
    <td>特定の EdgeID にパブリッシュする必要があり、デフォルトのトピックは空です。エッジは Receive を呼び出してメッセージを受信し、処理後に msg.Done() または msg.Error(err) を呼び出してメッセージの整合性を確保する必要があります。</td>
  </tr>
  <tr>
    <td>Edge</td>
    <td>Service または External MQ</td>
    <td>Publish</td>
    <td>Topic</td>
    <td>トピックにパブリッシュする必要があり、Frontier はトピックに基づいて特定の Service または MQ を選択します。</td>
  </tr>
  <tr>
    <td rowspan="2">RPCer</td>
    <td>Service</td>
    <td>Edge</td>
    <td>Call</td>
    <td>EdgeID+Method</td>
    <td>特定の EdgeID を呼び出す必要があり、メソッド名を伴います。</td>
  </tr>
  <tr>
    <td>Edge</td>
    <td>Service</td>
    <td>Call</td>
    <td>Method</td>
    <td>メソッドを呼び出す必要があり、Frontier はメソッド名に基づいて特定の Service を選択します。</td>
  </tr>
  <tr>
    <td rowspan="2">Multiplexer</td>
    <td>Service</td>
    <td>Edge</td>
    <td>OpenStream</td>
    <td>EdgeID</td>
    <td>特定の EdgeID に対してストリームを開く必要があります。</td>
  </tr>
  <tr>
    <td>Edge</td>
    <td>Service</td>
    <td>OpenStream</td>
    <td>ServiceName</td>
    <td>ServiceName に対してストリームを開く必要があり、Service 初期化時に service.OptionServiceName で指定します。</td>
  </tr>
</tbody></table>

**主要な設計原則**:

1. すべてのメッセージ、RPC、ストリームはポイントツーポイント伝送です。
	- マイクロサービスからエッジへは、エッジノード ID を指定する必要があります。
	- エッジからマイクロサービスへは、Frontier が Topic と Method に基づいてルーティングし、最終的にハッシュにより特定のマイクロサービスまたは外部 MQ を選択します。デフォルトは edgeid に基づくハッシュですが、random や srcip を選択できます。
2. メッセージは受信側による明示的な確認が必要です。
	- メッセージ配信のセマンティクスを保証するため、受信側は msg.Done() または msg.Error(err) を呼び出して配信の整合性を確保する必要があります。
3. Multiplexer によって開かれたストリームは、論理的にマイクロサービスとエッジノード間の直接通信を表します。
	- 相手側がストリームを受信すると、このストリーム上のすべての機能は Frontier のルーティングポリシーをバイパスして直接相手側に到達します。

## 使い方

詳細な使い方ガイド: [docs/USAGE.md](./docs/USAGE.md)

## 設定

詳細な設定ガイド: [docs/CONFIGURATION.md](./docs/CONFIGURATION.md)

## デプロイ

単一の Frontier インスタンスでは、環境に応じて以下の方法から選んでインスタンスをデプロイできます。

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

Kubernetes 環境の場合、Helm を使用して素早くインスタンスをデプロイできます。

```bash
git clone https://github.com/singchia/frontier.git
cd dist/helm
helm install frontier ./ -f values.yaml
```

マイクロサービスは ```service/frontier-servicebound-svc:30011``` に接続し、エッジノードは `:30012` の NodePort に接続できます。

### Systemd

Systemd 専用ドキュメントを参照:

[dist/systemd/README.md](./dist/systemd/README.md)

### Operator

後述のクラスタデプロイセクションを参照してください。

## クラスタ

### Frontier + Frontlas アーキテクチャ

<img src="./docs/diagram/frontlas.png" width="100%">

追加の Frontlas コンポーネントはクラスタの構築に使用されます。Frontlas もステートレスなコンポーネントで、情報をメモリに保存しないため、Redis への追加依存が必要です。Frontlas に Redis 接続情報を提供する必要があり、`redis`、`sentinel`、`redis-cluster` をサポートします。

- _Frontier_: マイクロサービスとエッジデータプレーン間の通信コンポーネント。
- _Frontlas_: Frontier Atlas の略で、マイクロサービスとエッジのメタデータおよびアクティブ情報を Redis に記録するクラスタ管理コンポーネント。

Frontier は能動的に Frontlas に接続し、自身、マイクロサービス、エッジのアクティブ状態を報告する必要があります。Frontlas のデフォルトポート:

- `:40011` マイクロサービス接続用、単一 Frontier インスタンスの 30011 ポートを置き換えます。
- `:40012` Frontier がステータス報告のために接続するポート。

必要に応じて任意の数の Frontier インスタンスをデプロイでき、Frontlas は状態を保持せず一貫性の問題がないため、2 インスタンスを別々にデプロイすることで HA（高可用性）を保証できます。

### 設定

**Frontier** の `frontier.yaml` に以下の設定を追加する必要があります:

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
  # Frontier クラスタ内で一意の ID
  frontier_id: frontier01
```

Frontier は Frontlas に接続して、自身、マイクロサービス、エッジのアクティブ状態を報告する必要があります。

**Frontlas** の `frontlas.yaml` の最小構成:

```yaml
control_plane:
  listen:
    # マイクロサービスはこのアドレスに接続してクラスタ内のエッジを発見します
    network: tcp
    addr: 0.0.0.0:40011
frontier_plane:
  # Frontier はこのアドレスに接続します
  listen:
    network: tcp
    addr: 0.0.0.0:40012
  expiration:
    # Redis 内のマイクロサービスメタデータの有効期限
    service_meta: 30
    # Redis 内のエッジメタデータの有効期限
    edge_meta: 30
redis:
  # standalone、sentinel、cluster 接続をサポート
  mode: standalone
  standalone:
    network: tcp
    addr: redis:6379
    db: 0
```

### 使い方

Frontlas は利用可能な Frontier を発見するために使用されるため、マイクロサービスは次のように調整する必要があります:

**マイクロサービスが Service を取得**

```golang
package main

import (
  "net"
  "github.com/singchia/frontier/api/dataplane/v1/service"
)

func main() {
  // NewClusterService を使用して Service を取得
  svc, err := service.NewClusterService("127.0.0.1:40011")
  // service の使用開始、その他はすべて変更なし
}
```

**エッジノードの接続アドレス取得**

エッジノードは引き続き Frontier に接続しますが、Frontlas から利用可能な Frontier のアドレスを取得できます。Frontlas は Frontier インスタンスを一覧取得するインターフェースを提供します:

```bash
curl -X GET http://127.0.0.1:40011/cluster/v1/frontiers
```

このインターフェースをラップしてエッジノードにロードバランシングや高可用性を提供したり、mTLS を追加してエッジノードに直接提供したり（非推奨）できます。

コントロールプレーン gRPC は [Protobuf 定義](./api/controlplane/frontlas/v1/cluster.proto) を参照。

Frontlas のコントロールプレーンは Frontier のそれとは異なり、クラスタ指向のコントロールプレーンであり、現在はクラスタの読み取りインターフェースのみを提供します。

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

**CRD と Operator のインストール**

次の手順で Operator を .kubeconfig 環境にインストール／デプロイします:

```bash
git clone https://github.com/singchia/frontier.git
cd dist/crd
kubectl apply -f install.yaml
```

CRD を確認:

```bash
kubectl get crd frontierclusters.frontier.singchia.io
```

Operator を確認:

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
    # 単一インスタンス Frontier
    replicas: 2
    # マイクロサービス側ポート
    servicebound:
      port: 30011
    # エッジノード側ポート
    edgebound:
      port: 30012
  frontlas:
    # 単一インスタンス Frontlas
    replicas: 1
    # コントロールプレーンポート
    controlplane:
      port: 40011
    redis:
      # 依存する Redis の構成
      addrs:
        - rfs-redisfailover:26379
      password: your-password
      masterName: mymaster
      redisType: sentinel
```

`frontiercluster.yaml` として保存し、

```
kubectl apply -f frontiercluster.yaml
```

1 分以内に、2 インスタンスの Frontier ＋ 1 インスタンスの Frontlas クラスタが得られます。

次のコマンドでリソースのデプロイ状況を確認:

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

マイクロサービスは `service/frontiercluster-frontlas-svc:40011` に接続し、エッジノードは `:30012` の NodePort に接続できます。

## 開発

### ロードマップ

[ROADMAP](./ROADMAP.md) を参照。

### コントリビューション

バグを発見した場合は issue を開いてください。プロジェクトメンテナが速やかに対応します。

機能の提出や、プロジェクト課題のより迅速な解決をご希望の場合は、次のシンプルな条件のもとで PR を提出いただけます:

- コードスタイルの一貫性を保つこと
- 1 回の提出につき 1 つの機能とすること
- 提出するコードにユニットテストを含めること

## テスト

### Stream 機能

<img src="./docs/diagram/stream.png" width="100%">

## コミュニティ

<p align=center>
<img src="./docs/diagram/wechat.JPG" width="30%">
</p>

議論とサポートのために WeChat グループへご参加ください。

## ライセンス

 [Apache License 2.0](https://github.com/singchia/geminio/blob/main/LICENSE) の下で公開されています。

---
Star ⭐️ をいただけると大変嬉しいです ♥️
