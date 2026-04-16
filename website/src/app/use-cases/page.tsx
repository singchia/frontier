import { Server, Smartphone, Zap, FileUp, Workflow, Radio } from 'lucide-react';

export default function UseCases() {
  return (
    <div className="w-full max-w-7xl mx-auto px-6 lg:px-8 py-24">
      <div className="max-w-3xl mb-20">
        <h1 className="text-4xl md:text-6xl font-extrabold tracking-tighter text-white mb-6">
          Built for real-world <br />
          <span className="text-transparent bg-clip-text bg-gradient-to-r from-blue-400 to-cyan-300">edge challenges</span>
        </h1>
        <p className="text-xl text-zinc-400 leading-relaxed font-light">
          See how Frontier solves complex communication patterns that traditional gateways struggle with.
        </p>
      </div>

      <div className="space-y-16">
        {/* Case 1: Service-to-Client RPC */}
        <div className="bg-[#0D0D0D] border border-zinc-800 rounded-3xl overflow-hidden hover:border-zinc-700 transition-colors">
          <div className="grid md:grid-cols-2">
            <div className="p-10 md:p-12 flex flex-col justify-center">
              <div className="w-12 h-12 rounded-2xl bg-blue-500/10 flex items-center justify-center mb-6 border border-blue-500/20">
                <Zap className="w-6 h-6 text-blue-400" />
              </div>
              <h2 className="text-2xl font-bold text-white mb-4">Command & Control (RPC)</h2>
              <p className="text-zinc-400 mb-8 leading-relaxed">
                In traditional setups, clients poll servers for commands. With Frontier, the microservice can issue a direct RPC call to a specific Edge node by its ID. Perfect for IoT device management, game server commands, or remote agent execution.
              </p>
              <div className="bg-[#111] border border-zinc-800/50 rounded-xl p-5 font-mono text-sm">
                <div className="text-zinc-500 mb-2"># Call a method on a specific edge node</div>
                <div className="text-blue-300">response, err := frontier.Call(</div>
                <div className="text-zinc-300 pl-4">edgeID,</div>
                <div className="text-emerald-300 pl-4">&quot;RebootDevice&quot;,</div>
                <div className="text-zinc-300 pl-4">payload,</div>
                <div className="text-blue-300">)</div>
              </div>
            </div>

            {/* Visual representation of RPC */}
            <div className="bg-[#111] p-10 flex items-center justify-center border-l border-zinc-800">
              <div className="w-full max-w-sm">
                <div className="flex items-center justify-between mb-8">
                  <div className="flex flex-col items-center gap-2">
                    <div className="w-16 h-16 rounded-2xl bg-blue-500/20 border border-blue-500/30 flex items-center justify-center shadow-[0_0_30px_rgba(59,130,246,0.15)] z-10">
                      <Server className="w-8 h-8 text-blue-400" />
                    </div>
                    <span className="text-xs text-zinc-500 font-mono">Microservice</span>
                  </div>

                  <div className="flex-1 relative flex items-center justify-center px-4">
                    <div className="w-full h-[2px] bg-zinc-800 absolute"></div>
                    <div className="w-full h-[2px] bg-gradient-to-r from-blue-500 to-emerald-400 absolute animate-[pulse_2s_ease-in-out_infinite]"></div>
                    <div className="bg-[#111] px-3 z-10 text-xs text-zinc-400 font-mono border border-zinc-800 rounded-full">RPC.Call</div>
                  </div>

                  <div className="flex flex-col items-center gap-2">
                    <div className="w-16 h-16 rounded-2xl bg-emerald-500/20 border border-emerald-500/30 flex items-center justify-center shadow-[0_0_30px_rgba(16,185,129,0.15)] z-10">
                      <Smartphone className="w-8 h-8 text-emerald-400" />
                    </div>
                    <span className="text-xs text-zinc-500 font-mono">Edge Node</span>
                  </div>
                </div>

                <div className="bg-zinc-900 border border-zinc-800 rounded-xl p-4 text-xs font-mono text-zinc-400">
                  <div className="flex justify-between border-b border-zinc-800 pb-2 mb-2">
                    <span>Target:</span>
                    <span className="text-white">Edge-X792</span>
                  </div>
                  <div className="flex justify-between">
                    <span>Method:</span>
                    <span className="text-emerald-400">RebootDevice</span>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>

        {/* Case 2: Edge Telemetry */}
        <div className="bg-[#0D0D0D] border border-zinc-800 rounded-3xl overflow-hidden hover:border-zinc-700 transition-colors">
          <div className="grid md:grid-cols-2">
            {/* Visual representation of Messaging */}
            <div className="bg-[#111] p-10 flex items-center justify-center border-r border-zinc-800 order-2 md:order-1">
              <div className="w-full max-w-xs">
                {/* Publisher */}
                <div className="flex items-center gap-3 mb-2">
                  <div className="w-10 h-10 rounded-xl bg-purple-500/10 border border-purple-500/20 flex items-center justify-center shadow-[0_0_20px_rgba(168,85,247,0.15)] relative flex-shrink-0">
                    <Smartphone className="w-5 h-5 text-purple-400" />
                    <div className="absolute -top-1 -right-1 w-2.5 h-2.5 bg-emerald-500 rounded-full border-2 border-[#111]"></div>
                  </div>
                  <div>
                    <div className="text-xs font-mono text-zinc-200">Edge Node</div>
                    <div className="text-[10px] font-mono text-zinc-500">Publisher</div>
                  </div>
                </div>

                {/* Arrow down + label */}
                <div className="flex items-center gap-2 ml-5 mb-2">
                  <div className="flex flex-col items-center">
                    <div className="w-[2px] h-4 bg-gradient-to-b from-purple-500 to-purple-500/30 rounded-full"></div>
                    <div className="w-0 h-0 border-l-[3px] border-r-[3px] border-t-[5px] border-l-transparent border-r-transparent border-t-purple-400"></div>
                  </div>
                  <span className="text-[10px] font-mono text-zinc-500 bg-zinc-900 px-2 py-0.5 rounded border border-zinc-800">Publish(&quot;sensor/tmp&quot;, data)</span>
                </div>

                {/* Frontier broker box */}
                <div className="rounded-xl border border-zinc-700 bg-zinc-900/80 p-3 mb-2 relative overflow-hidden">
                  <div className="absolute inset-0 bg-gradient-to-br from-purple-500/5 to-transparent pointer-events-none"></div>
                  <div className="flex items-center gap-2 mb-1">
                    <div className="w-1.5 h-1.5 bg-purple-400 rounded-full animate-[pulse_2s_ease-in-out_infinite]"></div>
                    <span className="text-[10px] font-mono text-zinc-400 uppercase tracking-widest">Frontier Broker</span>
                  </div>
                  <div className="flex items-center gap-2 mt-1">
                    <div className="bg-purple-500/10 border border-purple-500/20 rounded-lg px-3 py-1.5 flex-1 text-center">
                      <div className="text-[9px] text-zinc-500 font-mono leading-none mb-0.5">TOPIC</div>
                      <div className="text-[11px] font-mono text-purple-300">sensor/tmp</div>
                    </div>
                    <div className="text-zinc-600 text-xs">→</div>
                    <div className="text-[10px] font-mono text-zinc-400 flex flex-col gap-0.5">
                      <div className="bg-zinc-800 rounded px-1.5 py-0.5 text-zinc-300">route</div>
                    </div>
                  </div>
                </div>

                {/* Fan-out arrows + subscribers */}
                <div className="ml-5 flex flex-col gap-2">
                  {/* Subscriber 1 */}
                  <div className="flex items-center gap-2">
                    <div className="flex items-center gap-0.5">
                      <div className="w-3 h-[2px] bg-blue-500/50"></div>
                      <div className="w-0 h-0 border-l-[4px] border-y-[3px] border-l-blue-400 border-y-transparent"></div>
                    </div>
                    <div className="flex items-center gap-2 bg-zinc-900/80 border border-zinc-800 hover:border-blue-500/30 transition-colors rounded-lg px-3 py-2 flex-1">
                      <div className="p-1 rounded bg-blue-500/10">
                        <Server className="w-3 h-3 text-blue-400" />
                      </div>
                      <span className="text-[11px] font-mono text-zinc-300">Service A</span>
                      <span className="ml-auto text-[9px] font-mono text-blue-500/70">subscriber</span>
                    </div>
                  </div>

                  {/* Subscriber 2 */}
                  <div className="flex items-center gap-2">
                    <div className="flex items-center gap-0.5">
                      <div className="w-3 h-[2px] bg-amber-500/50"></div>
                      <div className="w-0 h-0 border-l-[4px] border-y-[3px] border-l-amber-400 border-y-transparent"></div>
                    </div>
                    <div className="flex items-center gap-2 bg-zinc-900/80 border border-zinc-800 hover:border-amber-500/30 transition-colors rounded-lg px-3 py-2 flex-1">
                      <div className="p-1 rounded bg-amber-500/10">
                        <Radio className="w-3 h-3 text-amber-400" />
                      </div>
                      <div className="flex flex-col leading-none">
                        <span className="text-[11px] font-mono text-zinc-300">Kafka MQ</span>
                        <span className="text-[9px] font-mono text-zinc-500">forward</span>
                      </div>
                      <span className="ml-auto text-[9px] font-mono text-amber-500/70">external</span>
                    </div>
                  </div>
                </div>
              </div>
            </div>

            <div className="p-10 md:p-12 flex flex-col justify-center order-1 md:order-2">
              <div className="w-12 h-12 rounded-2xl bg-purple-500/10 flex items-center justify-center mb-6 border border-purple-500/20">
                <Workflow className="w-6 h-6 text-purple-400" />
              </div>
              <h2 className="text-2xl font-bold text-white mb-4">Edge Telemetry & Routing</h2>
              <p className="text-zinc-400 mb-8 leading-relaxed">
                Edge nodes publish telemetry or events to Topics. Frontier intelligently routes these to the appropriate backend microservice or forwards them directly to an external MQ (like Kafka), ensuring guaranteed delivery with explicit ACKs.
              </p>
              <div className="bg-[#111] border border-zinc-800/50 rounded-xl p-5 font-mono text-sm">
                <div className="text-zinc-500 mb-2"># Edge publishes to a topic</div>
                <div className="text-purple-300">frontier.Publish(</div>
                <div className="text-emerald-300 pl-4">&quot;sensor/temperature&quot;,</div>
                <div className="text-zinc-300 pl-4">[]byte(`&#123;&quot;temp&quot;: 42.5&#125;`),</div>
                <div className="text-purple-300">)</div>
              </div>
            </div>
          </div>
        </div>

        {/* Case 3: Streaming */}
        <div className="bg-[#0D0D0D] border border-zinc-800 rounded-3xl overflow-hidden hover:border-zinc-700 transition-colors">
          <div className="grid md:grid-cols-2">
            <div className="p-10 md:p-12 flex flex-col justify-center">
              <div className="w-12 h-12 rounded-2xl bg-cyan-500/10 flex items-center justify-center mb-6 border border-cyan-500/20">
                <FileUp className="w-6 h-6 text-cyan-400" />
              </div>
              <h2 className="text-2xl font-bold text-white mb-4">Traffic Proxying & Files</h2>
              <p className="text-zinc-400 mb-8 leading-relaxed">
                Need to transfer large files or proxy custom protocol traffic (like SSH/VNC/RTMP) to a device behind a NAT? Open a point-to-point stream. The stream acts as a raw multiplexed socket, bypassing all routing overhead.
              </p>
              <div className="bg-[#111] border border-zinc-800/50 rounded-xl p-5 font-mono text-sm">
                <div className="text-zinc-500 mb-2"># Open a raw byte stream to edge</div>
                <div className="text-zinc-300">stream, err := <span className="text-cyan-300">frontier.OpenStream(</span>edgeID<span className="text-cyan-300">)</span></div>
                <div className="text-zinc-500 mt-4 mb-2"># Stream a file or pipe SSH directly</div>
                <div className="text-zinc-300">io.Copy(stream, largeFile)</div>
              </div>
            </div>

            {/* Visual representation of Stream */}
            <div className="bg-[#111] p-10 flex items-center justify-center border-l border-zinc-800 relative overflow-hidden">
              <img
                src="/docs/diagram/stream.png"
                alt="Frontier Stream Flow"
                className="w-full max-w-sm rounded-xl border border-zinc-800 shadow-2xl relative z-10 mix-blend-screen opacity-90"
              />
              {/* Background ambient glow matching the image */}
              <div className="absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 w-[200px] h-[200px] bg-cyan-500/20 blur-[80px] rounded-full pointer-events-none z-0"></div>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
