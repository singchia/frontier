import { chromium } from 'playwright';
import path from 'path';
import { fileURLToPath } from 'url';

const __dirname = path.dirname(fileURLToPath(import.meta.url));

const pages = [
  { name: 'home', url: 'http://localhost:3000' },
  { name: 'why-frontier', url: 'http://localhost:3000/why-frontier' },
  { name: 'architecture', url: 'http://localhost:3000/architecture' },
  { name: 'use-cases', url: 'http://localhost:3000/use-cases' },
  { name: 'examples', url: 'http://localhost:3000/examples' },
  { name: 'docs', url: 'http://localhost:3000/docs' },
  { name: 'docs-usage', url: 'http://localhost:3000/docs/usage' },
  { name: 'docs-configuration', url: 'http://localhost:3000/docs/configuration' },
  { name: 'docs-cluster', url: 'http://localhost:3000/docs/cluster' },
  { name: 'docs-operator', url: 'http://localhost:3000/docs/operator' },
];

(async () => {
  const browser = await chromium.launch();
  const context = await browser.newContext({
    viewport: { width: 1440, height: 900 },
  });

  for (const page of pages) {
    const p = await context.newPage();
    await p.goto(page.url, { waitUntil: 'networkidle' });
    await p.waitForTimeout(500);
    await p.screenshot({
      path: path.join(__dirname, `screenshots/${page.name}.png`),
      fullPage: true,
    });
    console.log(`✓ ${page.name}`);
    await p.close();
  }

  await browser.close();
  console.log('Done.');
})();
