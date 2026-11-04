import { beforeEach, describe, expect, it, vi } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { CreateCredentialModal } from './CreateCredentialModal';
import { useUIStore } from '@/store/uiStore';

const setup = (onSubmit = vi.fn().mockResolvedValue(true)) => {
  const onClose = vi.fn();
  render(<CreateCredentialModal isOpen onClose={onClose} onSubmit={onSubmit} />);
  return { onClose, onSubmit, user: userEvent.setup() };
};

const labelled = (label: string) => {
  const lbl = screen.getByText(label, { selector: 'label, label *' }).closest('label')!;
  return lbl.parentElement!.querySelector('input, textarea, select') as HTMLInputElement;
};

const typeButton = (label: string) => screen.getByRole('button', { name: new RegExp(`^${label}`) });
const createBtn = () => screen.getByRole('button', { name: 'Create Credential' });

describe('CreateCredentialModal', () => {
  beforeEach(() => useUIStore.setState({ language: 'en' }));

  it('starts on the type selector and lists all five credential types', () => {
    setup();
    expect(screen.getByRole('heading', { name: 'Create New Credential' })).toBeInTheDocument();
    for (const label of ['API Key', 'Basic Auth', 'OAuth2', 'Service Account', 'Custom']) {
      expect(typeButton(label)).toBeInTheDocument();
    }
    expect(screen.queryByRole('button', { name: 'Create Credential' })).not.toBeInTheDocument();
  });

  it('creates an API key credential with the request body the backend expects', async () => {
    const { user, onSubmit, onClose } = setup();
    await user.click(typeButton('API Key'));
    expect(screen.getByRole('heading', { name: 'Create New Credential - API Key' })).toBeInTheDocument();
    expect(createBtn()).toBeDisabled();

    await user.type(screen.getByPlaceholderText('e.g., OpenAI API Key'), ' My key ');
    expect(createBtn()).toBeDisabled();
    await user.type(screen.getByPlaceholderText('sk-...'), 'sk-secret');
    await user.type(screen.getByPlaceholderText('Optional description...'), 'desc');
    await user.selectOptions(screen.getByRole('combobox'), 'openai');
    expect(createBtn()).toBeEnabled();

    await user.click(createBtn());
    expect(onSubmit).toHaveBeenCalledWith('api_key', {
      name: 'My key',
      description: 'desc',
      provider: 'openai',
      api_key: 'sk-secret',
    });
    await waitFor(() => expect(onClose).toHaveBeenCalledTimes(1));
    // Form is reset back to the type selector.
    expect(typeButton('API Key')).toBeInTheDocument();
  });

  it('creates basic auth credentials', async () => {
    const { user, onSubmit } = setup();
    await user.click(typeButton('Basic Auth'));
    await user.type(screen.getByPlaceholderText('e.g., OpenAI API Key'), 'Svc');
    await user.type(labelled('Username'), 'bob');
    expect(createBtn()).toBeDisabled();
    await user.type(labelled('Password'), 'pw');
    expect(labelled('Password')).toHaveAttribute('type', 'password');
    await user.click(createBtn());
    expect(onSubmit).toHaveBeenCalledWith('basic_auth', {
      name: 'Svc',
      description: '',
      provider: '',
      username: 'bob',
      password: 'pw',
    });
  });

  it('creates OAuth2 credentials, omitting empty optional fields', async () => {
    const { user, onSubmit } = setup();
    await user.click(typeButton('OAuth2'));
    await user.type(screen.getByPlaceholderText('e.g., OpenAI API Key'), 'Google');
    await user.type(labelled('Client ID'), 'cid');
    expect(createBtn()).toBeDisabled();
    await user.type(labelled('Client Secret'), 'csecret');
    await user.type(labelled('Access Token'), 'at');
    await user.type(labelled('Refresh Token'), 'rt');
    await user.type(screen.getByPlaceholderText('https://oauth2.example.com/token'), 'https://t');
    await user.type(screen.getByPlaceholderText('read write'), 'read');
    await user.click(createBtn());
    expect(onSubmit).toHaveBeenCalledWith('oauth2', {
      name: 'Google',
      description: '',
      provider: '',
      client_id: 'cid',
      client_secret: 'csecret',
      access_token: 'at',
      refresh_token: 'rt',
      token_url: 'https://t',
      scopes: 'read',
    });
  });

  it('creates service account credentials from a JSON key', async () => {
    const { user, onSubmit } = setup();
    await user.click(typeButton('Service Account'));
    await user.type(screen.getByPlaceholderText('e.g., OpenAI API Key'), 'GCP');
    const json = screen.getByPlaceholderText('{"type": "service_account", ...}');
    await user.click(json);
    await user.paste('{"type":"service_account"}');
    await user.click(createBtn());
    expect(onSubmit).toHaveBeenCalledWith('service_account', {
      name: 'GCP',
      description: '',
      provider: '',
      json_key: '{"type":"service_account"}',
    });
  });

  it('creates custom credentials from key/value rows and resets them afterwards', async () => {
    const { user, onSubmit } = setup();
    await user.click(typeButton('Custom'));
    await user.type(screen.getByPlaceholderText('e.g., OpenAI API Key'), 'Custom one');
    expect(screen.queryByRole('button', { name: /Remove field/ })).not.toBeInTheDocument();

    await user.type(screen.getByPlaceholderText('Key'), ' token ');
    expect(createBtn()).toBeDisabled();
    await user.type(screen.getByPlaceholderText('Value (secret)'), 'v1');
    expect(createBtn()).toBeEnabled();

    await user.click(screen.getByRole('button', { name: '+ Add Field' }));
    const keys = screen.getAllByPlaceholderText('Key');
    expect(keys).toHaveLength(2);
    await user.type(keys[1], 'ignored-without-value');
    await user.click(screen.getByRole('button', { name: '+ Add Field' }));
    await user.type(screen.getAllByPlaceholderText('Key')[2], 'gone');
    await user.click(screen.getByRole('button', { name: 'Remove field 3' }));
    expect(screen.getAllByPlaceholderText('Key')).toHaveLength(2);

    await user.click(createBtn());
    expect(onSubmit).toHaveBeenCalledWith('custom', {
      name: 'Custom one',
      description: '',
      provider: '',
      data: { token: 'v1' },
    });

    // Regression: the custom rows used to be mutated in place, so the secret leaked into the next form.
    await waitFor(() => expect(typeButton('Custom')).toBeInTheDocument());
    await user.click(typeButton('Custom'));
    expect(screen.getAllByPlaceholderText('Key')).toHaveLength(1);
    expect(screen.getByPlaceholderText('Key')).toHaveValue('');
    expect(screen.getByPlaceholderText('Value (secret)')).toHaveValue('');
  });

  it('keeps the form when creation fails, and Back returns to the type selector keeping input', async () => {
    const { user, onSubmit, onClose } = setup(vi.fn().mockResolvedValue(false));
    await user.click(typeButton('API Key'));
    await user.type(screen.getByPlaceholderText('e.g., OpenAI API Key'), 'K');
    await user.type(screen.getByPlaceholderText('sk-...'), 'x');
    await user.click(createBtn());
    await waitFor(() => expect(createBtn()).toBeEnabled());
    expect(onSubmit).toHaveBeenCalledTimes(1);
    expect(onClose).not.toHaveBeenCalled();

    await user.click(screen.getByRole('button', { name: 'Back' }));
    expect(screen.getByRole('heading', { name: 'Create New Credential' })).toBeInTheDocument();
    await user.click(typeButton('API Key'));
    expect(screen.getByPlaceholderText('e.g., OpenAI API Key')).toHaveValue('K');
  });

  it('ignores close while submitting, then cancel resets everything', async () => {
    let resolve!: (v: boolean) => void;
    const { user, onClose } = setup(vi.fn(() => new Promise<boolean>((r) => (resolve = r))));
    await user.click(typeButton('API Key'));
    await user.type(screen.getByPlaceholderText('e.g., OpenAI API Key'), 'K');
    await user.type(screen.getByPlaceholderText('sk-...'), 'x');
    await user.click(createBtn());
    expect(screen.getByRole('button', { name: 'Cancel' })).toBeDisabled();
    await user.keyboard('{Escape}');
    expect(onClose).not.toHaveBeenCalled();

    resolve(false);
    await waitFor(() => expect(screen.getByRole('button', { name: 'Cancel' })).toBeEnabled());
    await user.click(screen.getByRole('button', { name: 'Cancel' }));
    expect(onClose).toHaveBeenCalledTimes(1);
    expect(typeButton('API Key')).toBeInTheDocument();
    await user.click(typeButton('API Key'));
    expect(screen.getByPlaceholderText('e.g., OpenAI API Key')).toHaveValue('');
  });
});
