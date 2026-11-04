import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { EditRentalKeyModal } from './EditRentalKeyModal';
import { mockApi, type ApiMock } from '@/test/apiMock';
import { adminRentalKey } from '@/test/adminFixtures';
import { useUIStore } from '@/store/uiStore';

const field = (name: string) => document.querySelector(`[name="${name}"]`) as HTMLInputElement;

describe('EditRentalKeyModal', () => {
  let api: ApiMock;

  beforeEach(() => {
    useUIStore.setState({ language: 'en' });
    api = mockApi().on('PUT', '/admin/rental-keys/k1', ({ data }) => ({ data: adminRentalKey('k1', data as object) }));
  });

  afterEach(() => api.restore());

  const renderModal = (key = adminRentalKey('k1', { description: 'old', daily_request_limit: 10, monthly_token_limit: 1000 })) => {
    const onClose = vi.fn();
    const onUpdated = vi.fn();
    render(<EditRentalKeyModal isOpen onClose={onClose} onUpdated={onUpdated} rentalKey={key} />);
    return { onClose, onUpdated };
  };

  it('prefills the form from the key', () => {
    renderModal();
    expect(field('name')).toHaveValue('Key k1');
    expect(field('description')).toHaveValue('old');
    expect(field('status')).toHaveValue('active');
    expect(field('daily_request_limit')).toHaveValue(10);
    expect(field('monthly_token_limit')).toHaveValue(1000);
  });

  it('sends only the changed fields', async () => {
    const { onClose, onUpdated } = renderModal();
    await userEvent.clear(field('description'));
    await userEvent.type(field('description'), 'new');
    await userEvent.selectOptions(field('status'), 'suspended');
    await userEvent.clear(field('daily_request_limit'));
    await userEvent.type(field('daily_request_limit'), '25');
    await userEvent.clear(field('monthly_token_limit'));
    await userEvent.type(field('monthly_token_limit'), '2000');

    await userEvent.click(screen.getByRole('button', { name: 'Save' }));
    await waitFor(() => expect(onUpdated).toHaveBeenCalled());
    expect(onClose).toHaveBeenCalled();
    expect(api.last()).toMatchObject({
      method: 'PUT',
      url: '/admin/rental-keys/k1',
      data: { description: 'new', status: 'suspended', daily_request_limit: 25, monthly_token_limit: 2000 },
    });
  });

  it('sends an empty update when nothing changed (key without description or limits)', async () => {
    const { onUpdated } = renderModal(adminRentalKey('k1'));
    expect(field('description')).toHaveValue('');
    expect(field('daily_request_limit')).toHaveValue(null);
    await userEvent.click(screen.getByRole('button', { name: 'Save' }));
    await waitFor(() => expect(onUpdated).toHaveBeenCalled());
    expect(api.last()!.data).toEqual({});
  });

  it('shows the API error', async () => {
    api.once('PUT', '/admin/rental-keys/k1', { status: 404, data: { message: 'rental key not found' } });
    const { onClose, onUpdated } = renderModal();
    await userEvent.type(field('name'), '!');
    await userEvent.click(screen.getByRole('button', { name: 'Save' }));
    expect(await screen.findByText('rental key not found')).toBeInTheDocument();
    expect(api.last()!.data).toEqual({ name: 'Key k1!' });
    expect(onUpdated).not.toHaveBeenCalled();
    expect(onClose).not.toHaveBeenCalled();
  });

  it('closes from Cancel', async () => {
    const { onClose } = renderModal();
    await userEvent.click(screen.getByRole('button', { name: 'Cancel' }));
    expect(onClose).toHaveBeenCalled();
  });
});
