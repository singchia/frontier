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

[English](./README.md) | [简体中文](./README_zh.md) | [日本語](./README_ja.md) | [한국어](./README_ko.md) | [Español](./README_es.md) | Français | [Deutsch](./README_de.md)

</div>


Frontier est une passerelle de connexion longue durée open source **full-duplex**, écrite en Go. Elle permet aux microservices d'atteindre directement les nœuds de périphérie ou les clients, et inversement. Elle fournit un **RPC bidirectionnel** full-duplex, de la **messagerie** et des **flux point à point**. Frontier suit les principes d'architecture **cloud-native**, prend en charge un déploiement rapide de clusters via Operator, et est conçue pour la **haute disponibilité** et la **mise à l'échelle élastique** jusqu'à des millions de nœuds de périphérie ou de clients en ligne.

## Table des matières

- [Fonctionnalités](#fonctionnalités)
- [Démarrage rapide](#démarrage-rapide)
- [Architecture](#architecture)
- [Utilisation](#utilisation)
- [Configuration](#configuration)
- [Déploiement](#déploiement)
- [Cluster](#cluster)
- [Kubernetes](#kubernetes)
- [Développement](#développement)
- [Tests](#tests)
- [Communauté](#communauté)
- [Licence](#licence)

## Démarrage rapide

1. Lancer une instance unique de Frontier :

```bash
docker run -d --name frontier -p 30011:30011 -p 30012:30012 singchia/frontier:1.1.0
```

2. Compiler et exécuter les exemples :

```bash
make examples
```

Exécuter l'exemple de salon de discussion :

```bash
# Terminal 1
./bin/chatroom_service

# Terminal 2
./bin/chatroom_agent
```

Vidéo de démonstration :

https://github.com/singchia/frontier/assets/15531166/18b01d96-e30b-450f-9610-917d65259c30

## Fonctionnalités

- **RPC bidirectionnel** : les services et les bords peuvent s'appeler mutuellement avec équilibrage de charge.
- **Messagerie** : publication/réception par sujet entre services, bords et MQ externe.
- **Flux point à point** : ouverture de flux directs pour le proxy, le transfert de fichiers et le trafic personnalisé.
- **Déploiement cloud-native** : exécution via Docker, Compose, Helm ou Operator.
- **Haute disponibilité et mise à l'échelle** : prise en charge de la reconnexion, du clustering et de la mise à l'échelle horizontale avec Frontlas.
- **Authentification et présence** : authentification des bords et notifications en ligne/hors ligne.
- **APIs du plan de contrôle** : APIs gRPC et REST pour interroger et gérer les nœuds en ligne.


## Architecture

**Composant Frontier**

<img src="./docs/diagram/frontier.png" width="100%">

- _Service End_ : point d'entrée pour les fonctions des microservices, se connecte par défaut.
- _Edge End_ : point d'entrée pour les fonctions des nœuds de périphérie ou des clients.
- _Publish/Receive_ : publication et réception de messages.
- _Call/Register_ : appel et enregistrement de fonctions.
- _OpenStream/AcceptStream_ : ouverture et acceptation de flux point à point (connexions).
- _External MQ_ : Frontier prend en charge, selon la configuration, le transfert des messages publiés depuis les nœuds de périphérie vers des sujets MQ externes.


Frontier exige que les microservices et les nœuds de périphérie se connectent activement à Frontier. Les métadonnées de Service et Edge (sujets de réception, RPC, noms de service, etc.) peuvent être transportées lors de la connexion. Les ports de connexion par défaut sont :

- :30011 : pour que les microservices se connectent et obtiennent Service.
- :30012 : pour que les nœuds de périphérie se connectent et obtiennent Edge.
- :30010 : pour les opérateurs ou programmes qui utilisent le plan de contrôle.


### Fonctionnalités

<table><thead>
  <tr>
    <th>Fonction</th>
    <th>Initiateur</th>
    <th>Destinataire</th>
    <th>Méthode</th>
    <th>Méthode de routage</th>
    <th>Description</th>
  </tr></thead>
<tbody>
  <tr>
    <td rowspan="2">Messager</td>
    <td>Service</td>
    <td>Edge</td>
    <td>Publish</td>
    <td>EdgeID+Topic</td>
    <td>Doit publier vers un EdgeID spécifique, le sujet par défaut est vide. Le bord appelle Receive pour recevoir le message, puis après traitement, doit appeler msg.Done() ou msg.Error(err) pour garantir la cohérence du message.</td>
  </tr>
  <tr>
    <td>Edge</td>
    <td>Service ou External MQ</td>
    <td>Publish</td>
    <td>Topic</td>
    <td>Doit publier vers un sujet ; Frontier sélectionne un Service ou un MQ spécifique selon le sujet.</td>
  </tr>
  <tr>
    <td rowspan="2">RPCer</td>
    <td>Service</td>
    <td>Edge</td>
    <td>Call</td>
    <td>EdgeID+Method</td>
    <td>Doit appeler un EdgeID spécifique, en transportant le nom de la méthode.</td>
  </tr>
  <tr>
    <td>Edge</td>
    <td>Service</td>
    <td>Call</td>
    <td>Method</td>
    <td>Doit appeler une méthode ; Frontier sélectionne un Service spécifique selon le nom de la méthode.</td>
  </tr>
  <tr>
    <td rowspan="2">Multiplexer</td>
    <td>Service</td>
    <td>Edge</td>
    <td>OpenStream</td>
    <td>EdgeID</td>
    <td>Doit ouvrir un flux vers un EdgeID spécifique.</td>
  </tr>
  <tr>
    <td>Edge</td>
    <td>Service</td>
    <td>OpenStream</td>
    <td>ServiceName</td>
    <td>Doit ouvrir un flux vers un ServiceName, spécifié via service.OptionServiceName lors de l'initialisation du Service.</td>
  </tr>
</tbody></table>

**Principes de conception clés** :

1. Tous les messages, RPC et flux sont des transmissions point à point.
	- Des microservices vers les bords, l'identifiant du nœud de périphérie doit être spécifié.
	- Des bords vers les microservices, Frontier route selon Topic et Method, et sélectionne finalement un microservice ou un MQ externe par hachage. Par défaut, le hachage est basé sur edgeid, mais vous pouvez choisir random ou srcip.
2. Les messages nécessitent un accusé de réception explicite du destinataire.
	- Afin de garantir la sémantique de livraison, le destinataire doit appeler msg.Done() ou msg.Error(err) pour garantir la cohérence de la livraison.
3. Les flux ouverts par le Multiplexer représentent logiquement une communication directe entre microservices et nœuds de périphérie.
	- Une fois que l'autre partie reçoit le flux, toutes les fonctionnalités de ce flux atteignent directement l'autre partie, en contournant les politiques de routage de Frontier.

## Utilisation

Guide d'utilisation détaillé : [docs/USAGE.md](./docs/USAGE.md)

## Configuration

Guide de configuration détaillé : [docs/CONFIGURATION.md](./docs/CONFIGURATION.md)

## Déploiement

Pour une instance Frontier unique, vous pouvez choisir les méthodes suivantes pour déployer votre instance selon votre environnement.

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

Dans un environnement Kubernetes, vous pouvez utiliser Helm pour déployer rapidement une instance.

```bash
git clone https://github.com/singchia/frontier.git
cd dist/helm
helm install frontier ./ -f values.yaml
```

Votre microservice devrait se connecter à ```service/frontier-servicebound-svc:30011```, et votre nœud de périphérie peut se connecter au NodePort où se trouve `:30012`.

### Systemd

Utilisez la documentation Systemd dédiée :

[dist/systemd/README.md](./dist/systemd/README.md)

### Operator

Voir la section de déploiement en cluster ci-dessous.

## Cluster

### Architecture Frontier + Frontlas

<img src="./docs/diagram/frontlas.png" width="100%">

Le composant supplémentaire Frontlas sert à construire le cluster. Frontlas est également un composant sans état qui ne stocke pas d'autres informations en mémoire, il nécessite donc une dépendance supplémentaire à Redis. Vous devez fournir à Frontlas les informations de connexion Redis, en prenant en charge `redis`, `sentinel` et `redis-cluster`.

- _Frontier_ : composant de communication entre les microservices et le plan de données des bords.
- _Frontlas_ : nommé Frontier Atlas, un composant de gestion de cluster qui enregistre dans Redis les métadonnées et les informations d'activité des microservices et des bords.

Frontier doit se connecter de manière proactive à Frontlas pour signaler son propre état ainsi que l'activité et l'état des microservices et des bords. Les ports par défaut de Frontlas sont :

- `:40011` pour la connexion des microservices, remplaçant le port 30011 d'une instance Frontier unique.
- `:40012` pour la connexion de Frontier afin de rapporter l'état.

Vous pouvez déployer autant d'instances Frontier que nécessaire ; pour Frontlas, déployer deux instances séparément peut garantir la haute disponibilité (HA), car il ne stocke pas d'état et ne présente pas de problèmes de cohérence.

### Configuration

Le `frontier.yaml` de **Frontier** doit ajouter la configuration suivante :

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
  # Identifiant unique au sein du cluster Frontier
  frontier_id: frontier01
```

Frontier doit se connecter à Frontlas pour signaler son propre état ainsi que celui des microservices et des bords.

Configuration minimale du `frontlas.yaml` de **Frontlas** :

```yaml
control_plane:
  listen:
    # Les microservices se connectent à cette adresse pour découvrir les bords du cluster
    network: tcp
    addr: 0.0.0.0:40011
frontier_plane:
  # Frontier se connecte à cette adresse
  listen:
    network: tcp
    addr: 0.0.0.0:40012
  expiration:
    # Durée d'expiration des métadonnées des microservices dans Redis
    service_meta: 30
    # Durée d'expiration des métadonnées des bords dans Redis
    edge_meta: 30
redis:
  # Prise en charge des connexions standalone, sentinel et cluster
  mode: standalone
  standalone:
    network: tcp
    addr: redis:6379
    db: 0
```

### Utilisation

Comme Frontlas sert à découvrir les Frontiers disponibles, les microservices doivent s'adapter ainsi :

**Microservice obtenant Service**

```golang
package main

import (
  "net"
  "github.com/singchia/frontier/api/dataplane/v1/service"
)

func main() {
  // Utiliser NewClusterService pour obtenir Service
  svc, err := service.NewClusterService("127.0.0.1:40011")
  // Commencer à utiliser service, tout le reste reste inchangé
}
```

**Nœud de périphérie obtenant l'adresse de connexion**

Les nœuds de périphérie se connectent toujours à Frontier mais peuvent obtenir les adresses des Frontiers disponibles auprès de Frontlas. Frontlas fournit une interface permettant de lister les instances Frontier :

```bash
curl -X GET http://127.0.0.1:40011/cluster/v1/frontiers
```

Vous pouvez encapsuler cette interface pour fournir l'équilibrage de charge ou la haute disponibilité aux nœuds de périphérie, ou ajouter du mTLS pour l'exposer directement aux nœuds (non recommandé).

Pour le gRPC du plan de contrôle, voir la [définition Protobuf](./api/controlplane/frontlas/v1/cluster.proto).

Le plan de contrôle de Frontlas diffère de celui de Frontier car il est orienté cluster, et ne fournit actuellement que des interfaces de lecture pour le cluster.

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

**Installer CRD et Operator**

Suivez ces étapes pour installer et déployer l'Operator dans votre environnement .kubeconfig :

```bash
git clone https://github.com/singchia/frontier.git
cd dist/crd
kubectl apply -f install.yaml
```

Vérifier la CRD :

```bash
kubectl get crd frontierclusters.frontier.singchia.io
```

Vérifier l'Operator :

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
    # Frontier en instance unique
    replicas: 2
    # Port côté microservice
    servicebound:
      port: 30011
    # Port côté nœud de périphérie
    edgebound:
      port: 30012
  frontlas:
    # Frontlas en instance unique
    replicas: 1
    # Port du plan de contrôle
    controlplane:
      port: 40011
    redis:
      # Configuration Redis dont dépend le système
      addrs:
        - rfs-redisfailover:26379
      password: your-password
      masterName: mymaster
      redisType: sentinel
```

Enregistrez sous `frontiercluster.yaml`, puis

```
kubectl apply -f frontiercluster.yaml
```

En 1 minute, vous obtiendrez un cluster avec 2 instances Frontier + 1 instance Frontlas.

Vérifiez l'état du déploiement des ressources avec :

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

Votre microservice devrait se connecter à `service/frontiercluster-frontlas-svc:40011`, et votre nœud de périphérie peut se connecter au NodePort où se trouve `:30012`.

## Développement

### Feuille de route

Voir [ROADMAP](./ROADMAP.md).

### Contributions

Si vous trouvez un bug, ouvrez une issue et les mainteneurs du projet répondront rapidement.

Si vous souhaitez soumettre des fonctionnalités ou résoudre plus rapidement des problèmes du projet, les PR sont les bienvenues sous ces conditions simples :

- Le style de code reste cohérent
- Chaque soumission inclut une seule fonctionnalité
- Le code soumis inclut des tests unitaires

## Tests

### Fonction Stream

<img src="./docs/diagram/stream.png" width="100%">

## Communauté

<p align=center>
<img src="./docs/diagram/wechat.JPG" width="30%">
</p>

Rejoignez notre groupe WeChat pour les discussions et le support.

## Licence

 Publié sous la [licence Apache 2.0](https://github.com/singchia/geminio/blob/main/LICENSE).

---
Une étoile ⭐️ serait très appréciée ♥️
