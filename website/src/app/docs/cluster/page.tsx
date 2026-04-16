export default function ClusterArchitecture() {
  return (
    <div className="pb-16 max-w-4xl mx-auto">
      <div className="mb-12">
        <h1 className="text-4xl font-bold text-white mb-4">Cluster Architecture</h1>
        <p className="text-xl text-zinc-400">
          Scale Frontier horizontally to handle millions of connections using Frontlas.
        </p>
      </div>

      <div className="prose prose-invert prose-blue max-w-none prose-img:rounded-xl prose-img:border prose-img:border-zinc-800">
        <p>
          While a standalone Frontier instance is powerful, production environments requiring High Availability (HA) and horizontal scaling should deploy the <strong>Frontier + Frontlas</strong> cluster architecture.
        </p>

        <img src="/docs/diagram/frontlas.png" alt="Frontlas Cluster Architecture" />

        <h2>What is Frontlas?</h2>
        <p>
          <strong>Frontlas</strong> (Frontier Atlas) is a stateless cluster management component. It acts as the registry and control plane for multiple Frontier gateway instances.
        </p>
        <ul>
          <li><strong>Stateless:</strong> Frontlas does not store state in memory. It uses Redis to maintain metadata and active connection states.</li>
          <li><strong>Port 40011:</strong> Microservices connect here to discover available Edge nodes across the entire cluster.</li>
          <li><strong>Port 40012:</strong> Frontier instances connect here to report their status and the status of edges connected to them.</li>
        </ul>

        <h2>How it works</h2>
        <ol>
          <li>You can deploy any number of Frontier instances.</li>
          <li>Each Frontier instance connects to Frontlas to report heartbeat and metadata.</li>
          <li>Edge nodes connect to any available Frontier instance (usually balanced by a load balancer or NodePort).</li>
          <li>Microservices query Frontlas to find out which Frontier instance currently holds the connection to the target Edge node, and then establishes communication.</li>
        </ol>

        <h2>Cluster Configuration</h2>

        <h3>1. Frontier Configuration</h3>
        <p>In your <code>frontier.yaml</code>, enable Frontlas and set the instance ID:</p>
        <pre><code className="language-yaml">{`frontlas:
  enable: true
  dial:
    network: tcp
    addr:
      - 127.0.0.1:40012

daemon:
  # Must be unique within the cluster
  frontier_id: frontier01`}</code></pre>

        <h3>2. Frontlas Configuration</h3>
        <p>Configure <code>frontlas.yaml</code> to point to your Redis instance:</p>
        <pre><code className="language-yaml">{`control_plane:
  listen:
    network: tcp
    addr: 0.0.0.0:40011

frontier_plane:
  listen:
    network: tcp
    addr: 0.0.0.0:40012

redis:
  mode: standalone # Supports standalone, sentinel, and cluster
  standalone:
    network: tcp
    addr: redis:6379
    db: 0`}</code></pre>

        <h2>Updating Microservices for Cluster Mode</h2>
        <p>When running in cluster mode, microservices must initialize using <code>NewClusterService</code> pointing to Frontlas, rather than pointing to a single Frontier instance.</p>
        <pre><code className="language-go">{`package main

import (
    "github.com/singchia/frontier/api/dataplane/v1/service"
)

func main() {
    // Point to Frontlas (Port 40011) instead of Frontier (Port 30011)
    svc, err := service.NewClusterService("127.0.0.1:40011")

    // The rest of the usage remains exactly the same!
    // svc.Publish(...)
    // svc.Call(...)
}`}</code></pre>
      </div>
    </div>
  );
}