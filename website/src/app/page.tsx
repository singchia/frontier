import Link from 'next/link';
import { ArrowRight, Shield, Activity, Zap, Workflow, CheckCircle2 } from 'lucide-react';

export default function Home() {
  return (
    <div className="flex flex-col items-center">
      {/* Hero Section */}
      <section className="relative w-full max-w-7xl mx-auto px-6 lg:px-8 pt-32 pb-24 flex flex-col items-center text-center">
        <div className="absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 w-[800px] h-[500px] bg-blue-500/20 blur-[120px] rounded-full pointer-events-none"></div>

        <a href="https://github.com/singchia/frontier/releases" target="_blank" rel="noreferrer" className="inline-flex items-center gap-2 rounded-full px-4 py-1.5 text-sm font-medium text-blue-400 bg-blue-500/10 border border-blue-500/20 mb-8 hover:bg-blue-500/20 transition-colors cursor-pointer">
          <span className="flex h-2 w-2 rounded-full bg-blue-500 animate-pulse"></span>
          Latest stable release: Frontier v1.2.2
          <ArrowRight className="w-4 h-4" />
        </a>

        <h1 className="text-6xl md:text-8xl font-extrabold tracking-tighter text-white mb-8 leading-[1.1]">
          Backend services need to <br />
          <span className="text-transparent bg-clip-text bg-gradient-to-r from-blue-400 via-cyan-300 to-emerald-300">
            reach online edge nodes
          </span>
        </h1>

        <p className="mt-4 text-xl md:text-2xl text-zinc-400 max-w-3xl mx-auto mb-12 font-light leading-relaxed">
          Frontier is a service-to-edge gateway for long-lived connections. Use it when backend services and edge nodes both need to actively call, notify, and open streams to each other.
        </p>

        <div className="flex flex-col sm:flex-row gap-4 justify-center items-center w-full sm:w-auto">
          <a href="https://github.com/singchia/frontier" target="_blank" rel="noreferrer" className="group w-full sm:w-auto rounded-xl bg-zinc-100 px-8 py-4 text-sm font-semibold text-zinc-900 shadow-sm hover:bg-white focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-white transition-all flex items-center justify-center gap-2">
            Star on GitHub
            <ArrowRight className="w-4 h-4 group-hover:translate-x-1 transition-transform" />
          </a>
          <Link href="/examples" className="w-full sm:w-auto rounded-xl bg-zinc-900/50 px-8 py-4 text-sm font-semibold text-white hover:bg-zinc-800 border border-zinc-800 transition-all flex items-center justify-center gap-2 backdrop-blur-sm">
            Run the 3-minute demos
          </Link>
          <Link href="/docs" className="w-full sm:w-auto rounded-xl bg-zinc-900/50 px-8 py-4 text-sm font-semibold text-white hover:bg-zinc-800 border border-zinc-800 transition-all flex items-center justify-center gap-2 backdrop-blur-sm">
            Read the docs
          </Link>
        </div>

        <div className="mt-8 flex flex-wrap items-center justify-center gap-3 text-xs sm:text-sm text-zinc-400">
          <span className="rounded-full border border-zinc-800 bg-zinc-900/60 px-3 py-1.5">Service -&gt; specific edge RPC</span>
          <span className="rounded-full border border-zinc-800 bg-zinc-900/60 px-3 py-1.5">Edge -&gt; service callbacks</span>
          <span className="rounded-full border border-zinc-800 bg-zinc-900/60 px-3 py-1.5">Messaging + streams on one data plane</span>
        </div>
      </section>

      {/* Terminal Preview Section */}
      <section className="w-full max-w-5xl mx-auto px-6 lg:px-8 pb-32">
        <div className="rounded-2xl bg-[#0D0D0D] border border-zinc-800 shadow-2xl shadow-black/50 overflow-hidden">
          <div className="flex items-center px-4 py-3 border-b border-zinc-800 bg-[#111]">
            <div className="flex gap-2">
              <div className="w-3 h-3 rounded-full bg-red-500/80"></div>
              <div className="w-3 h-3 rounded-full bg-yellow-500/80"></div>
              <div className="w-3 h-3 rounded-full bg-green-500/80"></div>
            </div>
            <div className="mx-auto text-xs text-zinc-500 font-mono">frontier-start.sh</div>
          </div>
          <div className="p-6 font-mono text-sm leading-relaxed overflow-x-auto">
            <div className="flex gap-4">
              <span className="text-zinc-600 select-none">1</span>
              <span className="text-zinc-400"># Start Frontier and run the fastest demo path</span>
            </div>
            <div className="flex gap-4">
              <span className="text-zinc-600 select-none">2</span>
              <span className="text-emerald-400">docker</span> <span className="text-zinc-300">run -d --name frontier \</span>
            </div>
            <div className="flex gap-4">
              <span className="text-zinc-600 select-none">3</span>
              <span className="text-zinc-300">  -p 30011:30011</span> <span className="text-zinc-500"># Service-bound port</span>
            </div>
            <div className="flex gap-4">
              <span className="text-zinc-600 select-none">4</span>
              <span className="text-zinc-300">  -p 30012:30012</span> <span className="text-zinc-500"># Edge-bound port</span>
            </div>
            <div className="flex gap-4">
              <span className="text-zinc-600 select-none">5</span>
              <span className="text-blue-300">  singchia/frontier:1.2.2</span>
            </div>
            <div className="flex gap-4">
              <span className="text-zinc-600 select-none">6</span>
              <span className="text-emerald-400">make</span> <span className="text-zinc-300">examples</span>
            </div>
            <div className="flex gap-4">
              <span className="text-zinc-600 select-none">7</span>
              <span className="text-zinc-300">./bin/chatroom_service</span> <span className="text-zinc-500"># service side</span>
            </div>
            <div className="flex gap-4">
              <span className="text-zinc-600 select-none">8</span>
              <span className="text-zinc-300">./bin/chatroom_agent</span> <span className="text-zinc-500"># edge side</span>
            </div>
          </div>
        </div>
      </section>

      <section className="w-full max-w-7xl mx-auto px-6 lg:px-8 pb-24">
        <div className="grid lg:grid-cols-2 gap-6">
          <div className="rounded-3xl border border-emerald-500/20 bg-emerald-500/[0.04] p-8">
            <h2 className="text-2xl font-bold text-white mb-5">Use Frontier when</h2>
            <div className="space-y-4">
              {[
                'A backend service needs to call a specific online device, agent, or connector',
                'Edge nodes need to call backend services without opening inbound ports',
                'You need RPC, messaging, and streams on the same long-lived connection model',
                'Your system is service <-> edge, not just service <-> service',
              ].map((item) => (
                <div key={item} className="flex items-start gap-3 text-zinc-300">
                  <CheckCircle2 className="w-5 h-5 text-emerald-400 mt-0.5 flex-shrink-0" />
                  <span>{item}</span>
                </div>
              ))}
            </div>
          </div>

          <div className="rounded-3xl border border-zinc-800 bg-zinc-900/40 p-8">
            <h2 className="text-2xl font-bold text-white mb-5">Do not use Frontier when</h2>
            <div className="space-y-4 text-zinc-400">
              <p>You only need service-to-service RPC. Use gRPC.</p>
              <p>You only need HTTP ingress or proxying. Use Envoy or an API gateway.</p>
              <p>You only need pub/sub or event streaming. Use NATS or Kafka.</p>
              <p>You only need a generic tunnel. Use frp or another tunnel tool.</p>
            </div>
          </div>
        </div>
      </section>

      {/* Core Features */}
      <section className="w-full max-w-7xl mx-auto px-6 lg:px-8 py-24 border-t border-zinc-800/50">
        <div className="text-center mb-16">
          <h2 className="text-3xl md:text-5xl font-bold text-white tracking-tight mb-4">One connection model, three primitives</h2>
          <p className="text-zinc-400 text-lg">The reason Frontier feels different is that RPC, messaging, and streams are part of the same service-to-edge model.</p>
        </div>

        <div className="grid md:grid-cols-3 gap-6">
          <div className="group rounded-3xl bg-zinc-900/40 border border-zinc-800/50 p-8 hover:bg-zinc-800/50 hover:border-zinc-700 transition-all">
            <div className="w-12 h-12 rounded-2xl bg-blue-500/10 flex items-center justify-center mb-6 group-hover:scale-110 group-hover:bg-blue-500/20 transition-all">
              <Activity className="w-6 h-6 text-blue-400" />
            </div>
            <h3 className="text-xl font-bold text-white mb-3">Bidirectional RPC</h3>
            <p className="text-zinc-400 leading-relaxed text-sm">
              Address a specific online edge node from a backend service, or let edge nodes call backend services back over the same communication model.
            </p>
          </div>

          <div className="group rounded-3xl bg-zinc-900/40 border border-zinc-800/50 p-8 hover:bg-zinc-800/50 hover:border-zinc-700 transition-all">
            <div className="w-12 h-12 rounded-2xl bg-purple-500/10 flex items-center justify-center mb-6 group-hover:scale-110 group-hover:bg-purple-500/20 transition-all">
              <Workflow className="w-6 h-6 text-purple-400" />
            </div>
            <h3 className="text-xl font-bold text-white mb-3">Topic Messaging</h3>
            <p className="text-zinc-400 leading-relaxed text-sm">
              Push telemetry, events, and notifications between services and edges, with explicit acknowledgments and optional forwarding to external MQ.
            </p>
          </div>

          <div className="group rounded-3xl bg-zinc-900/40 border border-zinc-800/50 p-8 hover:bg-zinc-800/50 hover:border-zinc-700 transition-all">
            <div className="w-12 h-12 rounded-2xl bg-cyan-500/10 flex items-center justify-center mb-6 group-hover:scale-110 group-hover:bg-cyan-500/20 transition-all">
              <Zap className="w-6 h-6 text-cyan-400" />
            </div>
            <h3 className="text-xl font-bold text-white mb-3">P2P Multiplexing</h3>
            <p className="text-zinc-400 leading-relaxed text-sm">
              Open direct streams for proxying, file transfer, media relay, or custom protocols when RPC is not enough.
            </p>
          </div>
        </div>
      </section>

      {/* Deployment */}
      <section className="w-full border-t border-zinc-800/50 bg-zinc-900/20 py-24">
        <div className="max-w-7xl mx-auto px-6 lg:px-8 text-center">
          <h2 className="text-3xl md:text-4xl font-bold text-white mb-6">Cloud-Native by Design</h2>
          <p className="text-zinc-400 max-w-2xl mx-auto mb-16 text-lg">Start with a single container, then move to clustered deployment when your service-to-edge fleet grows.</p>

          <div className="grid grid-cols-2 md:grid-cols-4 gap-4 max-w-4xl mx-auto">
            <div className="px-6 py-8 bg-[#0A0A0A] border border-zinc-800 rounded-2xl hover:border-zinc-700 transition-colors">
              <span className="text-white font-mono font-medium block mb-2 text-lg">Docker</span>
              <span className="text-sm text-zinc-500">Standalone container</span>
            </div>
            <div className="px-6 py-8 bg-[#0A0A0A] border border-zinc-800 rounded-2xl hover:border-zinc-700 transition-colors">
              <span className="text-white font-mono font-medium block mb-2 text-lg">Compose</span>
              <span className="text-sm text-zinc-500">Local cluster</span>
            </div>
            <div className="px-6 py-8 bg-[#0A0A0A] border border-zinc-800 rounded-2xl hover:border-zinc-700 transition-colors">
              <span className="text-white font-mono font-medium block mb-2 text-lg">Helm</span>
              <span className="text-sm text-zinc-500">K8s deployment</span>
            </div>
            <div className="px-6 py-8 bg-[#0A0A0A] border border-zinc-800 rounded-2xl hover:border-zinc-700 transition-colors relative overflow-hidden">
              <div className="absolute top-0 right-0 p-2">
                <Shield className="w-4 h-4 text-emerald-500/50" />
              </div>
              <span className="text-white font-mono font-medium block mb-2 text-lg">Operator</span>
              <span className="text-sm text-zinc-500">HA & Scale</span>
            </div>
          </div>
        </div>
      </section>
    </div>
  );
}
