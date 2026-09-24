/// <reference types="vite/client" />

interface ImportMetaEnv {
  /** Base URL for REST API calls (default: `/api/v1`, proxied) */
  readonly VITE_API_URL?: string;
  /** Base URL for WebSocket connections (default: `/ws`, proxied) */
  readonly VITE_WS_URL?: string;
  /** Backend origin used by the Vite dev-server proxy */
  readonly VITE_BACKEND_URL?: string;
}

interface ImportMeta {
  readonly env: ImportMetaEnv;
}
