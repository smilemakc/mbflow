import { beforeEach, describe, expect, it, vi } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { EditCredentialModal } from './EditCredentialModal';
import { useUIStore } from '@/store/uiStore';
import { credential } from '@/test/resourcesFixtures';
import type { CredentialType } from '@/services/credentialsService';

describe('EditCredentialModal', () => {
  beforeEach(() => useUIStore.setState({ language: 'en' }));

  const nameInput = () => screen.getByPlaceholderText('e.g., OpenAI API Key');
  const descInput = () => screen.getByPlaceholderText('Optional description...');

  it('prefills the current name/description and shows type and provider', () => {
    render(<EditCredentialModal isOpen credential={credential()} onClose={vi.fn()} onSubmit={vi.fn()} />);
    expect(screen.getByRole('heading', { name: 'Edit Credential' })).toBeInTheDocument();
    expect(nameInput()).toHaveValue('OpenAI key');
    expect(descInput()).toHaveValue('Production key');
    expect(screen.getByText('API Key')).toBeInTheDocument();
    expect(screen.getByText('openai')).toBeInTheDocument();
  });

  it.each<[CredentialType, string]>([
    ['basic_auth', 'Basic Auth'],
    ['oauth2', 'OAuth2'],
    ['service_account', 'Service Account'],
    ['custom', 'Custom'],
    ['other' as CredentialType, 'other'],
  ])('renders the %s type label without provider', (type, label) => {
    render(
      <EditCredentialModal
        isOpen
        credential={credential({ credential_type: type, provider: undefined, description: undefined as never })}
        onClose={vi.fn()}
        onSubmit={vi.fn()}
      />,
    );
    expect(screen.getByText(label)).toBeInTheDocument();
    expect(screen.queryByText('openai')).not.toBeInTheDocument();
    expect(descInput()).toHaveValue('');
  });

  it('disables save for a blank name', async () => {
    const user = userEvent.setup();
    render(<EditCredentialModal isOpen credential={credential()} onClose={vi.fn()} onSubmit={vi.fn()} />);
    await user.clear(nameInput());
    await user.type(nameInput(), '  ');
    expect(screen.getByRole('button', { name: 'Save' })).toBeDisabled();
  });

  it('submits trimmed values (blank description -> undefined) and closes on success', async () => {
    const user = userEvent.setup();
    const onClose = vi.fn();
    let resolve!: (v: boolean) => void;
    const onSubmit = vi.fn(() => new Promise<boolean>((r) => (resolve = r)));
    render(<EditCredentialModal isOpen credential={credential()} onClose={onClose} onSubmit={onSubmit} />);

    await user.clear(nameInput());
    await user.type(nameInput(), '  Renamed  ');
    await user.clear(descInput());
    await user.type(descInput(), '   ');
    await user.click(screen.getByRole('button', { name: 'Save' }));
    expect(onSubmit).toHaveBeenCalledWith('cred-1', 'Renamed', undefined);

    await user.click(screen.getByRole('button', { name: 'Cancel' }));
    expect(onClose).not.toHaveBeenCalled();

    resolve(true);
    await waitFor(() => expect(onClose).toHaveBeenCalledTimes(1));
  });

  it('stays open when the update fails and closes on cancel', async () => {
    const user = userEvent.setup();
    const onClose = vi.fn();
    const onSubmit = vi.fn().mockResolvedValue(false);
    render(<EditCredentialModal isOpen credential={credential()} onClose={onClose} onSubmit={onSubmit} />);
    await user.click(screen.getByRole('button', { name: 'Save' }));
    expect(onSubmit).toHaveBeenCalledWith('cred-1', 'OpenAI key', 'Production key');
    await waitFor(() => expect(screen.getByRole('button', { name: 'Save' })).toBeEnabled());
    expect(onClose).not.toHaveBeenCalled();

    await user.click(screen.getByRole('button', { name: 'Cancel' }));
    expect(onClose).toHaveBeenCalledTimes(1);
  });
});
