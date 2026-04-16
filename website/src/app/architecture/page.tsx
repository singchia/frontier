export default function Architecture() {
  return (
    <div className="w-full max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-16">
      <div className="max-w-3xl mb-16">
        <h1 className="text-4xl md:text-5xl font-bold text-white mb-6">Architecture</h1>
        <p className="text-xl text-slate-400">
          A deep dive into how Frontier routes traffic, manages state, and scales.
        </p>
      </div>

      <div className="bg-white/5 border border-white/10 rounded-2xl p-8 mb-16 flex flex-col items-center justify-center min-h-[400px]">
        {/* Placeholder for the architecture diagram */}
        <img src="/docs/diagram/frontier.png" alt="Frontier Architecture" className="w-full max-w-4xl rounded-lg" />
      </div>

      <div className="grid md:grid-cols-2 gap-12">
        <div>
          <h2 className="text-2xl font-bold text-white mb-6">Connection Model</h2>
          <p className="text-slate-400 mb-4">
            Unlike traditional gateways where clients connect to the gateway and the gateway connects to upstream services, in Frontier, <strong>both microservices and edge nodes actively connect to Frontier</strong>.
          </p>
          <ul className="space-y-3 text-slate-400 list-disc list-inside">
            <li><strong className="text-white">Port 30011:</strong> For microservices to connect.</li>
            <li><strong className="text-white">Port 30012:</strong> For edge nodes to connect.</li>
            <li><strong className="text-white">Port 30010:</strong> Control plane APIs for operators.</li>
          </ul>
        </div>

        <div>
          <h2 className="text-2xl font-bold text-white mb-6">Routing Model</h2>
          <p className="text-slate-400 mb-4">
            All Messages, RPCs, and Streams are point-to-point transmissions:
          </p>
          <ul className="space-y-3 text-slate-400 list-disc list-inside">
            <li><strong>Service to Edge:</strong> Must specify the Edge ID.</li>
            <li><strong>Edge to Service:</strong> Frontier routes based on Topic and Method, selecting a service via hashing (default by edgeid, or random/srcip).</li>
            <li><strong>Multiplexer Streams:</strong> Bypass routing completely for direct byte-level proxying.</li>
          </ul>
        </div>
      </div>

      <div className="mt-16 pt-16 border-t border-white/10">
        <h2 className="text-2xl font-bold text-white mb-6">Consistency & Reliability</h2>
        <div className="bg-blue-500/10 border border-blue-500/20 p-6 rounded-xl">
          <h3 className="text-lg font-bold text-blue-400 mb-2">Explicit Acknowledgment</h3>
          <p className="text-slate-300">
            To ensure message delivery semantics, Frontier requires the receiving end to explicitly call <code>msg.Done()</code> or <code>msg.Error(err)</code>. This guarantees consistency across distributed edge environments.
          </p>
        </div>
      </div>
    </div>
  );
}
