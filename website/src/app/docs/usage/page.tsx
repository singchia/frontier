import Link from 'next/link';

export default function UsageGuide() {
  return (
    <div className="pb-16 max-w-4xl mx-auto">
      <div className="mb-12">
        <h1 className="text-4xl font-bold text-white mb-4">Usage Guide</h1>
        <p className="text-xl text-zinc-400">
          Learn the fastest mental model first, then copy the service-side and edge-side patterns you need.
        </p>
      </div>

      <div className="grid md:grid-cols-3 gap-4 mb-12">
        <Link href="/examples" className="rounded-2xl border border-zinc-800 bg-[#111] p-5 hover:border-zinc-700 transition-colors">
          <div className="text-sm text-blue-400 mb-2">Fastest path</div>
          <h2 className="text-lg font-bold text-white mb-2">Run examples</h2>
          <p className="text-sm text-zinc-400">Use chatroom for messaging and presence, or RTMP for streams.</p>
        </Link>
        <div className="rounded-2xl border border-zinc-800 bg-[#111] p-5">
          <div className="text-sm text-emerald-400 mb-2">Mental model</div>
          <p className="text-sm text-zinc-300 leading-relaxed">Services connect to <code>:30011</code>. Edges connect to <code>:30012</code>. Service -&gt; edge usually targets a specific <code>edgeID</code>.</p>
        </div>
        <div className="rounded-2xl border border-zinc-800 bg-[#111] p-5">
          <div className="text-sm text-cyan-400 mb-2">Why it feels different</div>
          <p className="text-sm text-zinc-300 leading-relaxed">Frontier uses one service-to-edge model for RPC, messaging, and streams instead of treating them as separate systems.</p>
        </div>
      </div>

      <div className="prose prose-invert prose-blue max-w-none prose-pre:bg-[#0D0D0D] prose-pre:border prose-pre:border-zinc-800">
        <h2>Start with the right example</h2>
        <ul>
          <li>Choose <strong>chatroom</strong> if you want to understand presence, messaging, and the basic long-lived connection model.</li>
          <li>Choose <strong>rtmp</strong> if you care about proxying traffic, file transfer, media relay, or other point-to-point stream use cases.</li>
          <li>Return to this page once you know which SDK pattern you need to copy into production code.</li>
        </ul>

        <h2>Using Frontier in Microservices</h2>

        <h3>1. Getting the Service Client</h3>
        <p>To connect your microservice to Frontier, use the <code>NewService</code> method with a dialer connecting to the Service port (default: 30011).</p>
        <pre><code className="language-go">{`package main

import (
    "net"
    "github.com/singchia/frontier/api/dataplane/v1/service"
)

func main() {
    dialer := func() (net.Conn, error) {
        return net.Dial("tcp", "127.0.0.1:30011")
    }
    svc, err := service.NewService(dialer)
    // Start using the service
}`}</code></pre>

        <h3>2. Handling Edge Presence (Online/Offline)</h3>
        <p>You can register callbacks to know when Edge nodes connect or disconnect from the gateway.</p>
        <pre><code className="language-go">{`func main() {
    // ... dialer setup ...
    svc, _ := service.NewService(dialer)

    // Register lifecycle hooks
    svc.RegisterGetEdgeID(context.TODO(), getID)
    svc.RegisterEdgeOnline(context.TODO(), online)
    svc.RegisterEdgeOffline(context.TODO(), offline)
}

// Service can assign IDs based on edge metadata
func getID(meta []byte) (uint64, error) {
    return 0, nil
}

func online(edgeID uint64, meta []byte, addr net.Addr) error {
    fmt.Printf("Edge %d is online\\n", edgeID)
    return nil
}

func offline(edgeID uint64, meta []byte, addr net.Addr) error {
    fmt.Printf("Edge %d is offline\\n", edgeID)
    return nil
}`}</code></pre>

        <h3>3. Service to Edge RPC (Command & Control)</h3>
        <p>A Service can execute a method directly on an Edge node by its <code>edgeID</code>.</p>
        <pre><code className="language-go">{`func main() {
    // ... dialer setup ...
    svc, _ := service.NewService(dialer)

    req := svc.NewRequest([]byte("payload data"))

    // Call the "reboot" method on Edge ID 1001
    rsp, err := svc.Call(context.TODO(), 1001, "reboot", req)
    if err != nil {
        log.Fatal(err)
    }
}`}</code></pre>

        <hr className="my-12 border-zinc-800" />

        <h2>Using Frontier on Edge Nodes</h2>

        <h3>1. Getting the Edge Client</h3>
        <p>Edge nodes (IoT devices, agents, clients) connect to the Edge port (default: 30012). You must provide a unique Edge ID if the Service isn&apos;t allocating them.</p>
        <pre><code className="language-go">{`package main

import (
    "net"
    "github.com/singchia/frontier/api/dataplane/v1/edge"
)

func main() {
    dialer := func() (net.Conn, error) {
        return net.Dial("tcp", "127.0.0.1:30012")
    }

    // Connect with a specific Edge ID
    opt := edge.OptionEdgeID("edge-node-01")
    eg, err := edge.NewEdge(dialer, opt)
    if err != nil {
        panic(err)
    }
}`}</code></pre>

        <h3>2. Edge Registers RPC for Service to Call</h3>
        <p>For the Service to be able to issue commands to the Edge, the Edge must register methods.</p>
        <pre><code className="language-go">{`func main() {
    // ... edge setup ...

    // Register the "reboot" method
    eg.Register(context.TODO(), "reboot", handleReboot)
}

func handleReboot(ctx context.Context, req geminio.Request, rsp geminio.Response) {
    // Execute reboot logic
    log.Println("Received reboot command")

    // Send response back
    rsp.SetData([]byte("rebooting..."))
}`}</code></pre>

        <h3>3. Edge Publishes Telemetry (Messaging)</h3>
        <p>Edges can publish data to specific Topics. Frontier routes this to Services that have subscribed to the topic, or forwards it to an external MQ like Kafka.</p>
        <pre><code className="language-go">{`func main() {
    // ... edge setup ...

    msg := eg.NewMessage([]byte(\`{"temp": 42.5}\`))

    // Publish to the "sensor/temperature" topic
    err := eg.Publish(context.TODO(), "sensor/temperature", msg)
}`}</code></pre>

        <hr className="my-12 border-zinc-800" />

        <h2>Advanced: Point-to-Point Streams</h2>
        <p>If you need to transfer large files or proxy custom protocol traffic such as SSH, VNC, or RTMP, you can open a direct multiplexed stream.</p>

        <h3>Service Opening a Stream to an Edge</h3>
        <pre><code className="language-go">{`func main() {
    // ... service setup ...

    // Open a new stream to Edge ID 1001.
    // st implements net.Conn, so you can use it like a raw TCP socket!
    st, err := svc.OpenStream(context.TODO(), 1001)

    // Proxy SSH traffic, relay media, or write files directly
    io.Copy(st, fileData)
}`}</code></pre>

      </div>
    </div>
  );
}
