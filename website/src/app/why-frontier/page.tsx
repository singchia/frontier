import { CheckCircle2, XCircle, Zap, Cpu, Network, Server } from 'lucide-react';

export default function WhyFrontier() {
  return (
    <div className="w-full max-w-7xl mx-auto px-6 lg:px-8 py-24">
      <div className="max-w-3xl mb-24">
        <div className="inline-flex items-center gap-2 rounded-full px-4 py-1.5 text-sm font-medium text-emerald-400 bg-emerald-500/10 border border-emerald-500/20 mb-6">
          <Zap className="w-4 h-4" /> The structural advantage
        </div>
        <h1 className="text-4xl md:text-6xl font-extrabold tracking-tighter text-white mb-6">
          Why build with Frontier?
        </h1>
        <p className="text-xl text-zinc-400 leading-relaxed font-light">
          Understand the fundamental differences between Frontier and traditional reverse proxies or message queues, and see why it excels in edge computing scenarios.
        </p>
      </div>

      {/* Comparison Table */}
      <section className="mb-32">
        <div className="flex items-center gap-4 mb-8">
          <div className="w-10 h-10 rounded-xl bg-zinc-800 flex items-center justify-center border border-zinc-700">
            <svg className="w-5 h-5 text-zinc-300" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z"></path></svg>
          </div>
          <h2 className="text-2xl md:text-3xl font-bold text-white tracking-tight">Structural Differences</h2>
        </div>

        <div className="overflow-x-auto rounded-3xl border border-zinc-800 bg-[#0D0D0D] shadow-2xl">
          <table className="w-full text-left border-collapse min-w-[800px]">
            <thead>
              <tr className="border-b border-zinc-800 bg-zinc-900/50">
                <th className="py-6 px-8 text-sm font-medium text-zinc-400 w-1/4">Capability</th>
                <th className="py-6 px-8 text-sm font-semibold text-blue-400 bg-blue-950/20 w-1/4">Frontier</th>
                <th className="py-6 px-8 text-sm font-medium text-zinc-500 w-1/4">Reverse Proxy (Nginx/Envoy)</th>
                <th className="py-6 px-8 text-sm font-medium text-zinc-500 w-1/4">Message Queue (Kafka/RabbitMQ)</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-zinc-800/50 text-sm">
              <tr className="hover:bg-zinc-900/30 transition-colors">
                <td className="py-6 px-8 font-medium text-white">Full-Duplex Native</td>
                <td className="py-6 px-8 bg-blue-950/10">
                  <div className="flex items-center text-blue-400 font-medium"><CheckCircle2 className="w-5 h-5 mr-3" /> Yes (Bi-directional RPC)</div>
                </td>
                <td className="py-6 px-8">
                  <div className="flex items-center text-zinc-500"><XCircle className="w-5 h-5 mr-3" /> Client to Server only</div>
                </td>
                <td className="py-6 px-8">
                  <div className="flex items-center text-zinc-500"><XCircle className="w-5 h-5 mr-3" /> Pub/Sub only, no RPC</div>
                </td>
              </tr>
              <tr className="hover:bg-zinc-900/30 transition-colors">
                <td className="py-6 px-8 font-medium text-white">Edge Node Presence</td>
                <td className="py-6 px-8 bg-blue-950/10">
                  <div className="flex items-center text-blue-400 font-medium"><CheckCircle2 className="w-5 h-5 mr-3" /> Built-in Online State</div>
                </td>
                <td className="py-6 px-8">
                  <div className="flex items-center text-zinc-500"><XCircle className="w-5 h-5 mr-3" /> Stateless</div>
                </td>
                <td className="py-6 px-8">
                  <div className="flex items-center text-zinc-500"><XCircle className="w-5 h-5 mr-3" /> Topic-based only</div>
                </td>
              </tr>
              <tr className="hover:bg-zinc-900/30 transition-colors">
                <td className="py-6 px-8 font-medium text-white">Point-to-Point Stream</td>
                <td className="py-6 px-8 bg-blue-950/10">
                  <div className="flex items-center text-blue-400 font-medium"><CheckCircle2 className="w-5 h-5 mr-3" /> Direct multiplexed streams</div>
                </td>
                <td className="py-6 px-8">
                  <div className="flex items-center text-zinc-500"><XCircle className="w-5 h-5 mr-3" /> Proxy forwarding only</div>
                </td>
                <td className="py-6 px-8">
                  <div className="flex items-center text-zinc-500"><XCircle className="w-5 h-5 mr-3" /> Not supported</div>
                </td>
              </tr>
              <tr className="hover:bg-zinc-900/30 transition-colors">
                <td className="py-6 px-8 font-medium text-white">Control Plane API</td>
                <td className="py-6 px-8 bg-blue-950/10">
                  <div className="flex items-center text-blue-400 font-medium"><CheckCircle2 className="w-5 h-5 mr-3" /> Query online nodes & state</div>
                </td>
                <td className="py-6 px-8">
                  <div className="flex items-center text-zinc-500"><XCircle className="w-5 h-5 mr-3" /> Config management only</div>
                </td>
                <td className="py-6 px-8">
                  <div className="flex items-center text-zinc-500"><XCircle className="w-5 h-5 mr-3" /> Consumer group metrics only</div>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </section>

      {/* Target Scenarios */}
      <section>
        <div className="flex items-center gap-4 mb-12">
          <div className="w-10 h-10 rounded-xl bg-zinc-800 flex items-center justify-center border border-zinc-700">
            <svg className="w-5 h-5 text-zinc-300" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M19.428 15.428a2 2 0 00-1.022-.547l-2.387-.477a6 6 0 00-3.86.517l-.318.158a6 6 0 01-3.86.517L6.05 15.21a2 2 0 00-1.806.547M8 4h8l-1 1v5.172a2 2 0 00.586 1.414l5 5c1.26 1.26.367 3.414-1.415 3.414H4.828c-1.782 0-2.674-2.154-1.414-3.414l5-5A2 2 0 009 10.172V5L8 4z"></path></svg>
          </div>
          <h2 className="text-2xl md:text-3xl font-bold text-white tracking-tight">Purpose-built for Edge Scenarios</h2>
        </div>

        <div className="grid md:grid-cols-2 gap-6">
          <div className="bg-[#0D0D0D] border border-zinc-800 p-8 rounded-3xl hover:border-zinc-700 transition-colors group">
            <Cpu className="w-8 h-8 text-blue-400 mb-6 group-hover:scale-110 transition-transform" />
            <h3 className="text-xl font-bold text-white mb-3">IoT & Edge Devices</h3>
            <p className="text-zinc-400 leading-relaxed">Maintain massive numbers of long connections. Services need to push commands to specific devices (RPC) and devices need to upload telemetry (Messaging) efficiently.</p>
          </div>

          <div className="bg-[#0D0D0D] border border-zinc-800 p-8 rounded-3xl hover:border-zinc-700 transition-colors group">
            <Network className="w-8 h-8 text-purple-400 mb-6 group-hover:scale-110 transition-transform" />
            <h3 className="text-xl font-bold text-white mb-3">Remote Access & Tunneling</h3>
            <p className="text-zinc-400 leading-relaxed">Need to expose a service behind NAT? Devices connect out to Frontier, and services can open a point-to-point stream to the device safely behind the firewall.</p>
          </div>

          <div className="bg-[#0D0D0D] border border-zinc-800 p-8 rounded-3xl hover:border-zinc-700 transition-colors group">
            <Zap className="w-8 h-8 text-amber-400 mb-6 group-hover:scale-110 transition-transform" />
            <h3 className="text-xl font-bold text-white mb-3">Realtime Chat & Sync</h3>
            <p className="text-zinc-400 leading-relaxed">Clients maintain persistent presence. Services can route messages to specific clients based on Edge ID, with guaranteed delivery acknowledgment built into the protocol.</p>
          </div>

          <div className="bg-[#0D0D0D] border border-zinc-800 p-8 rounded-3xl hover:border-zinc-700 transition-colors group">
            <Server className="w-8 h-8 text-emerald-400 mb-6 group-hover:scale-110 transition-transform" />
            <h3 className="text-xl font-bold text-white mb-3">Multi-tenant Agents</h3>
            <p className="text-zinc-400 leading-relaxed">Deploying agents into customer VPCs? Use Frontier as the central control plane where all agents connect, allowing cloud services to manage them securely without exposing internal ports.</p>
          </div>
        </div>
      </section>
    </div>
  );
}
