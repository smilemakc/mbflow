import { afterEach, beforeEach, describe, expect, it } from 'vitest';
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import { NodeLibrary } from './NodeLibrary';
import { ApiOperationsLibrary } from './ApiOperationsLibrary';
import { NODE_TEMPLATE_MIME, NODE_TYPE_MIME } from './dragData';
import { useUIStore } from '@/store/uiStore';
import { useApiClientStore } from '@/store/apiClientStore';
import { NodeType } from '@/types';
import { mockApi, type ApiMock } from '@/test/apiMock';
import { resetBuilderStores } from '@/test/builderFlowMocks';

const initialApiClients = useApiClientStore.getState();

const summary = (id: string, name: string, updated_at = '1') => ({ id, name, title: `${name} API`, operation_count: 2, updated_at });
const op = (id: string, method: string, path: string, extra: Record<string, unknown> = {}) => ({ id, method, path, ...extra });
const PETS = {
  ...summary('c1', 'Pets'),
  operations: [
    op('listPets', 'get', '/pets', { summary: 'List pets', tags: ['pets'], description: 'All pets' }),
    op('oldPet', 'delete', '/pets/{id}', { tags: ['pets'], deprecated: true }),
    op('health', 'get', '/health'),
  ],
};
const STORE = { ...summary('c2', 'Store'), operations: [op('order', 'post', '/orders', { summary: 'Place order' })] };

const renderWithRouter = (ui: React.ReactElement) =>
  render(
    <MemoryRouter initialEntries={['/builder']} future={{ v7_startTransition: true, v7_relativeSplatPath: true }}>
      <Routes>
        <Route path="/builder" element={ui} />
        <Route path="/api-clients" element={<div>API clients page</div>} />
      </Routes>
    </MemoryRouter>,
  );

const dragData = (el: Element) => {
  const data: Record<string, string> = {};
  const dataTransfer = { setData: (k: string, v: string) => (data[k] = v), effectAllowed: '' };
  fireEvent.dragStart(el, { dataTransfer });
  return { data, dataTransfer };
};

let api: ApiMock;
beforeEach(() => {
  resetBuilderStores();
  useApiClientStore.setState(initialApiClients, true);
  api = mockApi();
});
afterEach(() => api.restore());

const withClients = (...clients: (typeof PETS)[]) => {
  api.on('GET', '/api-clients', { data: { data: clients.map(({ operations: _o, ...s }) => s) } });
  clients.forEach((c) => api.on('GET', `/api-clients/${c.id}`, { data: c }));
};

describe('NodeLibrary', () => {
  it('lists node categories, filters by search and sets drag data', async () => {
    withClients();
    useUIStore.setState({ isNodeLibraryOpen: true });
    const { container } = renderWithRouter(<NodeLibrary />);
    expect((container.firstChild as HTMLElement).className).toContain('translate-x-0');
    for (const heading of ['TRIGGERS', 'ACTIONS', 'TELEGRAM', 'LOGIC', 'STORAGE', 'ADAPTERS']) {
      expect(screen.getByText(heading)).toBeInTheDocument();
    }
    expect(screen.getByText('RSS')).toBeInTheDocument(); // missing translation falls back to the key

    const { data, dataTransfer } = dragData(screen.getByText('HTTP Request').closest('[draggable]')!);
    expect(data[NODE_TYPE_MIME]).toBe(NodeType.HTTP);
    expect(dataTransfer.effectAllowed).toBe('move');

    await userEvent.type(screen.getByPlaceholderText('Search...'), 'telegram');
    expect(screen.getByText('Telegram Send')).toBeInTheDocument();
    expect(screen.queryByText('HTTP Request')).not.toBeInTheDocument();
    expect(screen.queryByText('ACTIONS')).not.toBeInTheDocument();
    expect(screen.getByText('Drag to add')).toBeInTheDocument();
  });

  it('closes via the header button', async () => {
    withClients();
    useUIStore.setState({ isNodeLibraryOpen: true });
    renderWithRouter(<NodeLibrary />);
    await userEvent.click(screen.getAllByRole('button')[0]);
    expect(useUIStore.getState().isNodeLibraryOpen).toBe(false);
  });
});

