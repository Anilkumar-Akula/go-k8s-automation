# Dashboard — Frontend (Phase 2)

React + Vite + TypeScript shell for the [Go Dashboard API](../backend):
sidebar navigation and routed pages, not yet wired to any data (that's
Phase 3+).

## Run locally

```sh
npm install
npm run dev
```

## Pages

`Overview`, `Pod Auto-Healer`, `Rollout Manager`, `Resource Optimizer`,
`Auto-Scaling Controller`, `Live Events` — each a placeholder under
`src/pages/`, routed from `src/App.tsx` through `src/components/Layout.tsx`
(sidebar + content area).
