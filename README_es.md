<p align=center>
<img src="./docs/diagram/frontier-logo.png" width="30%">
</p>

<div align="center">

[![Go](https://github.com/singchia/frontier/actions/workflows/go.yml/badge.svg)](https://github.com/singchia/frontier/actions/workflows/go.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/singchia/frontier)](https://goreportcard.com/report/github.com/singchia/frontier)
[![Go Reference](https://pkg.go.dev/badge/badge/github.com/singchia/frontier.svg)](https://pkg.go.dev/github.com/singchia/frontier/api/dataplane/v1/service)
[![License](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)

[English](./README.md) | [简体中文](./README_zh.md) | [日本語](./README_ja.md) | [한국어](./README_ko.md) | Español | [Français](./README_fr.md) | [Deutsch](./README_de.md)

</div>


Frontier es una pasarela de conexión persistente de **full-dúplex** y código abierto, escrita en Go. Permite que los microservicios lleguen directamente a los nodos de borde o clientes, y viceversa. Ofrece **RPC bidireccional** full-dúplex, **mensajería** y **flujos punto a punto**. Frontier sigue los principios de arquitectura **cloud-native**, admite un despliegue rápido de clústeres mediante Operator y está diseñado para **alta disponibilidad** y **escalado elástico** hasta millones de nodos de borde o clientes en línea.

## Tabla de contenidos

- [Características](#características)
- [Inicio rápido](#inicio-rápido)
- [Arquitectura](#arquitectura)
- [Uso](#uso)
- [Configuración](#configuración)
- [Despliegue](#despliegue)
- [Clúster](#clúster)
- [Kubernetes](#kubernetes)
- [Desarrollo](#desarrollo)
- [Pruebas](#pruebas)
- [Comunidad](#comunidad)
- [Licencia](#licencia)

## Inicio rápido

1. Ejecutar una única instancia de Frontier:

```bash
docker run -d --name frontier -p 30011:30011 -p 30012:30012 singchia/frontier:1.1.0
```

2. Compilar y ejecutar los ejemplos:

```bash
make examples
```

Ejecutar el ejemplo de sala de chat:

```bash
# Terminal 1
./bin/chatroom_service

# Terminal 2
./bin/chatroom_agent
```

Vídeo de demostración:

https://github.com/singchia/frontier/assets/15531166/18b01d96-e30b-450f-9610-917d65259c30

## Características

- **RPC bidireccional**: servicios y bordes pueden invocarse entre sí con balanceo de carga.
- **Mensajería**: publicación/recepción basada en temas entre servicios, bordes y MQ externo.
- **Flujos punto a punto**: apertura de flujos directos para proxy, transferencia de archivos y tráfico personalizado.
- **Despliegue cloud-native**: ejecución mediante Docker, Compose, Helm u Operator.
- **Alta disponibilidad y escalado**: soporte de reconexión, clustering y escalado horizontal con Frontlas.
- **Autenticación y presencia**: autenticación de bordes y notificaciones de conexión/desconexión.
- **APIs del plano de control**: APIs gRPC y REST para consultar y gestionar los nodos en línea.


## Arquitectura

**Componente Frontier**

<img src="./docs/diagram/frontier.png" width="100%">

- _Service End_: el punto de entrada para funciones de microservicios, que se conecta por defecto.
- _Edge End_: el punto de entrada para funciones de nodos de borde o clientes.
- _Publish/Receive_: publicación y recepción de mensajes.
- _Call/Register_: invocación y registro de funciones.
- _OpenStream/AcceptStream_: apertura y aceptación de flujos punto a punto (conexiones).
- _External MQ_: Frontier admite el reenvío, según la configuración, de los mensajes publicados desde los nodos de borde hacia temas de MQ externos.


Frontier requiere que tanto los microservicios como los nodos de borde se conecten activamente a Frontier. Durante la conexión se pueden transportar los metadatos de Service y Edge (temas de recepción, RPC, nombres de servicio, etc.). Los puertos de conexión por defecto son:

- :30011: para que los microservicios se conecten y obtengan Service.
- :30012: para que los nodos de borde se conecten y obtengan Edge.
- :30010: para que el personal de operaciones o los programas utilicen el plano de control.


### Funcionalidad

<table><thead>
  <tr>
    <th>Función</th>
    <th>Iniciador</th>
    <th>Receptor</th>
    <th>Método</th>
    <th>Método de enrutamiento</th>
    <th>Descripción</th>
  </tr></thead>
<tbody>
  <tr>
    <td rowspan="2">Messager</td>
    <td>Service</td>
    <td>Edge</td>
    <td>Publish</td>
    <td>EdgeID+Topic</td>
    <td>Debe publicar a un EdgeID específico; el tema por defecto está vacío. El borde llama a Receive para recibir el mensaje y, tras procesarlo, debe llamar a msg.Done() o msg.Error(err) para garantizar la consistencia del mensaje.</td>
  </tr>
  <tr>
    <td>Edge</td>
    <td>Service o External MQ</td>
    <td>Publish</td>
    <td>Topic</td>
    <td>Debe publicar a un tema; Frontier selecciona un Service o MQ concreto según el tema.</td>
  </tr>
  <tr>
    <td rowspan="2">RPCer</td>
    <td>Service</td>
    <td>Edge</td>
    <td>Call</td>
    <td>EdgeID+Method</td>
    <td>Debe invocar a un EdgeID específico, indicando el nombre del método.</td>
  </tr>
  <tr>
    <td>Edge</td>
    <td>Service</td>
    <td>Call</td>
    <td>Method</td>
    <td>Debe invocar un método; Frontier selecciona un Service concreto según el nombre del método.</td>
  </tr>
  <tr>
    <td rowspan="2">Multiplexer</td>
    <td>Service</td>
    <td>Edge</td>
    <td>OpenStream</td>
    <td>EdgeID</td>
    <td>Debe abrir un flujo hacia un EdgeID específico.</td>
  </tr>
  <tr>
    <td>Edge</td>
    <td>Service</td>
    <td>OpenStream</td>
    <td>ServiceName</td>
    <td>Debe abrir un flujo hacia un ServiceName, especificado mediante service.OptionServiceName durante la inicialización del Service.</td>
  </tr>
</tbody></table>

**Principios clave de diseño**:

1. Todos los mensajes, RPCs y flujos son transmisiones punto a punto.
	- De los microservicios a los bordes, debe especificarse el ID del nodo de borde.
	- De los bordes a los microservicios, Frontier enruta según Topic y Method, y finalmente selecciona un microservicio o un MQ externo mediante hashing. Por defecto, el hashing se basa en edgeid, pero puede elegirse random o srcip.
2. Los mensajes requieren un reconocimiento explícito por parte del receptor.
	- Para garantizar la semántica de entrega, el receptor debe llamar a msg.Done() o msg.Error(err) para asegurar la consistencia de entrega.
3. Los flujos abiertos por el Multiplexer representan lógicamente una comunicación directa entre microservicios y nodos de borde.
	- Una vez que el otro lado recibe el flujo, toda la funcionalidad sobre este flujo llegará directamente al otro lado, omitiendo las políticas de enrutamiento de Frontier.

## Uso

Guía de uso detallada: [docs/USAGE.md](./docs/USAGE.md)

## Configuración

Guía de configuración detallada: [docs/CONFIGURATION.md](./docs/CONFIGURATION.md)

## Despliegue

Para una única instancia de Frontier, puede elegir los siguientes métodos para desplegarla según su entorno.

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

Si está en un entorno Kubernetes, puede usar Helm para desplegar rápidamente una instancia.

```bash
git clone https://github.com/singchia/frontier.git
cd dist/helm
helm install frontier ./ -f values.yaml
```

Su microservicio debería conectarse a ```service/frontier-servicebound-svc:30011```, y su nodo de borde puede conectarse al NodePort donde se expone `:30012`.

### Systemd

Consulte la documentación dedicada a Systemd:

[dist/systemd/README.md](./dist/systemd/README.md)

### Operator

Véase la sección de despliegue en clúster más abajo.

## Clúster

### Arquitectura Frontier + Frontlas

<img src="./docs/diagram/frontlas.png" width="100%">

El componente adicional Frontlas se utiliza para construir el clúster. Frontlas también es un componente sin estado y no almacena otra información en memoria, por lo que requiere una dependencia adicional de Redis. Debe proporcionar información de conexión a Redis a Frontlas, con soporte para `redis`, `sentinel` y `redis-cluster`.

- _Frontier_: componente de comunicación entre los microservicios y el plano de datos de los bordes.
- _Frontlas_: siglas de Frontier Atlas, un componente de gestión del clúster que registra en Redis los metadatos y la información activa de los microservicios y los bordes.

Frontier debe conectarse proactivamente a Frontlas para reportar su estado y el de los microservicios y los bordes. Los puertos por defecto de Frontlas son:

- `:40011` para la conexión de microservicios, reemplazando al puerto 30011 de una única instancia Frontier.
- `:40012` para la conexión de Frontier para reportar estado.

Puede desplegar el número de instancias Frontier que necesite; en el caso de Frontlas, desplegar dos instancias por separado puede garantizar HA (alta disponibilidad), ya que no guarda estado ni presenta problemas de consistencia.

### Configuración

El `frontier.yaml` de **Frontier** necesita añadir la siguiente configuración:

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
  # ID único dentro del clúster de Frontier
  frontier_id: frontier01
```

Frontier debe conectarse a Frontlas para reportar su estado y el de los microservicios y bordes.

Configuración mínima del `frontlas.yaml` de **Frontlas**:

```yaml
control_plane:
  listen:
    # Los microservicios se conectan a esta dirección para descubrir bordes en el clúster
    network: tcp
    addr: 0.0.0.0:40011
frontier_plane:
  # Frontier se conecta a esta dirección
  listen:
    network: tcp
    addr: 0.0.0.0:40012
  expiration:
    # Tiempo de expiración de los metadatos de microservicio en Redis
    service_meta: 30
    # Tiempo de expiración de los metadatos de borde en Redis
    edge_meta: 30
redis:
  # Soporte para conexiones standalone, sentinel y cluster
  mode: standalone
  standalone:
    network: tcp
    addr: redis:6379
    db: 0
```

### Uso

Dado que Frontlas se utiliza para descubrir los Frontiers disponibles, los microservicios deben ajustarse de la siguiente manera:

**Obtención de Service desde un microservicio**

```golang
package main

import (
  "net"
  "github.com/singchia/frontier/api/dataplane/v1/service"
)

func main() {
  // Usar NewClusterService para obtener Service
  svc, err := service.NewClusterService("127.0.0.1:40011")
  // Empezar a usar service; todo lo demás permanece igual
}
```

**Obtención de la dirección de conexión por parte de los nodos de borde**

Los nodos de borde siguen conectándose a Frontier, pero pueden obtener las direcciones de los Frontier disponibles desde Frontlas. Frontlas proporciona una interfaz para listar las instancias de Frontier:

```bash
curl -X GET http://127.0.0.1:40011/cluster/v1/frontiers
```

Puede envolver esta interfaz para proporcionar balanceo de carga o alta disponibilidad a los nodos de borde, o añadir mTLS para entregarla directamente a los nodos de borde (no recomendado).

Para el gRPC del plano de control, véase la [definición Protobuf](./api/controlplane/frontlas/v1/cluster.proto).

El plano de control de Frontlas difiere del de Frontier en que está orientado al clúster, y actualmente solo proporciona interfaces de lectura para el clúster.

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

**Instalar CRD y Operator**

Siga estos pasos para instalar y desplegar el Operator en su entorno .kubeconfig:

```bash
git clone https://github.com/singchia/frontier.git
cd dist/crd
kubectl apply -f install.yaml
```

Comprobar CRD:

```bash
kubectl get crd frontierclusters.frontier.singchia.io
```

Comprobar Operator:

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
    # Frontier de instancia única
    replicas: 2
    # Puerto del lado de microservicio
    servicebound:
      port: 30011
    # Puerto del lado de nodo de borde
    edgebound:
      port: 30012
  frontlas:
    # Frontlas de instancia única
    replicas: 1
    # Puerto del plano de control
    controlplane:
      port: 40011
    redis:
      # Configuración de Redis del que se depende
      addrs:
        - rfs-redisfailover:26379
      password: your-password
      masterName: mymaster
      redisType: sentinel
```

Guarde como `frontiercluster.yaml` y ejecute

```
kubectl apply -f frontiercluster.yaml
```

En 1 minuto tendrá un clúster con 2 instancias de Frontier + 1 de Frontlas.

Compruebe el estado del despliegue de los recursos con:

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

Su microservicio debería conectarse a `service/frontiercluster-frontlas-svc:40011`, y su nodo de borde puede conectarse al NodePort donde se expone `:30012`.

## Desarrollo

### Roadmap

Véase [ROADMAP](./ROADMAP.md).

### Contribuciones

Si encuentra algún error, abra una issue y los mantenedores responderán con prontitud.

Si desea enviar funcionalidades o resolver problemas del proyecto con mayor rapidez, los PR son bienvenidos bajo estas sencillas condiciones:

- Mantener un estilo de código coherente
- Cada envío incluye una única funcionalidad
- El código enviado incluye pruebas unitarias

## Pruebas

### Funcionalidad Stream

<img src="./docs/diagram/stream.png" width="100%">

## Comunidad

<p align=center>
<img src="./docs/diagram/wechat.JPG" width="30%">
</p>

Únase a nuestro grupo de WeChat para debates y soporte.

## Licencia

 Publicado bajo la [Apache License 2.0](https://github.com/singchia/geminio/blob/main/LICENSE).

---
¡Una estrella ⭐️ sería muy apreciada ♥️!
