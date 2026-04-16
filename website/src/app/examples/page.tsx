import { Terminal, Code2, Radio, ArrowRight } from 'lucide-react';

export default function Examples() {
  return (
    <div className="w-full max-w-7xl mx-auto px-6 lg:px-8 py-24">
      <div className="max-w-3xl mb-16">
        <h1 className="text-4xl md:text-6xl font-extrabold tracking-tighter text-white mb-6">
          Examples & Demos
        </h1>
        <p className="text-xl text-zinc-400 font-light leading-relaxed">
          The fastest way to decide whether Frontier fits your system is to run the example closest to your use case.
        </p>
      </div>

      <div className="grid lg:grid-cols-2 gap-8">
        {/* Chatroom Demo */}
        <div className="bg-[#0D0D0D] border border-zinc-800 rounded-3xl overflow-hidden flex flex-col group">
          <div className="bg-zinc-900/50 p-8 border-b border-zinc-800">
            <div className="flex items-center justify-between mb-4">
              <div className="w-12 h-12 rounded-2xl bg-blue-500/10 flex items-center justify-center border border-blue-500/20">
                <Terminal className="w-6 h-6 text-blue-400" />
              </div>
              <span className="px-3 py-1 rounded-full bg-zinc-800 text-xs font-medium text-zinc-300 border border-zinc-700">Go</span>
            </div>
            <h2 className="text-2xl font-bold text-white mb-2">
              Chatroom Demo
            </h2>
            <p className="text-zinc-400 text-sm">Start here if you care about command flow, notifications, presence, and service &lt;-&gt; edge messaging.</p>
          </div>

          <div className="p-8 flex-1 flex flex-col">
            <div className="bg-[#111] border border-zinc-800/50 rounded-xl p-5 font-mono text-sm text-zinc-300 mb-8 overflow-x-auto">
              <div className="text-zinc-500 mb-2"># 1. Start Frontier Gateway</div>
              <div className="mb-6 text-emerald-400">docker run -d -p 30011:30011 -p 30012:30012 singchia/frontier:1.2.2</div>

              <div className="text-zinc-500 mb-2"># 2. Build the examples</div>
              <div className="mb-6 text-emerald-400">make examples</div>

              <div className="text-zinc-500 mb-2"># 3. Terminal 1: Run the Chatroom Service</div>
              <div className="mb-6 text-white">./bin/chatroom_service</div>

              <div className="text-zinc-500 mb-2"># 4. Terminal 2: Run the Agent (Edge)</div>
              <div className="text-white">./bin/chatroom_agent</div>
            </div>

            <div className="mt-auto flex gap-4">
              <a href="https://github.com/singchia/frontier/tree/main/examples/chatroom" target="_blank" rel="noreferrer" className="flex-1 flex items-center justify-center gap-2 rounded-xl bg-zinc-100 px-4 py-3 text-sm font-semibold text-zinc-900 hover:bg-white transition-all">
                <Code2 className="w-4 h-4" /> View Source
              </a>
            </div>
          </div>
        </div>

        <div className="bg-[#0D0D0D] border border-zinc-800 rounded-3xl overflow-hidden flex flex-col group lg:col-span-2">
          <div className="bg-zinc-900/50 p-8 border-b border-zinc-800">
            <div className="flex items-center justify-between mb-4">
              <div className="w-12 h-12 rounded-2xl bg-cyan-500/10 flex items-center justify-center border border-cyan-500/20">
                <Radio className="w-6 h-6 text-cyan-400" />
              </div>
              <span className="px-3 py-1 rounded-full bg-zinc-800 text-xs font-medium text-zinc-300 border border-zinc-700">Stream</span>
            </div>
            <h2 className="text-2xl font-bold text-white mb-2">
              RTMP Relay Demo
            </h2>
            <p className="text-zinc-400 text-sm">Start here if you care about point-to-point streams, media relay, or using Frontier as a transport for custom traffic.</p>
          </div>

          <div className="p-8 flex-1 flex flex-col">
            <div className="bg-[#111] border border-zinc-800/50 rounded-xl p-5 font-mono text-sm text-zinc-300 mb-8 overflow-x-auto">
              <div className="text-zinc-500 mb-2"># 1. Start Frontier Gateway</div>
              <div className="mb-6 text-emerald-400">docker run -d -p 30011:30011 -p 30012:30012 singchia/frontier:1.2.2</div>

              <div className="text-zinc-500 mb-2"># 2. Build the examples</div>
              <div className="mb-6 text-emerald-400">make examples</div>

              <div className="text-zinc-500 mb-2"># 3. Terminal 1: Run the RTMP service</div>
              <div className="mb-6 text-white">./bin/rtmp_service</div>

              <div className="text-zinc-500 mb-2"># 4. Terminal 2: Run the RTMP edge</div>
              <div className="text-white">./bin/rtmp_edge</div>
            </div>

            <div className="mt-auto flex gap-4">
              <a href="https://github.com/singchia/frontier/tree/main/examples/rtmp" target="_blank" rel="noreferrer" className="flex-1 flex items-center justify-center gap-2 rounded-xl bg-zinc-100 px-4 py-3 text-sm font-semibold text-zinc-900 hover:bg-white transition-all">
                <Code2 className="w-4 h-4" /> View Source
              </a>
            </div>
          </div>
        </div>

        <div className="bg-[#0D0D0D] border border-zinc-800 rounded-3xl overflow-hidden flex flex-col group lg:col-span-2">
          <div className="bg-zinc-900/50 p-8 border-b border-zinc-800 flex items-center justify-between gap-6 flex-wrap">
            <div>
              <h2 className="text-2xl font-bold text-white mb-2">
                Which demo should you run first?
              </h2>
              <p className="text-zinc-400 text-sm">Choose by the job you need Frontier to do, not by the API name.</p>
            </div>
            <a href="https://github.com/singchia/frontier" target="_blank" rel="noreferrer" className="inline-flex items-center gap-2 rounded-xl border border-zinc-700 px-4 py-3 text-sm font-semibold text-white hover:bg-zinc-900 transition-colors">
              Visit GitHub
              <ArrowRight className="w-4 h-4" />
            </a>
          </div>

          <div className="grid md:grid-cols-3 gap-px bg-zinc-800">
            <div className="bg-[#0D0D0D] p-6">
              <div className="text-sm text-blue-400 mb-2">Messaging</div>
              <h3 className="text-lg font-bold text-white mb-2">Chatroom</h3>
              <p className="text-sm text-zinc-400">For command flow, messaging, edge online/offline state, and the core service-to-edge model.</p>
            </div>
            <div className="bg-[#0D0D0D] p-6">
              <div className="text-sm text-cyan-400 mb-2">Streams</div>
              <h3 className="text-lg font-bold text-white mb-2">RTMP</h3>
              <p className="text-sm text-zinc-400">For traffic relay, media transport, and understanding how streams differ from RPC and messaging.</p>
            </div>
            <div className="bg-[#0D0D0D] p-6">
              <div className="text-sm text-emerald-400 mb-2">SDK usage</div>
              <h3 className="text-lg font-bold text-white mb-2">Docs</h3>
              <p className="text-sm text-zinc-400">Once the examples click, move to the usage guide and copy the exact service-side or edge-side SDK pattern.</p>
            </div>
          </div>
        </div>

        {/* Video Player */}
        <div className="bg-[#0D0D0D] border border-zinc-800 rounded-3xl overflow-hidden flex flex-col group">
          <div className="bg-zinc-900/50 p-8 border-b border-zinc-800">
            <h2 className="text-2xl font-bold text-white mb-2">
              Live Demonstration
            </h2>
            <p className="text-zinc-400 text-sm">Watch the Chatroom example connecting Edge agents with the Service.</p>
          </div>

          <div className="p-6 flex-1 flex flex-col items-center justify-center bg-[#050505]">
            <div className="w-full rounded-2xl overflow-hidden border border-zinc-800 shadow-2xl bg-black aspect-video relative flex items-center justify-center group-hover:border-zinc-700 transition-colors">
              <video
                controls
                preload="metadata"
                className="w-full h-full object-cover"
                poster="/docs/diagram/frontier.png"
              >
                <source src="https://github.com/singchia/frontier/assets/15531166/18b01d96-e30b-450f-9610-917d65259c30" type="video/mp4" />
                Your browser does not support the video tag.
              </video>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
