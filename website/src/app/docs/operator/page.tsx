export default function OperatorGuide() {
  return (
    <div className="pb-16 max-w-4xl mx-auto">
      <div className="mb-12">
        <h1 className="text-4xl font-bold text-white mb-4">Kubernetes Operator</h1>
        <p className="text-xl text-zinc-400">
          The official Kubernetes Operator for deploying Frontier clusters.
        </p>
      </div>

      <div className="prose prose-invert prose-blue max-w-none prose-pre:bg-[#0D0D0D] prose-pre:border prose-pre:border-zinc-800">
        <p>
          The Frontier Operator automates the deployment, provisioning, and scaling of Frontier and Frontlas clusters inside Kubernetes.
        </p>

        <h2>1. Install the CRD and Operator</h2>
        <p>First, clone the repository and apply the Custom Resource Definitions (CRDs):</p>
        <pre><code className="language-bash">{`git clone https://github.com/singchia/frontier.git
cd frontier/dist/crd
kubectl apply -f install.yaml`}</code></pre>

        <p>Verify that the CRD is installed:</p>
        <pre><code className="language-bash">{`kubectl get crd frontierclusters.frontier.singchia.io`}</code></pre>

        <p>Verify that the Operator is running:</p>
        <pre><code className="language-bash">{`kubectl get all -n frontier-system`}</code></pre>

        <h2>2. Deploy a FrontierCluster</h2>
        <p>Create a file named <code>frontiercluster.yaml</code>. This example deploys 2 Frontier gateways and 1 Frontlas control plane relying on an existing Redis sentinel.</p>

        <pre><code className="language-yaml">{`apiVersion: frontier.singchia.io/v1alpha1
kind: FrontierCluster
metadata:
  name: my-frontier-cluster
spec:
  frontier:
    replicas: 2
    servicebound:
      port: 30011
    edgebound:
      port: 30012
  frontlas:
    replicas: 1
    controlplane:
      port: 40011
    redis:
      addrs:
        - rfs-redisfailover:26379
      password: your-password
      masterName: mymaster
      redisType: sentinel`}</code></pre>

        <p>Apply the cluster configuration:</p>
        <pre><code className="language-bash">{`kubectl apply -f frontiercluster.yaml`}</code></pre>

        <h2>3. Verify the Cluster</h2>
        <p>Within a minute, your HA cluster will be ready. You can check the status of the pods:</p>
        <pre><code className="language-bash">{`kubectl get all -l app=frontiercluster-frontier
kubectl get all -l app=frontiercluster-frontlas`}</code></pre>

        <h2>4. Connect your workloads</h2>
        <ul>
          <li><strong>Microservices:</strong> Should connect to <code>service/frontiercluster-frontlas-svc:40011</code></li>
          <li><strong>Edge Nodes:</strong> Can connect to the NodePort where <code>:30012</code> is exposed externally.</li>
        </ul>
      </div>
    </div>
  );
}