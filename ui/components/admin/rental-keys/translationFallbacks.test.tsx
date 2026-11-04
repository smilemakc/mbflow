/**
 * The rental key admin screens fall back to English text for any translation
 * key that is missing from the active dictionary.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { render, screen, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { mockApi, type ApiMock } from '@/test/apiMock';
import { ADMIN_USERS, adminRentalKey } from '@/test/adminFixtures';

vi.mock('@/store/translations', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/store/translations')>();
  return { ...actual, useTranslation: () => ({}) };
});

const { RentalKeyAdminList } = await import('./RentalKeyAdminList');

describe('rental key admin without translations', () => {
  let api: ApiMock;

  beforeEach(() => {
    api = mockApi()
      .on('GET', '/admin/users', { data: { data: ADMIN_USERS, total: 3 } })
      .on('GET', /^\/admin\/rental-keys/, { data: { rental_keys: [adminRentalKey('k1', { name: 'Prod' })], total: 1 } });
  });

  afterEach(() => api.restore());

  it('shows English fallbacks across the list and its modals', async () => {
    render(<RentalKeyAdminList />);
    expect(await screen.findByText('Prod')).toBeInTheDocument();
    expect(screen.getByText('Rental Keys Management')).toBeInTheDocument();
    ['Name', 'Owner', 'Provider', 'Status', 'Usage', 'Created'].forEach((header) =>
      expect(screen.getByRole('columnheader', { name: header })).toBeInTheDocument(),
    );
    expect(screen.getByRole('option', { name: 'All Providers' })).toBeInTheDocument();
    ['All Statuses', 'Active', 'Suspended', 'Deleted'].forEach((name) =>
      expect(screen.getByRole('option', { name })).toBeInTheDocument(),
    );
    expect(screen.getByText('All Owners')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /Refresh/ })).toBeInTheDocument();

    await userEvent.click(screen.getByRole('button', { name: /Create Key/ }));
    expect(screen.getByRole('heading', { name: 'Create Rental Key' })).toBeInTheDocument();
    for (const label of ['Owner *', 'Name *', 'Description', 'Provider *', 'API Key *', 'Daily Request Limit', 'Monthly Token Limit', 'Provisioner Type']) {
      expect(screen.getAllByText(label).length).toBeGreaterThan(0);
    }
    expect(screen.getByText('This key will be encrypted and never shown again.')).toBeInTheDocument();
    await userEvent.click(screen.getByText('Select user...').closest('button')!);
    expect(screen.getByPlaceholderText('Search users...')).toBeInTheDocument();
    await userEvent.type(screen.getByPlaceholderText('Search users...'), 'zzz');
    expect(screen.getAllByText('No users found').length).toBeGreaterThan(0);
    await userEvent.type(screen.getByPlaceholderText('Rental key name'), 'k');
    await userEvent.type(screen.getByPlaceholderText('The actual API key (will be encrypted)'), 'sk');
    await userEvent.click(screen.getByRole('button', { name: 'Create' }));
    expect(screen.getByText('Owner: Select user...')).toBeInTheDocument();
    await userEvent.click(screen.getByRole('button', { name: 'Cancel' }));

    const row = screen.getByText('Prod').closest('tr')!;
    await userEvent.click(within(row).getByTitle('Edit'));
    expect(screen.getByRole('heading', { name: 'Edit Rental Key' })).toBeInTheDocument();
    expect(screen.getAllByRole('option', { name: 'Suspended' })).toHaveLength(2);
    expect(screen.getByRole('button', { name: 'Save' })).toBeInTheDocument();
    await userEvent.click(screen.getByRole('button', { name: 'Cancel' }));

    await userEvent.click(within(row).getByTitle('Rotate API Key'));
    expect(screen.getByRole('heading', { name: 'Rotate API Key' })).toBeInTheDocument();
    expect(screen.getByText('Warning')).toBeInTheDocument();
    expect(screen.getByText('Rotating key for:')).toBeInTheDocument();
    expect(screen.getByText('New API Key *')).toBeInTheDocument();
    await userEvent.click(screen.getByRole('button', { name: 'Cancel' }));

    await userEvent.click(within(row).getByTitle('Delete'));
    expect(screen.getByText('Delete Rental Key')).toBeInTheDocument();
    expect(screen.getByText('Are you sure you want to delete "Prod"?')).toBeInTheDocument();
  });

  it('shows the English empty state', async () => {
    api.once('GET', /^\/admin\/rental-keys/, { data: { rental_keys: [], total: 0 } });
    render(<RentalKeyAdminList />);
    expect(await screen.findByText('No rental keys found')).toBeInTheDocument();
    expect(screen.getByText('Create your first rental key to get started')).toBeInTheDocument();
  });
});
