import { Navbar } from '@/components/Navbar';
import './globals.css';
import type { Metadata } from 'next';

export const metadata: Metadata = {
  title: 'Frontier - The Edge-to-Service Communication Gateway',
  description: 'Full-duplex bidirectional RPC, messaging, and streams. Built for cloud-native edge environments.',
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="en" className="dark h-full antialiased scroll-smooth">
      <body className="bg-[#0A0A0A] text-zinc-300 min-h-screen flex flex-col font-sans selection:bg-blue-500/30">
        <div className="fixed inset-0 z-[-1] bg-[radial-gradient(ellipse_at_top_right,_var(--tw-gradient-stops))] from-blue-900/20 via-[#0A0A0A] to-[#0A0A0A] pointer-events-none"></div>
        <Navbar />
        <main className="flex-1 pt-20">
          {children}
        </main>
        <footer className="border-t border-white/5 bg-[#0A0A0A] py-12 mt-24">
          <div className="max-w-7xl mx-auto px-6 lg:px-8 text-center text-zinc-500 text-sm">
            <p className="mb-2">Built with modern cloud-native architectures in mind.</p>
            <p>&copy; {new Date().getFullYear()} Frontier. Released under Apache 2.0.</p>
          </div>
        </footer>
      </body>
    </html>
  );
}
