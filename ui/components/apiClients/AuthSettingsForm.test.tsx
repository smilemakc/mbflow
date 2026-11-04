import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter } from 'react-router-dom';
import React, { useState } from 'react';
import { AuthSettingsForm, authFromScheme } from './AuthSettingsForm';
import { mockApi, type ApiMock } from '@/test/apiMock';
import { credential } from '@/test/adminFixtures';
import { useUIStore } from '@/store/uiStore';
import type { ApiClientAuth, ApiSecurityScheme } from '@/services/apiClientService';

const CREDENTIALS = [
  credential('c-key', 'api_key', 'Stripe key'),
  credential('c-basic', 'basic_auth', 'Admin login'),
  credential('c-oauth', 'oauth2', 'Google'),
  credential('c-custom', 'custom', 'Anything'),
  credential('c-tg', 'telegram' as never, 'Bot'),
];

const selectFor = (label: string) => screen.getByText(label).parentElement!.querySelector('select')!;
const optionLabels = (select: HTMLSelectElement) => Array.from(select.options).map((o) => o.textContent);

describe('authFromScheme', () => {
  const scheme = (s: Partial<ApiSecurityScheme>): ApiSecurityScheme => ({ name: 'S', type: 'apiKey', ...s });

  it('maps OpenAPI security schemes to auth settings', () => {
    expect(authFromScheme(scheme({ in: 'header', param_name: 'X-Key' }))).toEqual({
      type: 'api_key',
      in: 'header',
      name: 'X-Key',
      scheme_name: 'S',
    });
    expect(authFromScheme(scheme({ in: 'body', param_name: 'x' }))).toBeNull();
    expect(authFromScheme(scheme({ in: 'query' }))).toBeNull();
    expect(authFromScheme(scheme({ type: 'http', scheme: 'basic' }))).toEqual({ type: 'basic', scheme_name: 'S' });
    expect(authFromScheme(scheme({ type: 'HTTP', scheme: 'bearer' }))).toEqual({ type: 'bearer', scheme_name: 'S' });
    expect(authFromScheme(scheme({ type: 'http', scheme: 'digest' }))).toBeNull();
    expect(authFromScheme(scheme({ type: 'oauth2', token_url: 'https://t', scopes: ['a'] }))).toEqual({
      type: 'oauth2_client_credentials',
      token_url: 'https://t',
      scopes: ['a'],
      scheme_name: 'S',
    });
    expect(authFromScheme(scheme({ type: 'oauth2' }))).toEqual({ type: 'bearer', scheme_name: 'S' });
    expect(authFromScheme(scheme({ type: 'openIdConnect' }))).toEqual({ type: 'bearer', scheme_name: 'S' });
    expect(authFromScheme(scheme({ type: 'mutualTLS' }))).toBeNull();
  });
});

