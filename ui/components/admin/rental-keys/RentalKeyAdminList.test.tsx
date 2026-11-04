import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { render, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { RentalKeyAdminList } from './RentalKeyAdminList';
import { mockApi, type ApiMock } from '@/test/apiMock';
import { ADMIN_USERS, adminRentalKey } from '@/test/adminFixtures';
import { useUIStore } from '@/store/uiStore';
import { useToastStore } from '@/lib/toast';

const KEYS = [
  adminRentalKey('k1', {
    name: 'Prod OpenAI',
    description: 'main key',
    owner_id: 'alice',
    total_requests: 12_345,
    total_cost: 3.5,
  }),
  adminRentalKey('k2', {
    name: 'Claude',
    provider: 'anthropic',
    status: 'suspended',
    owner_id: '9f1c2d3e-aaaa-bbbb-cccc-000000000000',
  }),
  adminRentalKey('k3', { name: 'Gone', status: 'deleted', provider: 'google_ai', owner_id: 'bob' }),
  adminRentalKey('k4', { name: 'Odd', status: 'weird' as never, owner_id: 'carol' }),
];

const LIST_URL = /^\/admin\/rental-keys(\?.*)?$/;
const rowFor = (name: string) => screen.getByText(name).closest('tr')!;
const confirmDelete = () => screen.getAllByRole('button', { name: 'Delete' }).find((b) => !b.hasAttribute('title'))!;
const listCalls = (api: ApiMock) => api.calls.filter((c) => c.method === 'GET' && LIST_URL.test(c.url));

describe('RentalKeyAdminList', () => {
  let api: ApiMock;

  beforeEach(() => {
    useUIStore.setState({ language: 'en' });
    useToastStore.getState().clearAll();
    api = mockApi()
      .on('GET', '/admin/users', { data: { data: ADMIN_USERS, total: 3, limit: 500, offset: 0 } })
      .on('GET', LIST_URL, { data: { rental_keys: KEYS, total: 4, limit: 20, offset: 0 } });
  });

  afterEach(() => api.restore());

  it('lists keys with owners resolved from the user list', async () => {
    render(<RentalKeyAdminList className="custom" />);
    expect(await screen.findByText('Prod OpenAI')).toBeInTheDocument();
    expect(listCalls(api)[0].url).toBe('/admin/rental-keys?limit=20&offset=0');
    expect(api.calls.find((c) => c.url === '/admin/users')?.params).toEqual({ limit: 500, offset: 0 });

    const first = rowFor('Prod OpenAI');
    expect(within(first).getByText('main key')).toBeInTheDocument();
    expect(await within(first).findByText('Alice Liddell')).toBeInTheDocument();
    expect(within(first).getByText('@alice')).toBeInTheDocument();
    expect(within(first).getByText('OpenAI')).toBeInTheDocument();
    expect(within(first).getByText('12.35K')).toBeInTheDocument();
    expect(within(first).getByText('$3.50')).toBeInTheDocument();

    const second = rowFor('Claude');
    expect(within(second).getByText('9f1c2d3e...')).toHaveAttribute('title', KEYS[1].owner_id);
    expect(within(second).getByText('Anthropic')).toBeInTheDocument();
    expect(within(second).getByText('suspended').className).toContain('yellow');
    expect(within(rowFor('Gone')).getByText('deleted').className).toContain('red');
    expect(within(rowFor('Gone')).getByText('Bob')).toBeInTheDocument();
    expect(within(rowFor('Odd')).getByText('carol')).toBeInTheDocument();
    expect(within(rowFor('Odd')).getByText('weird').className).toContain('gray');
  });

  it('shows the empty state and still works when users fail to load', async () => {
    const error = vi.spyOn(console, 'error').mockImplementation(() => {});
    api.once('GET', '/admin/users', { status: 500, data: { message: 'boom' } });
    api.once('GET', LIST_URL, { data: { total: 0 } });
    render(<RentalKeyAdminList />);
    expect(await screen.findByText('No rental keys found')).toBeInTheDocument();
    await waitFor(() => expect(error).toHaveBeenCalledWith('Failed to load users:', expect.anything()));
  });

  it('shows the load error', async () => {
    api.once('GET', LIST_URL, { status: 500, data: { message: 'failed to list rental keys' } });
    render(<RentalKeyAdminList />);
    expect(await screen.findByText('failed to list rental keys')).toBeInTheDocument();
  });

  it('filters by provider, status and owner through query parameters', async () => {
    render(<RentalKeyAdminList />);
    await screen.findByText('Prod OpenAI');
    const [providerSelect, statusSelect] = screen.getAllByRole('combobox');

    await userEvent.selectOptions(providerSelect, 'anthropic');
    await waitFor(() => expect(listCalls(api).at(-1)!.url).toBe('/admin/rental-keys?provider=anthropic&limit=20&offset=0'));

    await userEvent.selectOptions(statusSelect, 'suspended');
    await waitFor(() =>
      expect(listCalls(api).at(-1)!.url).toBe('/admin/rental-keys?provider=anthropic&status=suspended&limit=20&offset=0'),
    );

    await userEvent.click(screen.getByText('All Owners'));
    await userEvent.click(await screen.findByText('@bob • bob@example.com'));
    await waitFor(() =>
      expect(listCalls(api).at(-1)!.url).toBe(
        '/admin/rental-keys?provider=anthropic&status=suspended&owner_id=bob&limit=20&offset=0',
      ),
    );

    await userEvent.click(screen.getByRole('button', { name: 'Clear selection' }));
    await userEvent.selectOptions(providerSelect, '');
    await userEvent.selectOptions(statusSelect, '');
    await waitFor(() => expect(listCalls(api).at(-1)!.url).toBe('/admin/rental-keys?limit=20&offset=0'));
  });

  it('refreshes on demand', async () => {
    render(<RentalKeyAdminList />);
    await screen.findByText('Prod OpenAI');
    const before = listCalls(api).length;
    await userEvent.click(screen.getByRole('button', { name: /Refresh/ }));
    await waitFor(() => expect(listCalls(api).length).toBe(before + 1));
  });

  it('deletes a key after confirmation and reloads the list', async () => {
    api.on('DELETE', '/admin/rental-keys/k1', { data: { message: 'rental key deleted successfully' } });
    render(<RentalKeyAdminList />);
    await screen.findByText('Prod OpenAI');

    await userEvent.click(within(rowFor('Prod OpenAI')).getByTitle('Delete'));
    expect(screen.getByText('Are you sure you want to delete "Prod OpenAI"?')).toBeInTheDocument();
    await userEvent.click(screen.getByRole('button', { name: 'Cancel' }));
    expect(screen.queryByText(/Are you sure you want to delete/)).not.toBeInTheDocument();

    await userEvent.click(within(rowFor('Prod OpenAI')).getByTitle('Delete'));
    const before = listCalls(api).length;
    await userEvent.click(confirmDelete());
    await waitFor(() => expect(api.calls.some((c) => c.method === 'DELETE' && c.url === '/admin/rental-keys/k1')).toBe(true));
    await waitFor(() => expect(listCalls(api).length).toBe(before + 1));
    expect(screen.queryByText(/Are you sure you want to delete/)).not.toBeInTheDocument();
  });

  it('reports a failed delete with a toast', async () => {
    vi.spyOn(console, 'error').mockImplementation(() => {});
    api.on('DELETE', '/admin/rental-keys/k2', { status: 404, data: { message: 'rental key not found' } });
    render(<RentalKeyAdminList />);
    await screen.findByText('Claude');
    await userEvent.click(within(rowFor('Claude')).getByTitle('Delete'));
    await userEvent.click(confirmDelete());
    await waitFor(() =>
      expect(useToastStore.getState().toasts).toEqual(
        expect.arrayContaining([expect.objectContaining({ type: 'error', title: 'Delete Failed', message: 'rental key not found' })]),
      ),
    );
  });

  it('opens the create, edit and rotate modals and refreshes after saving', async () => {
    api
      .on('PUT', '/admin/rental-keys/k1', ({ data }) => ({ data: { ...KEYS[0], ...(data as object) } }))
      .on('POST', '/admin/rental-keys/k1/rotate-key', { data: { message: 'API key rotated successfully' } });
    render(<RentalKeyAdminList />);
    await screen.findByText('Prod OpenAI');

    await userEvent.click(screen.getByRole('button', { name: /Create Key/ }));
    expect(screen.getByRole('heading', { name: 'Create Rental Key' })).toBeInTheDocument();
    await userEvent.click(screen.getByRole('button', { name: 'Cancel' }));
    expect(screen.queryByRole('heading', { name: 'Create Rental Key' })).not.toBeInTheDocument();

    await userEvent.click(within(rowFor('Prod OpenAI')).getByTitle('Edit'));
    expect(screen.getByRole('heading', { name: 'Edit Rental Key' })).toBeInTheDocument();
    const name = screen.getByDisplayValue('Prod OpenAI');
    await userEvent.clear(name);
    await userEvent.type(name, 'Renamed');
    let before = listCalls(api).length;
    await userEvent.click(screen.getByRole('button', { name: 'Save' }));
    await waitFor(() => expect(screen.queryByRole('heading', { name: 'Edit Rental Key' })).not.toBeInTheDocument());
    expect(api.calls.find((c) => c.method === 'PUT')?.data).toEqual({ name: 'Renamed' });
    await waitFor(() => expect(listCalls(api).length).toBe(before + 1));

    await userEvent.click(within(rowFor('Prod OpenAI')).getByTitle('Rotate API Key'));
    expect(screen.getByRole('heading', { name: 'Rotate API Key' })).toBeInTheDocument();
    await userEvent.type(screen.getByPlaceholderText('Enter the new API key'), 'sk-new');
    before = listCalls(api).length;
    await userEvent.click(screen.getByRole('button', { name: 'Rotate Key' }));
    await waitFor(() => expect(screen.queryByRole('heading', { name: 'Rotate API Key' })).not.toBeInTheDocument());
    await waitFor(() => expect(listCalls(api).length).toBe(before + 1));
  });
});
