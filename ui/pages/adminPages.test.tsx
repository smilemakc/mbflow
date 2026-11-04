/**
 * Admin-only pages: user management and rental keys administration.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { UsersPage } from './UsersPage';
import { RentalKeysAdminPage } from './RentalKeysAdminPage';
import { useUIStore } from '@/store/uiStore';
import { useToastStore } from '@/lib/toast';
import { mockApi, type ApiMock } from '@/test/apiMock';
import { makeUser, renderPage, signIn } from '@/test/pagesRouter';
import type { User } from '@/types/auth';

vi.mock('@/components/admin/rental-keys', () => ({
  RentalKeyAdminList: () => <div>rental key admin list</div>,
}));

const toasts = () => useToastStore.getState().toasts;
const rows = () => screen.getAllByRole('row').slice(1);
const rowFor = (text: string) => rows().find((r) => within(r).queryByText(text))!;

const USERS: User[] = [
  makeUser({ id: 'u1', username: 'ada', full_name: 'Ada Lovelace', email: 'ada@example.com', is_admin: true, roles: ['admin'] }),
  makeUser({ id: 'u2', username: 'grace', full_name: undefined, email: 'grace@example.com', is_active: false, roles: undefined as never }),
];

let api: ApiMock;

beforeEach(() => {
  useUIStore.setState({ language: 'en', activeModal: null });
  useToastStore.getState().clearAll();
  signIn(makeUser({ is_admin: true }));
  api = mockApi()
    .on('GET', '/admin/users', ({ params }) => ({
      data: { data: params?.offset ? [makeUser({ id: 'u99', username: 'page2user' })] : USERS, total: 2 },
    }))
    .on('GET', '/admin/roles', { data: { roles: [{ id: 'r1', name: 'admin' }] } });
});

afterEach(() => api.restore());

describe('UsersPage', () => {
  it('lists users with roles and status from the API', async () => {
    renderPage(<UsersPage />);
    expect(await screen.findByText('Ada Lovelace')).toBeInTheDocument();
    expect(api.calls.find((c) => c.url === '/admin/users')?.params).toEqual({ limit: 20, offset: 0 });

    const ada = rowFor('Ada Lovelace');
    expect(within(ada).getByText('AD')).toBeInTheDocument();
    expect(within(ada).getByText('@ada')).toBeInTheDocument();
    expect(within(ada).getByText('Admin')).toBeInTheDocument();
    expect(within(ada).getByText('admin')).toBeInTheDocument();
    expect(within(ada).getByText('Active')).toBeInTheDocument();
    expect(within(ada).getByTitle('Deactivate')).toBeInTheDocument();

    const grace = rowFor('@grace');
    expect(within(grace).getByText('grace')).toBeInTheDocument();
    expect(within(grace).getByText('Inactive')).toBeInTheDocument();
    expect(within(grace).getByTitle('Activate')).toBeInTheDocument();
    expect(screen.queryByText(/Showing \d+ to/)).toBeNull();
  });

  it('redirects non-admins to the home page', async () => {
    signIn(makeUser({ is_admin: false }));
    renderPage(<UsersPage />, { at: '/admin/users' });
    await waitFor(() => expect(screen.getByTestId('location')).toHaveTextContent(/^\/$/));
  });

  it('activates and deactivates users', async () => {
    const user = userEvent.setup();
    api.on('PUT', /^\/admin\/users\//, ({ data }) => ({ data: { user: data } }));
    renderPage(<UsersPage />);
    await screen.findByText('Ada Lovelace');

    await user.click(within(rowFor('Ada Lovelace')).getByTitle('Deactivate'));
    await waitFor(() => expect(toasts().at(-1)).toMatchObject({ type: 'success', title: 'User deactivated' }));
    expect(api.calls.find((c) => c.method === 'PUT')).toMatchObject({
      url: '/admin/users/u1',
      data: { email: 'ada@example.com', username: 'ada', full_name: 'Ada Lovelace', is_active: false, is_admin: true },
    });

    await user.click(within(rowFor('@grace')).getByTitle('Activate'));
    await waitFor(() => expect(toasts().at(-1)).toMatchObject({ type: 'success', title: 'User activated' }));
    expect(api.calls.filter((c) => c.url === '/admin/users')).toHaveLength(3);
  });

  it('reports update failures', async () => {
    const user = userEvent.setup();
    api.on('PUT', /^\/admin\/users\//, { status: 500, data: { message: 'boom' } });
    renderPage(<UsersPage />);
    await screen.findByText('Ada Lovelace');
    await user.click(within(rowFor('Ada Lovelace')).getByTitle('Deactivate'));
    await waitFor(() => expect(toasts().at(-1)).toMatchObject({ type: 'error', title: 'Failed to update user' }));
  });

  it('deletes a user after confirmation', async () => {
    const user = userEvent.setup();
    api.once('DELETE', '/admin/users/u2', { data: {} }).on('DELETE', /^\/admin\/users\//, { status: 500 });
    renderPage(<UsersPage />);
    await screen.findByText('Ada Lovelace');

    await user.click(within(rowFor('@grace')).getByTitle('Delete'));
    expect(screen.getByText(/Are you sure you want to delete this user/)).toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: 'Cancel' }));
    expect(api.calls.some((c) => c.method === 'DELETE')).toBe(false);

    await user.click(within(rowFor('@grace')).getByTitle('Delete'));
    await user.click(within(screen.getByRole('dialog')).getByRole('button', { name: 'Delete' }));
    await waitFor(() => expect(toasts().at(-1)).toMatchObject({ type: 'success', title: 'User deleted successfully' }));
    expect(screen.queryByText(/Are you sure you want to delete this user/)).toBeNull();

    await user.click(within(rowFor('Ada Lovelace')).getByTitle('Delete'));
    await user.click(within(screen.getByRole('dialog')).getByRole('button', { name: 'Delete' }));
    await waitFor(() => expect(toasts().at(-1)).toMatchObject({ type: 'error', title: 'Failed to delete user' }));
  });

  it('pages through users', async () => {
    api.restore();
    api = mockApi().on('GET', '/admin/roles', { data: { roles: [] } });
    api.once('GET', '/admin/users', { data: { data: USERS, total: 45 } });
    api.on('GET', '/admin/users', ({ params }) => ({
      data: { data: [makeUser({ id: `p${params?.offset}`, username: `user${params?.offset}` })], total: 45 },
    }));
    const user = userEvent.setup();
    renderPage(<UsersPage />);

    expect(await screen.findByText('Showing 1 to 20 of 45 users')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Previous' })).toBeDisabled();

    await user.click(screen.getByRole('button', { name: 'Next' }));
    expect(await screen.findByText('Showing 21 to 40 of 45 users')).toBeInTheDocument();
    expect(screen.getByText('@user20')).toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: 'Next' }));
    expect(await screen.findByText('Showing 41 to 45 of 45 users')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Next' })).toBeDisabled();

    await user.click(screen.getByRole('button', { name: 'Previous' }));
    expect(await screen.findByText('Showing 21 to 40 of 45 users')).toBeInTheDocument();
  });

  it('shows a toast when users cannot be loaded', async () => {
    api.once('GET', '/admin/roles', { status: 500 });
    renderPage(<UsersPage />);
    await waitFor(() => expect(toasts().at(-1)).toMatchObject({ type: 'error', title: 'Failed to load users' }));
    expect(screen.getByRole('table')).toBeInTheDocument();
    expect(rows()).toHaveLength(0);
  });

  it('keeps the add/edit buttons clickable', async () => {
    const user = userEvent.setup();
    renderPage(<UsersPage />);
    await screen.findByText('Ada Lovelace');
    await user.click(screen.getByRole('button', { name: /Add User/ }));
    await user.click(within(rowFor('Ada Lovelace')).getByTitle('Edit'));
    expect(screen.getByText('Ada Lovelace')).toBeInTheDocument();
  });
});

describe('RentalKeysAdminPage', () => {
  it('renders the admin list inside the page layout', () => {
    renderPage(<RentalKeysAdminPage />);
    expect(screen.getByRole('heading', { name: 'Rental Keys Administration' })).toBeInTheDocument();
    expect(screen.getByText('rental key admin list')).toBeInTheDocument();
  });

  it('redirects non-admins', async () => {
    signIn(makeUser({ is_admin: false }));
    renderPage(<RentalKeysAdminPage />, { at: '/admin/rental-keys' });
    await waitFor(() => expect(screen.getByTestId('location')).toHaveTextContent(/^\/$/));
  });
});
