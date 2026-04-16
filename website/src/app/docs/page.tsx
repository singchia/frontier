import Link from 'next/link';

export default function DocsOverview() {
  return (
    <div className="pb-16">
      <div className="mb-12">
        <h1 className="text-4xl font-bold text-white mb-4">Frontier Documentation</h1>
        <p className="text-xl text-zinc-400">
          Learn the service-to-edge model, run the fastest demos, and then integrate the SDKs.
        </p>
      </div>

      <p className="text-zinc-300 leading-relaxed mb-10">
        Frontier is easiest to understand in this order: first the communication model, then the examples, then the SDK snippets. If your backend services need to directly reach online edge nodes and edge nodes also need to reach services back, you are in the right place.
      </p>

      <div className="grid md:grid-cols-3 gap-4 mb-14">
        <Link href="/examples" className="rounded-2xl border border-zinc-800 bg-[#111] p-6 hover:border-zinc-700 transition-colors">
          <div className="text-sm text-blue-400 mb-2">Start here</div>
          <h2 className="text-xl font-bold text-white mb-2">Examples</h2>
          <p className="text-sm text-zinc-400">Use chatroom for messaging and presence, or RTMP for stream transport.</p>
        </Link>
        <Link href="/why-frontier" className="rounded-2xl border border-zinc-800 bg-[#111] p-6 hover:border-zinc-700 transition-colors">
          <div className="text-sm text-emerald-400 mb-2">Positioning</div>
          <h2 className="text-xl font-bold text-white mb-2">Why Frontier</h2>
          <p className="text-sm text-zinc-400">See when Frontier is the right model, and when another tool is simpler.</p>
        </Link>
        <Link href="/docs/usage" className="rounded-2xl border border-zinc-800 bg-[#111] p-6 hover:border-zinc-700 transition-colors">
          <div className="text-sm text-cyan-400 mb-2">Integrate</div>
          <h2 className="text-xl font-bold text-white mb-2">Usage Guide</h2>
          <p className="text-sm text-zinc-400">Copy the service-side and edge-side SDK patterns you actually need.</p>
        </Link>
      </div>

      <h2 className="text-2xl font-bold text-white mt-12 mb-6">Quick Start Guide</h2>

      <div className="space-y-12">
        <div className="bg-[#111] border border-zinc-800 rounded-2xl p-8 relative overflow-hidden">
          <div className="absolute top-0 left-0 w-1 h-full bg-blue-500"></div>
          <h3 className="text-lg font-bold text-white mb-4 flex items-center gap-2">
            <span className="flex items-center justify-center w-6 h-6 rounded-full bg-blue-500/20 text-blue-400 text-xs">1</span>
            Start the Gateway
          </h3>
          <p className="text-zinc-400 mb-4 text-sm">Run a standalone Frontier instance using Docker. This exposes the Service port (30011) and the Edge port (30012).</p>
          <div className="bg-[#050505] border border-zinc-800 rounded-xl p-4 font-mono text-sm text-emerald-400 overflow-x-auto">
            docker run -d --name frontier -p 30011:30011 -p 30012:30012 singchia/frontier:1.2.2
          </div>
        </div>

        <div className="bg-[#111] border border-zinc-800 rounded-2xl p-8 relative overflow-hidden">
          <div className="absolute top-0 left-0 w-1 h-full bg-purple-500"></div>
          <h3 className="text-lg font-bold text-white mb-4 flex items-center gap-2">
            <span className="flex items-center justify-center w-6 h-6 rounded-full bg-purple-500/20 text-purple-400 text-xs">2</span>
            Microservice Integration
          </h3>
          <p className="text-zinc-400 mb-4 text-sm">Microservices connect to port 30011. Start with a dialer, then create the service client.</p>
          <div className="bg-[#050505] border border-zinc-800 rounded-xl p-4 font-mono text-sm overflow-x-auto leading-relaxed">
            <div className="text-zinc-400 mb-2">import (</div>
            <div className="pl-4 text-zinc-400">&quot;net&quot;</div>
            <div className="pl-4 text-zinc-400">&quot;github.com/singchia/frontier/api/dataplane/v1/service&quot;</div>
            <div className="text-zinc-400 mb-2">)</div>
            <div><span className="text-blue-400">func</span> <span className="text-emerald-400">main</span>() &#123;</div>
            <div className="pl-4 text-zinc-500">dialer := func() (net.Conn, error) &#123;</div>
            <div className="pl-8 text-zinc-300">return net.Dial(<span className="text-amber-300">&quot;tcp&quot;</span>, <span className="text-amber-300">&quot;127.0.0.1:30011&quot;</span>)</div>
            <div className="pl-4 text-zinc-500">&#125;</div>
            <div className="pl-4 text-zinc-300">svc, err := service.<span className="text-cyan-300">NewService</span>(dialer)</div>
            <div className="pl-4 text-zinc-300">if err != nil &#123; panic(err) &#125;</div>
            <br/>
            <div className="pl-4 text-zinc-500">{'// Now the service can receive messages, register RPC, or open streams'}</div>
            <div className="text-zinc-300">&#125;</div>
          </div>
        </div>

        <div className="bg-[#111] border border-zinc-800 rounded-2xl p-8 relative overflow-hidden">
          <div className="absolute top-0 left-0 w-1 h-full bg-cyan-500"></div>
          <h3 className="text-lg font-bold text-white mb-4 flex items-center gap-2">
            <span className="flex items-center justify-center w-6 h-6 rounded-full bg-cyan-500/20 text-cyan-400 text-xs">3</span>
            Edge Node Integration
          </h3>
          <p className="text-zinc-400 mb-4 text-sm">Edge nodes (IoT, agents, clients) connect to port 30012. Then they can publish, register RPC, call services, or accept streams.</p>
          <div className="bg-[#050505] border border-zinc-800 rounded-xl p-4 font-mono text-sm overflow-x-auto leading-relaxed">
            <div className="text-zinc-400 mb-2">import (</div>
            <div className="pl-4 text-zinc-400">&quot;net&quot;</div>
            <div className="pl-4 text-zinc-400">&quot;github.com/singchia/frontier/api/dataplane/v1/edge&quot;</div>
            <div className="text-zinc-400 mb-2">)</div>
            <div><span className="text-blue-400">func</span> <span className="text-emerald-400">main</span>() &#123;</div>
            <div className="pl-4 text-zinc-500">dialer := func() (net.Conn, error) &#123;</div>
            <div className="pl-8 text-zinc-300">return net.Dial(<span className="text-amber-300">&quot;tcp&quot;</span>, <span className="text-amber-300">&quot;127.0.0.1:30012&quot;</span>)</div>
            <div className="pl-4 text-zinc-500">&#125;</div>
            <div className="pl-4 text-zinc-300">eg, err := edge.<span className="text-cyan-300">NewEdge</span>(dialer)</div>
            <br/>
            <div className="pl-4 text-zinc-500">{'// Now the edge can publish, register methods, call services, or open streams'}</div>
            <div className="text-zinc-300">&#125;</div>
          </div>
        </div>
      </div>

      <div className="mt-16 flex items-center justify-between pt-8 border-t border-zinc-800">
        <div></div>
        <Link href="/docs/usage" className="inline-flex items-center text-blue-400 hover:text-blue-300 font-medium">
          Next: Usage Guide &rarr;
        </Link>
      </div>
    </div>
  );
}
