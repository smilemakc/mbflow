import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter } from 'react-router-dom';
import { ImportApiClientModal } from './ImportApiClientModal';
import { mockApi, type ApiMock, type RecordedCall } from '@/test/apiMock';
import { apiClientFixture, specPreview } from '@/test/adminFixtures';
import { useUIStore } from '@/store/uiStore';
import { useToastStore } from '@/lib/toast';

const specFile = (name = 'petstore.yaml') => new File(['openapi: 3.0.3'], name, { type: 'application/yaml' });
const fileInput = () => document.querySelector('input[type="file"]') as HTMLInputElement;
const formOf = (call: RecordedCall) => call.data as FormData;
const toasts = () => useToastStore.getState().toasts;

describe('ImportApiClientModal', () => {
  let api: ApiMock;

  beforeEach(() => {
    useUIStore.setState({ language: 'en' });
    useToastStore.getState().clearAll();
    api = mockApi().on('GET', '/credentials', { data: { credentials: [] } });
  });

  afterEach(() => api.restore());

  const renderModal = () => {
    const onClose = vi.fn();
    const onImported = vi.fn();
    render(
      <MemoryRouter>
        <ImportApiClientModal isOpen onClose={onClose} onImported={onImported} />
      </MemoryRouter>,
    );
    return { onClose, onImported };
  };

  it('previews a chosen file, then imports it with the edited settings', async () => {
    api
      .on('POST', '/api-clients/preview', {
        data: specPreview({
          api_version: '1.2.0',
          base_url: 'https://petstore.io/v1',
          servers: [{ url: 'https://petstore.io/v1' }, { url: 'https://staging.petstore.io' }],
          suggested_auth: { type: 'bearer', scheme_name: 'Bearer' },
          security_schemes: [{ name: 'Bearer', type: 'http', scheme: 'bearer' }],
          warnings: [{ code: 'UNSUPPORTED', message: 'callbacks are ignored' }],
        }),
      })
      .on('POST', '/api-clients', { status: 201, data: apiClientFixture('new', { operation_count: 3 }) });
    const { onClose, onImported } = renderModal();

    expect(screen.getByText('Import an API from OpenAPI')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Import' })).toBeDisabled();
    expect(fileInput()).toHaveAttribute('accept', expect.stringContaining('.yaml'));

    await userEvent.upload(fileInput(), specFile());
    expect(await screen.findByRole('heading', { name: 'Petstore' })).toBeInTheDocument();
    const preview = api.calls.find((c) => c.url === '/api-clients/preview')!;
    expect((formOf(preview).get('file') as File).name).toBe('petstore.yaml');
    expect(preview.config.timeout).toBe(120_000);

    expect(screen.getByText('v1.2.0')).toBeInTheDocument();
    expect(screen.getByText(/OpenAPI 3\.0\.3 · 3 operations/)).toBeInTheDocument();
    expect(screen.getByText(/petstore\.yaml/)).toBeInTheDocument();
    expect(screen.getByText('callbacks are ignored')).toBeInTheDocument();
    expect(screen.getByDisplayValue('Petstore')).toBeInTheDocument();
    expect(screen.getByDisplayValue('https://petstore.io/v1')).toBeInTheDocument();
    // Suggested auth is preselected.
    expect(screen.getByDisplayValue('Bearer token')).toBeInTheDocument();

    const name = screen.getByDisplayValue('Petstore');
    await userEvent.clear(name);
    expect(screen.getByRole('button', { name: 'Import' })).toBeDisabled();
    await userEvent.type(name, '  Pets  ');
    await userEvent.click(screen.getByRole('button', { name: 'https://staging.petstore.io' }));

    await userEvent.click(screen.getByRole('button', { name: 'Import' }));
    await waitFor(() => expect(onImported).toHaveBeenCalledWith(expect.objectContaining({ id: 'new' })));
    expect(onClose).toHaveBeenCalled();

    const create = formOf(api.calls.find((c) => c.method === 'POST' && c.url === '/api-clients')!);
    expect((create.get('file') as File).name).toBe('petstore.yaml');
    expect(create.get('name')).toBe('Pets');
    expect(create.get('base_url')).toBe('https://staging.petstore.io');
    expect(JSON.parse(create.get('auth') as string)).toEqual({ type: 'bearer', scheme_name: 'Bearer' });
    expect(create.get('default_headers')).toBeNull();
    expect(toasts()).toEqual([
      expect.objectContaining({
        type: 'success',
        title: 'API imported',
        message: '3 operations are now available in the node library.',
      }),
    ]);
  });

  it('sends no base URL when it is empty and labels Swagger 2.0', async () => {
    api
      .on('POST', '/api-clients/preview', { data: specPreview({ spec_version: '2.0', suggested_auth: undefined as never }) })
      .on('POST', '/api-clients', { status: 201, data: apiClientFixture('new') });
    renderModal();
    await userEvent.upload(fileInput(), specFile('swagger.json'));
    expect(await screen.findByText(/Swagger 2\.0/)).toBeInTheDocument();
    expect(screen.getByText(/Base URL is not set/)).toBeInTheDocument();
    expect(screen.getByDisplayValue('No authentication')).toBeInTheDocument();
    await userEvent.click(screen.getByRole('button', { name: 'Import' }));
    await waitFor(() => expect(api.calls.some((c) => c.url === '/api-clients')).toBe(true));
    const create = formOf(api.calls.find((c) => c.url === '/api-clients')!);
    expect(create.get('base_url')).toBeNull();
    expect(JSON.parse(create.get('auth') as string)).toEqual({ type: 'none' });
  });

  it('shows a parse error and accepts a dropped file afterwards', async () => {
    api.once('POST', '/api-clients/preview', { status: 400, data: { message: 'not an OpenAPI document' } });
    api.on('POST', '/api-clients/preview', { data: specPreview({ title: 'Dropped' }) });
    renderModal();

    await userEvent.upload(fileInput(), specFile('bad.yaml'));
    expect(await screen.findByText('not an OpenAPI document')).toBeInTheDocument();
    expect(screen.getByText('bad.yaml')).toBeInTheDocument();

    const dropZone = screen.getByText('Drop a specification file here or click to choose').closest('button')!;
    fireEvent.dragOver(dropZone);
    expect(dropZone.className).toContain('border-blue-500');
    fireEvent.dragLeave(dropZone);
    expect(dropZone.className).not.toContain('border-blue-500');
    fireEvent.drop(dropZone, { dataTransfer: { files: [] } });
    fireEvent.drop(dropZone, { dataTransfer: { files: [specFile('dropped.json')] } });

    expect(await screen.findByRole('heading', { name: 'Dropped' })).toBeInTheDocument();
    expect(screen.queryByText('not an OpenAPI document')).not.toBeInTheDocument();
  });

  it('shows the parsing state while the preview is in flight', async () => {
    let resolve: (r: { data: unknown }) => void = () => {};
    api.on('POST', '/api-clients/preview', () => new Promise((r) => (resolve = r)));
    renderModal();
    await userEvent.upload(fileInput(), specFile());
    expect(screen.getByText('Reading the specification...')).toBeInTheDocument();
    expect(screen.getByText('Reading the specification...').closest('button')).toBeDisabled();
    resolve({ data: specPreview() });
    expect(await screen.findByRole('heading', { name: 'Petstore' })).toBeInTheDocument();
  });

  it('opens the file picker from the drop zone and "Choose another file"', async () => {
    api.on('POST', '/api-clients/preview', { data: specPreview() });
    renderModal();
    const click = vi.spyOn(fileInput(), 'click');
    await userEvent.click(screen.getByText('Drop a specification file here or click to choose'));
    expect(click).toHaveBeenCalledTimes(1);
    await userEvent.upload(fileInput(), specFile());
    await userEvent.click(await screen.findByRole('button', { name: 'Choose another file' }));
    expect(click).toHaveBeenCalledTimes(2);
    // An empty selection is ignored.
    fireEvent.change(fileInput(), { target: { files: [] } });
    expect(api.calls.filter((c) => c.url === '/api-clients/preview')).toHaveLength(1);
  });

  it('keeps the modal open and toasts when the import fails', async () => {
    api
      .on('POST', '/api-clients/preview', { data: specPreview() })
      .on('POST', '/api-clients', { status: 413, data: { message: 'payload too large' } });
    const { onClose, onImported } = renderModal();
    await userEvent.upload(fileInput(), specFile());
    await userEvent.click(await screen.findByRole('button', { name: 'Import' }));
    await waitFor(() =>
      expect(toasts()).toEqual([expect.objectContaining({ type: 'error', title: 'Error', message: 'payload too large' })]),
    );
    expect(onImported).not.toHaveBeenCalled();
    expect(onClose).not.toHaveBeenCalled();
    expect(screen.getByRole('heading', { name: 'Petstore' })).toBeInTheDocument();
  });

  it('resets the state when cancelled', async () => {
    api.on('POST', '/api-clients/preview', { data: specPreview() });
    const { onClose } = renderModal();
    await userEvent.upload(fileInput(), specFile());
    await screen.findByRole('heading', { name: 'Petstore' });
    await userEvent.click(screen.getByRole('button', { name: 'Cancel' }));
    expect(onClose).toHaveBeenCalled();
    expect(screen.queryByRole('heading', { name: 'Petstore' })).not.toBeInTheDocument();
    expect(screen.getByText('Drop a specification file here or click to choose')).toBeInTheDocument();
  });
});
