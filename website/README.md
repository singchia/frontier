# Frontier Website

This directory contains the marketing site and docs site for Frontier, built with Next.js 16.

## Local Development

Install dependencies and start the development server:

```bash
npm install
npm run dev
```

Open `http://localhost:3000`.

Useful commands:

```bash
npm run lint
npm run build
```

## Deploying to Vercel

This app is intended to be deployed from the `website/` directory as its project root.

Recommended Vercel project settings:

- Framework Preset: `Next.js`
- Root Directory: `website`
- Install Command: `npm install`
- Build Command: `npm run build`
- Output Directory: leave empty
- Node version: use the project default from Vercel unless you need to pin it

Once linked, you can deploy with:

```bash
npx vercel
npx vercel --prod
```

## What This Site Covers

- Homepage positioning for Frontier
- Why Frontier / use cases
- Examples and docs pages
- Architecture and deployment docs
