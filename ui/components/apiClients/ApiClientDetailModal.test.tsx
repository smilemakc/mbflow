import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { fireEvent, render, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter } from 'react-router-dom';
import { ApiClientDetailModal } from './ApiClientDetailModal';
import { mockApi, type ApiMock } from '@/test/apiMock';
import { apiClientFixture, operation } from '@/test/adminFixtures';
import { useUIStore } from '@/store/uiStore';
import { useToastStore } from '@/lib/toast';
import { useApiClientStore } from '@/store/apiClientStore';
import type { ApiClient } from '@/services/apiClientService';

const CLIENT: ApiClient = apiClientFixture('c1', {
  name: 'Pets API',
  title: 'Petstore',
  api_version: '1.0',
  description: 'pets',
  base_url: 'https://petstore.io',
  auth: { type: 'bearer' },
  default_headers: { 'X-Team': 'core' },
  servers: [{ url: 'https://petstore.io' }],
  warnings: [{ code: 'W', message: 'callbacks ignored' }],
  schemas: { Pet: { type: 'object', properties: { name: { type: 'string' } } } },
  operations: [
    operation('listPets', {
      path: '/pets',
      summary: 'List pets',
      tags: ['pets'],
      description: 'Returns all pets',
      parameters: [
        { name: 'limit', in: 'query', schema: { type: 'integer', format: 'int32' } },
        { name: 'id', in: 'path', required: true, schema: { type: 'string' } },
      ],
      responses: [{ status: '200', content: [{ content_type: 'application/json', schema: { type: 'array', items: { $ref: '#/components/schemas/Pet' } } }] }],
    }),
    operation('createPet', {
      method: 'POST',
      path: '/pets',
      tags: ['pets'],
      deprecated: true,
      no_auth: true,
      request_body: { content: [{ content_type: 'application/json', schema: { $ref: '#/components/schemas/Pet' } }] },
      responses: [{ status: '201', description: 'Created' }],
    }),
    operation('health', { path: '/health', request_body: { content: [{ content_type: 'text/plain' }] } }),
  ],
  operation_count: 3,
});

const toasts = () => useToastStore.getState().toasts;
const specInput = () => document.querySelector('input[type="file"]') as HTMLInputElement;

