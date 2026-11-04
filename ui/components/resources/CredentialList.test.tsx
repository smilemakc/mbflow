import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { render, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { CredentialList } from './CredentialList';
import { useUIStore } from '@/store/uiStore';
import { mockApi, type ApiMock } from '@/test/apiMock';
import { credential } from '@/test/resourcesFixtures';
import type { Credential } from '@/services/credentialsService';

const CREDS: Credential[] = [
  credential({ id: 'c1', name: 'OpenAI key', credential_type: 'api_key', provider: 'openai' }),
  credential({ id: 'c2', name: 'Stripe key', credential_type: 'api_key', provider: 'stripe' }),
  credential({ id: 'c3', name: 'Jenkins', credential_type: 'basic_auth', provider: undefined, fields: ['username', 'password'] }),
];

const submitCreate = () => within(screen.getByRole('dialog')).getByRole('button', { name: 'Create Credential' });
const openCreate = () => screen.getAllByRole('button', { name: 'Create Credential' })[0];

const cardFor = (name: string) =>
  screen.getByRole('heading', { name }).closest('div.rounded-xl') as HTMLElement;

describe('CredentialList', () => {
  let api: ApiMock;

  beforeEach(() => {
    useUIStore.setState({ language: 'en' });
    api = mockApi();
  });
  afterEach(() => api.restore());

  it('loads credentials and groups them by type', async () => {
    api.on('GET', '/credentials', { data: { credentials: CREDS } });
    render(<CredentialList className="custom" />);
    expect(document.querySelector('.animate-spin')).not.toBeNull();

    expect(await screen.findByText('API Key (2)')).toBeInTheDocument();
    expect(screen.getByText('Basic Auth (1)')).toBeInTheDocument();
    expect(screen.getByRole('heading', { name: 'Jenkins' })).toBeInTheDocument();
    expect(document.querySelector('.custom')).not.toBeNull();
    expect(api.calls[0].url).toBe('/credentials');
  });

  it('filters by type client-side and by provider server-side', async () => {
    const user = userEvent.setup();
    api.on('GET', '/credentials', { data: { credentials: CREDS } });
    api.on('GET', '/credentials?provider=stripe', { data: { credentials: [CREDS[1]] } });
    render(<CredentialList />);
    await screen.findByText('API Key (2)');

    await user.selectOptions(screen.getByRole('combobox'), 'basic_auth');
    await waitFor(() => expect(screen.queryByText('API Key (2)')).not.toBeInTheDocument());
    expect(screen.getByText('Basic Auth (1)')).toBeInTheDocument();

    await user.selectOptions(screen.getByRole('combobox'), '');
    await screen.findByText('API Key (2)');
    await user.type(screen.getByPlaceholderText('Filter by provider...'), 'stripe');
    expect(await screen.findByText('API Key (1)')).toBeInTheDocument();
    expect(screen.queryByRole('heading', { name: 'OpenAI key' })).not.toBeInTheDocument();
    expect(api.last()!.url).toBe('/credentials?provider=stripe');
  });

  it('shows the empty state when the API returns no credentials (or a null list)', async () => {
    const user = userEvent.setup();
    api.on('GET', '/credentials', { data: { credentials: null } });
    render(<CredentialList />);
    expect(await screen.findByText('No credentials yet')).toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: 'Create Your First Credential' }));
    expect(screen.getByRole('heading', { name: 'Create New Credential' })).toBeInTheDocument();
  });

  it('shows load errors and retries', async () => {
    const user = userEvent.setup();
    api.once('GET', '/credentials', { status: 500, data: { code: 'INTERNAL', message: 'db down' } });
    api.on('GET', '/credentials', { data: { credentials: CREDS } });
    render(<CredentialList />);
    expect(await screen.findByText('db down')).toBeInTheDocument();
    expect(screen.queryByText('No credentials yet')).not.toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: 'Retry' }));
    expect(await screen.findByText('API Key (2)')).toBeInTheDocument();
    expect(screen.queryByText('db down')).not.toBeInTheDocument();
  });

  it.each([
    ['API Key', '/credentials/api-key', { 'sk-...': 'sk-x' }],
  ])('creates a %s credential via POST %s and reloads', async (typeLabel, url, inputs) => {
    const user = userEvent.setup();
    api.on('GET', '/credentials', { data: { credentials: [] } });
    api.on('POST', url, { status: 201, data: credential() });
    render(<CredentialList />);
    await screen.findByText('No credentials yet');

    await user.click(openCreate());
    await user.click(screen.getByRole('button', { name: new RegExp(`^${typeLabel}`) }));
    await user.type(screen.getByPlaceholderText('e.g., OpenAI API Key'), 'New');
    for (const [placeholder, value] of Object.entries(inputs)) {
      await user.type(screen.getByPlaceholderText(placeholder), value);
    }
    const before = api.calls.filter((c) => c.method === 'GET').length;
    await user.click(submitCreate());

    await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument());
    const post = api.calls.find((c) => c.method === 'POST')!;
    expect(post.url).toBe(url);
    expect(post.data).toEqual({ name: 'New', description: '', provider: '', api_key: 'sk-x' });
    expect(api.calls.filter((c) => c.method === 'GET').length).toBe(before + 1);
  });

  it('routes every credential type to its create endpoint', async () => {
    const user = userEvent.setup();
    api.on('GET', '/credentials', { data: { credentials: [] } });
    api.on('POST', /^\/credentials\//, { status: 201, data: credential() });
    render(<CredentialList />);
    await screen.findByText('No credentials yet');

    const flows: [string, string, (u: typeof user) => Promise<void>][] = [
      ['Basic Auth', '/credentials/basic-auth', async (u) => {
        const [username, password] = screen.getAllByRole('dialog')[0].querySelectorAll('input:not([placeholder])');
        await u.type(username as HTMLElement, 'u');
        await u.type(password as HTMLElement, 'p');
      }],
      ['OAuth2', '/credentials/oauth2', async (u) => {
        const [id, secret] = screen.getAllByRole('dialog')[0].querySelectorAll('input:not([placeholder])');
        await u.type(id as HTMLElement, 'i');
        await u.type(secret as HTMLElement, 's');
      }],
      ['Service Account', '/credentials/service-account', async (u) => {
        await u.type(screen.getByPlaceholderText('{"type": "service_account", ...}'), 'k');
      }],
      ['Custom', '/credentials/custom', async (u) => {
        await u.type(screen.getByPlaceholderText('Key'), 'k');
        await u.type(screen.getByPlaceholderText('Value (secret)'), 'v');
      }],
    ];

    for (const [label, url, fill] of flows) {
      await user.click(openCreate());
      await user.click(screen.getByRole('button', { name: new RegExp(`^${label}`) }));
      await user.type(screen.getByPlaceholderText('e.g., OpenAI API Key'), 'N');
      await fill(user);
      await user.click(submitCreate());
      await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument());
      expect(api.calls.filter((c) => c.method === 'POST').pop()!.url).toBe(url);
    }
  });

  it('keeps the create modal open when the API rejects the credential', async () => {
    const user = userEvent.setup();
    const err = vi.spyOn(console, 'error').mockImplementation(() => {});
    api.on('GET', '/credentials', { data: { credentials: [] } });
    api.on('POST', '/credentials/api-key', { status: 400, data: { message: 'bad' } });
    render(<CredentialList />);
    await screen.findByText('No credentials yet');
    await user.click(openCreate());
    await user.click(screen.getByRole('button', { name: /^API Key/ }));
    await user.type(screen.getByPlaceholderText('e.g., OpenAI API Key'), 'N');
    await user.type(screen.getByPlaceholderText('sk-...'), 'k');
    await user.click(submitCreate());
    await waitFor(() => expect(err).toHaveBeenCalledWith('Failed to create credential:', expect.anything()));
    expect(screen.getByRole('dialog')).toBeInTheDocument();
  });

  it('edits a credential via PUT /credentials/:id', async () => {
    const user = userEvent.setup();
    api.on('GET', '/credentials', { data: { credentials: CREDS } });
    api.on('PUT', '/credentials/c3', { data: credential({ id: 'c3' }) });
    render(<CredentialList />);
    await screen.findByText('Basic Auth (1)');

    await user.click(within(cardFor('Jenkins')).getByRole('button', { name: 'Edit' }));
    const name = screen.getByDisplayValue('Jenkins');
    await user.clear(name);
    await user.type(name, 'Jenkins CI');
    await user.click(screen.getByRole('button', { name: 'Save' }));

    await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument());
    const put = api.calls.find((c) => c.method === 'PUT')!;
    expect(put.data).toEqual({ name: 'Jenkins CI', description: 'Production key' });
  });

  it('keeps the edit modal open on update failure and closes it on cancel', async () => {
    const user = userEvent.setup();
    const err = vi.spyOn(console, 'error').mockImplementation(() => {});
    api.on('GET', '/credentials', { data: { credentials: CREDS } });
    api.on('PUT', '/credentials/c1', { status: 403, data: { error: 'access denied' } });
    render(<CredentialList />);
    await screen.findByText('API Key (2)');
    await user.click(within(cardFor('OpenAI key')).getByRole('button', { name: 'Edit' }));
    await user.click(screen.getByRole('button', { name: 'Save' }));
    await waitFor(() => expect(err).toHaveBeenCalledWith('Failed to update credential:', expect.anything()));
    expect(screen.getByRole('dialog')).toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: 'Cancel' }));
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
  });

  it('opens and closes the secrets viewer', async () => {
    const user = userEvent.setup();
    api.on('GET', '/credentials', { data: { credentials: CREDS } });
    api.on('GET', '/credentials/c2/secrets', { data: { data: { api_key: 'sk-live-123456789' } } });
    render(<CredentialList />);
    await screen.findByText('API Key (2)');
    await user.click(within(cardFor('Stripe key')).getByRole('button', { name: 'View Secrets' }));
    expect(await screen.findByText('sk-l••••••••6789')).toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: 'Close modal' }));
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
  });

  it('confirms and deletes a credential, then reloads', async () => {
    const user = userEvent.setup();
    let list = CREDS;
    api.on('GET', '/credentials', () => ({ data: { credentials: list } }));
    api.on('DELETE', '/credentials/c1', () => {
      list = CREDS.slice(1);
      return { data: { message: 'credential deleted successfully' } };
    });
    render(<CredentialList />);
    await screen.findByText('API Key (2)');

    await user.click(within(cardFor('OpenAI key')).getByRole('button', { name: 'Delete' }));
    expect(screen.getByText('Are you sure you want to delete "OpenAI key"? This action cannot be undone.')).toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: 'Cancel' }));
    expect(screen.queryByText(/Are you sure/)).not.toBeInTheDocument();

    await user.click(within(cardFor('OpenAI key')).getByRole('button', { name: 'Delete' }));
    const confirm = screen.getByText('Delete Credential').parentElement!;
    await user.click(within(confirm).getByRole('button', { name: 'Delete' }));
    expect(await screen.findByText('API Key (1)')).toBeInTheDocument();
    expect(screen.queryByText(/Are you sure/)).not.toBeInTheDocument();
    expect(screen.queryByRole('heading', { name: 'OpenAI key' })).not.toBeInTheDocument();
  });

  it('keeps the delete confirmation open when deletion fails', async () => {
    const user = userEvent.setup();
    const err = vi.spyOn(console, 'error').mockImplementation(() => {});
    api.on('GET', '/credentials', { data: { credentials: CREDS } });
    api.on('DELETE', '/credentials/c1', { status: 500, data: { error: 'boom' } });
    render(<CredentialList />);
    await screen.findByText('API Key (2)');
    await user.click(within(cardFor('OpenAI key')).getByRole('button', { name: 'Delete' }));
    const confirm = screen.getByText('Delete Credential').parentElement!;
    await user.click(within(confirm).getByRole('button', { name: 'Delete' }));
    await waitFor(() => expect(err).toHaveBeenCalledWith('Failed to delete credential:', expect.anything()));
    expect(screen.getByText('Delete Credential')).toBeInTheDocument();
  });
});