describe('AuthSettingsForm', () => {
  let api: ApiMock;

  beforeEach(() => {
    useUIStore.setState({ language: 'en' });
    api = mockApi().on('GET', '/credentials', { data: { credentials: CREDENTIALS } });
  });

  afterEach(() => api.restore());

  const Harness: React.FC<{ initial: ApiClientAuth; schemes?: ApiSecurityScheme[]; spy: (a: ApiClientAuth) => void }> = ({
    initial,
    schemes,
    spy,
  }) => {
    const [auth, setAuth] = useState(initial);
    return (
      <MemoryRouter>
        <AuthSettingsForm
          value={auth}
          securitySchemes={schemes}
          onChange={(a) => {
            setAuth(a);
            spy(a);
          }}
        />
      </MemoryRouter>
    );
  };

  it('hides credentials for "none" and switches to API key auth with defaults', async () => {
    const spy = vi.fn();
    render(<Harness initial={{ type: 'none' }} spy={spy} />);
    expect(screen.queryByText('Credential')).not.toBeInTheDocument();

    await userEvent.selectOptions(selectFor('Type'), 'api_key');
    expect(spy).toHaveBeenLastCalledWith({
      type: 'api_key',
      credential_id: undefined,
      in: 'header',
      name: 'X-API-Key',
      token_url: undefined,
      scopes: undefined,
    });
    await waitFor(() =>
      expect(optionLabels(selectFor('Credential'))).toEqual([
        '— not selected —',
        'Stripe key (api_key)',
        'Google (oauth2)',
        'Anything (custom)',
      ]),
    );

    await userEvent.selectOptions(selectFor('Send the key in'), 'query');
    expect(spy).toHaveBeenLastCalledWith(expect.objectContaining({ in: 'query' }));
    const keyName = screen.getByPlaceholderText('X-API-Key');
    await userEvent.clear(keyName);
    await userEvent.type(keyName, 'k');
    expect(spy).toHaveBeenLastCalledWith(expect.objectContaining({ name: 'k' }));

    await userEvent.selectOptions(selectFor('Credential'), 'c-key');
    expect(spy).toHaveBeenLastCalledWith(expect.objectContaining({ credential_id: 'c-key' }));
    await userEvent.selectOptions(selectFor('Credential'), '');
    expect(spy).toHaveBeenLastCalledWith(expect.objectContaining({ credential_id: undefined }));
  });

  it('keeps a compatible credential when the type changes and drops an incompatible one', async () => {
    const spy = vi.fn();
    render(<Harness initial={{ type: 'bearer', credential_id: 'c-custom' }} spy={spy} />);
    await waitFor(() => expect(selectFor('Credential')).toHaveValue('c-custom'));

    await userEvent.selectOptions(selectFor('Type'), 'basic');
    expect(spy).toHaveBeenLastCalledWith(expect.objectContaining({ type: 'basic', credential_id: 'c-custom' }));
    expect(optionLabels(selectFor('Credential'))).toEqual(['— not selected —', 'Admin login (basic_auth)', 'Anything (custom)']);

    await userEvent.selectOptions(selectFor('Credential'), 'c-basic');
    await userEvent.selectOptions(selectFor('Type'), 'bearer');
    expect(spy).toHaveBeenLastCalledWith(expect.objectContaining({ type: 'bearer', credential_id: undefined }));
  });

  it('edits OAuth2 token URL and scopes', async () => {
    const spy = vi.fn();
    render(<Harness initial={{ type: 'oauth2_client_credentials', scopes: ['read'] }} spy={spy} />);
    expect(screen.getByDisplayValue('read')).toBeInTheDocument();

    await userEvent.type(screen.getByPlaceholderText('https://auth.example.com/oauth/token'), 'https://t');
    expect(spy).toHaveBeenLastCalledWith(expect.objectContaining({ token_url: 'https://t' }));
    await userEvent.clear(screen.getByPlaceholderText('https://auth.example.com/oauth/token'));
    expect(spy).toHaveBeenLastCalledWith(expect.objectContaining({ token_url: undefined }));

    const scopes = screen.getByPlaceholderText('read write');
    await userEvent.type(scopes, ' write');
    expect(spy).toHaveBeenLastCalledWith(expect.objectContaining({ scopes: ['read', 'write'] }));
    await userEvent.clear(scopes);
    expect(spy).toHaveBeenLastCalledWith(expect.objectContaining({ scopes: undefined }));

    await userEvent.selectOptions(selectFor('Type'), 'oauth2_client_credentials');
    expect(spy).toHaveBeenLastCalledWith(expect.objectContaining({ type: 'oauth2_client_credentials', in: undefined }));
  });

  it('shows scopes chosen from a specification scheme', async () => {
    const spy = vi.fn();
    const schemes: ApiSecurityScheme[] = [
      { name: 'OAuth', type: 'oauth2', token_url: 'https://auth/token', scopes: ['admin', 'audit'] },
    ];
    render(<Harness initial={{ type: 'oauth2_client_credentials', scopes: ['read'] }} schemes={schemes} spy={spy} />);
    await userEvent.click(screen.getByRole('button', { name: 'From the specification: OAuth' }));
    expect(screen.getByPlaceholderText('read write')).toHaveValue('admin audit');
    expect(screen.getByPlaceholderText('https://auth.example.com/oauth/token')).toHaveValue('https://auth/token');
  });

  it('offers security schemes from the specification as shortcuts', async () => {
    const spy = vi.fn();
    const schemes: ApiSecurityScheme[] = [
      { name: 'ApiKeyAuth', type: 'apiKey', in: 'header', param_name: 'X-Key', description: 'key header' },
      { name: 'Digest', type: 'http', scheme: 'digest' },
      { name: 'BearerAuth', type: 'http', scheme: 'bearer' },
    ];
    render(<Harness initial={{ type: 'none', credential_id: 'keep' }} schemes={schemes} spy={spy} />);
    expect(screen.queryByText(/Digest/)).not.toBeInTheDocument();

    const apiKeyButton = screen.getByRole('button', { name: 'From the specification: ApiKeyAuth' });
    expect(apiKeyButton).toHaveAttribute('title', 'key header');
    await userEvent.click(apiKeyButton);
    expect(spy).toHaveBeenLastCalledWith({
      type: 'api_key',
      in: 'header',
      name: 'X-Key',
      scheme_name: 'ApiKeyAuth',
      credential_id: 'keep',
    });
    expect(screen.getByRole('button', { name: 'From the specification: ApiKeyAuth' }).className).toContain('border-blue-500');
    expect(screen.getByRole('button', { name: 'From the specification: BearerAuth' }).className).not.toContain('border-blue-500');
  });

  it('links to Resources when no credential fits', async () => {
    api.once('GET', '/credentials', { data: {} });
    render(<Harness initial={{ type: 'basic' }} spy={vi.fn()} />);
    expect(await screen.findByText(/No suitable credentials/)).toBeInTheDocument();
    expect(screen.getByRole('link', { name: /Create credential/ })).toHaveAttribute('href', '/resources');
  });

  it('explains when credentials are unavailable', async () => {
    api.once('GET', '/credentials', { status: 503, data: { message: 'no encryption key' } });
    render(<Harness initial={{ type: 'bearer' }} spy={vi.fn()} />);
    expect(await screen.findByText(/Credentials are unavailable/)).toBeInTheDocument();
    expect(screen.queryByText(/Secrets stay encrypted/)).not.toBeInTheDocument();
  });

  it('ignores credential responses after unmount', async () => {
    let resolve: (v: { data: unknown }) => void = () => {};
    let reject: (v: { status: number }) => void = () => {};
    api.once('GET', '/credentials', () => new Promise((r) => (resolve = r)));
    const first = render(<Harness initial={{ type: 'bearer' }} spy={vi.fn()} />);
    await waitFor(() => expect(api.calls).toHaveLength(1));
    first.unmount();
    resolve({ data: { credentials: CREDENTIALS } });

    api.once('GET', '/credentials', () => new Promise((_, rj) => (reject = (v) => rj(v))));
    const second = render(<Harness initial={{ type: 'bearer' }} spy={vi.fn()} />);
    await waitFor(() => expect(api.calls).toHaveLength(2));
    second.unmount();
    reject({ status: 500 });
    await new Promise((r) => setTimeout(r, 0));
  });
});
