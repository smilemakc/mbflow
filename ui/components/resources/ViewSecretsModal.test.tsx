import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { act, render, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { ViewSecretsModal } from './ViewSecretsModal';
import { useUIStore } from '@/store/uiStore';
import { mockApi, type ApiMock } from '@/test/apiMock';
import { credential } from '@/test/resourcesFixtures';
import type { CredentialType } from '@/services/credentialsService';

const SECRETS = { api_key: 'sk-1234567890abcd', pin: 'short' };

describe('ViewSecretsModal', () => {
  let api: ApiMock;

  beforeEach(() => {
    useUIStore.setState({ language: 'en' });
    api = mockApi();
  });
  afterEach(() => api.restore());

  const fieldBox = (field: string) => screen.getByText(field, { selector: 'label' }).closest('div.border') as HTMLElement;

  it('loads secrets from GET /credentials/:id/secrets and masks them', async () => {
    api.on('GET', '/credentials/cred-1/secrets', { data: { ...credential(), data: SECRETS } });
    const lastUsed = '2026-04-01T12:00:00Z';
    render(<ViewSecretsModal isOpen credential={credential({ last_used_at: lastUsed })} onClose={vi.fn()} />);

    expect(screen.getByText('Sensitive Information')).toBeInTheDocument();
    expect(document.querySelector('.animate-spin')).not.toBeNull();
    expect(screen.getByText('openai')).toBeInTheDocument();
    expect(screen.getByText(new Date(lastUsed).toLocaleString())).toBeInTheDocument();

    expect(await screen.findByText('sk-1••••••••abcd')).toBeInTheDocument();
    expect(within(fieldBox('pin')).getByText('••••••••')).toBeInTheDocument();
    expect(api.calls).toHaveLength(1);
  });

  it('toggles visibility per field', async () => {
    const user = userEvent.setup();
    api.on('GET', '/credentials/cred-1/secrets', { data: { data: SECRETS } });
    render(<ViewSecretsModal isOpen credential={credential()} onClose={vi.fn()} />);
    await screen.findByText('sk-1••••••••abcd');

    await user.click(within(fieldBox('api_key')).getByTitle('Show'));
    expect(screen.getByText('sk-1234567890abcd')).toBeInTheDocument();
    expect(within(fieldBox('pin')).getByText('••••••••')).toBeInTheDocument();

    await user.click(within(fieldBox('api_key')).getByTitle('Hide'));
    expect(screen.queryByText('sk-1234567890abcd')).not.toBeInTheDocument();
  });

  it('copies a secret to the clipboard and briefly shows a check mark', async () => {
    vi.useFakeTimers({ shouldAdvanceTime: true });
    try {
      const user = userEvent.setup({ advanceTimers: vi.advanceTimersByTime });
      const writeText = vi.fn().mockResolvedValue(undefined);
      Object.defineProperty(navigator, 'clipboard', { value: { writeText }, configurable: true });
      api.on('GET', '/credentials/cred-1/secrets', { data: { data: SECRETS } });
      render(<ViewSecretsModal isOpen credential={credential()} onClose={vi.fn()} />);
      await screen.findByText('sk-1••••••••abcd');

      const copy = within(fieldBox('pin')).getByTitle('Copy');
      await user.click(copy);
      expect(writeText).toHaveBeenCalledWith('short');
      await waitFor(() => expect(copy.querySelector('.text-green-500')).not.toBeNull());

      act(() => {
        vi.advanceTimersByTime(2100);
      });
      expect(copy.querySelector('.text-green-500')).toBeNull();
    } finally {
      vi.useRealTimers();
    }
  });

  it('logs and survives a clipboard failure', async () => {
    const user = userEvent.setup();
    const err = vi.spyOn(console, 'error').mockImplementation(() => {});
    Object.defineProperty(navigator, 'clipboard', {
      value: { writeText: vi.fn().mockRejectedValue(new Error('denied')) },
      configurable: true,
    });
    api.on('GET', '/credentials/cred-1/secrets', { data: { data: SECRETS } });
    render(<ViewSecretsModal isOpen credential={credential()} onClose={vi.fn()} />);
    await screen.findByText('sk-1••••••••abcd');

    const copy = within(fieldBox('api_key')).getByTitle('Copy');
    await user.click(copy);
    await waitFor(() => expect(err).toHaveBeenCalledWith('Failed to copy:', expect.any(Error)));
    expect(copy.querySelector('.text-green-500')).toBeNull();
  });

  it('shows the API error (e.g. 410 for expired credentials) and retries', async () => {
    const user = userEvent.setup();
    api.once('GET', '/credentials/cred-1/secrets', { status: 410, data: { error: 'credential has expired' } });
    api.on('GET', '/credentials/cred-1/secrets', { data: { data: { token: 'abcdefghijkl' } } });
    render(<ViewSecretsModal isOpen credential={credential({ provider: undefined })} onClose={vi.fn()} />);

    expect(await screen.findByText('credential has expired')).toBeInTheDocument();
    expect(screen.queryByText('Provider:')).not.toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: 'Retry' }));
    expect(await screen.findByText('abcd••••••••ijkl')).toBeInTheDocument();
    expect(screen.queryByText('credential has expired')).not.toBeInTheDocument();
    expect(api.calls).toHaveLength(2);
  });

  it('does not fetch while closed and calls onClose from the footer', async () => {
    const user = userEvent.setup();
    const onClose = vi.fn();
    api.on('GET', '/credentials/cred-1/secrets', { data: { data: SECRETS } });
    const { rerender } = render(<ViewSecretsModal isOpen={false} credential={credential()} onClose={onClose} />);
    expect(api.calls).toHaveLength(0);

    rerender(<ViewSecretsModal isOpen credential={credential()} onClose={onClose} />);
    await screen.findByText('sk-1••••••••abcd');
    const footerClose = screen.getAllByRole('button', { name: 'Close' }).find((b) => b.textContent === 'Close')!;
    await user.click(footerClose);
    expect(onClose).toHaveBeenCalledTimes(1);
  });

  it.each<CredentialType>(['basic_auth', 'oauth2', 'service_account', 'custom', 'x' as CredentialType])(
    'renders the %s type icon in the title',
    async (type) => {
      api.on('GET', '/credentials/cred-1/secrets', { data: { data: {} } });
      render(<ViewSecretsModal isOpen credential={credential({ credential_type: type, name: 'Typed' })} onClose={vi.fn()} />);
      const heading = screen.getByRole('heading', { name: 'Typed' });
      expect(heading.querySelector('svg')).not.toBeNull();
      await waitFor(() => expect(document.querySelector('.animate-spin')).toBeNull());
    },
  );
});
