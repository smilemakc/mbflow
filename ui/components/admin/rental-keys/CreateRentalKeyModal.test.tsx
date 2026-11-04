import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { CreateRentalKeyModal } from './CreateRentalKeyModal';
import { mockApi, type ApiMock } from '@/test/apiMock';
import { ADMIN_USERS, adminRentalKey } from '@/test/adminFixtures';
import { useUIStore } from '@/store/uiStore';

const ownerToggle = () => screen.getByText('Select user...').closest('button')!;
const field = (name: string) => document.querySelector(`[name="${name}"]`) as HTMLInputElement;

describe('CreateRentalKeyModal', () => {
  let api: ApiMock;

  beforeEach(() => {
    useUIStore.setState({ language: 'en' });
    api = mockApi().on('GET', '/admin/users', { data: { data: ADMIN_USERS, total: 3 } });
  });

  afterEach(() => api.restore());

  const renderModal = (props: Partial<React.ComponentProps<typeof CreateRentalKeyModal>> = {}) => {
    const onClose = vi.fn();
    const onCreated = vi.fn();
    const utils = render(<CreateRentalKeyModal isOpen onClose={onClose} onCreated={onCreated} {...props} />);
    return { onClose, onCreated, ...utils };
  };

  it('renders nothing and loads no users while closed', () => {
    render(<CreateRentalKeyModal isOpen={false} onClose={vi.fn()} onCreated={vi.fn()} />);
    expect(screen.queryByRole('heading', { name: 'Create Rental Key' })).not.toBeInTheDocument();
    expect(api.calls).toHaveLength(0);
  });

  it('creates a key with the request shape the backend binds', async () => {
    api.on('POST', '/admin/rental-keys', ({ data }) => ({ status: 201, data: adminRentalKey('new', data as object) }));
    const { onClose, onCreated } = renderModal();
    await waitFor(() => expect(api.calls[0]).toMatchObject({ method: 'GET', url: '/admin/users', params: { limit: 500, offset: 0 } }));

    await userEvent.click(ownerToggle());
    const search = screen.getByPlaceholderText('Search users...');
    await userEvent.type(search, 'ALICE');
    expect(screen.queryByText('Bob')).not.toBeInTheDocument();
    expect(screen.getByText('Admin')).toBeInTheDocument();
    await userEvent.click(screen.getByText('Alice Liddell'));
    expect(screen.queryByPlaceholderText('Search users...')).not.toBeInTheDocument();
    expect(screen.getByText('alice@example.com')).toBeInTheDocument();

    await userEvent.type(screen.getByPlaceholderText('Rental key name'), 'Team key');
    await userEvent.type(screen.getByPlaceholderText('Optional description'), 'for the team');
    await userEvent.selectOptions(field('provider'), 'anthropic');
    await userEvent.type(screen.getByPlaceholderText('The actual API key (will be encrypted)'), 'sk-ant-1');
    await userEvent.type(field('daily_request_limit'), '100');
    await userEvent.type(field('monthly_token_limit'), '5000');
    await userEvent.clear(field('monthly_token_limit'));
    await userEvent.selectOptions(field('provisioner_type'), 'auto_anthropic');

    await userEvent.click(screen.getByRole('button', { name: 'Create' }));
    await waitFor(() => expect(onCreated).toHaveBeenCalled());
    expect(onClose).toHaveBeenCalled();

    const post = api.calls.find((c) => c.method === 'POST')!;
    expect(post.data).toEqual({
      owner_id: 'alice',
      name: 'Team key',
      description: 'for the team',
      provider: 'anthropic',
      api_key: 'sk-ant-1',
      daily_request_limit: 100,
      provisioner_type: 'auto_anthropic',
    });

    // The form is reset for the next key.
    expect(screen.getByPlaceholderText('Rental key name')).toHaveValue('');
    expect(screen.getByText('Select user...')).toBeInTheDocument();
  });

  it('requires an owner before sending the request', async () => {
    renderModal();
    await userEvent.type(screen.getByPlaceholderText('Rental key name'), 'Key');
    await userEvent.type(screen.getByPlaceholderText('The actual API key (will be encrypted)'), 'sk');
    await userEvent.click(screen.getByRole('button', { name: 'Create' }));
    expect(await screen.findByText('Owner: Select user...')).toBeInTheDocument();
    expect(api.calls.some((c) => c.method === 'POST')).toBe(false);
  });

  it('shows the API error and keeps the modal open', async () => {
    api.on('POST', '/admin/rental-keys', { status: 400, data: { message: 'invalid owner_id' } });
    const { onClose, onCreated } = renderModal();
    await userEvent.click(ownerToggle());
    await userEvent.click(await screen.findByText('Bob'));
    // Re-opening shows the selection marked in the list.
    await userEvent.click(screen.getByText('Bob').closest('button')!);
    expect(screen.getAllByText('Bob')[1].closest('button')!.className).toContain('bg-blue-50');
    await userEvent.click(screen.getAllByText('Bob')[1]);

    await userEvent.type(screen.getByPlaceholderText('Rental key name'), 'Key');
    await userEvent.type(screen.getByPlaceholderText('The actual API key (will be encrypted)'), 'sk');
    await userEvent.click(screen.getByRole('button', { name: 'Create' }));
    expect(await screen.findByText('invalid owner_id')).toBeInTheDocument();
    expect(onCreated).not.toHaveBeenCalled();
    expect(onClose).not.toHaveBeenCalled();
  });

  it('shows loading and empty states of the owner list and closes it on an outside click', async () => {
    const error = vi.spyOn(console, 'error').mockImplementation(() => {});
    let release: () => void = () => {};
    api.once('GET', '/admin/users', () => new Promise((resolve) => {
      release = () => resolve({ status: 500, data: { message: 'nope' } });
    }));
    renderModal();
    await userEvent.click(ownerToggle());
    expect(screen.getByText('Loading')).toBeInTheDocument();
    release();
    expect(await screen.findByText('No users found')).toBeInTheDocument();
    expect(error).toHaveBeenCalledWith('Failed to load users:', expect.anything());

    fireEvent.mouseDown(screen.getByPlaceholderText('Search users...'));
    expect(screen.getByPlaceholderText('Search users...')).toBeInTheDocument();
    fireEvent.mouseDown(screen.getByRole('heading', { name: 'Create Rental Key' }));
    expect(screen.queryByPlaceholderText('Search users...')).not.toBeInTheDocument();

    await userEvent.click(ownerToggle());
    await userEvent.click(ownerToggle());
    expect(screen.queryByPlaceholderText('Search users...')).not.toBeInTheDocument();
  });

  it('closes from the header button and Cancel', async () => {
    const { onClose } = renderModal();
    const buttons = screen.getAllByRole('button');
    await userEvent.click(buttons.find((b) => b.closest('form') && b.querySelector('svg') && !b.textContent)!);
    await userEvent.click(screen.getByRole('button', { name: 'Cancel' }));
    expect(onClose).toHaveBeenCalledTimes(2);
  });
});