describe('ApiOperationsLibrary', () => {
  it('shows a loader, then an empty state linking to the API clients page', async () => {
    withClients();
    const { container } = renderWithRouter(<ApiOperationsLibrary searchQuery="" categoryLabel="API" />);
    expect(container.querySelector('.animate-spin')).not.toBeNull();
    expect(await screen.findByText('Import an OpenAPI specification to get a node for every endpoint.')).toBeInTheDocument();
    await userEvent.click(screen.getByRole('button', { name: 'Import API' }));
    expect(screen.getByText('API clients page')).toBeInTheDocument();
  });

  it('navigates to manage APIs', async () => {
    withClients();
    renderWithRouter(<ApiOperationsLibrary searchQuery="" categoryLabel="API" />);
    await userEvent.click(screen.getByTitle('Manage APIs'));
    expect(screen.getByText('API clients page')).toBeInTheDocument();
  });

  it('expands a single client by default and drags operations as openapi nodes', async () => {
    withClients(PETS);
    renderWithRouter(<ApiOperationsLibrary searchQuery="" categoryLabel="API" />);
    expect(await screen.findByText('List pets')).toBeInTheDocument();
    expect(screen.getByText('pets')).toBeInTheDocument();
    expect(screen.getByText('Other')).toBeInTheDocument();
    expect(screen.getByText('delete /pets/{id}').className).toContain('line-through');
    expect(screen.getByText('List pets').closest('[draggable]')).toHaveAttribute('title', 'get /pets\n\nAll pets');

    const { data } = dragData(screen.getByText('List pets').closest('[draggable]')!);
    expect(data[NODE_TYPE_MIME]).toBe(NodeType.OPENAPI);
    expect(JSON.parse(data[NODE_TEMPLATE_MIME])).toEqual({
      label: 'List pets',
      description: 'Pets',
      config: { client_id: 'c1', operation_id: 'listPets' },
    });

    const toggle = screen.getByRole('button', { name: /Pets/ });
    expect(toggle).toHaveAttribute('aria-expanded', 'true');
    await userEvent.click(toggle);
    expect(toggle).toHaveAttribute('aria-expanded', 'false');
    expect(screen.queryByText('List pets')).not.toBeInTheDocument();
  });

  it('collapses several clients by default and shows a loader for missing details', async () => {
    // A never-answered request stays shared in the store module, so use an id no other test loads.
    api.on('GET', '/api-clients', { data: { data: [summary('c1', 'Pets'), summary('slow', 'Store')] } });
    api.on('GET', '/api-clients/c1', { data: PETS });
    api.on('GET', '/api-clients/slow', () => new Promise(() => {}));
    const { container } = renderWithRouter(<ApiOperationsLibrary searchQuery="" categoryLabel="API" />);
    const store = await screen.findByRole('button', { name: /Store/ });
    expect(store).toHaveAttribute('aria-expanded', 'false');
    await userEvent.click(store);
    expect(store).toHaveAttribute('aria-expanded', 'true');
    expect(container.querySelectorAll('.animate-spin').length).toBeGreaterThan(0);
  });

  it('filters operations by search, by client name, and hides when nothing matches', async () => {
    withClients(PETS, STORE);
    const { rerender } = renderWithRouter(<ApiOperationsLibrary searchQuery="health" categoryLabel="API" />);
    expect(await screen.findByText('get /health')).toBeInTheDocument();
    expect(screen.queryByText('List pets')).not.toBeInTheDocument();
    expect(screen.queryByRole('button', { name: /Store/ })).not.toBeInTheDocument();

    rerender(
      <MemoryRouter initialEntries={['/builder']} future={{ v7_startTransition: true, v7_relativeSplatPath: true }}>
        <ApiOperationsLibrary searchQuery="store" categoryLabel="API" />
      </MemoryRouter>,
    );
    expect(await screen.findByText('Place order')).toBeInTheDocument();

    rerender(
      <MemoryRouter initialEntries={['/builder']} future={{ v7_startTransition: true, v7_relativeSplatPath: true }}>
        <ApiOperationsLibrary searchQuery="nothing-here" categoryLabel="API" />
      </MemoryRouter>,
    );
    await waitFor(() => expect(screen.queryByText('API')).not.toBeInTheDocument());
  });
});
