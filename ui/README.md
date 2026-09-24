# MBFlow UI

Web interface for MBFlow — a React 19 + TypeScript single-page app built with Vite,
Tailwind CSS v4, React Flow and Zustand. It talks to the MBFlow Go backend
(`../go`) over REST (`/api/v1`) and WebSocket (`/ws`).

## Requirements

- Node.js 22+ and npm
- A running MBFlow backend (default `http://localhost:8585`)

## Development

```bash
npm ci
cp .env.example .env.local   # optional, adjust values
npm run dev                  # http://localhost:3435
```

The Vite dev server proxies `/api` and `/ws` to `VITE_BACKEND_URL`
(default `http://localhost:8585`), so the browser only ever talks to the dev server:

```bash
VITE_BACKEND_URL=http://localhost:8585 npm run dev
```

## Scripts

| Command             | Description                                    |
|---------------------|------------------------------------------------|
| `npm run dev`       | Start the dev server with HMR on port 3435     |
| `npm run typecheck` | Run the TypeScript compiler (`tsc --noEmit`)   |
| `npm run build`     | Type-check, then build static assets to `dist/`|
| `npm run preview`   | Serve the production build locally             |

Production builds strip `console.log/debug/info` calls and `debugger` statements
(`console.warn`/`console.error` are kept) and split vendor libraries into separate chunks.

## Environment variables

See [`.env.example`](.env.example). Variables prefixed with `VITE_` are read at **build time**.

| Variable           | Used by          | Default                 | Purpose                                                 |
|--------------------|------------------|-------------------------|---------------------------------------------------------|
| `VITE_BACKEND_URL` | dev server proxy | `http://localhost:8585` | Backend origin that `/api` and `/ws` are proxied to     |
| `VITE_API_URL`     | app (build time) | `/api/v1`               | REST API base URL; set only if the API is on another origin |
| `VITE_WS_URL`      | app (build time) | `/ws`                   | WebSocket base URL; set only if WS is on another origin |
| `VITE_PORT`        | dev server       | `3435`                  | Dev server port                                         |
| `VITE_HOST`        | dev server       | `0.0.0.0`               | Dev server bind address                                 |

Leaving `VITE_API_URL` / `VITE_WS_URL` unset (recommended) keeps all requests same-origin,
which works both with the Vite proxy and behind the nginx image below.

## Production (Docker)

The [`Dockerfile`](Dockerfile) builds the static assets with Node and serves them with
unprivileged nginx on port **8080**. [`nginx.conf`](nginx.conf) serves the SPA (with
history-API fallback) and proxies `/api/` and `/ws/` (including WebSocket upgrades) to the
backend service `mbflow-api:8585`. A health endpoint is exposed at `/healthz`.

The easiest way to run it is via the `mbflow-ui` service in
[`../go/docker-compose.yml`](../go/docker-compose.yml):

```bash
cd ../go
docker compose up -d --build mbflow-ui   # UI at http://localhost:8080
```

To build the image standalone:

```bash
docker build -t mbflow-ui:local .
```

When running standalone, the container must be able to resolve a host named `mbflow-api`
(e.g. attach it to the same Docker network as the backend).
