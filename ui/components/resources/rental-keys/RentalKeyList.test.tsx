import { afterEach, beforeEach, describe, expect, it } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { RentalKeyList } from './RentalKeyList';
import { useUIStore } from '@/store/uiStore';
import { mockApi, type ApiMock } from '@/test/apiMock';
import { rentalKey, usageRecord } from '@/test/resourcesFixtures';

const KEYS = [
  rentalKey({ id: 'k1', name: 'GPT key', provider: 'openai' }),
  rentalKey({ id: 'k2', name: 'Claude key', provider: 'anthropic' }),
];

describe('RentalKeyList', () => {
  let api: ApiMock;

  beforeEach(() => {
    useUIStore.setState({ language: 'en' });
    api = mockApi();
  });
  afterEach(() => api.restore());

  it('shows a spinner, then the keys from GET /rental-keys', async () => {
    api.on('GET', '/rental-keys', { data: { rental_keys: KEYS } });
    const { container } = render(<RentalKeyList className="extra" />);
    expect(container.querySelector('.animate-spin')).not.toBeNull();
    expect(await screen.findByRole('heading', { name: 'GPT key' })).toBeInTheDocument();
    expect(screen.getByRole('heading', { name: 'Claude key' })).toBeInTheDocument();
    expect(screen.getByText('Rental Keys')).toBeInTheDocument();
    expect(container.firstElementChild).toHaveClass('extra');
  });

  it('filters by provider through the query string and shows a provider-specific empty state', async () => {
    const user = userEvent.setup();
    api.on('GET', '/rental-keys', { data: { rental_keys: KEYS } });
    api.on('GET', '/rental-keys?provider=google_ai', { data: { rental_keys: [] } });
    api.on('GET', '/rental-keys?provider=anthropic', { data: { rental_keys: [KEYS[1]] } });
    render(<RentalKeyList />);
    await screen.findByRole('heading', { name: 'GPT key' });

    const select = screen.getByRole('combobox');
    expect(Array.from((select as HTMLSelectElement).options).map((o) => o.text)).toEqual([
      'All Providers',
      'OpenAI',
      'Anthropic',
      'Google AI',
    ]);

    await user.selectOptions(select, 'anthropic');
    await waitFor(() => expect(screen.queryByRole('heading', { name: 'GPT key' })).not.toBeInTheDocument());
    expect(screen.getByRole('heading', { name: 'Claude key' })).toBeInTheDocument();

    await user.selectOptions(select, 'google_ai');
    expect(await screen.findByText('No rental keys found for this provider.')).toBeInTheDocument();
    expect(screen.getByText('No Rental Keys')).toBeInTheDocument();
    expect(api.calls.map((c) => c.url)).toEqual([
      '/rental-keys',
      '/rental-keys?provider=anthropic',
      '/rental-keys?provider=google_ai',
    ]);
  });

  it('shows the generic empty state when the user has no keys (null list)', async () => {
    api.on('GET', '/rental-keys', { data: { rental_keys: null } });
    render(<RentalKeyList />);
    expect(await screen.findByText(/You don't have any rental keys assigned yet/)).toBeInTheDocument();
  });

  it('shows errors and recovers on refresh', async () => {
    const user = userEvent.setup();
    api.once('GET', '/rental-keys', { status: 500, data: { error: 'failed to list rental keys' } });
    api.on('GET', '/rental-keys', { data: { rental_keys: KEYS } });
    render(<RentalKeyList />);
    expect(await screen.findByText('failed to list rental keys')).toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: 'Refresh' }));
    expect(await screen.findByRole('heading', { name: 'GPT key' })).toBeInTheDocument();
    expect(screen.queryByText('failed to list rental keys')).not.toBeInTheDocument();
  });

  it('opens and closes the usage modal for a key', async () => {
    const user = userEvent.setup();
    api.on('GET', '/rental-keys', { data: { rental_keys: KEYS } });
    api.on('GET', /^\/rental-keys\/k2\/usage/, { data: { usage: [usageRecord({ model: 'claude-sonnet' })] } });
    render(<RentalKeyList />);
    await screen.findByRole('heading', { name: 'Claude key' });

    const buttons = screen.getAllByRole('button', { name: 'View Usage' });
    await user.click(buttons[1]);
    expect(await screen.findByText('claude-sonnet')).toBeInTheDocument();
    expect(screen.getByText('Claude key - Anthropic')).toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: 'Close modal' }));
    expect(screen.queryByText('Usage History')).not.toBeInTheDocument();
  });
});
