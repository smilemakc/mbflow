import { beforeEach, describe, expect, it, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { CredentialCard } from './CredentialCard';
import { useUIStore } from '@/store/uiStore';
import { credential } from '@/test/resourcesFixtures';
import type { Credential, CredentialType } from '@/services/credentialsService';

const renderCard = (cred: Credential) => {
  const handlers = { onView: vi.fn(), onEdit: vi.fn(), onDelete: vi.fn() };
  const utils = render(<CredentialCard credential={cred} {...handlers} />);
  return { ...utils, ...handlers };
};

describe('CredentialCard', () => {
  beforeEach(() => useUIStore.setState({ language: 'en' }));

  it('renders metadata, provider, field names and stats', () => {
    const lastUsed = '2026-04-01T12:00:00Z';
    renderCard(credential({ fields: ['api_key', 'org_id'], last_used_at: lastUsed }));
    expect(screen.getByRole('heading', { name: 'OpenAI key' })).toBeInTheDocument();
    expect(screen.getByText('API Key')).toBeInTheDocument();
    expect(screen.getByText('Production key')).toBeInTheDocument();
    expect(screen.getByText('openai')).toBeInTheDocument();
    expect(screen.getByText('api_key')).toBeInTheDocument();
    expect(screen.getByText('org_id')).toBeInTheDocument();
    expect(screen.getByText('Usage:').parentElement).toHaveTextContent('Usage: 7');
    expect(screen.getByText('Last used:').parentElement).toHaveTextContent(new Date(lastUsed).toLocaleDateString());
    expect(screen.getByText('active').className).toContain('green');
    expect(screen.queryByText('Expires:')).not.toBeInTheDocument();
  });

  it('tolerates the backend omitting empty fields/description/provider', () => {
    const cred = credential({ description: '', provider: undefined, credential_type: 'custom' });
    delete (cred as Partial<Credential>).fields; // `fields` is omitempty in CredentialResponse
    renderCard(cred);
    expect(screen.getByText('Custom')).toBeInTheDocument();
    expect(screen.queryByText('Provider:')).not.toBeInTheDocument();
    expect(screen.queryByText('Last used:')).not.toBeInTheDocument();
  });

  it('flags expired credentials', () => {
    renderCard(credential({ expires_at: '2000-01-01T00:00:00Z', status: 'active' }));
    expect(screen.getByText('expired').className).toContain('red');
    expect(screen.getByText('Expires:').parentElement).toHaveTextContent('(Expired)');
  });

  it('shows a future expiry without the expired marker', () => {
    renderCard(credential({ expires_at: '2999-01-01T00:00:00Z' }));
    expect(screen.getByText('Expires:').parentElement).not.toHaveTextContent('Expired');
    expect(screen.getByText('active')).toBeInTheDocument();
  });

  it.each<[string, string]>([
    ['suspended', 'yellow'],
    ['deleted', 'red'],
    ['archived', 'slate'],
  ])('colours the %s status badge', (status, colour) => {
    renderCard(credential({ status }));
    expect(screen.getByText(status).className).toContain(colour);
  });

  it.each<[CredentialType, string]>([
    ['basic_auth', 'Basic Auth'],
    ['oauth2', 'OAuth2'],
    ['service_account', 'Service Account'],
    ['weird' as CredentialType, 'weird'],
  ])('labels %s credentials', (type, label) => {
    renderCard(credential({ credential_type: type }));
    expect(screen.getByText(label)).toBeInTheDocument();
  });

  it('fires view/edit/delete callbacks', async () => {
    const user = userEvent.setup();
    const { onView, onEdit, onDelete } = renderCard(credential());
    await user.click(screen.getByRole('button', { name: 'View Secrets' }));
    await user.click(screen.getByRole('button', { name: 'Edit' }));
    await user.click(screen.getByRole('button', { name: 'Delete' }));
    expect(onView).toHaveBeenCalledTimes(1);
    expect(onEdit).toHaveBeenCalledTimes(1);
    expect(onDelete).toHaveBeenCalledTimes(1);
  });
});
