export default function ConfigurationGuide() {
  return (
    <div className="pb-16 max-w-4xl mx-auto">
      <div className="mb-12">
        <h1 className="text-4xl font-bold text-white mb-4">Configuration</h1>
        <p className="text-xl text-zinc-400">
          Customize your Frontier gateway with <code>frontier.yaml</code>.
        </p>
      </div>

      <div className="prose prose-invert prose-blue max-w-none prose-pre:bg-[#0D0D0D] prose-pre:border prose-pre:border-zinc-800">
        <p>
          Save your configuration as <code>frontier.yaml</code> and mount it to the container at <code>/usr/conf/frontier.yaml</code> to take effect.
        </p>

        <h2>Minimal Configuration</h2>
        <p>To get started, you only need to configure the listening addresses for microservices and edge nodes:</p>
        <pre><code className="language-yaml">{`# Microservice configuration
servicebound:
  listen:
    network: tcp
    addr: 0.0.0.0:30011

# Edge node configuration
edgebound:
  listen:
    network: tcp
    addr: 0.0.0.0:30012
  # Allow Frontier to allocate edgeID if no ID service is registered
  edgeid_alloc_when_no_idservice_on: true`}</code></pre>

        <h2>TLS and mTLS</h2>
        <p>Frontier supports TLS for both microservices and edge nodes. It also supports mTLS where Frontier verifies the client certificate.</p>
        <pre><code className="language-yaml">{`edgebound:
  listen:
    addr: 0.0.0.0:30012
    network: tcp
    tls:
      enable: true
      certs:
        - cert: edgebound.cert
          key: edgebound.key
      insecure_skip_verify: false

      # Enable mTLS: verify client certificates against a CA
      mtls: true
      ca_certs:
        - ca1.cert`}</code></pre>

        <h2>External Message Queue (MQ) Routing</h2>
        <p>Frontier can automatically forward messages published by Edge nodes directly to external Message Queues based on topics.</p>

        <h3>Kafka</h3>
        <p>If an Edge node publishes to a topic listed in this array, Frontier will forward it to Kafka. If microservices also declare the same topic, Frontier hashes the traffic between them.</p>
        <pre><code className="language-yaml">{`mqm:
  kafka:
    enable: true
    addrs:
      - "kafka:9092"
    producer:
      topics:
        - "sensor/temperature"
        - "device/events"`}</code></pre>

        <h3>Redis</h3>
        <pre><code className="language-yaml">{`mqm:
  redis:
    enable: true
    addrs:
      - "redis:6379"
    db: 0
    password: "your-password"
    producer:
      channels:
        - "alerts"
        - "metrics"`}</code></pre>
        <p>Frontier also supports <strong>AMQP (RabbitMQ)</strong>, <strong>NATS</strong> (with Jetstream support), and <strong>NSQ</strong>.</p>

        <h2>Routing Strategy & Database</h2>
        <pre><code className="language-yaml">{`dao:
  # Supports buntdb and sqlite3. Both use in-memory mode to keep Frontier stateless.
  # buntdb is recommended for maximum performance.
  backend: buntdb

exchange:
  # How Frontier hashes and routes traffic when multiple microservices subscribe
  # to the same topic/method. Options: edgeid, srcip, random
  hashby: edgeid`}</code></pre>
      </div>
    </div>
  );
}