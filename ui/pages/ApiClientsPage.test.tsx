import { beforeEach, describe, expect, it, vi } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { ApiClientsPage } from './ApiClientsPage';
import { apiClientService, type ApiClientSummary } from '@/services/apiClientService';
import { useApiClientStore } from '@/store/apiClientStore';
import { useUIStore } from '@/store/uiStore';
import { apiError } from '@/test/apiMock';

// The modals have their own tests; stand-ins expose the props the page passes.
vi.mock('@/components/apiClients', () => ({
  ImportApiClientModal: ({ isOpen, onClose, onImported }: any) =>
    isOpen ? (
      <div role="dialog" aria-label="import">
        <button onClick={() => onImported({ id: 'imported-1' })}>finish import</button>
        <button onClick={onClose}>close import</button>
      </div>
    ) : null,
  ApiClientDetailModal: ({ clientId, onClose }: any) =>
    clientId ? (
      <div role="dialog" aria-label="detail">
        detail of {clientId}
        <button onClick={onClose}>close detail</button>
      </div>
    ) : null,
}));

const client = (id: string, overrides: Partial<ApiClientSummary> = {}): ApiClientSummary => ({
  id,
  name: `Client ${id}`,
  title: 'Petstore',
  spec_version: '3.0.3',
  auth: { type: 'none' } as never,
  operation_count: 3,
  warnings: [],
  created_at: '2026-01-01T00:00:00Z',
  updated_at: '2026-01-02T00:00:00Z',
  ...overrides,
});

const listResponse = (clients: ApiClientSummary[]) => ({ data: { data: clients } }) as never;

describe('ApiClientsPage', () => {
  let list: ReturnType<typeof vi.spyOn>;

  beforeEach(() => {
    useUIStore.setState({ language: 'en' });
    useApiClientStore.setState({ clients: [], details: {}, missing: {}, isLoading: false, isLoaded: false, error: null });
    list = vi.spyOn(apiClientService, 'list').mockResolvedValue(
      listResponse([
        client('a', {
          api_version: '1.2',
          description: 'Pets API',
          base_url: 'https://pets.example.com',
          auth: { type: 'bearer' } as never,
          warnings: [{ message: 'x' }, { message: 'y' }] as never,
        }),
        client('b', { spec_version: '2.0', operation_count: 1 }),
      ]),
    );
  });

  it('lists imported clients with their spec, auth and warnings', async () => {
    render(<ApiClientsPage />);
    expect(await screen.findByText('Client a')).toBeInTheDocument();

    expect(screen.getByText('Petstore v1.2')).toBeInTheDocument();
    expect(screen.getByText('OpenAPI 3.0.3')).toBeInTheDocument();
    expect(screen.getByText('Pets API')).toBeInTheDocument();
    expect(screen.getByText('https://pets.example.com')).toBeInTheDocument();
    expect(screen.getByText('Bearer token')).toBeInTheDocument();
    expect(screen.getByText('2')).toBeInTheDocument();
    expect(screen.getByText('3 operations')).toBeInTheDocument();

    expect(screen.getByText('Swagger 2.0')).toBeInTheDocument();
    expect(screen.getByText('Base URL is not set: requests will fail until you set it.')).toBeInTheDocument();
    expect(screen.getAllByText(/^Updated:/)).toHaveLength(2);
  });

  it('opens a client detail and closes it', async () => {
    const user = userEvent.setup();
    render(<ApiClientsPage />);
    await user.click(await screen.findByRole('button', { name: /Client b/ }));
    expect(screen.getByRole('dialog', { name: 'detail' })).toHaveTextContent('detail of b');
    await user.click(screen.getByRole('button', { name: 'close detail' }));
    expect(screen.queryByRole('dialog', { name: 'detail' })).toBeNull();
  });

  it('shows the empty state and opens the import dialog, then the imported client', async () => {
    list.mockResolvedValue(listResponse([]));
    const user = userEvent.setup();
    render(<ApiClientsPage />);

    expect(await screen.findByText('No API clients yet')).toBeInTheDocument();
    const importButtons = screen.getAllByRole('button', { name: 'Import OpenAPI' });
    expect(importButtons).toHaveLength(2);

    await user.click(importButtons[1]);
    await user.click(screen.getByRole('button', { name: 'close import' }));
    expect(screen.queryByRole('dialog', { name: 'import' })).toBeNull();

    await user.click(importButtons[0]);
    await user.click(screen.getByRole('button', { name: 'finish import' }));
    expect(screen.getByRole('dialog', { name: 'detail' })).toHaveTextContent('detail of imported-1');
  });

  it('reloads on refresh and shows load errors', async () => {
    const user = userEvent.setup();
    render(<ApiClientsPage />);
    await screen.findByText('Client a');
    expect(list).toHaveBeenCalledTimes(1);

    list.mockRejectedValue(apiError(500, { message: 'db offline' }));
    await user.click(screen.getByRole('button', { name: 'Refresh' }));
    expect(await screen.findByText(/Failed to load API clients: db offline/)).toBeInTheDocument();
    expect(list).toHaveBeenCalledTimes(2);
  });

  it('shows a spinner during the first load', async () => {
    let resolve!: (v: unknown) => void;
    list.mockReturnValue(new Promise((r) => (resolve = r)) as never);
    render(<ApiClientsPage />);
    expect(document.querySelector('.animate-spin')).not.toBeNull();
    expect(screen.getByRole('button', { name: 'Refresh' })).toBeDisabled();
    resolve(listResponse([]));
    await waitFor(() => expect(document.querySelector('.animate-spin')).toBeNull());
  });
});
