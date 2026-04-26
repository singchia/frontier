<p align=center>
<picture>
  <source media="(prefers-color-scheme: dark)" srcset="./docs/diagram/frontier-logo-dark.png">
  <img src="./docs/diagram/frontier-logo.png" width="30%">
</picture>
</p>

<div align="center">

[![Go](https://github.com/singchia/frontier/actions/workflows/go.yml/badge.svg)](https://github.com/singchia/frontier/actions/workflows/go.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/singchia/frontier)](https://goreportcard.com/report/github.com/singchia/frontier)
[![Go Reference](https://pkg.go.dev/badge/badge/github.com/singchia/frontier.svg)](https://pkg.go.dev/github.com/singchia/frontier/api/dataplane/v1/service)
[![License](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)

[English](./README.md) | [简体中文](./README_zh.md) | [日本語](./README_ja.md) | [한국어](./README_ko.md) | [Español](./README_es.md) | [Français](./README_fr.md) | Deutsch

</div>


Frontier ist ein in Go geschriebenes, **vollduplexfähiges** Open-Source-Gateway für Langzeitverbindungen. Es ermöglicht Microservices, Edge-Knoten oder Clients direkt zu erreichen – und umgekehrt. Es bietet vollduplexfähige **bidirektionale RPCs**, **Messaging** und **Punkt-zu-Punkt-Streams**. Frontier folgt den Prinzipien einer **Cloud-Native**-Architektur, unterstützt eine schnelle Cluster-Bereitstellung über den Operator und ist für **Hochverfügbarkeit** und **elastische Skalierung** auf Millionen aktiver Edge-Knoten oder Clients ausgelegt.

## Inhaltsverzeichnis

- [Funktionen](#funktionen)
- [Schnellstart](#schnellstart)
- [Architektur](#architektur)
- [Verwendung](#verwendung)
- [Konfiguration](#konfiguration)
- [Bereitstellung](#bereitstellung)
- [Cluster](#cluster)
- [Kubernetes](#kubernetes)
- [Entwicklung](#entwicklung)
- [Tests](#tests)
- [Community](#community)
- [Lizenz](#lizenz)

## Schnellstart

1. Eine einzelne Frontier-Instanz starten:

```bash
docker run -d --name frontier -p 30011:30011 -p 30012:30012 singchia/frontier:1.1.0
```

2. Beispiele bauen und ausführen:

```bash
make examples
```

Das Chatroom-Beispiel ausführen:

```bash
# Terminal 1
./bin/chatroom_service

# Terminal 2
./bin/chatroom_agent
```

Demovideo:

https://github.com/singchia/frontier/assets/15531166/18b01d96-e30b-450f-9610-917d65259c30

## Funktionen

- **Bidirektionale RPCs**: Services und Edges können sich gegenseitig aufrufen, inklusive Lastverteilung.
- **Messaging**: Topic-basiertes Publish/Receive zwischen Services, Edges und externem MQ.
- **Punkt-zu-Punkt-Streams**: direkte Streams für Proxying, Dateiübertragung und benutzerdefinierten Verkehr.
- **Cloud-Native-Bereitstellung**: Ausführung über Docker, Compose, Helm oder Operator.
- **Hochverfügbarkeit und Skalierung**: Unterstützung für Reconnect, Clustering und horizontale Skalierung mit Frontlas.
- **Authentifizierung und Präsenz**: Edge-Authentifizierung sowie Online-/Offline-Benachrichtigungen.
- **Control-Plane-APIs**: gRPC- und REST-APIs zum Abfragen und Verwalten aktiver Knoten.


## Architektur

**Frontier-Komponente**

<img src="./docs/diagram/frontier.png" width="100%">

- _Service End_: Einstiegspunkt für Microservice-Funktionen, standardmäßig verbunden.
- _Edge End_: Einstiegspunkt für Edge-Knoten- oder Client-Funktionen.
- _Publish/Receive_: Nachrichten veröffentlichen und empfangen.
- _Call/Register_: Funktionen aufrufen und registrieren.
- _OpenStream/AcceptStream_: Öffnen und Annehmen von Punkt-zu-Punkt-Streams (Verbindungen).
- _External MQ_: Frontier unterstützt gemäß Konfiguration die Weiterleitung von Nachrichten, die von Edge-Knoten veröffentlicht werden, an Topics eines externen MQ.


Frontier verlangt, dass sich sowohl Microservices als auch Edge-Knoten aktiv zu Frontier verbinden. Beim Verbindungsaufbau können Metadaten von Service und Edge (Empfangs-Topics, RPC, Service-Namen etc.) mitgegeben werden. Die Standardverbindungsports sind:

- :30011: Microservices verbinden sich darüber und erhalten Service.
- :30012: Edge-Knoten verbinden sich darüber und erhalten Edge.
- :30010: für Betriebsmitarbeiter oder Programme zur Nutzung der Control Plane.


### Funktionalität

<table><thead>
  <tr>
    <th>Funktion</th>
    <th>Initiator</th>
    <th>Empfänger</th>
    <th>Methode</th>
    <th>Routing-Verfahren</th>
    <th>Beschreibung</th>
  </tr></thead>
<tbody>
  <tr>
    <td rowspan="2">Messager</td>
    <td>Service</td>
    <td>Edge</td>
    <td>Publish</td>
    <td>EdgeID+Topic</td>
    <td>Muss an eine bestimmte EdgeID veröffentlicht werden, das Standard-Topic ist leer. Der Edge ruft Receive auf, um die Nachricht zu empfangen, und muss nach der Verarbeitung msg.Done() oder msg.Error(err) aufrufen, um die Nachrichtenkonsistenz sicherzustellen.</td>
  </tr>
  <tr>
    <td>Edge</td>
    <td>Service oder External MQ</td>
    <td>Publish</td>
    <td>Topic</td>
    <td>Muss an ein Topic veröffentlicht werden; Frontier wählt anhand des Topics einen konkreten Service oder MQ aus.</td>
  </tr>
  <tr>
    <td rowspan="2">RPCer</td>
    <td>Service</td>
    <td>Edge</td>
    <td>Call</td>
    <td>EdgeID+Method</td>
    <td>Muss eine bestimmte EdgeID aufrufen und dabei den Methodennamen mitführen.</td>
  </tr>
  <tr>
    <td>Edge</td>
    <td>Service</td>
    <td>Call</td>
    <td>Method</td>
    <td>Muss eine Methode aufrufen; Frontier wählt anhand des Methodennamens einen konkreten Service aus.</td>
  </tr>
  <tr>
    <td rowspan="2">Multiplexer</td>
    <td>Service</td>
    <td>Edge</td>
    <td>OpenStream</td>
    <td>EdgeID</td>
    <td>Muss einen Stream zu einer bestimmten EdgeID öffnen.</td>
  </tr>
  <tr>
    <td>Edge</td>
    <td>Service</td>
    <td>OpenStream</td>
    <td>ServiceName</td>
    <td>Muss einen Stream zu einem ServiceName öffnen, der bei der Service-Initialisierung über service.OptionServiceName angegeben wird.</td>
  </tr>
</tbody></table>

**Zentrale Designprinzipien**:

1. Alle Nachrichten, RPCs und Streams sind Punkt-zu-Punkt-Übertragungen.
	- Von Microservices zu Edges muss die Edge-Knoten-ID angegeben werden.
	- Von Edges zu Microservices routet Frontier anhand von Topic und Method und wählt schließlich per Hashing einen Microservice oder externen MQ aus. Standardmäßig wird auf Basis von edgeid gehasht, wahlweise auch random oder srcip.
2. Nachrichten erfordern eine explizite Bestätigung durch den Empfänger.
	- Zur Sicherstellung der Zustellsemantik muss der Empfänger msg.Done() oder msg.Error(err) aufrufen, um die Zustellkonsistenz zu gewährleisten.
3. Vom Multiplexer geöffnete Streams repräsentieren logisch eine direkte Kommunikation zwischen Microservices und Edge-Knoten.
	- Sobald die Gegenseite den Stream empfängt, erreichen alle Funktionen auf diesem Stream die Gegenseite direkt und umgehen die Routing-Richtlinien von Frontier.

## Verwendung

Ausführliche Nutzungsanleitung: [docs/USAGE.md](./docs/USAGE.md)

## Konfiguration

Ausführliche Konfigurationsanleitung: [docs/CONFIGURATION.md](./docs/CONFIGURATION.md)

## Bereitstellung

Bei einer einzelnen Frontier-Instanz können Sie Ihre Instanz je nach Umgebung mit den folgenden Methoden bereitstellen.

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

In einer Kubernetes-Umgebung können Sie mit Helm schnell eine Instanz bereitstellen.

```bash
git clone https://github.com/singchia/frontier.git
cd dist/helm
helm install frontier ./ -f values.yaml
```

Ihr Microservice sollte sich mit ```service/frontier-servicebound-svc:30011``` verbinden, und Ihr Edge-Knoten kann sich mit dem NodePort verbinden, auf dem `:30012` liegt.

### Systemd

Nutzen Sie die separate Systemd-Dokumentation:

[dist/systemd/README.md](./dist/systemd/README.md)

### Operator

Siehe den Abschnitt zur Cluster-Bereitstellung weiter unten.

## Cluster

### Frontier + Frontlas-Architektur

<img src="./docs/diagram/frontlas.png" width="100%">

Die zusätzliche Frontlas-Komponente dient zum Aufbau des Clusters. Frontlas ist ebenfalls zustandslos und speichert keine weiteren Informationen im Speicher, weshalb eine zusätzliche Abhängigkeit zu Redis erforderlich ist. Sie müssen Frontlas Redis-Verbindungsdaten bereitstellen; unterstützt werden `redis`, `sentinel` und `redis-cluster`.

- _Frontier_: Kommunikationskomponente zwischen Microservices und der Edge-Datenebene.
- _Frontlas_: kurz für Frontier Atlas, eine Cluster-Management-Komponente, die Metadaten und Aktivitätsinformationen von Microservices und Edges in Redis ablegt.

Frontier muss sich proaktiv mit Frontlas verbinden, um seinen eigenen Status sowie die Aktivität und den Status von Microservices und Edges zu melden. Die Standardports von Frontlas sind:

- `:40011` für Microservice-Verbindungen, ersetzt den Port 30011 einer einzelnen Frontier-Instanz.
- `:40012` für die Verbindung von Frontier zur Statusmeldung.

Sie können beliebig viele Frontier-Instanzen bereitstellen; für Frontlas genügt es, zwei Instanzen getrennt bereitzustellen, um HA (Hochverfügbarkeit) zu gewährleisten, da es keinen Zustand speichert und keine Konsistenzprobleme aufweist.

### Konfiguration

In der `frontier.yaml` von **Frontier** muss folgende Konfiguration ergänzt werden:

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
  # Innerhalb des Frontier-Clusters eindeutige ID
  frontier_id: frontier01
```

Frontier muss sich mit Frontlas verbinden, um seinen eigenen Status sowie den von Microservices und Edges zu melden.

Minimale Konfiguration der `frontlas.yaml` von **Frontlas**:

```yaml
control_plane:
  listen:
    # Microservices verbinden sich mit dieser Adresse, um Edges im Cluster zu entdecken
    network: tcp
    addr: 0.0.0.0:40011
frontier_plane:
  # Frontier verbindet sich mit dieser Adresse
  listen:
    network: tcp
    addr: 0.0.0.0:40012
  expiration:
    # Ablaufzeit der Microservice-Metadaten in Redis
    service_meta: 30
    # Ablaufzeit der Edge-Metadaten in Redis
    edge_meta: 30
redis:
  # Unterstützt Standalone-, Sentinel- und Cluster-Verbindungen
  mode: standalone
  standalone:
    network: tcp
    addr: redis:6379
    db: 0
```

### Verwendung

Da Frontlas dazu dient, verfügbare Frontiers zu finden, müssen sich Microservices folgendermaßen anpassen:

**Microservice bezieht Service**

```golang
package main

import (
  "net"
  "github.com/singchia/frontier/api/dataplane/v1/service"
)

func main() {
  // NewClusterService verwenden, um Service zu beziehen
  svc, err := service.NewClusterService("127.0.0.1:40011")
  // Service verwenden, alles andere bleibt unverändert
}
```

**Edge-Knoten erhält die Verbindungsadresse**

Edge-Knoten verbinden sich weiterhin mit Frontier, können aber verfügbare Frontier-Adressen von Frontlas erhalten. Frontlas stellt eine Schnittstelle zum Auflisten der Frontier-Instanzen bereit:

```bash
curl -X GET http://127.0.0.1:40011/cluster/v1/frontiers
```

Sie können diese Schnittstelle kapseln, um den Edge-Knoten Lastverteilung oder Hochverfügbarkeit bereitzustellen, oder mTLS ergänzen, um sie den Edge-Knoten direkt zur Verfügung zu stellen (nicht empfohlen).

gRPC für die Control Plane siehe [Protobuf-Definition](./api/controlplane/frontlas/v1/cluster.proto).

Die Control Plane von Frontlas unterscheidet sich von der von Frontier, da sie cluster-orientiert ist und derzeit nur lesende Schnittstellen für den Cluster bereitstellt.

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

**CRD und Operator installieren**

Folgen Sie diesen Schritten, um den Operator in Ihrer .kubeconfig-Umgebung zu installieren und bereitzustellen:

```bash
git clone https://github.com/singchia/frontier.git
cd dist/crd
kubectl apply -f install.yaml
```

CRD prüfen:

```bash
kubectl get crd frontierclusters.frontier.singchia.io
```

Operator prüfen:

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
    # Frontier als einzelne Instanz
    replicas: 2
    # Port auf der Microservice-Seite
    servicebound:
      port: 30011
    # Port auf der Edge-Knoten-Seite
    edgebound:
      port: 30012
  frontlas:
    # Frontlas als einzelne Instanz
    replicas: 1
    # Port der Control Plane
    controlplane:
      port: 40011
    redis:
      # Konfiguration des abhängigen Redis
      addrs:
        - rfs-redisfailover:26379
      password: your-password
      masterName: mymaster
      redisType: sentinel
```

Als `frontiercluster.yaml` speichern und

```
kubectl apply -f frontiercluster.yaml
```

Innerhalb einer Minute steht Ihnen ein Cluster mit 2 Frontier-Instanzen + 1 Frontlas-Instanz zur Verfügung.

Den Bereitstellungsstatus der Ressourcen prüfen Sie mit:

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

Ihr Microservice sollte sich mit `service/frontiercluster-frontlas-svc:40011` verbinden, und Ihr Edge-Knoten kann sich mit dem NodePort verbinden, auf dem `:30012` liegt.

## Entwicklung

### Roadmap

Siehe [ROADMAP](./ROADMAP.md).

### Beiträge

Wenn Sie einen Fehler finden, öffnen Sie bitte ein Issue; die Projekt-Maintainer antworten zeitnah.

Wenn Sie Features einreichen oder Projektprobleme schneller lösen möchten, sind PRs unter diesen einfachen Bedingungen willkommen:

- Der Code-Stil bleibt konsistent
- Jeder Commit enthält ein Feature
- Der eingereichte Code enthält Unit-Tests

## Tests

### Stream-Funktion

<img src="./docs/diagram/stream.png" width="100%">

## Community

<p align=center>
<img src="./docs/diagram/wechat.JPG" width="30%">
</p>

Treten Sie unserer WeChat-Gruppe bei, um zu diskutieren und Unterstützung zu erhalten.

## Lizenz

 Veröffentlicht unter der [Apache License 2.0](https://github.com/singchia/geminio/blob/main/LICENSE).

---
Ein Stern ⭐️ wäre sehr willkommen ♥️
