import { create } from 'zustand';
import { apiClientService, ApiClient, ApiClientSummary } from '@/services/apiClientService';
import { getErrorMessage } from '@/lib/api';

interface ApiClientState {
  clients: ApiClientSummary[];
  /** Full clients with operations and schemas, keyed by id. */
  details: Record<string, ApiClient>;
  /** Ids whose details failed to load because the client does not exist. */
  missing: Record<string, boolean>;
  isLoading: boolean;
  isLoaded: boolean;
  error: string | null;

  loadClients: (force?: boolean) => Promise<void>;
  loadClient: (id: string, force?: boolean) => Promise<ApiClient | null>;
  /** Loads the list and the details of every client, refetching details that are out of date. */
  loadAllDetails: () => Promise<void>;
  upsertClient: (client: ApiClient) => void;
  removeClient: (id: string) => void;
}

// Requests in flight, so components asking for the same client share one request.
const pendingDetails = new Map<string, Promise<ApiClient | null>>();
let pendingList: Promise<void> | null = null;

const toSummary = (client: ApiClient): ApiClientSummary => ({
  id: client.id,
  name: client.name,
  description: client.description,
  title: client.title,
  api_version: client.api_version,
  spec_version: client.spec_version,
  base_url: client.base_url,
  auth: client.auth,
  operation_count: client.operation_count,
  warnings: client.warnings,
  created_at: client.created_at,
  updated_at: client.updated_at,
});

export const useApiClientStore = create<ApiClientState>((set, get) => ({
  clients: [],
  details: {},
  missing: {},
  isLoading: false,
  isLoaded: false,
  error: null,

  loadClients: async (force = false) => {
    if (!force && get().isLoaded) return;
    if (pendingList) return pendingList;
    set({ isLoading: true, error: null });
    pendingList = (async () => {
      try {
        const response = await apiClientService.list();
        set({ clients: response.data.data ?? [], isLoaded: true });
      } catch (error) {
        set({ error: getErrorMessage(error) });
      } finally {
        set({ isLoading: false });
        pendingList = null;
      }
    })();
    return pendingList;
  },

  loadClient: async (id, force = false) => {
    const cached = get().details[id];
    if (cached && !force) return cached;
    const pending = pendingDetails.get(id);
    if (pending) return pending;

    const request = (async () => {
      try {
        const response = await apiClientService.get(id);
        get().upsertClient(response.data);
        return response.data;
      } catch (error: any) {
        const status = error?.response?.status;
        if (status === 404 || status === 400) {
          set((state) => ({ missing: { ...state.missing, [id]: true } }));
        }
        return null;
      } finally {
        pendingDetails.delete(id);
      }
    })();
    pendingDetails.set(id, request);
    return request;
  },

  loadAllDetails: async () => {
    await get().loadClients(true);
    const { clients, details } = get();
    // A client changed elsewhere (another tab, user or the API) has a newer updated_at.
    await Promise.all(
      clients.map((client) => get().loadClient(client.id, details[client.id]?.updated_at !== client.updated_at)),
    );
  },

  upsertClient: (client) => {
    set((state) => {
      const summary = toSummary(client);
      const exists = state.clients.some((c) => c.id === client.id);
      const { [client.id]: _, ...missing } = state.missing;
      return {
        details: { ...state.details, [client.id]: client },
        clients: exists
          ? state.clients.map((c) => (c.id === client.id ? summary : c))
          : [summary, ...state.clients],
        missing,
      };
    });
  },

  removeClient: (id) => {
    set((state) => {
      const { [id]: _, ...details } = state.details;
      return {
        details,
        clients: state.clients.filter((c) => c.id !== id),
        missing: { ...state.missing, [id]: true },
      };
    });
  },
}));
