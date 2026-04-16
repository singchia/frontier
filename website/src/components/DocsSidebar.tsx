'use client';
import Link from 'next/link';
import { usePathname } from 'next/navigation';

type DocLink = {
  href: string;
  label: string;
  external?: boolean;
};

type DocSection = {
  title: string;
  links: DocLink[];
};

const navItems: DocSection[] = [
  {
    title: 'Getting Started',
    links: [
      { href: '/docs', label: 'Overview & Quick Start' },
      { href: '/docs/usage', label: 'Usage Guide' },
      { href: '/docs/configuration', label: 'Configuration' },
    ]
  },
  {
    title: 'Deployment',
    links: [
      { href: '/docs/cluster', label: 'Cluster Architecture' },
      { href: '/docs/operator', label: 'Kubernetes Operator' },
    ]
  },
  {
    title: 'API Reference',
    links: [
      { href: 'https://pkg.go.dev/github.com/singchia/frontier/api/dataplane/v1/service', label: 'Service API', external: true },
      { href: 'https://pkg.go.dev/github.com/singchia/frontier/api/dataplane/v1/edge', label: 'Edge API', external: true },
    ]
  }
];

export function DocsSidebar() {
  const pathname = usePathname();

  return (
    <nav className="sticky top-28 space-y-8 pr-6">
      {navItems.map((section) => (
        <div key={section.title}>
          <h4 className="font-semibold text-white mb-3 uppercase tracking-wider text-xs">{section.title}</h4>
          <ul className="space-y-1.5 text-sm">
            {section.links.map((link) => {
              const isActive = pathname === link.href;
              return (
                <li key={link.href}>
                  <Link
                    href={link.href}
                    target={link.external ? '_blank' : undefined}
                    className={`flex items-center py-1.5 px-3 -mx-3 rounded-lg transition-colors ${
                      isActive
                        ? 'bg-blue-500/10 text-blue-400 font-medium'
                        : 'text-zinc-400 hover:text-white hover:bg-zinc-800/50'
                    }`}
                  >
                    {link.label}
                    {link.external && (
                      <svg className="w-3 h-3 ml-1.5 opacity-50" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14" />
                      </svg>
                    )}
                  </Link>
                </li>
              );
            })}
          </ul>
        </div>
      ))}
    </nav>
  );
}