describe('ApiClientDetailModal', () => {
  let api: ApiMock;

  beforeEach(() => {
    useUIStore.setState({ language: 'en' });
    useToastStore.getState().clearAll();
    useApiClientStore.setState({ clients: [], details: {}, missing: {}, isLoaded: false, error: null });
    api = mockApi()
      .on('GET', '/api-clients/c1', { data: CLIENT })
      .on('GET', '/credentials', { data: { credentials: [] } });
  });

  afterEach(() => api.restore());

  const renderModal = (clientId: string | null = 'c1') => {
    const onClose = vi.fn();
    const utils = render(
      <MemoryRouter>
        <ApiClientDetailModal clientId={clientId} onClose={onClose} />
      </MemoryRouter>,
    );
    return { onClose, ...utils };
  };

  it('renders nothing without a client id', () => {
    renderModal(null);
    expect(screen.queryByText('Loading')).not.toBeInTheDocument();
    expect(api.calls).toHaveLength(0);
  });

  it('loads the client and lists its operations grouped by tag', async () => {
    renderModal();
    expect(screen.getByText('Loading')).toBeInTheDocument();
    expect(await screen.findByText('Pets API')).toBeInTheDocument();
    expect(api.calls[0]).toMatchObject({ method: 'GET', url: '/api-clients/c1' });
    expect(screen.getByText(/Petstore v1\.0 · OpenAPI 3\.0\.3/)).toBeInTheDocument();
    expect(screen.getByText('Warnings (1)')).toBeInTheDocument();
    expect(screen.getByText('callbacks ignored')).toBeInTheDocument();
    expect(screen.getByRole('tab', { name: 'Operations (3)' })).toHaveAttribute('aria-selected', 'true');
    expect(screen.getByText('pets')).toBeInTheDocument();
    expect(screen.getByText('Other')).toBeInTheDocument();

    const createRow = screen.getByText('POST').closest('button')!;
    expect(within(createRow).getByText('/pets').className).toContain('line-through');
    expect(within(createRow).getByTitle('no auth')).toBeInTheDocument();

    const listRow = screen.getByText('List pets').closest('button')!;
    expect(listRow).toHaveAttribute('aria-expanded', 'false');
    await userEvent.click(listRow);
    expect(listRow).toHaveAttribute('aria-expanded', 'true');
    expect(screen.getByText('listPets')).toBeInTheDocument();
    expect(screen.getByText('Returns all pets')).toBeInTheDocument();
    expect(screen.getByText(': integer (int32)')).toBeInTheDocument();
    expect(screen.getByText('*')).toBeInTheDocument();
    expect(screen.getByText('array<Pet>')).toBeInTheDocument();

    await userEvent.click(createRow);
    expect(listRow).toHaveAttribute('aria-expanded', 'false');
    expect(screen.getByText('deprecated')).toBeInTheDocument();
    expect(screen.getByText(': Pet')).toBeInTheDocument();
    expect(screen.getByText('Created')).toBeInTheDocument();
    await userEvent.click(createRow);
    expect(createRow).toHaveAttribute('aria-expanded', 'false');

    await userEvent.click(screen.getAllByText('/health')[0].closest('button')!);
    expect(screen.getByText('text/plain')).toBeInTheDocument();

    await userEvent.type(screen.getByPlaceholderText('Search operations...'), 'nothing-matches');
    expect(screen.getByText('No operations match the search')).toBeInTheDocument();
  });

  it('hides the no-auth marker when the client has no auth and labels Swagger 2.0', async () => {
    api.once('GET', '/api-clients/c1', { data: { ...CLIENT, auth: { type: 'none' }, spec_version: '2.0', api_version: undefined, warnings: [] } });
    renderModal();
    expect(await screen.findByText(/Petstore · Swagger 2\.0/)).toBeInTheDocument();
    expect(screen.queryByTitle('no auth')).not.toBeInTheDocument();
    expect(screen.queryByText(/Warnings/)).not.toBeInTheDocument();
  });

  it('saves the settings with the update request shape', async () => {
    api.on('PUT', '/api-clients/c1', ({ data }) => ({ data: { ...CLIENT, ...(data as object), updated_at: '2024-02-02T00:00:00Z' } }));
    renderModal();
    await userEvent.click(await screen.findByRole('tab', { name: 'Settings' }));
    expect(screen.getByRole('tab', { name: 'Settings' })).toHaveAttribute('aria-selected', 'true');

    const name = screen.getByDisplayValue('Pets API');
    await userEvent.clear(name);
    expect(screen.getByRole('button', { name: 'Save' })).toBeDisabled();
    await userEvent.type(name, 'Pets v2');
    await userEvent.clear(screen.getByDisplayValue('pets'));
    await userEvent.type(document.querySelector('textarea')!, 'new description');
    fireEvent.change(screen.getByDisplayValue('https://petstore.io'), { target: { value: 'https://api.pets.io' } });
    await userEvent.type(screen.getByDisplayValue('core'), '-x');

    await userEvent.click(screen.getByRole('button', { name: 'Save' }));
    await waitFor(() => expect(toasts()).toEqual([expect.objectContaining({ type: 'success', title: 'Settings saved' })]));
    expect(api.last()).toMatchObject({
      method: 'PUT',
      url: '/api-clients/c1',
      data: {
        name: 'Pets v2',
        description: 'new description',
        base_url: 'https://api.pets.io',
        auth: { type: 'bearer' },
        default_headers: { 'X-Team': 'core-x' },
      },
    });
    expect(useApiClientStore.getState().details.c1.name).toBe('Pets v2');
    expect(screen.getByText('Pets v2')).toBeInTheDocument();
  });

  it('uses defaults for a client without optional settings and reports save errors', async () => {
    api.once('GET', '/api-clients/c1', {
      data: { ...CLIENT, description: undefined, base_url: undefined, auth: {} as never, default_headers: undefined },
    });
    api.on('PUT', '/api-clients/c1', { status: 400, data: { message: 'credential of type basic_auth cannot be used' } });
    renderModal();
    await userEvent.click(await screen.findByRole('tab', { name: 'Settings' }));
    expect(screen.getByDisplayValue('No authentication')).toBeInTheDocument();
    expect(screen.getByText(/Base URL is not set/)).toBeInTheDocument();
    await userEvent.click(screen.getByRole('button', { name: 'Save' }));
    await waitFor(() =>
      expect(toasts()).toEqual([
        expect.objectContaining({ type: 'error', message: 'credential of type basic_auth cannot be used' }),
      ]),
    );
    expect(api.last()!.data).toEqual({ name: 'Pets API', description: '', base_url: '', auth: { type: 'none' }, default_headers: {} });
  });

  it('replaces the specification and lists removed operations', async () => {
    api.on('PUT', '/api-clients/c1/spec', {
      data: { client: { ...CLIENT, updated_at: '2024-03-03T00:00:00Z' }, changes: { added: ['a', 'b'], removed: ['GET /old'] } },
    });
    renderModal();
    await screen.findByText('Pets API');
    const click = vi.spyOn(specInput(), 'click');
    await userEvent.click(screen.getByRole('button', { name: /Update specification/ }));
    expect(click).toHaveBeenCalled();

    await userEvent.upload(specInput(), new File(['{}'], 'v2.json', { type: 'application/json' }));
    await waitFor(() =>
      expect(toasts()).toEqual([
        expect.objectContaining({ type: 'success', title: 'Specification updated', message: 'Added operations: 2, removed: 1.' }),
      ]),
    );
    const call = api.calls.find((c) => c.url === '/api-clients/c1/spec' && c.method === 'PUT')!;
    expect(((call.data as FormData).get('file') as File).name).toBe('v2.json');
    expect(screen.getByText('GET /old')).toBeInTheDocument();
    expect(screen.getByText(/Nodes that use these operations will fail/)).toBeInTheDocument();

    fireEvent.change(specInput(), { target: { files: [] } });
    expect(api.calls.filter((c) => c.url === '/api-clients/c1/spec')).toHaveLength(1);
  });

  it('reports a failed specification replacement', async () => {
    api.on('PUT', '/api-clients/c1/spec', { status: 400, data: { message: 'invalid spec' } });
    renderModal();
    await screen.findByText('Pets API');
    await userEvent.upload(specInput(), new File(['x'], 'bad.yaml'));
    await waitFor(() => expect(toasts()).toEqual([expect.objectContaining({ type: 'error', message: 'invalid spec' })]));
  });

  it('downloads the specification with the server file name', async () => {
    const createObjectURL = vi.fn(() => 'blob:spec');
    const revokeObjectURL = vi.fn();
    Object.assign(URL, { createObjectURL, revokeObjectURL });
    const anchorClick = vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(() => {});
    const blob = new Blob(['openapi: 3.0.3']);
    // once() routes are a stack: the last registered answers first.
    api.once('GET', '/api-clients/c1/spec', { data: blob });
    api.once('GET', '/api-clients/c1/spec', { data: blob, headers: { 'content-disposition': 'attachment; filename="petstore.yaml"' } });

    renderModal();
    await screen.findByText('Pets API');
    await userEvent.click(screen.getByRole('button', { name: /Download specification/ }));
    await waitFor(() => expect(anchorClick).toHaveBeenCalledTimes(1));
    const call = api.calls.find((c) => c.url === '/api-clients/c1/spec')!;
    expect(call.config.responseType).toBe('blob');
    expect(createObjectURL).toHaveBeenCalledWith(blob);
    expect(revokeObjectURL).toHaveBeenCalledWith('blob:spec');
    expect((anchorClick.mock.contexts[0] as HTMLAnchorElement).download).toBe('petstore.yaml');

    await userEvent.click(screen.getByRole('button', { name: /Download specification/ }));
    await waitFor(() => expect(anchorClick).toHaveBeenCalledTimes(2));
    expect((anchorClick.mock.contexts[1] as HTMLAnchorElement).download).toBe('Pets API.yaml');

    api.once('GET', '/api-clients/c1/spec', { status: 404, data: { message: 'API client not found' } });
    await userEvent.click(screen.getByRole('button', { name: /Download specification/ }));
    await waitFor(() => expect(toasts()).toEqual([expect.objectContaining({ type: 'error', message: 'API client not found' })]));
  });

  it('deletes the client after confirmation', async () => {
    api.on('DELETE', '/api-clients/c1', { data: { message: 'API client deleted successfully' } });
    const { onClose } = renderModal();
    await screen.findByText('Pets API');
    await userEvent.click(screen.getByRole('button', { name: 'Delete' }));
    expect(screen.getByText('Workflow nodes that use this API client will stop working.')).toBeInTheDocument();
    await userEvent.click(screen.getByRole('button', { name: 'Cancel' }));
    expect(screen.queryByText('Workflow nodes that use this API client will stop working.')).not.toBeInTheDocument();

    await userEvent.click(screen.getByRole('button', { name: 'Delete' }));
    const dialogButtons = screen.getAllByRole('button', { name: 'Delete' });
    await userEvent.click(dialogButtons[dialogButtons.length - 1]);
    await waitFor(() => expect(onClose).toHaveBeenCalled());
    expect(api.calls.some((c) => c.method === 'DELETE' && c.url === '/api-clients/c1')).toBe(true);
    expect(useApiClientStore.getState().details.c1).toBeUndefined();
    expect(toasts()).toEqual([expect.objectContaining({ type: 'success', title: 'API client deleted' })]);
  });

  it('keeps the dialog open when the delete fails', async () => {
    api.on('DELETE', '/api-clients/c1', { status: 500, data: { message: 'boom' } });
    const { onClose } = renderModal();
    await screen.findByText('Pets API');
    await userEvent.click(screen.getByRole('button', { name: 'Delete' }));
    const dialogButtons = screen.getAllByRole('button', { name: 'Delete' });
    await userEvent.click(dialogButtons[dialogButtons.length - 1]);
    await waitFor(() => expect(toasts()).toEqual([expect.objectContaining({ type: 'error', message: 'boom' })]));
    expect(onClose).not.toHaveBeenCalled();
  });
});
