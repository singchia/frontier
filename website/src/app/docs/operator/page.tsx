export default function OperatorGuide() {
  return (
    <div className="pb-16 max-w-4xl mx-auto">
      <div className="mb-12">
        <h1 className="text-4xl font-bold text-white mb-4">Kubernetes Operator & CRD</h1>
        <p className="text-xl text-zinc-400">
          Run Frontier on Kubernetes with a single <code>FrontierCluster</code> resource. The operator handles deployments, TLS secrets, services, probes, security context, and graceful shutdown.
        </p>
      </div>

      <div className="prose prose-invert prose-blue max-w-none prose-pre:bg-[#0D0D0D] prose-pre:border prose-pre:border-zinc-800">

        <h2 id="overview">1. Overview</h2>
        <p>
          The Frontier operator manages a two-tier deployment: <strong>Frontier</strong> (data plane, stateless edge gateway) and <strong>Frontlas</strong> (control plane, Redis-backed coordinator). Both are reconciled from a single namespaced custom resource <code>FrontierCluster</code> in the API group <code>frontier.singchia.io/v1alpha1</code>.
        </p>
        <p>
          With one CR you get: two Deployments, three Services (servicebound, edgebound, controlplane), TLS material copied into operator-managed Secrets, sane production defaults (probes, preStop, non-root, drop-all caps, preferred anti-affinity), and a status that surfaces ready replicas and Conditions.
        </p>

        <h2 id="install">2. Installation</h2>
        <h3>2.1 Install the CRD and operator</h3>
        <pre><code className="language-bash">{`# Apply the CRD + operator deployment + RBAC in one shot
kubectl apply -f https://raw.githubusercontent.com/singchia/frontier/main/pkg/operator/dist/install.yaml

# Or from a local checkout
git clone https://github.com/singchia/frontier.git
kubectl apply -f frontier/pkg/operator/dist/install.yaml`}</code></pre>

        <p>Verify the CRD and operator pod:</p>
        <pre><code className="language-bash">{`kubectl get crd frontierclusters.frontier.singchia.io
kubectl get all -n frontier-operator-system`}</code></pre>

        <h3>2.2 RBAC</h3>
        <p>The bundled <code>install.yaml</code> creates a ClusterRole granting the operator:</p>
        <ul>
          <li><code>frontier.singchia.io/frontierclusters</code> &mdash; full CRUD + status</li>
          <li><code>apps/deployments</code> &mdash; full CRUD (manages frontier &amp; frontlas Deployments)</li>
          <li><code>core/services, secrets, pods</code> &mdash; full CRUD (services, TLS material, pod inspection)</li>
          <li><code>core/events</code> &mdash; create + patch (used by EventRecorder)</li>
        </ul>
        <p>If you tighten this further, keep at least <code>get;list;watch</code> on those resources or reconcile will fail.</p>

        <h3 id="install-helm">2.3 Alternative: install with Helm</h3>
        <p>
          Prefer Helm? The chart at <code>dist/helm/</code> deploys both <strong>frontier</strong> and <strong>frontlas</strong> in one shot, with an optional bundled <code>bitnami/redis</code> subchart. Defaults track the operator&apos;s production-grade settings (non-root UID 65532, drop-all capabilities, preferred host anti-affinity, preStop sleep, configurable drain window, observability endpoints on 9091/9092). Pick this path if your platform standardizes on Helm/ArgoCD/Flux and you don&apos;t need the operator&apos;s reconcile-driven self-healing for TLS Secrets and Status conditions.
        </p>

        <p><strong>Quick install</strong> with the bundled Redis:</p>
        <pre><code className="language-bash">{`# 1. Pull the bitnami/redis subchart
cd frontier/dist/helm
helm repo add bitnami https://charts.bitnami.com/bitnami
helm dependency update

# 2. Install
helm install frontier . \\
  --namespace frontier --create-namespace \\
  --set frontier.replicaCount=2 \\
  --set frontlas.replicaCount=1`}</code></pre>

        <p><strong>Bring your own Redis</strong> (set <code>redis.enabled: false</code> and point Frontlas at it):</p>
        <pre><code className="language-yaml">{`# my-values.yaml
redis:
  enabled: false

frontlas:
  externalRedis:
    addrs:
      - redis.shared:6379
    redisType: standalone
    passwordSecret:
      name: redis-creds      # must already exist in the release namespace
      key:  password`}</code></pre>

        <pre><code className="language-bash">{`helm install frontier . -n frontier --create-namespace -f my-values.yaml`}</code></pre>

        <p><strong>Common knobs</strong> in <code>values.yaml</code> (full listing: <code>helm show values dist/helm/</code>):</p>
        <table>
          <thead>
            <tr><th>Path</th><th>Default</th><th>Notes</th></tr>
          </thead>
          <tbody>
            <tr><td><code>frontier.replicaCount</code> / <code>frontlas.replicaCount</code></td><td>1 / 1</td><td>Independent scaling per component</td></tr>
            <tr><td><code>frontier.image.tag</code> / <code>frontlas.image.tag</code></td><td><code>{`""`}</code> &rarr; <code>Chart.AppVersion</code></td><td>Override to pin a specific binary version</td></tr>
            <tr><td><code>global.registry</code></td><td><code>singchia</code></td><td>Mirror to your private registry</td></tr>
            <tr><td><code>global.imagePullSecrets</code></td><td><code>[]</code></td><td>Private registry credentials</td></tr>
            <tr><td><code>frontier.service.edgebound.type</code></td><td>NodePort</td><td>Switch to LoadBalancer for cloud edge ingress</td></tr>
            <tr><td><code>frontier.podSecurityContext</code> / <code>containerSecurityContext</code></td><td>nonRoot UID 65532, drop ALL caps</td><td>Set to <code>{`{}`}</code> if your custom image needs root</td></tr>
            <tr><td><code>frontier.terminationGracePeriodSeconds</code> / <code>frontier.drainSeconds</code></td><td>60 / 50</td><td>Long-lived edge connections; drain &lt; grace</td></tr>
            <tr><td><code>frontier.autoscaling.enabled</code></td><td>false</td><td>HPA on the frontier Deployment</td></tr>
            <tr><td><code>observability.frontier.enabled</code> / <code>observability.frontlas.enabled</code></td><td>true / true</td><td>Toggle the <code>/healthz</code> <code>/readyz</code> <code>/metrics</code> endpoints (ports 9091 / 9092)</td></tr>
            <tr><td><code>serviceMonitor.enabled</code></td><td>false</td><td>Opt in if prometheus-operator is installed</td></tr>
            <tr><td><code>redis.enabled</code></td><td>true</td><td>Set false to use external Redis (configure under <code>frontlas.externalRedis</code>)</td></tr>
          </tbody>
        </table>

        <p><strong>Operator vs Helm — pick one:</strong></p>
        <table>
          <thead>
            <tr><th>Concern</th><th>Operator</th><th>Helm</th></tr>
          </thead>
          <tbody>
            <tr><td>Custom Resource (declarative)</td><td>✅ FrontierCluster CR</td><td>❌ values.yaml + Deployments directly</td></tr>
            <tr><td>Status &amp; Conditions per cluster</td><td>✅ Available / Progressing / Degraded</td><td>❌ Inspect underlying Deployments</td></tr>
            <tr><td>Self-healing on TLS Secret rotation</td><td>✅ Reconciler watches Secrets</td><td>❌ Manual <code>helm upgrade</code></td></tr>
            <tr><td>Multiple clusters in one namespace</td><td>✅ Each CR is isolated</td><td>⚠️ Need separate releases</td></tr>
            <tr><td>Bundled Redis</td><td>❌ BYO</td><td>✅ <code>bitnami/redis</code> subchart</td></tr>
            <tr><td>ArgoCD / Flux GitOps</td><td>✅ (commit the CR)</td><td>✅ (commit values.yaml)</td></tr>
            <tr><td>Initial install footprint</td><td>Operator pod + CRD + RBAC</td><td>No long-running operator</td></tr>
          </tbody>
        </table>
        <p>
          You can also <strong>publish the chart</strong> for downstream consumers: <code>helm package dist/helm/ -d /path/to/repo</code> produces <code>frontier-1.2.5.tgz</code>; serve the directory with any HTTP server or <code>helm push</code> to OCI.
        </p>

        <h2 id="quickstart">3. Quick start</h2>
        <p>Minimum viable cluster: 2 frontier replicas, 1 frontlas replica, external Redis. Save as <code>frontiercluster.yaml</code>:</p>
        <pre><code className="language-yaml">{`apiVersion: frontier.singchia.io/v1alpha1
kind: FrontierCluster
metadata:
  name: prod
  namespace: frontier
spec:
  frontier:
    replicas: 2
  frontlas:
    replicas: 1
    redis:
      addrs:
        - redis.frontier:6379
      passwordSecret:
        name: redis-creds
        key: password
      redisType: standalone`}</code></pre>

        <pre><code className="language-bash">{`kubectl create namespace frontier
kubectl -n frontier create secret generic redis-creds --from-literal=password=...
kubectl apply -f frontiercluster.yaml`}</code></pre>

        <p>Wait for it to come up, then check:</p>
        <pre><code className="language-bash">{`kubectl -n frontier get fc                  # short name 'fc' is registered
NAME   PHASE     FRONTIER   FRONTLAS   AGE
prod   Running   2          1          47s

kubectl -n frontier describe fc prod         # see Conditions + Events
kubectl -n frontier get pods                 # frontier + frontlas pods
kubectl -n frontier get svc                  # 3 services rendered`}</code></pre>

        <h2 id="crd-reference">4. CRD field reference</h2>
        <p>Everything below lives under <code>spec</code>. All fields outside <code>frontier.servicebound</code>, <code>frontier.edgebound</code>, <code>frontlas.controlplane</code>, and <code>frontlas.redis</code> are optional &mdash; sensible defaults apply.</p>

        <h3>4.1 <code>spec.frontier</code></h3>
        <table>
          <thead>
            <tr><th>Field</th><th>Type</th><th>Default</th><th>Notes</th></tr>
          </thead>
          <tbody>
            <tr><td><code>replicas</code></td><td>int</td><td>1</td><td>Frontier pod count</td></tr>
            <tr><td><code>image</code></td><td>string</td><td><code>singchia/frontier:1.1.0</code></td><td>Override to pin a specific tag</td></tr>
            <tr><td><code>servicebound.port</code></td><td>int</td><td>30011</td><td>Service-side TCP/gRPC port</td></tr>
            <tr><td><code>servicebound.service</code></td><td>string</td><td><code>{`<name>-servicebound-svc`}</code></td><td>Service name override</td></tr>
            <tr><td><code>servicebound.serviceType</code></td><td>string</td><td>ClusterIP</td><td>ClusterIP / NodePort / LoadBalancer</td></tr>
            <tr><td><code>edgebound.port</code></td><td>int</td><td>30012</td><td>Edge-side port (typically external)</td></tr>
            <tr><td><code>edgebound.serviceName</code></td><td>string</td><td><code>{`<name>-edgebound-svc`}</code></td><td>Service name override</td></tr>
            <tr><td><code>edgebound.serviceType</code></td><td>string</td><td>NodePort</td><td>Use LoadBalancer for cloud egress</td></tr>
            <tr><td><code>edgebound.tls.enabled</code></td><td>bool</td><td>false</td><td>Enable TLS on edgebound</td></tr>
            <tr><td><code>edgebound.tls.optional</code></td><td>bool</td><td>false</td><td>If true, both TLS and plain accepted</td></tr>
            <tr><td><code>edgebound.tls.mtls</code></td><td>bool</td><td>false</td><td>Enable client cert verification</td></tr>
            <tr><td><code>edgebound.tls.certificateKeySecretRef.name</code></td><td>string</td><td>&mdash;</td><td>User Secret with <code>tls.crt</code>, <code>tls.key</code></td></tr>
            <tr><td><code>edgebound.tls.caCertificateSecretRef.name</code></td><td>string</td><td>&mdash;</td><td>User Secret with <code>ca.crt</code> (mTLS only)</td></tr>
            <tr><td><code>nodeAffinity</code></td><td>NodeAffinity</td><td>nil</td><td>Legacy &mdash; use <code>pod.affinity</code> instead</td></tr>
            <tr><td><code>pod</code></td><td>PodOverrides</td><td>see &sect;4.3</td><td>Production-grade overrides for the frontier pod</td></tr>
          </tbody>
        </table>

        <h3>4.2 <code>spec.frontlas</code></h3>
        <table>
          <thead>
            <tr><th>Field</th><th>Type</th><th>Default</th><th>Notes</th></tr>
          </thead>
          <tbody>
            <tr><td><code>replicas</code></td><td>int</td><td>1</td><td>Frontlas pod count</td></tr>
            <tr><td><code>image</code></td><td>string</td><td><code>singchia/frontlas:1.1.0</code></td><td>Image override</td></tr>
            <tr><td><code>controlplane.port</code></td><td>int</td><td>40011</td><td>Service-side control plane port</td></tr>
            <tr><td><code>controlplane.frontierPlanePort</code></td><td>int</td><td>40012</td><td>Port used by frontier nodes to talk to frontlas</td></tr>
            <tr><td><code>controlplane.service</code></td><td>string</td><td><code>{`<name>-frontlas-svc`}</code></td><td>Service name override</td></tr>
            <tr><td><code>controlplane.serviceType</code></td><td>string</td><td>ClusterIP</td><td>Internal only by default</td></tr>
            <tr><td><code>redis.addrs</code></td><td>[]string</td><td><strong>required</strong></td><td>One or more Redis addrs</td></tr>
            <tr><td><code>redis.redisType</code></td><td>string</td><td><strong>required</strong></td><td><code>standalone</code> / <code>sentinel</code> / <code>cluster</code></td></tr>
            <tr><td><code>redis.db</code></td><td>int</td><td>0</td><td>DB index (standalone only)</td></tr>
            <tr><td><code>redis.user</code></td><td>string</td><td>&quot;&quot;</td><td>For Redis ACL</td></tr>
            <tr><td><code>redis.password</code></td><td>string</td><td>&quot;&quot;</td><td><strong>Deprecated</strong> &mdash; use <code>passwordSecret</code></td></tr>
            <tr><td><code>redis.passwordSecret</code></td><td>SecretKeySelector</td><td>nil</td><td>Recommended. Wins over <code>password</code>; injected via <code>valueFrom.secretKeyRef</code></td></tr>
            <tr><td><code>redis.masterName</code></td><td>string</td><td>&quot;&quot;</td><td>Sentinel only</td></tr>
            <tr><td><code>nodeAffinity</code></td><td>NodeAffinity</td><td>nil</td><td>Legacy &mdash; use <code>pod.affinity</code></td></tr>
            <tr><td><code>pod</code></td><td>PodOverrides</td><td>see &sect;4.3</td><td>Production-grade overrides for the frontlas pod</td></tr>
          </tbody>
        </table>

        <h3>4.3 <code>spec.frontier.pod</code> / <code>spec.frontlas.pod</code> (PodOverrides)</h3>
        <p>Every override is optional. When unset, the operator applies a production-grade default.</p>
        <table>
          <thead>
            <tr><th>Field</th><th>Type</th><th>Operator default</th><th>Use case</th></tr>
          </thead>
          <tbody>
            <tr><td><code>resources</code></td><td>ResourceRequirements</td><td>nil (BestEffort QoS)</td><td>Set CPU/memory requests + limits for production</td></tr>
            <tr><td><code>nodeSelector</code></td><td>map[string]string</td><td>nil</td><td>Pin to nodes by label</td></tr>
            <tr><td><code>tolerations</code></td><td>[]Toleration</td><td>nil</td><td>Run on tainted nodes</td></tr>
            <tr><td><code>topologySpreadConstraints</code></td><td>[]TopologySpreadConstraint</td><td>nil</td><td>Cross-zone / cross-node spread</td></tr>
            <tr><td><code>affinity</code></td><td>Affinity</td><td>only PodAntiAffinity (preferred host spread)</td><td>Setting this fully replaces the default and the legacy <code>nodeAffinity</code> field</td></tr>
            <tr><td><code>priorityClassName</code></td><td>string</td><td>&quot;&quot;</td><td>Critical workload priority</td></tr>
            <tr><td><code>serviceAccountName</code></td><td>string</td><td>default</td><td>Bind workload identity</td></tr>
            <tr><td><code>imagePullSecrets</code></td><td>[]LocalObjectReference</td><td>nil</td><td>Private registry credentials</td></tr>
            <tr><td><code>imagePullPolicy</code></td><td>string</td><td><code>IfNotPresent</code></td><td>Use <code>Always</code> in dev when pinning <code>latest</code></td></tr>
            <tr><td><code>annotations</code></td><td>map[string]string</td><td>nil</td><td>Pod annotations &mdash; cert-manager, Prometheus scrape config, sidecar opt-in</td></tr>
            <tr><td><code>labels</code></td><td>map[string]string</td><td>app=&hellip;</td><td>Extra labels (merged with selector labels)</td></tr>
            <tr><td><code>podSecurityContext</code></td><td>PodSecurityContext</td><td>runAsNonRoot=true, UID/GID/FSGroup=65532, RuntimeDefault seccomp</td><td>Override only when an image needs root or a different UID</td></tr>
            <tr><td><code>containerSecurityContext</code></td><td>SecurityContext</td><td>drop ALL caps, AllowPrivilegeEscalation=false, runAsNonRoot=true</td><td>Override to add a specific capability back</td></tr>
            <tr><td><code>terminationGracePeriodSeconds</code></td><td>int64</td><td>frontier=60, frontlas=30</td><td>Long-lived edge connections need at least 60</td></tr>
            <tr><td><code>livenessProbe</code></td><td>Probe</td><td>TCP socket on edge port (frontier) / control port (frontlas)</td><td>Replace with HTTP probe in M3+</td></tr>
            <tr><td><code>readinessProbe</code></td><td>Probe</td><td>TCP socket on service port (frontier) / HTTP <code>/cluster/v1/health</code> (frontlas)</td><td>HTTP <code>/readyz</code> available since M3</td></tr>
            <tr><td><code>lifecycle</code></td><td>Lifecycle</td><td>preStop: <code>sleep 10</code> (frontier) / <code>sleep 5</code> (frontlas)</td><td>Lets kube-proxy remove pod from Service Endpoints before SIGTERM</td></tr>
          </tbody>
        </table>

        <h2 id="scenarios">5. Common scenarios</h2>

        <h3>5.1 Edge mTLS</h3>
        <p>Provide both a server cert/key and a CA. The operator copies them into namespace-scoped Secrets and mounts them into the frontier pod at <code>/app/conf/edgebound/tls/secret</code> and <code>/app/conf/edgebound/tls/ca</code>.</p>
        <pre><code className="language-yaml">{`apiVersion: v1
kind: Secret
metadata:
  name: edge-server-cert
type: kubernetes.io/tls
data:
  tls.crt: ...    # PEM cert
  tls.key: ...    # PEM key
---
apiVersion: v1
kind: Secret
metadata:
  name: edge-ca
data:
  ca.crt: ...     # PEM CA
---
apiVersion: frontier.singchia.io/v1alpha1
kind: FrontierCluster
metadata:
  name: prod
spec:
  frontier:
    edgebound:
      port: 8443
      serviceType: LoadBalancer
      tls:
        enabled: true
        mtls: true
        certificateKeySecretRef:
          name: edge-server-cert
        caCertificateSecretRef:
          name: edge-ca
  frontlas: { ... }`}</code></pre>

        <h3>5.2 Production resources + scheduling</h3>
        <pre><code className="language-yaml">{`spec:
  frontier:
    replicas: 6
    pod:
      resources:
        requests: { cpu: "500m", memory: "512Mi" }
        limits:   { cpu: "2",    memory: "2Gi" }
      tolerations:
        - key: workload
          operator: Equal
          value: edge-gateway
          effect: NoSchedule
      topologySpreadConstraints:
        - maxSkew: 1
          topologyKey: topology.kubernetes.io/zone
          whenUnsatisfiable: ScheduleAnyway
          labelSelector:
            matchLabels: { app: prod-frontier }
      priorityClassName: frontend-critical
      terminationGracePeriodSeconds: 120`}</code></pre>

        <h3>5.3 Private image registry</h3>
        <pre><code className="language-yaml">{`spec:
  frontier:
    image: my-registry.example.com/frontier:1.2.4
    pod:
      imagePullSecrets:
        - name: my-registry-creds
      imagePullPolicy: IfNotPresent
  frontlas:
    image: my-registry.example.com/frontlas:1.2.4
    pod:
      imagePullSecrets:
        - name: my-registry-creds`}</code></pre>

        <h3>5.4 Annotations for Prometheus + cert-manager</h3>
        <pre><code className="language-yaml">{`spec:
  frontier:
    pod:
      annotations:
        prometheus.io/scrape: "true"
        prometheus.io/port: "9091"
        prometheus.io/path: "/metrics"
  frontlas:
    pod:
      annotations:
        cert-manager.io/inject-ca-from: frontier/frontier-ca`}</code></pre>

        <h3>5.5 Override SecurityContext for legacy images</h3>
        <p>If your custom frontier image needs root or a non-65532 UID, opt out of the default explicitly:</p>
        <pre><code className="language-yaml">{`spec:
  frontier:
    pod:
      podSecurityContext: {}                    # drop the default nonRoot/UID
      containerSecurityContext:
        runAsNonRoot: false
        capabilities:
          drop: []                              # keep capabilities`}</code></pre>

        <h2 id="status">6. Status &amp; Conditions</h2>
        <p>The CRD has a <code>status</code> subresource (read-only for users):</p>
        <pre><code className="language-yaml">{`status:
  phase: Running                # Pending | Running | Failed (deprecated, kept for printcolumn)
  message: "Good to go!"
  observedGeneration: 7         # spec.generation that this status reflects
  frontierReadyReplicas: 6
  frontlasReadyReplicas: 2
  conditions:
    - type: Available
      status: "True"
      reason: AllComponentsReady
      lastTransitionTime: "2026-05-02T01:23:45Z"
      observedGeneration: 7
    - type: Progressing
      status: "False"
      reason: ReconcileSucceeded
    - type: Degraded
      status: "False"`}</code></pre>

        <p>Three conditions are maintained:</p>
        <ul>
          <li><strong>Available</strong> &mdash; True when both Deployments report all replicas ready.</li>
          <li><strong>Progressing</strong> &mdash; True while the operator is still reconciling toward desired state.</li>
          <li><strong>Degraded</strong> &mdash; True when reconcile failed (TLS Secret missing, deployment error, etc.). Inspect <code>kubectl describe fc</code> for the Events stream.</li>
        </ul>

        <h2 id="observability">7. Observability endpoints (since M3)</h2>
        <p>Both <code>frontier</code> and <code>frontlas</code> expose three HTTP endpoints on a separate port:</p>
        <table>
          <thead>
            <tr><th>Endpoint</th><th>Frontier port</th><th>Frontlas port</th><th>Semantics</th></tr>
          </thead>
          <tbody>
            <tr><td><code>/healthz</code></td><td>9091</td><td>9092</td><td>Liveness &mdash; 200 if process responds</td></tr>
            <tr><td><code>/readyz</code></td><td>9091</td><td>9092</td><td>Readiness &mdash; 503 with details when not ready (e.g. Redis unreachable for frontlas)</td></tr>
            <tr><td><code>/metrics</code></td><td>9091</td><td>9092</td><td>Prometheus default registry &mdash; Go runtime + process metrics</td></tr>
          </tbody>
        </table>
        <p>Configure via the <code>observability</code> block in <code>frontier.yaml</code> / <code>frontlas.yaml</code>:</p>
        <pre><code className="language-yaml">{`observability:
  enable: true
  addr: 0.0.0.0:9091`}</code></pre>
        <p>The default behavior is on; set <code>enable: false</code> to disable.</p>

        <h2 id="kubectl">8. Common operations</h2>
        <pre><code className="language-bash">{`# CRUD with the short name
kubectl get fc
kubectl describe fc prod
kubectl edit fc prod
kubectl delete fc prod

# Inspect Conditions
kubectl get fc prod -o jsonpath='{.status.conditions}' | jq

# Watch reconcile events
kubectl describe fc prod | tail -20

# Patch the replica count without an editor
kubectl patch fc prod --type=merge -p '{"spec":{"frontier":{"replicas":4}}}'`}</code></pre>

        <h2 id="behavior">9. Operator behavior</h2>
        <ul>
          <li><strong>Reconcile order.</strong> Service &rarr; TLS Secrets &rarr; Frontlas Deployment &rarr; (wait until ready) &rarr; Frontier Deployment.</li>
          <li><strong>Owner references.</strong> Deployments + Services + operator-managed Secrets all carry the FrontierCluster as owner; deleting the CR cascades to all of them.</li>
          <li><strong>Graceful shutdown.</strong> Frontier honors <code>FRONTIER_DRAIN_SECONDS</code> (operator injects <code>terminationGracePeriodSeconds - 10</code>): on SIGTERM it waits this many seconds before tearing connections down, letting kube-proxy fully drop the pod from Service Endpoints first.</li>
          <li><strong>Events.</strong> Each meaningful state transition emits a Kubernetes Event: <code>ServiceEnsureFailed</code>, <code>TLSEnsureFailed</code>, <code>DeploymentEnsureFailed</code>, <code>Available</code> (one-shot when the cluster first becomes ready).</li>
        </ul>

        <h2 id="troubleshooting">10. Troubleshooting</h2>
        <table>
          <thead>
            <tr><th>Symptom</th><th>Likely cause</th><th>Where to look</th></tr>
          </thead>
          <tbody>
            <tr><td>Frontier pod CrashLoopBackOff with <code>connect: connection refused</code> on the frontier-plane port</td><td>Frontlas not yet ready, or Redis unreachable from frontlas</td><td><code>kubectl describe fc</code> Conditions; <code>kubectl logs deploy/{`<name>`}-frontlas</code></td></tr>
            <tr><td>Frontier pod fails to start: container can&apos;t run as nonRoot</td><td>Custom image without a non-root USER directive</td><td>Override <code>spec.frontier.pod.podSecurityContext</code> + <code>containerSecurityContext</code>, or use <code>singchia/frontier:1.2.4+</code> which ships with USER 65532</td></tr>
            <tr><td>Status stays Pending for minutes</td><td>One of the Deployments not converging on ready replicas</td><td><code>kubectl describe fc</code> + <code>kubectl get pods</code> + pod Events</td></tr>
            <tr><td>TLS-enabled cluster can&apos;t serve mTLS</td><td>Missing <code>ca.crt</code> in the user CA Secret, or the operator-managed Secret was deleted manually</td><td>Operator log: <code>Error ensuring tls secret</code>; check user Secret keys exactly match <code>tls.crt</code>, <code>tls.key</code>, <code>ca.crt</code></td></tr>
            <tr><td>Cluster keeps re-reconciling but never settles</td><td>Some required spec field changed (e.g. ServiceType) and K8s rejects the update</td><td>Operator log + <code>kubectl get events</code></td></tr>
            <tr><td>Redis password is visible in <code>kubectl describe pod</code></td><td>Using deprecated <code>spec.frontlas.redis.password</code> instead of <code>passwordSecret</code></td><td>Move to <code>passwordSecret</code> &mdash; injected via <code>valueFrom.secretKeyRef</code> with no plaintext leak</td></tr>
          </tbody>
        </table>

        <h2 id="limitations">11. Known limitations</h2>
        <ul>
          <li><strong>No <code>kubectl scale</code></strong> &mdash; the spec has two replica fields (frontier &amp; frontlas) so the scale subresource isn&apos;t enabled. Patch <code>spec.frontier.replicas</code> directly. HPA targets the underlying Deployments instead.</li>
          <li><strong>v1alpha1</strong> &mdash; no compatibility guarantees between alpha versions. The next bump goes to <code>v1beta1</code> alongside conversion machinery.</li>
          <li><strong>Helm chart only ships frontier templates</strong> &mdash; the operator path (this page) is the recommended deployment route. Helm-only users should bring their own frontlas + Redis manifests until the chart catches up.</li>
          <li><strong>No webhook validation</strong> &mdash; bad input (e.g. negative replicas, invalid <code>redisType</code>) is caught at reconcile time, not at <code>kubectl apply</code>.</li>
        </ul>

        <h2 id="roadmap">12. Roadmap</h2>
        <p>This page reflects RFC-001 &ldquo;Cloud-native optimization&rdquo; through M3 (observability) and M4 (Status conditions + EventRecorder + CRD ergonomics). Open RFC content lives at <code>docs/rfc/RFC-001-cloud-native-optimization.md</code> in the repository.</p>

      </div>
    </div>
  );
}
