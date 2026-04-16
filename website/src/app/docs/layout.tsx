import { DocsSidebar } from '@/components/DocsSidebar';

export default function DocsLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <div className="w-full max-w-7xl mx-auto px-6 lg:px-8 py-12 flex flex-col md:flex-row gap-12">
      {/* Sidebar for Desktop */}
      <aside className="hidden md:block w-64 shrink-0 border-r border-zinc-800/50 min-h-[calc(100vh-120px)]">
        <DocsSidebar />
      </aside>

      {/* Main Content Area */}
      <main className="flex-1 min-w-0 prose prose-invert prose-blue max-w-4xl">
        {children}
      </main>
    </div>
  );
}